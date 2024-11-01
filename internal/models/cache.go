package models

import (
	"sync"
	"time"
)

type MetricsCache struct {
	sync.RWMutex
	LastUpdate time.Time
	Collecting bool
	Metrics    *BorgMetrics
	Timeout    time.Duration
}
