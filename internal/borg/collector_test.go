package borg

import (
	"testing"
	"time"

	"github.com/lefeverd/backup-exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestInfoArgs_SplitsMultiWordOpts(t *testing.T) {
	repo := Repository{Path: "/repo", Opts: "--lock-wait 5"}
	args := infoArgs(repo, 10)
	assert.Equal(t, []string{"--lock-wait", "5", "info", "--last", "10", "--json", "/repo"}, args)
}

func TestInfoArgs_NoOpts(t *testing.T) {
	repo := Repository{Path: "/repo"}
	args := infoArgs(repo, 10)
	assert.Equal(t, []string{"info", "--last", "10", "--json", "/repo"}, args)
}

func archiveAt(name string, start time.Time) InfoOutputArchive {
	return InfoOutputArchive{
		Name:     name,
		Duration: 12.5,
		Start:    BorgTime{Time: start},
		Stats: InfoOutputArchiveStats{
			CompressedSize:   100,
			DeduplicatedSize: 50,
			NFiles:           10,
			OriginalSize:     200,
		},
	}
}

func TestSetArchiveMetrics(t *testing.T) {
	metrics := models.NewBorgMetrics()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	archives := []InfoOutputArchive{
		archiveAt("archive-1", start),
		archiveAt("archive-2", start.Add(24*time.Hour)),
	}

	setArchiveMetrics(metrics, "repo-a", archives)

	if got := testutil.ToFloat64(metrics.ArchiveDuration.WithLabelValues("repo-a", "archive-1")); got != 12.5 {
		t.Errorf("ArchiveDuration archive-1 = %v, want 12.5", got)
	}
	if got := testutil.ToFloat64(metrics.ArchiveCompressedSize.WithLabelValues("repo-a", "archive-2")); got != 100 {
		t.Errorf("ArchiveCompressedSize archive-2 = %v, want 100", got)
	}
	if got := testutil.ToFloat64(metrics.ArchiveDeduplicatedSize.WithLabelValues("repo-a", "archive-1")); got != 50 {
		t.Errorf("ArchiveDeduplicatedSize archive-1 = %v, want 50", got)
	}
	if got := testutil.ToFloat64(metrics.ArchiveFiles.WithLabelValues("repo-a", "archive-1")); got != 10 {
		t.Errorf("ArchiveFiles archive-1 = %v, want 10", got)
	}
	if got := testutil.ToFloat64(metrics.ArchiveOriginalSize.WithLabelValues("repo-a", "archive-1")); got != 200 {
		t.Errorf("ArchiveOriginalSize archive-1 = %v, want 200", got)
	}
	if got := testutil.ToFloat64(metrics.ArchiveTimestamp.WithLabelValues("repo-a", "archive-1")); got != float64(start.Unix()) {
		t.Errorf("ArchiveTimestamp archive-1 = %v, want %v", got, float64(start.Unix()))
	}
	if got := testutil.ToFloat64(metrics.ArchiveTimestamp.WithLabelValues("repo-a", "archive-2")); got != float64(start.Add(24*time.Hour).Unix()) {
		t.Errorf("ArchiveTimestamp archive-2 = %v, want %v", got, float64(start.Add(24*time.Hour).Unix()))
	}
}

// TestSetArchiveMetrics_PrunesAgedOutArchives mirrors Collect's reset-then-set
// pattern: archives that fall outside the --last N window on a later
// collection must not linger as stale series.
func TestSetArchiveMetrics_PrunesAgedOutArchives(t *testing.T) {
	metrics := models.NewBorgMetrics()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	setArchiveMetrics(metrics, "repo-a", []InfoOutputArchive{
		archiveAt("archive-1", start),
		archiveAt("archive-2", start.Add(24*time.Hour)),
	})

	metrics.ArchiveDuration.Reset()
	metrics.ArchiveCompressedSize.Reset()
	metrics.ArchiveDeduplicatedSize.Reset()
	metrics.ArchiveFiles.Reset()
	metrics.ArchiveOriginalSize.Reset()
	metrics.ArchiveTimestamp.Reset()

	setArchiveMetrics(metrics, "repo-a", []InfoOutputArchive{
		archiveAt("archive-2", start.Add(24*time.Hour)),
		archiveAt("archive-3", start.Add(48*time.Hour)),
	})

	if testutil.ToFloat64(metrics.ArchiveDuration.WithLabelValues("repo-a", "archive-2")) != 12.5 {
		t.Error("expected archive-2 to still be present after pruning")
	}

	// archive-1 aged out: querying it creates a fresh zero-value series
	// rather than proving absence, so assert via the vector's collected count.
	count := testutil.CollectAndCount(metrics.ArchiveDuration)
	if count != 2 {
		t.Errorf("expected 2 archive series after pruning, got %d", count)
	}
}
