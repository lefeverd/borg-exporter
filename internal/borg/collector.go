package borg

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lefeverd/backup-exporter/internal/collector"
	"github.com/lefeverd/backup-exporter/internal/models"
)

// Repository is a single configured borg repository.
type Repository struct {
	Name string
	Path string
	Opts string
}

// Collector collects metrics from a set of borg repositories.
type Collector struct {
	Repositories        []Repository
	BorgPath            string
	CommandTimeout      time.Duration
	ArchiveHistoryLimit int
	Metrics             *models.BorgMetrics
	ExporterMetrics     *models.ExporterMetrics
	Cache               *models.MetricsCache[*models.BorgMetrics]
	Parser              BorgParserInterface
	Logger              *slog.Logger
}

const toolName = "borg"

// Collect collects the metrics from borg and refreshes them in the cache.
// It collects metrics from every configured borg repository.
// In case of error, it still tries to collect metrics of the remaining repositories.
// This is why it returns a slice of error, which can come from different repositories.
func (c *Collector) Collect(ctx context.Context) []error {
	c.Cache.Lock()
	defer c.Cache.Unlock()

	// Check if collection is already in progress
	if c.Cache.Collecting {
		c.Logger.Info("Metrics collection already in progress, skipping", "tool", toolName)
		return nil
	}

	c.Cache.Collecting = true
	defer func() {
		c.Cache.Collecting = false
	}()

	totalStartTime := time.Now()

	// Reset the domain metrics
	c.Metrics.LastBackupDuration.Reset()
	c.Metrics.LastBackupCompressedSize.Reset()
	c.Metrics.LastBackupDeduplicatedSize.Reset()
	c.Metrics.LastBackupFiles.Reset()
	c.Metrics.LastBackupOriginalSize.Reset()
	c.Metrics.LastBackupTimestamp.Reset()

	c.Metrics.ArchiveDuration.Reset()
	c.Metrics.ArchiveCompressedSize.Reset()
	c.Metrics.ArchiveDeduplicatedSize.Reset()
	c.Metrics.ArchiveFiles.Reset()
	c.Metrics.ArchiveOriginalSize.Reset()
	c.Metrics.ArchiveTimestamp.Reset()

	c.Metrics.TotalChunks.Reset()
	c.Metrics.TotalCompressedSize.Reset()
	c.Metrics.TotalSize.Reset()
	c.Metrics.TotalUniqueChunks.Reset()
	c.Metrics.DeduplicatedCompressedSize.Reset()
	c.Metrics.DeduplicatedSize.Reset()

	c.Metrics.LastArchiveInfo.Reset()
	c.Metrics.RepositoryInfo.Reset()

	// Create command with timeout
	ctx, cancel := context.WithTimeout(ctx, c.CommandTimeout)
	defer cancel()

	var errs []error
	for _, repo := range c.Repositories {
		startTime := time.Now()
		c.Logger.Debug("Collecting metrics", "repository", repo.Name)
		args := infoArgs(repo, c.ArchiveHistoryLimit)
		cmd := exec.CommandContext(ctx, c.BorgPath, args...)
		output, err := cmd.Output()
		c.ExporterMetrics.LastCollectDuration.WithLabelValues(toolName, repo.Name).Set(time.Since(startTime).Seconds())
		c.ExporterMetrics.LastCollectTimestamp.WithLabelValues(toolName, repo.Name).Set(float64(time.Now().Unix()))
		c.Logger.Debug("Collecting metrics done", "repository", repo.Name, "duration", time.Since(startTime), "error", err)

		if err != nil {
			c.ExporterMetrics.LastCollectError.WithLabelValues(toolName, repo.Name).Set(1)
			c.ExporterMetrics.CollectErrors.WithLabelValues(toolName, repo.Name).Inc()

			var stdErr string
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				if len(exitError.Stderr) > 0 {
					stdErr = string(exitError.Stderr)
				}
			}

			errs = append(errs, &collector.RepositoryCollectionError{
				Tool:       toolName,
				Repository: repo.Name,
				Msg:        "borg command error",
				Err:        err,
				StdErr:     stdErr,
			})
			continue
		}

		info, err := c.Parser.ParseInfo(output)
		if err != nil {
			c.ExporterMetrics.LastCollectError.WithLabelValues(toolName, repo.Name).Set(1)
			c.ExporterMetrics.CollectErrors.WithLabelValues(toolName, repo.Name).Inc()
			errs = append(errs, &collector.RepositoryCollectionError{
				Tool:       toolName,
				Repository: repo.Name,
				Msg:        "borg output parsing error",
				Err:        err,
			})
			continue
		}

		// Update metrics
		if len(info.Archives) > 0 {
			// Set archive metrics
			// borg info --last N returns archives oldest-first (verified on borg 1.4.4),
			// so the last element is always the most recent.
			latest := info.Archives[len(info.Archives)-1]

			c.Metrics.LastBackupDuration.WithLabelValues(repo.Name).Set(latest.Duration)
			c.Metrics.LastBackupCompressedSize.WithLabelValues(repo.Name).Set(float64(latest.Stats.CompressedSize))
			c.Metrics.LastBackupDeduplicatedSize.WithLabelValues(repo.Name).Set(float64(latest.Stats.DeduplicatedSize))
			c.Metrics.LastBackupFiles.WithLabelValues(repo.Name).Set(float64(latest.Stats.NFiles))
			c.Metrics.LastBackupOriginalSize.WithLabelValues(repo.Name).Set(float64(latest.Stats.OriginalSize))
			c.Metrics.LastBackupTimestamp.WithLabelValues(repo.Name).Set(float64(latest.Start.Unix()))

			// Set last archive info metric
			c.Metrics.LastArchiveInfo.WithLabelValues(
				repo.Name,
				latest.Comment,
				latest.Start.Format(time.RFC3339),
				latest.End.Format(time.RFC3339),
				latest.Hostname,
				latest.ID,
				latest.Name,
				latest.Username,
			).Set(1)

			setArchiveMetrics(c.Metrics, repo.Name, info.Archives)
		}

		// Set repository metrics
		c.Metrics.TotalChunks.WithLabelValues(repo.Name).Set(float64(info.Cache.Stats.TotalChunks))
		c.Metrics.TotalCompressedSize.WithLabelValues(repo.Name).Set(float64(info.Cache.Stats.TotalCompressedSize))
		c.Metrics.TotalSize.WithLabelValues(repo.Name).Set(float64(info.Cache.Stats.TotalSize))
		c.Metrics.TotalUniqueChunks.WithLabelValues(repo.Name).Set(float64(info.Cache.Stats.TotalUniqueChunks))
		c.Metrics.DeduplicatedCompressedSize.WithLabelValues(repo.Name).Set(float64(info.Cache.Stats.DeduplicatedCompressedSize))
		c.Metrics.DeduplicatedSize.WithLabelValues(repo.Name).Set(float64(info.Cache.Stats.DeduplicatedSize))

		// Set repository info metric
		c.Metrics.RepositoryInfo.WithLabelValues(
			repo.Name,
			info.Repository.ID,
			info.Repository.LastModified.Format(time.RFC3339),
			info.Repository.Location,
		).Set(1)

		c.ExporterMetrics.LastCollectError.WithLabelValues(toolName, repo.Name).Set(0)
		c.Cache.LastUpdate = time.Now()
	}

	c.Logger.Debug("Collecting metrics done for all repositories", "tool", toolName, "duration", time.Since(totalStartTime).Seconds())
	return errs
}

