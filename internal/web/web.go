package web

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lefeverd/backup-exporter/internal/borg"
	"github.com/lefeverd/backup-exporter/internal/collector"
	"github.com/lefeverd/backup-exporter/internal/config"
	"github.com/lefeverd/backup-exporter/internal/models"
	"github.com/lefeverd/backup-exporter/internal/restic"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type appConfig struct {
	listenAddress          string
	metricsPath            string
	configPath             string
	borgRefreshInterval    time.Duration
	resticRefreshInterval  time.Duration
	schedulerCheckInterval time.Duration
	commandTimeout         time.Duration
	borgPath               string
	resticPath             string
	logLevel               string
	archiveHistoryLimit    int
}

type application struct {
	logger   *slog.Logger
	logLevel *slog.LevelVar
	config   *appConfig
}

func Execute(Version string) {
	logLevel := &slog.LevelVar{} // INFO
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	app := &application{
		logger:   logger,
		logLevel: logLevel,
	}

	// Parse configuration
	var cfg appConfig
	flag.StringVar(&cfg.listenAddress, "listen-address", app.getEnv("LISTEN_ADDRESS", ":9099"), "http service address")
	flag.StringVar(&cfg.metricsPath, "metrics-path", app.getEnv("METRICS_PATH", "/metrics"), "metrics endpoint path")
	flag.StringVar(&cfg.configPath, "config", app.getEnv("CONFIG_FILE", ""), "path to the YAML repositories config file")
	flag.DurationVar(&cfg.borgRefreshInterval, "borg-refresh-interval", app.getDurationEnv("BORG_REFRESH_INTERVAL", 4*time.Hour), "borg metrics refresh interval (default 4h)")
	flag.DurationVar(&cfg.resticRefreshInterval, "restic-refresh-interval", app.getDurationEnv("RESTIC_REFRESH_INTERVAL", 4*time.Hour), "restic metrics refresh interval (default 4h)")
	flag.DurationVar(&cfg.schedulerCheckInterval, "scheduler-check-interval", app.getDurationEnv("SCHEDULER_CHECK_INTERVAL", 20*time.Second), "scheduler check interval (default 20s)")
	flag.DurationVar(&cfg.commandTimeout, "command-timeout", app.getDurationEnv("COMMAND_TIMEOUT", 120*time.Second), "command timeout, per collection cycle (default 120s)")
	flag.StringVar(&cfg.borgPath, "borg-path", app.getEnv("BORG_PATH", "borg"), "path to the borg binary (default borg)")
	flag.StringVar(&cfg.resticPath, "restic-path", app.getEnv("RESTIC_PATH", "restic"), "path to the restic binary (default restic)")
	flag.StringVar(&cfg.logLevel, "log-level", os.Getenv("LOG_LEVEL"), "log level")
	flag.IntVar(&cfg.archiveHistoryLimit, "archive-history-limit", app.getIntEnv("ARCHIVE_HISTORY_LIMIT", 10), "number of most recent archives/snapshots to expose per-archive metrics for (default 10)")

	var version bool
	flag.BoolVar(&version, "version", false, "prints the version")
	flag.Parse()

	if version {
		fmt.Println(Version)
		os.Exit(0)
	}

	app.logger.Info("Starting backup-exporter", "version", Version)
	app.config = &cfg

	if cfg.configPath == "" {
		app.logger.Error("No config file defined")
		os.Exit(1)
	}

	if cfg.archiveHistoryLimit < 1 {
		app.logger.Error("archive-history-limit must be a positive integer", "value", cfg.archiveHistoryLimit)
		os.Exit(1)
	}

	app.setLogLevel()

	repoConfig, err := config.Load(cfg.configPath)
	if err != nil {
		app.logger.Error("Could not load config file", "error", err)
		os.Exit(1)
	}
	for _, repo := range repoConfig.ResticRepositories() {
		if repo.UsesPlaintextPassword() {
			app.logger.Warn("restic repository uses the plaintext password field, password_file or password_command is recommended instead", "repository", repo.Name)
		}
	}

	reg := prometheus.NewRegistry()
	exporterMetrics := models.NewExporterMetrics()
	exporterMetrics.Register(reg)

	var runners []*collector.Runner

	if borgRepos := repoConfig.BorgRepositories(); len(borgRepos) > 0 {
		runners = append(runners, app.setupBorgCollector(borgRepos, exporterMetrics, reg))
	}

	if resticRepos := repoConfig.ResticRepositories(); len(resticRepos) > 0 {
		runners = append(runners, app.setupResticCollector(resticRepos, exporterMetrics, reg))
	}

	schedulerOpts := collector.NewTaskSchedulerOpts()
	schedulerOpts.CheckInterval = cfg.schedulerCheckInterval

	for _, runner := range runners {
		app.logger.Info("Starting initial metrics collection", "tool", runner.Tool)
		runner.CollectWrapper()
		app.logger.Info("Initial metrics collection done", "tool", runner.Tool)

		// Built after the initial collection so the refresh clock starts
		// counting from when that collection finished, not from before it
		// (which would shave its duration off the first refresh interval).
		runner.Scheduler = collector.NewTaskScheduler(runner.MetricsRefreshInterval, schedulerOpts)

		app.logger.Info("Start metrics collection routine", "tool", runner.Tool, "refresh interval", runner.MetricsRefreshInterval.String())
		go runner.Loop()
	}

	// Create our endpoints and start the web server
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.Handle(cfg.metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Printf("Starting backup-exporter on %s", cfg.listenAddress)
	log.Fatal(http.ListenAndServe(cfg.listenAddress, nil))
}

func (app *application) setupBorgCollector(repos []config.Repository, exporterMetrics *models.ExporterMetrics, reg *prometheus.Registry) *collector.Runner {
	borgMetrics := models.NewBorgMetrics()
	borgMetrics.Register(reg)

	var borgRepos []borg.Repository
	for _, repo := range repos {
		borgRepos = append(borgRepos, borg.Repository{Name: repo.Name, Path: repo.Path, Opts: repo.Opts})
	}

	version, err := getToolVersion(app.config.commandTimeout, app.config.borgPath, "--version")
	if err != nil {
		app.logger.Error("Could not get borg version", "error", err)
	}
	exporterMetrics.SystemInfo.WithLabelValues("borg", app.hostname(), version).Set(1)

	borgCollector := &borg.Collector{
		Repositories:        borgRepos,
		BorgPath:            app.config.borgPath,
		CommandTimeout:      app.config.commandTimeout,
		ArchiveHistoryLimit: app.config.archiveHistoryLimit,
		Metrics:             borgMetrics,
		ExporterMetrics:     exporterMetrics,
		Cache:               &models.MetricsCache[*models.BorgMetrics]{Metrics: borgMetrics},
		Parser:              &borg.BorgParser{},
		Logger:              app.logger,
	}

	return &collector.Runner{
		Tool:                   "borg",
		Collector:              borgCollector,
		MetricsRefreshInterval: app.config.borgRefreshInterval,
		Logger:                 app.logger,
	}
}

func (app *application) setupResticCollector(repos []config.Repository, exporterMetrics *models.ExporterMetrics, reg *prometheus.Registry) *collector.Runner {
	resticMetrics := models.NewResticMetrics()
	resticMetrics.Register(reg)

	var resticRepos []restic.Repository
	for _, repo := range repos {
		resticRepos = append(resticRepos, restic.Repository{
			Name:            repo.Name,
			Repository:      repo.ResticRepository,
			Password:        repo.Password,
			PasswordFile:    repo.PasswordFile,
			PasswordCommand: repo.PasswordCommand,
		})
	}

	version, err := getToolVersion(app.config.commandTimeout, app.config.resticPath, "version")
	if err != nil {
		app.logger.Error("Could not get restic version", "error", err)
	}
	exporterMetrics.SystemInfo.WithLabelValues("restic", app.hostname(), version).Set(1)

	resticCollector := &restic.Collector{
		Repositories:         resticRepos,
		ResticPath:           app.config.resticPath,
		CommandTimeout:       app.config.commandTimeout,
		SnapshotHistoryLimit: app.config.archiveHistoryLimit,
		Metrics:              resticMetrics,
		ExporterMetrics:      exporterMetrics,
		Cache:                &models.MetricsCache[*models.ResticMetrics]{Metrics: resticMetrics},
		Parser:               &restic.Parser{},
		Logger:               app.logger,
	}

	return &collector.Runner{
		Tool:                   "restic",
		Collector:              resticCollector,
		MetricsRefreshInterval: app.config.resticRefreshInterval,
		Logger:                 app.logger,
	}
}

func (app *application) hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getToolVersion(timeout time.Duration, path string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (app *application) getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (app *application) getDurationEnv(key string, fallback time.Duration) time.Duration {
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

func (app *application) getIntEnv(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.Atoi(value)
		if err != nil {
			app.logger.Error("Cannot parse int for config item", "item", key, "error", err)
			os.Exit(1)
		}
		return i
	}
	return fallback
}

func (app *application) setLogLevel() {
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
