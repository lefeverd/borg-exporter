package collector

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Collector collects metrics for a single backup tool across all of its
// configured repositories, updating that tool's metrics in place.
// It's tolerant of partial failure: it keeps collecting the remaining
// repositories after an error and returns every error encountered.
type Collector interface {
	Collect(ctx context.Context) []error
}

// Runner drives a Collector on its own schedule, independently of any other
// tool's Runner.
type Runner struct {
	Tool                   string
	Collector              Collector
	Scheduler              *TaskScheduler
	MetricsRefreshInterval time.Duration
	Logger                 *slog.Logger
}

// Loop executes the collector at every refresh interval, forever.
func (r *Runner) Loop() {
	for {
		if r.Scheduler.ShouldRun() {
			r.Logger.Info("Refreshing metrics", "tool", r.Tool)
			r.CollectWrapper()
			r.Logger.Info("Refreshing metrics done", "tool", r.Tool)
			r.Scheduler.UpdateLastRun()
		}
		r.Scheduler.WaitForNextRun()
	}
}

// CollectWrapper wraps the Collector's Collect method and logs any errors,
// retrying a handful of times within the same cycle when the refresh
// interval is long enough to make that worthwhile.
func (r *Runner) CollectWrapper() {
	const maxRetries = 5
	var attempt int

	for {
		errs := r.Collector.Collect(context.Background())
		if len(errs) == 0 {
			return
		}

		r.Logger.Error("Collection failed with the following error(s):", "tool", r.Tool)
		for _, err := range errs {
			var repositoryCollectionError *RepositoryCollectionError
			if errors.As(err, &repositoryCollectionError) {
				r.Logger.Error(repositoryCollectionError.Msg, "tool", r.Tool, "repository", repositoryCollectionError.Repository, "error", repositoryCollectionError.Err, "stdErr", repositoryCollectionError.StdErr)
				continue
			}
			r.Logger.Error(err.Error(), "tool", r.Tool)
		}

		// Not useful to retry if the refresh interval is smaller than 5 minutes
		if r.MetricsRefreshInterval < 5*time.Minute {
			r.Logger.Info("Metrics refresh interval is too short for retries, aborting and waiting for next refresh.", "tool", r.Tool)
			return
		}

		if attempt >= maxRetries {
			r.Logger.Error("Max retry limit reached for this cycle, aborting collection and waiting for next refresh.", "tool", r.Tool)
			return
		}

		r.Logger.Info("Retrying in a minute", "tool", r.Tool)
		time.Sleep(1 * time.Minute)
		attempt++
	}
}
