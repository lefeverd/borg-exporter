package web

import (
	"time"
)

// TimeProvider is a function type that returns the current time
type TimeProvider func() time.Time

type TaskScheduler struct {
	interval      time.Duration
	checkInterval time.Duration
	lastRun       time.Time
	now           TimeProvider
}

type TaskSchedulerOpts struct {
	CheckInterval time.Duration
	TimeProvider  TimeProvider
}

// NewTaskSchedulerOpts returns a TaskSchedulerOpts with default values.
func NewTaskSchedulerOpts() *TaskSchedulerOpts {
	return &TaskSchedulerOpts{
		CheckInterval: 20 * time.Second,
		TimeProvider:  time.Now,
	}
}

func NewTaskScheduler(interval time.Duration, opts *TaskSchedulerOpts) *TaskScheduler {
	return &TaskScheduler{
		interval:      interval,
		checkInterval: opts.CheckInterval,
		lastRun:       opts.TimeProvider(),
		now:           opts.TimeProvider,
	}
}

func (ts *TaskScheduler) ShouldRun() bool {
	elapsed := ts.now().Sub(ts.lastRun)
	return elapsed >= ts.interval
}

func (ts *TaskScheduler) UpdateLastRun() {
	ts.lastRun = ts.now()
}

func (ts *TaskScheduler) WaitForNextRun() {
	for {
		now := ts.now()
		nextRun := ts.lastRun.Add(ts.interval)

		if now.After(nextRun) || now.Equal(nextRun) {
			return
		}

		sleepTime := time.Until(nextRun)
		if sleepTime > ts.checkInterval {
			sleepTime = ts.checkInterval
		}

		time.Sleep(sleepTime)
	}
}
