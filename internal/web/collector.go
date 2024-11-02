package web

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

// Collect collects the metrics from borg and refreshes them in the cache.
// It collects metrics from the configured borg repositories.
// In case of error, it still tries to collect metrics of the remaining repositories.
// This is why it returns a slice of error, which can come from different repositories.
func (app *Application) Collect() []error {
	app.metricsCache.Lock()
	defer app.metricsCache.Unlock()

	// Check if collection is already in progress
	if app.metricsCache.Collecting {
		app.logger.Info("Metrics collection already in progress, skipping")
		return nil
	}

	app.metricsCache.Collecting = true
	defer func() {
		app.metricsCache.Collecting = false
	}()

	startTime := time.Now()

	// Reset the metrics
	app.metricsCache.Metrics.LastBackupDuration.Reset()
	app.metricsCache.Metrics.LastBackupCompressedSize.Reset()
	app.metricsCache.Metrics.LastBackupDeduplicatedSize.Reset()
	app.metricsCache.Metrics.LastBackupFiles.Reset()
	app.metricsCache.Metrics.LastBackupOriginalSize.Reset()
	app.metricsCache.Metrics.LastBackupTimestamp.Reset()

	app.metricsCache.Metrics.LastCollectDuration.Reset()
	app.metricsCache.Metrics.LastCollectError.Reset()
	app.metricsCache.Metrics.LastCollectTimestamp.Reset()
	// We don't reset CollectErrors as it is an incrementing errors counter

	app.metricsCache.Metrics.LastArchiveInfo.Reset()
	app.metricsCache.Metrics.RepositoryInfo.Reset()

	// Create command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), app.config.commandTimeout)
	defer cancel()

	var errs []error
	for _, borgRepository := range app.borgRepositories {
		app.logger.Debug("Collecting metrics", "repository", borgRepository)
		cmd := exec.CommandContext(ctx, "borg", "info", "--last", "1", "--json", borgRepository)
		output, err := cmd.Output()
		app.metricsCache.Metrics.LastCollectDuration.WithLabelValues(borgRepository).Set(time.Since(startTime).Seconds())
		app.metricsCache.Metrics.LastCollectTimestamp.WithLabelValues(borgRepository).Set(float64(time.Now().Unix()))
		app.logger.Debug("Collecting metrics", "repository", borgRepository, "duration", time.Since(startTime), "error", err)

		if err != nil {
			app.metricsCache.Metrics.LastCollectError.WithLabelValues(borgRepository).Set(1)
			app.metricsCache.Metrics.CollectErrors.WithLabelValues(borgRepository).Inc()

			var stdErr string
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				// Get stderr directly from the ExitError
				if len(exitError.Stderr) > 0 {
					stdErr = string(exitError.Stderr)
				}
			}

			errs = append(errs, &RepositoryCollectionError{
				Repository: borgRepository,
				Msg:        "borg command error",
				Err:        err,
				StdErr:     stdErr,
			})
			continue
		}

		info, err := app.borgParser.ParseInfo(output)
		if err != nil {
			app.metricsCache.Metrics.LastCollectError.WithLabelValues(borgRepository).Set(1)
			app.metricsCache.Metrics.CollectErrors.WithLabelValues(borgRepository).Inc()
			errs = append(errs, &RepositoryCollectionError{
				Repository: borgRepository,
				Msg:        "borg output parsing error",
				Err:        err,
			})
			continue
		}

		// Update metrics
		if len(info.Archives) > 0 {
			latest := info.Archives[len(info.Archives)-1]

			app.metricsCache.Metrics.LastBackupDuration.WithLabelValues(borgRepository).Set(latest.Duration)
			app.metricsCache.Metrics.LastBackupCompressedSize.WithLabelValues(borgRepository).Set(latest.Stats.CompressedSize)
			app.metricsCache.Metrics.LastBackupDeduplicatedSize.WithLabelValues(borgRepository).Set(latest.Stats.DeduplicatedSize)
			app.metricsCache.Metrics.LastBackupFiles.WithLabelValues(borgRepository).Set(float64(latest.Stats.NFiles))
			app.metricsCache.Metrics.LastBackupOriginalSize.WithLabelValues(borgRepository).Set(latest.Stats.OriginalSize)
			app.metricsCache.Metrics.LastBackupTimestamp.WithLabelValues(borgRepository).Set(float64(latest.Start.Unix()))

			// Set last archive info metric
			app.metricsCache.Metrics.LastArchiveInfo.WithLabelValues(
				borgRepository,
				latest.Comment,
				latest.Start.Format(time.RFC3339),
				latest.End.Format(time.RFC3339),
				latest.Hostname,
				latest.ID,
				latest.Name,
				latest.Username,
			).Set(1)
		}

		// Set repository info metric
		app.metricsCache.Metrics.RepositoryInfo.WithLabelValues(
			borgRepository,
			info.Repository.ID,
			info.Repository.LastModified.Format(time.RFC3339),
			info.Repository.Location,
		).Set(1)

		app.metricsCache.Metrics.LastCollectError.WithLabelValues(borgRepository).Set(0)
		app.metricsCache.LastUpdate = time.Now()
	}
	return errs
}
