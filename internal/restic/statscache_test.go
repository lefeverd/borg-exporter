package restic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatsCache_NewSnapshotNeedsRefresh(t *testing.T) {
	c := newStatsCache()
	assert.True(t, c.needsRefresh("repo-a", "snap-1", false))
}

func TestStatsCache_CachedNonLatestSkipsRefresh(t *testing.T) {
	c := newStatsCache()
	c.set("repo-a", "snap-1", Stats{TotalSize: 10, TotalFileCount: 1})

	assert.False(t, c.needsRefresh("repo-a", "snap-1", false))

	stats, ok := c.get("repo-a", "snap-1")
	assert.True(t, ok)
	assert.Equal(t, Stats{TotalSize: 10, TotalFileCount: 1}, stats)
}

func TestStatsCache_LatestAlwaysNeedsRefreshEvenIfCached(t *testing.T) {
	c := newStatsCache()
	c.set("repo-a", "snap-1", Stats{TotalSize: 10, TotalFileCount: 1})

	assert.True(t, c.needsRefresh("repo-a", "snap-1", true))
}

func TestStatsCache_SelfHealsDroppedSnapshots(t *testing.T) {
	c := newStatsCache()
	c.set("repo-a", "snap-1", Stats{TotalSize: 10})
	c.set("repo-a", "snap-2", Stats{TotalSize: 20})

	// snap-1 was forgotten/pruned upstream; only snap-2 remains.
	c.reconcile("repo-a", []string{"snap-2"})

	_, ok := c.get("repo-a", "snap-1")
	assert.False(t, ok, "expected snap-1 to be dropped from the cache")

	_, ok = c.get("repo-a", "snap-2")
	assert.True(t, ok, "expected snap-2 to remain cached")
}

func TestStatsCache_ReposAreIndependent(t *testing.T) {
	c := newStatsCache()
	c.set("repo-a", "snap-1", Stats{TotalSize: 10})
	c.set("repo-b", "snap-1", Stats{TotalSize: 20})

	c.reconcile("repo-a", []string{}) // repo-a's snap-1 is gone

	_, ok := c.get("repo-a", "snap-1")
	assert.False(t, ok)

	statsB, ok := c.get("repo-b", "snap-1")
	assert.True(t, ok, "repo-b's cache must be unaffected by repo-a's reconcile")
	assert.Equal(t, Stats{TotalSize: 20}, statsB)
}
