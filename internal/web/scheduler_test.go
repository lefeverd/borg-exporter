package web

import (
	"testing"
	"time"
)

type mockTime struct {
	currentTime time.Time
}

func (mt *mockTime) Now() time.Time {
	return mt.currentTime
}

func (mt *mockTime) advance(d time.Duration) {
	mt.currentTime = mt.currentTime.Add(d)
}

func TestNewTaskScheduler(t *testing.T) {
	interval := 1 * time.Minute
	scheduler := NewTaskScheduler(interval, NewTaskSchedulerOpts())

	if scheduler.interval != interval {
		t.Errorf("Expected interval %v, got %v", interval, scheduler.interval)
	}

	diff := time.Since(scheduler.lastRun)
	if diff > time.Second {
		t.Errorf("lastRun not set to recent time, diff: %v", diff)
	}
}

func TestShouldRun(t *testing.T) {
	tests := []struct {
		name           string
		interval       time.Duration
		timeAdvance    time.Duration
		expectedResult bool
	}{
		{
			name:           "should not run before interval",
			interval:       time.Minute,
			timeAdvance:    30 * time.Second,
			expectedResult: false,
		},
		{
			name:           "should run at exact interval",
			interval:       time.Minute,
			timeAdvance:    time.Minute,
			expectedResult: true,
		},
		{
			name:           "should run after interval",
			interval:       time.Minute,
			timeAdvance:    90 * time.Second,
			expectedResult: true,
		},
		{
			name:           "should not run immediately",
			interval:       time.Minute,
			timeAdvance:    0,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTime := &mockTime{
				currentTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			}

			opts := NewTaskSchedulerOpts()
			opts.TimeProvider = mockTime.Now
			scheduler := NewTaskScheduler(tt.interval, opts)

			// Advance time
			mockTime.advance(tt.timeAdvance)

			// Test if scheduler should run
			result := scheduler.ShouldRun()

			if result != tt.expectedResult {
				t.Errorf("Expected ShouldRun() to return %v, got %v for time advance of %v",
					tt.expectedResult, result, tt.timeAdvance)
			}
		})
	}
}

func TestUpdateLastRun(t *testing.T) {
	mockTime := &mockTime{
		currentTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	opts := NewTaskSchedulerOpts()
	opts.TimeProvider = mockTime.Now
	scheduler := NewTaskScheduler(time.Minute, opts)

	oldLastRun := scheduler.lastRun

	mockTime.advance(time.Second)
	scheduler.UpdateLastRun()

	if !scheduler.lastRun.After(oldLastRun) {
		t.Error("UpdateLastRun did not update lastRun time")
	}

	if scheduler.lastRun != mockTime.currentTime {
		t.Error("UpdateLastRun did not set correct time")
	}
}

func TestMultipleIntervals(t *testing.T) {
	mockTime := &mockTime{
		currentTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	opts := NewTaskSchedulerOpts()
	opts.TimeProvider = mockTime.Now
	scheduler := NewTaskScheduler(time.Minute, opts)

	// Test multiple intervals
	executionTimes := make([]time.Time, 0)

	for i := 1; i <= 5; i++ {
		mockTime.advance(time.Minute)

		if !scheduler.ShouldRun() {
			t.Errorf("Task should run at interval %d", i)
		}

		scheduler.UpdateLastRun()
		executionTimes = append(executionTimes, scheduler.lastRun)
	}

	// Verify intervals between executions
	for i := 1; i < len(executionTimes); i++ {
		interval := executionTimes[i].Sub(executionTimes[i-1])
		if interval != time.Minute {
			t.Errorf("Expected 1 minute interval, got %v between executions %d and %d",
				interval, i-1, i)
		}
	}
}