// infoArgs builds the `borg info` argument list for repo. repo.Opts is
// split on whitespace since exec.Command doesn't invoke a shell - passing
// it as a single argument would send e.g. "--lock-wait 5" to borg as one
// malformed flag instead of two separate arguments.
func infoArgs(repo Repository, historyLimit int) []string {
	var args []string
	if repo.Opts != "" {
		args = append(args, strings.Fields(repo.Opts)...)
	}
	args = append(args, "info", "--last", strconv.Itoa(historyLimit), "--json", repo.Path)
	return args
}

// setArchiveMetrics sets the per-archive metrics for the given repository's archives.
func setArchiveMetrics(metrics *models.BorgMetrics, repository string, archives []InfoOutputArchive) {
	for _, archive := range archives {
		metrics.ArchiveDuration.WithLabelValues(repository, archive.Name).Set(archive.Duration)
		metrics.ArchiveCompressedSize.WithLabelValues(repository, archive.Name).Set(float64(archive.Stats.CompressedSize))
		metrics.ArchiveDeduplicatedSize.WithLabelValues(repository, archive.Name).Set(float64(archive.Stats.DeduplicatedSize))
		metrics.ArchiveFiles.WithLabelValues(repository, archive.Name).Set(float64(archive.Stats.NFiles))
		metrics.ArchiveOriginalSize.WithLabelValues(repository, archive.Name).Set(float64(archive.Stats.OriginalSize))
		metrics.ArchiveTimestamp.WithLabelValues(repository, archive.Name).Set(float64(archive.Start.Unix()))
	}
}
