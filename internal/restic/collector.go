package restic

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/lefeverd/backup-exporter/internal/collector"
	"github.com/lefeverd/backup-exporter/internal/models"
)

const toolName = "restic"

// Repository is a single configured restic repository. Exactly one of
// Password/PasswordFile/PasswordCommand is expected to be set (enforced by
// config validation).
type Repository struct {
	Name            string
	Repository      string
	Password        string
	PasswordFile    string
	PasswordCommand string
}

// Collector collects metrics from a set of restic repositories.
type Collector struct {
	Repositories         []Repository
	ResticPath           string
	CommandTimeout       time.Duration
	SnapshotHistoryLimit int
	Metrics              *models.ResticMetrics
	ExporterMetrics      *models.ExporterMetrics
	Cache                *models.MetricsCache[*models.ResticMetrics]
	Parser               ParserInterface
	Logger               *slog.Logger

	stats *statsCache
}

// Collect collects the metrics from restic and refreshes them in the cache.
// It collects metrics from every configured restic repository.
// In case of error, it still tries to collect metrics of the remaining repositories.
// This is why it returns a slice of error, which can come from different repositories.
func (c *Collector) Collect(ctx context.Context) []error {
	c.Cache.Lock()
	defer c.Cache.Unlock()

	if c.Cache.Collecting {
		c.Logger.Info("Metrics collection already in progress, skipping", "tool", toolName)
		return nil
	}

	c.Cache.Collecting = true
	defer func() {
		c.Cache.Collecting = false
	}()

	if c.stats == nil {
		c.stats = newStatsCache()
	}

	totalStartTime := time.Now()

	c.Metrics.LastBackupDuration.Reset()
	c.Metrics.LastBackupOriginalSize.Reset()
	c.Metrics.LastBackupFiles.Reset()
	c.Metrics.LastBackupTimestamp.Reset()
	c.Metrics.LastBackupDataAdded.Reset()
	c.Metrics.LastBackupDataAddedPacked.Reset()

	c.Metrics.ArchiveDuration.Reset()
	c.Metrics.ArchiveOriginalSize.Reset()
	c.Metrics.ArchiveFiles.Reset()
	c.Metrics.ArchiveTimestamp.Reset()
	c.Metrics.ArchiveDataAdded.Reset()
	c.Metrics.ArchiveDataAddedPacked.Reset()

	c.Metrics.LastSnapshotInfo.Reset()

	ctx, cancel := context.WithTimeout(ctx, c.CommandTimeout)
	defer cancel()

	var errs []error
	for _, repo := range c.Repositories {
		startTime := time.Now()
		c.Logger.Debug("Collecting metrics", "repository", repo.Name)

		snapshots, err := c.listSnapshots(ctx, repo)
		c.ExporterMetrics.LastCollectDuration.WithLabelValues(toolName, repo.Name).Set(time.Since(startTime).Seconds())
		c.ExporterMetrics.LastCollectTimestamp.WithLabelValues(toolName, repo.Name).Set(float64(time.Now().Unix()))
		c.Logger.Debug("Collecting metrics done", "repository", repo.Name, "duration", time.Since(startTime), "error", err)

		if err != nil {
			c.ExporterMetrics.LastCollectError.WithLabelValues(toolName, repo.Name).Set(1)
			c.ExporterMetrics.CollectErrors.WithLabelValues(toolName, repo.Name).Inc()
			errs = append(errs, err)
			continue
		}

		sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].Time.Before(snapshots[j].Time) })
		if len(snapshots) > c.SnapshotHistoryLimit {
			snapshots = snapshots[len(snapshots)-c.SnapshotHistoryLimit:]
		}

		ids := make([]string, len(snapshots))
		for i, s := range snapshots {
			ids[i] = s.ID
		}
		c.stats.reconcile(repo.Name, ids)

		statsErr := c.collectStats(ctx, repo, snapshots)
		if statsErr != nil {
			c.ExporterMetrics.LastCollectError.WithLabelValues(toolName, repo.Name).Set(1)
			c.ExporterMetrics.CollectErrors.WithLabelValues(toolName, repo.Name).Inc()
			errs = append(errs, statsErr)
			continue
		}

		if len(snapshots) > 0 {
			latest := snapshots[len(snapshots)-1]
			// collectStats above always refreshes the latest snapshot, so it's
			// guaranteed to be cached here; the ok is never false in practice.
			latestStats, _ := c.stats.get(repo.Name, latest.ID)

			c.Metrics.LastBackupOriginalSize.WithLabelValues(repo.Name).Set(float64(latestStats.TotalSize))
			c.Metrics.LastBackupFiles.WithLabelValues(repo.Name).Set(float64(latestStats.TotalFileCount))
			c.Metrics.LastBackupTimestamp.WithLabelValues(repo.Name).Set(float64(latest.Time.Unix()))
			if latest.Summary != nil {
				c.Metrics.LastBackupDuration.WithLabelValues(repo.Name).Set(latest.Summary.BackupEnd.Sub(latest.Summary.BackupStart).Seconds())
				c.Metrics.LastBackupDataAdded.WithLabelValues(repo.Name).Set(float64(latest.Summary.DataAdded))
				c.Metrics.LastBackupDataAddedPacked.WithLabelValues(repo.Name).Set(float64(latest.Summary.DataAddedPacked))
			}

			c.Metrics.LastSnapshotInfo.WithLabelValues(
				repo.Name,
				latest.Hostname,
				latest.ID,
				strings.Join(latest.Paths, ","),
				latest.Username,
				strings.Join(latest.Tags, ","),
			).Set(1)

			setSnapshotMetrics(c.Metrics, c.stats, repo.Name, snapshots)
		}

		c.ExporterMetrics.LastCollectError.WithLabelValues(toolName, repo.Name).Set(0)
		c.Cache.LastUpdate = time.Now()
	}

	c.Logger.Debug("Collecting metrics done for all repositories", "tool", toolName, "duration", time.Since(totalStartTime).Seconds())
	return errs
}

