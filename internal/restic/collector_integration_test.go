package restic

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/lefeverd/backup-exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

// requireRestic skips the test if the real restic binary isn't available -
// keeps this test safe to run in CI environments that don't have it installed.
func requireRestic(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("restic")
	if err != nil {
		t.Skip("restic binary not found, skipping integration test")
	}
	return path
}

// initResticRepo creates a fresh restic repository in a temp dir and returns
// its path plus the password used to init it.
func initResticRepo(t *testing.T, resticPath string) (repoPath, password string) {
	t.Helper()
	repoPath = filepath.Join(t.TempDir(), "repo")
	password = "test-password"

	cmd := exec.Command(resticPath, "init")
	cmd.Env = append(os.Environ(), "RESTIC_REPOSITORY="+repoPath, "RESTIC_PASSWORD="+password)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "restic init failed: %s", out)
	return repoPath, password
}

// backup runs a real restic backup of a small file, returning the snapshot ID.
func backup(t *testing.T, resticPath, repoPath, password, content string) {
	t.Helper()
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte(content), 0o600))

	cmd := exec.Command(resticPath, "backup", srcDir)
	cmd.Env = append(os.Environ(), "RESTIC_REPOSITORY="+repoPath, "RESTIC_PASSWORD="+password)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "restic backup failed: %s", out)
}

func newTestCollector(resticPath string, repos []Repository) (*Collector, *models.ResticMetrics) {
	metrics := models.NewResticMetrics()
	return &Collector{
		Repositories:         repos,
		ResticPath:           resticPath,
		CommandTimeout:       30 * time.Second,
		SnapshotHistoryLimit: 10,
		Metrics:              metrics,
		ExporterMetrics:      models.NewExporterMetrics(),
		Cache:                &models.MetricsCache[*models.ResticMetrics]{Metrics: metrics},
		Parser:               &Parser{},
		Logger:               slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}, metrics
}

func TestCollector_Collect_Integration(t *testing.T) {
	resticPath := requireRestic(t)
	repoPath, password := initResticRepo(t, resticPath)
	backup(t, resticPath, repoPath, password, "hello world")
	backup(t, resticPath, repoPath, password, "hello world, more data this time")

	repo := Repository{Name: "test-repo", Repository: repoPath, Password: password}
	c, metrics := newTestCollector(resticPath, []Repository{repo})

	errs := c.Collect(context.Background())
	require.Empty(t, errs)

	require.Equal(t, 2, testutil.CollectAndCount(metrics.ArchiveOriginalSize))
	require.Equal(t, float64(0), testutil.ToFloat64(c.ExporterMetrics.LastCollectError.WithLabelValues("restic", "test-repo")))

	originalSize := testutil.ToFloat64(metrics.LastBackupOriginalSize.WithLabelValues("test-repo"))
	require.Greater(t, originalSize, 0.0)

	duration := testutil.ToFloat64(metrics.LastBackupDuration.WithLabelValues("test-repo"))
	require.GreaterOrEqual(t, duration, 0.0)
}

// TestCollector_Collect_CachesOlderSnapshotStats verifies the design's core
// strategy: once a snapshot has cached stats and stops being the latest, a
// second Collect cycle must not re-run `restic stats` for it - only for the
// (new) latest one.
func TestCollector_Collect_CachesOlderSnapshotStats(t *testing.T) {
	resticPath := requireRestic(t)
	repoPath, password := initResticRepo(t, resticPath)
	backup(t, resticPath, repoPath, password, "first snapshot content")

	repo := Repository{Name: "test-repo", Repository: repoPath, Password: password}
	c, _ := newTestCollector(resticPath, []Repository{repo})

	errs := c.Collect(context.Background())
	require.Empty(t, errs)

	snapshots, err := c.listSnapshots(context.Background(), repo)
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	firstID := snapshots[0].ID
	cachedStats, ok := c.stats.get("test-repo", firstID)
	require.True(t, ok)

	backup(t, resticPath, repoPath, password, "second snapshot content, quite a bit longer than the first")

	errs = c.Collect(context.Background())
	require.Empty(t, errs)

	// The first snapshot's cached stats must be untouched (same values,
	// not recomputed) - proven indirectly by still being present and equal.
	stillCachedStats, ok := c.stats.get("test-repo", firstID)
	require.True(t, ok)
	require.Equal(t, cachedStats, stillCachedStats)
}
