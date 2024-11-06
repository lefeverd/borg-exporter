package web

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/lefeverd/borg-exporter/internal/models"
	"github.com/lefeverd/borg-exporter/internal/parser"
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
	listenAddress          string
	metricsPath            string
	metricsRefreshInterval time.Duration
	schedulerCheckInterval time.Duration
	commandTimeout         time.Duration
	borgRepositories       string
	borgPath               string
	logLevel               string
}

type Application struct {
	logger           *slog.Logger
	logLevel         *slog.LevelVar
	config           *config
	borgRepositories []string
	metricsCache     *models.MetricsCache
	borgParser       parser.BorgParserInterface
}

func Execute(Version string) {
	logLevel := &slog.LevelVar{} // INFO
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	app := &Application{
		logger:   logger,
		logLevel: logLevel,
	}

	// Parse configuration
	var cfg config
	flag.StringVar(&cfg.listenAddress, "listen-address", app.getEnv("LISTEN_ADDRESS", ":9099"), "http service address")
	flag.StringVar(&cfg.metricsPath, "metrics-path", app.getEnv("METRICS_PATH", "/metrics"), "metrics endpoint path")
	flag.DurationVar(&cfg.metricsRefreshInterval, "metrics-refresh-interval", app.getDurationEnv("METRICS_REFRESH_INTERVAL", 4*time.Hour), "metrics refresh interval (default 4h)")
	flag.DurationVar(&cfg.schedulerCheckInterval, "scheduler-check-interval", app.getDurationEnv("SCHEDULER_CHECK_INTERVAL", 20*time.Second), "scheduler check interval (default 20s)")
	flag.DurationVar(&cfg.commandTimeout, "command-timeout", app.getDurationEnv("COMMAND_TIMEOUT", 120*time.Second), "borg command timeout (default 120s)")
	flag.StringVar(&cfg.borgRepositories, "borg-repositories", os.Getenv("BORG_REPOSITORIES"), "comma-separated list of borg repositories")
	flag.StringVar(&cfg.borgPath, "borg-path", app.getEnv("BORG_PATH", "borg"), "path to the borg binary (default borg)")
	flag.StringVar(&cfg.logLevel, "log-level", os.Getenv("LOG_LEVEL"), "log level")

	var version bool
	flag.BoolVar(&version, "version", false, "prints the version")
	flag.Parse()

	if version {
		fmt.Println(Version)
		os.Exit(0)
	}

	app.logger.Info("Starting borg-exporter")
	app.config = &cfg

	if cfg.borgRepositories == "" {
		app.logger.Error("No borg repositories defined")
		os.Exit(1)
	}
	app.borgRepositories = strings.Split(cfg.borgRepositories, ",")

	app.setLogLevel()

	// Setup our app by injecting our dependencies
	app.metricsCache = &models.MetricsCache{
		Metrics: models.NewBorgMetrics(app.getBorgVersion()),
	}
	app.borgParser = &parser.BorgParser{}

	// Create non-global registry and register our metrics
	reg := prometheus.NewRegistry()
	app.metricsCache.Metrics.Register(reg)

	// Trigger an initial metrics collection before starting the web server
	app.logger.Info("Starting initial metrics collection")
	app.CollectWrapper()
	app.logger.Info("Initial metrics collection done")

	app.logger.Info("Start metrics collection routine", "refresh interval", app.config.metricsRefreshInterval.String())
	go app.CollectLoop()

	// Create our endpoints and start the web server
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
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

func (app *Application) setLogLevel() {
	if app.config.logLevel == "" {
		return
	}
	level := strings.ToLower(app.config.logLevel)
	switch level {
	case "debug":
		app.logLevel.Set(slog.LevelDebug)
	case "warn":
		app.logLevel.Set(slog.LevelWarn)
	case "error":
		app.logLevel.Set(slog.LevelError)
	default:
		app.logLevel.Set(slog.LevelInfo)
	}
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

// CollectLoop executes the metrics collection at every refresh interval.
func (app *Application) CollectLoop() {
	opts := NewTaskSchedulerOpts()
	opts.CheckInterval = app.config.schedulerCheckInterval
	scheduler := NewTaskScheduler(app.config.metricsRefreshInterval, opts)
	for {
		if scheduler.ShouldRun() {
			app.logger.Info("Refreshing metrics")
			app.CollectWrapper()
			app.logger.Info("Refreshing metrics done")
			scheduler.UpdateLastRun()
		}
		scheduler.WaitForNextRun()
	}
}

// CollectWrapper wraps the Collect method and logs any errors
func (app *Application) CollectWrapper() {
	errs := app.Collect()
	if len(errs) != 0 {
		app.logger.Error("Collection failed with the following error(s):")
		for _, err := range errs {
			var repositoryCollectionError *RepositoryCollectionError
			if errors.As(err, &repositoryCollectionError) {
				app.logger.Error(repositoryCollectionError.Msg, "repository", repositoryCollectionError.Repository, "error", repositoryCollectionError.Err, "stdErr", repositoryCollectionError.StdErr)
				continue
			}

			app.logger.Error(err.Error())
		}
	}
}
