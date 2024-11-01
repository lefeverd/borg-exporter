package models

import (
	"github.com/prometheus/client_golang/prometheus"
	"os"
)

type BorgMetrics struct {
	// archive metrics
	LastBackupDuration         prometheus.Gauge
	LastBackupCompressedSize   prometheus.Gauge
	LastBackupDeduplicatedSize prometheus.Gauge
	LastBackupFiles            prometheus.Gauge
	LastBackupOriginalSize     prometheus.Gauge
	LastBackupTimestamp        prometheus.Gauge

	// exporter collection metrics
	CollectErrors       prometheus.Counter
	LastCollectError    prometheus.Gauge
	LastCollectDuration prometheus.Gauge

	// info metrics
	LastArchiveInfo *prometheus.GaugeVec
	RepositoryInfo  *prometheus.GaugeVec
	SystemInfo      *prometheus.GaugeVec
}

// NewBorgMetrics creates an empty BorgMetrics object and returns a pointer to it
func NewBorgMetrics(borgVersion string) *BorgMetrics {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	m := &BorgMetrics{
		LastBackupDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "borg_last_backup_duration_seconds",
			Help: "Duration of the last backup in seconds",
		}),
		LastBackupCompressedSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "borg_last_backup_compressed_size_bytes",
			Help: "Size of the last backup in bytes",
		}),
		LastBackupDeduplicatedSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "borg_last_backup_deduplicated_size_bytes",
			Help: "Size of the last backup in bytes",
		}),
		LastBackupFiles: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "borg_last_backup_files",
			Help: "Number of files that have been uploaded",
		}),
		LastBackupOriginalSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "borg_last_backup_original_size_bytes",
			Help: "Size of the last backup in bytes",
		}),
		LastBackupTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "borgmatic_last_backup_timestamp",
			Help: "Timestamp of the last backup",
		}),

		// Exporter collection metrics
		CollectErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "borg_collect_errors",
			Help: "Number of errors encountered by borg exporter",
		}),
		LastCollectError: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "borg_last_collect_error",
			Help: "1 if the last collection failed, 0 if successful",
		}),
		LastCollectDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "borg_last_collect_duration_seconds",
			Help: "Duration of the last metrics collection",
		}),

		// Info metrics
		LastArchiveInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_last_archive_info",
				Help: "Information about the last backup archive",
			},
			[]string{"comment", "start_time", "end_time", "hostname", "id", "name", "username"},
		),

		RepositoryInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_repository_info",
				Help: "Information about the backup repository",
			},
			[]string{"id", "last_modified", "location"},
		),

		SystemInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_system_info",
				Help: "Information about the borgmatic backup system",
			},
			[]string{"hostname", "borg_version"},
		),
	}

	// Set static system info
	m.SystemInfo.WithLabelValues(
		hostname,
		borgVersion,
	).Set(1)

	return m
}

// Register registers the metrics to the prometheus registry
func (m *BorgMetrics) Register(registry *prometheus.Registry) {
	registry.MustRegister(m.LastBackupDuration)
	registry.MustRegister(m.LastBackupCompressedSize)
	registry.MustRegister(m.LastBackupDeduplicatedSize)
	registry.MustRegister(m.LastBackupFiles)
	registry.MustRegister(m.LastBackupOriginalSize)
	registry.MustRegister(m.LastBackupTimestamp)
	registry.MustRegister(m.CollectErrors)
	registry.MustRegister(m.LastCollectError)
	registry.MustRegister(m.LastCollectDuration)
	registry.MustRegister(m.LastArchiveInfo)
	registry.MustRegister(m.RepositoryInfo)
	registry.MustRegister(m.SystemInfo)
}
