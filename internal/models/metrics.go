package models

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

type BorgMetrics struct {
	// last archive metrics
	LastBackupDuration         *prometheus.GaugeVec
	LastBackupCompressedSize   *prometheus.GaugeVec
	LastBackupDeduplicatedSize *prometheus.GaugeVec
	LastBackupFiles            *prometheus.GaugeVec
	LastBackupOriginalSize     *prometheus.GaugeVec
	LastBackupTimestamp        *prometheus.GaugeVec

	// per-archive metrics, for the last N archives (see ArchiveHistoryLimit)
	ArchiveDuration         *prometheus.GaugeVec
	ArchiveCompressedSize   *prometheus.GaugeVec
	ArchiveDeduplicatedSize *prometheus.GaugeVec
	ArchiveFiles            *prometheus.GaugeVec
	ArchiveOriginalSize     *prometheus.GaugeVec
	ArchiveTimestamp        *prometheus.GaugeVec

	// repository metrics (from borg info cache stats)
	TotalChunks                *prometheus.GaugeVec
	TotalCompressedSize        *prometheus.GaugeVec
	TotalSize                  *prometheus.GaugeVec
	TotalUniqueChunks          *prometheus.GaugeVec
	DeduplicatedCompressedSize *prometheus.GaugeVec // unique_csize
	DeduplicatedSize           *prometheus.GaugeVec // unique_size

	// exporter collection metrics
	CollectErrors        *prometheus.CounterVec
	LastCollectError     *prometheus.GaugeVec
	LastCollectDuration  *prometheus.GaugeVec
	LastCollectTimestamp *prometheus.GaugeVec

	// info metrics
	LastArchiveInfo *prometheus.GaugeVec
	RepositoryInfo  *prometheus.GaugeVec
	SystemInfo      *prometheus.GaugeVec
}

// NewBorgMetrics creates a BorgMetrics object containing all the metrics and returns a pointer to it
func NewBorgMetrics(borgVersion string) *BorgMetrics {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	m := &BorgMetrics{
		// archive metrics
		LastBackupDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_backup_duration_seconds",
			Help: "Duration of the last backup in seconds",
		}, []string{"repository"}),
		LastBackupCompressedSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_backup_compressed_size_bytes",
			Help: "Compressed size of the last backup in bytes",
		}, []string{"repository"}),
		LastBackupDeduplicatedSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_backup_deduplicated_size_bytes",
			Help: "Deduplicated size of the last backup in bytes",
		}, []string{"repository"}),
		LastBackupFiles: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_backup_files",
			Help: "Number of files in the last backup",
		}, []string{"repository"}),
		LastBackupOriginalSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_backup_original_size_bytes",
			Help: "Original size of the last backup in bytes",
		}, []string{"repository"}),
		LastBackupTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_backup_timestamp",
			Help: "Timestamp of the last backup",
		}, []string{"repository"}),

		// per-archive metrics
		ArchiveDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_archive_duration_seconds",
			Help: "Duration of the archive's backup in seconds",
		}, []string{"repository", "archive"}),
		ArchiveCompressedSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_archive_compressed_size_bytes",
			Help: "Compressed size of the archive in bytes",
		}, []string{"repository", "archive"}),
		ArchiveDeduplicatedSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_archive_deduplicated_size_bytes",
			Help: "Deduplicated size of the archive in bytes",
		}, []string{"repository", "archive"}),
		ArchiveFiles: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_archive_files",
			Help: "Number of files in the archive",
		}, []string{"repository", "archive"}),
		ArchiveOriginalSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_archive_original_size_bytes",
			Help: "Original size of the archive in bytes",
		}, []string{"repository", "archive"}),
		ArchiveTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_archive_timestamp",
			Help: "Timestamp of the archive's backup",
		}, []string{"repository", "archive"}),

		// repository metrics
		TotalChunks: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_total_chunks",
			Help: "Repository total chunks",
		}, []string{"repository"}),
		TotalCompressedSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_total_compressed_size_bytes",
			Help: "Repository total compressed size",
		}, []string{"repository"}),
		TotalSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_total_size_bytes",
			Help: "Repository total size",
		}, []string{"repository"}),
		TotalUniqueChunks: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_total_unique_chunks",
			Help: "Repository total unique chunks",
		}, []string{"repository"}),
		DeduplicatedCompressedSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_deduplicated_compressed_size_bytes",
			Help: "Repository deduplicated compressed size",
		}, []string{"repository"}),
		DeduplicatedSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_deduplicated_size_bytes",
			Help: "Repository deduplicated size",
		}, []string{"repository"}),

		// Exporter collection metrics
		CollectErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "borg_collect_errors",
			Help: "Number of errors encountered by borg exporter",
		}, []string{"repository"}),
		LastCollectError: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_collect_error",
			Help: "1 if the last collection failed, 0 if successful",
		}, []string{"repository"}),
		LastCollectDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_collect_duration_seconds",
			Help: "Duration of the last metrics collection",
		}, []string{"repository"}),
		LastCollectTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "borg_last_collect_timestamp",
			Help: "Timestamp of the last metrics collection",
		}, []string{"repository"}),

		// Info metrics
		LastArchiveInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_last_archive_info",
				Help: "Information about the last backup archive",
			},
			[]string{"repository", "comment", "start_time", "end_time", "hostname", "id", "name", "username"},
		),

		RepositoryInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_repository_info",
				Help: "Information about the backup repository",
			},
			[]string{"repository", "id", "last_modified", "location"},
		),

		SystemInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "borg_system_info",
				Help: "Information about the borg backup system",
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
	// archive metrics
	registry.MustRegister(m.LastBackupDuration)
	registry.MustRegister(m.LastBackupCompressedSize)
	registry.MustRegister(m.LastBackupDeduplicatedSize)
	registry.MustRegister(m.LastBackupFiles)
	registry.MustRegister(m.LastBackupOriginalSize)
	registry.MustRegister(m.LastBackupTimestamp)

	// per-archive metrics
	registry.MustRegister(m.ArchiveDuration)
	registry.MustRegister(m.ArchiveCompressedSize)
	registry.MustRegister(m.ArchiveDeduplicatedSize)
	registry.MustRegister(m.ArchiveFiles)
	registry.MustRegister(m.ArchiveOriginalSize)
	registry.MustRegister(m.ArchiveTimestamp)

	// repository metrics
	registry.MustRegister(m.TotalChunks)
	registry.MustRegister(m.TotalCompressedSize)
	registry.MustRegister(m.TotalSize)
	registry.MustRegister(m.TotalUniqueChunks)
	registry.MustRegister(m.DeduplicatedCompressedSize)
	registry.MustRegister(m.DeduplicatedSize)

	// exporter collection metrics
	registry.MustRegister(m.CollectErrors)
	registry.MustRegister(m.LastCollectError)
	registry.MustRegister(m.LastCollectDuration)
	registry.MustRegister(m.LastCollectTimestamp)

	// info metrics
	registry.MustRegister(m.LastArchiveInfo)
	registry.MustRegister(m.RepositoryInfo)
	registry.MustRegister(m.SystemInfo)
}
