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
		lastRun:       opts.TimeProvider().Round(0), // We round it to remove the monotonic part, see comments below
		now:           opts.TimeProvider,
	}
}

func (ts *TaskScheduler) ShouldRun() bool {
	elapsed := ts.now().Sub(ts.lastRun)
	return elapsed >= ts.interval
}

func (ts *TaskScheduler) UpdateLastRun() {
	// We round it to remove the monotonic part, which causes issues when the computer goes to sleep then wakes up.
	// See comment in WaitForNextRun
	ts.lastRun = ts.now().Round(0)
}

func (ts *TaskScheduler) WaitForNextRun() {
	for {
		now := ts.now()
		// We round it to remove the monotonic part, which causes issues when the computer goes to sleep then wakes up.
		// Indeed, it retained an old, pre-sleep monotonic component that no longer matches the current system time.
		// By stripping next's monotonic component, we only compare the wall-clock time.
		nextRun := ts.lastRun.Add(ts.interval).Round(0)

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
