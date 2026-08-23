package restic

import "sync"

// statsCache holds per-snapshot `restic stats` results across collection
// cycles, keyed by repository name then snapshot ID. Older snapshots'
// sizes don't change, so only new/latest snapshots need a fresh stats call
// each cycle - see needsRefresh.
type statsCache struct {
	mu     sync.Mutex
	byRepo map[string]map[string]Stats
}

func newStatsCache() *statsCache {
	return &statsCache{byRepo: make(map[string]map[string]Stats)}
}

// needsRefresh reports whether the snapshot's stats must be (re)computed
// this cycle: true for any snapshot not already cached, and always true for
// the latest snapshot regardless of caching.
func (c *statsCache) needsRefresh(repo, id string, isLatest bool) bool {
	if isLatest {
		return true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.byRepo[repo][id]
	return !ok
}

func (c *statsCache) get(repo, id string) (Stats, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats, ok := c.byRepo[repo][id]
	return stats, ok
}

func (c *statsCache) set(repo, id string, stats Stats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byRepo[repo] == nil {
		c.byRepo[repo] = make(map[string]Stats)
	}
	c.byRepo[repo][id] = stats
}

// reconcile drops cached entries for repo whose snapshot ID is no longer in
// currentIDs (self-heals against forget/prune on the upstream repository).
func (c *statsCache) reconcile(repo string, currentIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	existing, ok := c.byRepo[repo]
	if !ok {
		return
	}
	keep := make(map[string]bool, len(currentIDs))
	for _, id := range currentIDs {
		keep[id] = true
	}
	for id := range existing {
		if !keep[id] {
			delete(existing, id)
		}
	}
}
