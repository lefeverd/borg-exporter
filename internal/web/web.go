package web

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/lefeverd/borg-exporter/internal/models"
	"github.com/lefeverd/borg-exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type config struct {
	listenAddress    string
	metricsPath      string
	cacheTimeout     time.Duration
	commandTimeout   time.Duration
	borgRepositories string
}

type Application struct {
	logger           *slog.Logger
	config           *config
	borgRepositories []string
	metricsCache     *models.MetricsCache
	borgParser       utils.BorgParserInterface
}

func Execute() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := &Application{
		logger: logger,
	}

	app.logger.Info("Starting borg-exporter")

	var cfg config
	flag.StringVar(&cfg.listenAddress, "listen-address", app.getEnv("LISTEN_ADDRESS", ":9099"), "http service address")
	flag.StringVar(&cfg.metricsPath, "metrics-path", app.getEnv("METRICS_PATH", "/metrics"), "metrics endpoint path")
	flag.DurationVar(&cfg.cacheTimeout, "cache-timeout", app.getDurationEnv("CACHE_TIMEOUT", 12*time.Hour), "cache timeout (default 12h)")
	flag.DurationVar(&cfg.commandTimeout, "command-timeout", app.getDurationEnv("COMMAND_TIMEOUT", 120*time.Second), "borg command timeout (default 120s)")
	flag.StringVar(&cfg.borgRepositories, "borg-repositories", os.Getenv("BORG_REPOSITORIES"), "comma-separated list of borg repositories")
	flag.Parse()
	app.config = &cfg

	if cfg.borgRepositories == "" {
		app.logger.Error("No borg repositories defined")
		os.Exit(1)
	}
	app.borgRepositories = strings.Split(cfg.borgRepositories, ",")

	// Create the metrics cache
	app.metricsCache = &models.MetricsCache{
		Metrics: models.NewBorgMetrics(app.getBorgVersion()),
	}

	// Create the borg parser
	app.borgParser = &utils.BorgParser{}

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create non-global registry.
	reg := prometheus.NewRegistry()
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))

	// Register the metrics to the registry
	app.metricsCache.Metrics.Register(reg)

	app.logger.Info("Starting initial collection")
	err := app.Collect()
	if err != nil {
		app.logger.Error("Initial collection failed", "error", err)
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			// Get stderr directly from the ExitError
			if len(exitError.Stderr) > 0 {
				fmt.Printf("Command stderr: %s\n", string(exitError.Stderr))
			}
		}
		os.Exit(1)
	}
	app.logger.Info("Initial collection done")

	log.Printf("Starting borgmatic exporter on %s", cfg.listenAddress)
	log.Fatal(http.ListenAndServe(cfg.listenAddress, nil))
}

func (app *Application) getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (app *Application) getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		duration, err := time.ParseDuration(value)
		if err != nil {
			app.logger.Error("Cannot parse duration for config item", "item", key, "error", err)
			os.Exit(1)
		}
		return duration
	}
	return fallback
}

func (app *Application) getBorgVersion() string {
	// Create command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), app.config.commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "borg", "--version")
	output, err := cmd.Output()
	if err != nil {
		app.logger.Error("Could not get borg version")
		return ""
	}
	return strings.TrimSpace(string(output))
}

// Collect collects the metrics from borg and caches them, only if the last collection is older than
// the cache timeout.
func (app *Application) Collect() error {
	app.metricsCache.Lock()
	defer app.metricsCache.Unlock()

	// Check if collection is already in progress
	if app.metricsCache.Collecting {
		return nil
	}

	// Check if cache is still valid
	if time.Since(app.metricsCache.LastUpdate) < app.config.cacheTimeout {
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
	app.metricsCache.Metrics.CollectErrors.Reset()

	app.metricsCache.Metrics.LastArchiveInfo.Reset()
	app.metricsCache.Metrics.RepositoryInfo.Reset()

	// Create command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), app.config.commandTimeout)
	defer cancel()

	for _, borgRepository := range app.borgRepositories {
		cmd := exec.CommandContext(ctx, "borg", "info", "--last", "1", "--json", borgRepository)
		output, err := cmd.Output()
		app.metricsCache.Metrics.LastCollectDuration.WithLabelValues(borgRepository).Set(time.Since(startTime).Seconds())

		if err != nil {
			app.metricsCache.Metrics.LastCollectError.WithLabelValues(borgRepository).Set(1)
			app.metricsCache.Metrics.CollectErrors.WithLabelValues(borgRepository).Inc()
			return err
		}

		info, err := app.borgParser.ParseInfo(output)
		if err != nil {
			app.metricsCache.Metrics.LastCollectError.WithLabelValues(borgRepository).Set(1)
			app.metricsCache.Metrics.CollectErrors.WithLabelValues(borgRepository).Inc()
			return err
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
	return nil
}