// listSnapshots runs `restic snapshots --json` for repo and parses the result.
func (c *Collector) listSnapshots(ctx context.Context, repo Repository) ([]Snapshot, error) {
	cmd := exec.CommandContext(ctx, c.ResticPath, "snapshots", "--json")
	cmd.Env = resticEnv(repo)
	output, err := cmd.Output()
	if err != nil {
		return nil, &collector.RepositoryCollectionError{
			Tool:       toolName,
			Repository: repo.Name,
			Msg:        "restic snapshots command error",
			Err:        err,
			StdErr:     stderrOf(err),
		}
	}

	snapshots, err := c.Parser.ParseSnapshots(output)
	if err != nil {
		return nil, &collector.RepositoryCollectionError{
			Tool:       toolName,
			Repository: repo.Name,
			Msg:        "restic snapshots output parsing error",
			Err:        err,
		}
	}
	return snapshots, nil
}

// collectStats fetches `restic stats` for every snapshot that needs a
// refresh this cycle (new snapshots, plus the latest one unconditionally),
// reusing the cache for the rest.
func (c *Collector) collectStats(ctx context.Context, repo Repository, snapshots []Snapshot) error {
	for i, snap := range snapshots {
		isLatest := i == len(snapshots)-1
		if !c.stats.needsRefresh(repo.Name, snap.ID, isLatest) {
			continue
		}

		cmd := exec.CommandContext(ctx, c.ResticPath, "stats", snap.ID, "--mode", "restore-size", "--json")
		cmd.Env = resticEnv(repo)
		output, err := cmd.Output()
		if err != nil {
			return &collector.RepositoryCollectionError{
				Tool:       toolName,
				Repository: repo.Name,
				Msg:        "restic stats command error",
				Err:        err,
				StdErr:     stderrOf(err),
			}
		}

		stats, err := c.Parser.ParseStats(output)
		if err != nil {
			return &collector.RepositoryCollectionError{
				Tool:       toolName,
				Repository: repo.Name,
				Msg:        "restic stats output parsing error",
				Err:        err,
			}
		}

		c.stats.set(repo.Name, snap.ID, stats)
	}
	return nil
}

// setSnapshotMetrics sets the per-snapshot metrics for the given repository's snapshots.
func setSnapshotMetrics(metrics *models.ResticMetrics, stats *statsCache, repository string, snapshots []Snapshot) {
	for _, snap := range snapshots {
		snapStats, _ := stats.get(repository, snap.ID)

		metrics.ArchiveOriginalSize.WithLabelValues(repository, snap.ShortID).Set(float64(snapStats.TotalSize))
		metrics.ArchiveFiles.WithLabelValues(repository, snap.ShortID).Set(float64(snapStats.TotalFileCount))
		metrics.ArchiveTimestamp.WithLabelValues(repository, snap.ShortID).Set(float64(snap.Time.Unix()))
		if snap.Summary != nil {
			metrics.ArchiveDuration.WithLabelValues(repository, snap.ShortID).Set(snap.Summary.BackupEnd.Sub(snap.Summary.BackupStart).Seconds())
			metrics.ArchiveDataAdded.WithLabelValues(repository, snap.ShortID).Set(float64(snap.Summary.DataAdded))
			metrics.ArchiveDataAddedPacked.WithLabelValues(repository, snap.ShortID).Set(float64(snap.Summary.DataAddedPacked))
		}
	}
}

// resticEnv builds the environment for a restic invocation against repo,
// starting from a copy of the process environment with any stray RESTIC_*
// variables stripped so repositories can't cross-contaminate each other.
func resticEnv(repo Repository) []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "RESTIC_") {
			continue
		}
		env = append(env, kv)
	}

	env = append(env, "RESTIC_REPOSITORY="+repo.Repository)
	switch {
	case repo.PasswordCommand != "":
		env = append(env, "RESTIC_PASSWORD_COMMAND="+repo.PasswordCommand)
	case repo.PasswordFile != "":
		env = append(env, "RESTIC_PASSWORD_FILE="+repo.PasswordFile)
	default:
		env = append(env, "RESTIC_PASSWORD="+repo.Password)
	}
	return env
}

func stderrOf(err error) string {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && len(exitError.Stderr) > 0 {
		return string(exitError.Stderr)
	}
	return ""
}
