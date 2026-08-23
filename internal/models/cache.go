package models

import (
	"sync"
	"time"
)

// MetricsCache guards a tool's metrics object against concurrent collection
// runs and tracks basic collection state. T is the tool-specific metrics
// type (e.g. *BorgMetrics, *ResticMetrics).
type MetricsCache[T any] struct {
	sync.RWMutex
	LastUpdate time.Time
	Collecting bool
	Metrics    T
}
