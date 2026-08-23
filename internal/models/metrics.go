package models

import (
	"github.com/prometheus/client_golang/prometheus"
)

// BorgMetrics holds the borg-domain metrics (backup/archive/repository data).
// Collection-health and system-info metrics live in ExporterMetrics instead,
// shared across tools.
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

	// info metrics
	LastArchiveInfo *prometheus.GaugeVec
	RepositoryInfo  *prometheus.GaugeVec
}

// NewBorgMetrics creates a BorgMetrics object containing all the metrics and returns a pointer to it
func NewBorgMetrics() *BorgMetrics {
	return &BorgMetrics{
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
	}
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

	// info metrics
	registry.MustRegister(m.LastArchiveInfo)
	registry.MustRegister(m.RepositoryInfo)
}

// ResticMetrics holds the restic-domain metrics (backup/snapshot data).
// restic's CLI doesn't expose chunk-level dedup stats or a repository-info
// equivalent to borg's, so there's no ResticRepositoryInfo/TotalChunks etc.
// Duration/DataAdded/DataAddedPacked come from a snapshot's "summary" block,
// which isn't present on every snapshot (older restic, or snapshots taken
// before that field existed) - those series are simply left unset rather
// than faked with a 0 value.
type ResticMetrics struct {
	// last snapshot metrics
	LastBackupDuration        *prometheus.GaugeVec
	LastBackupOriginalSize    *prometheus.GaugeVec
	LastBackupFiles           *prometheus.GaugeVec
	LastBackupTimestamp       *prometheus.GaugeVec
	LastBackupDataAdded       *prometheus.GaugeVec
	LastBackupDataAddedPacked *prometheus.GaugeVec

	// per-snapshot metrics, for the last N snapshots (see ArchiveHistoryLimit)
	ArchiveDuration        *prometheus.GaugeVec
	ArchiveOriginalSize    *prometheus.GaugeVec
	ArchiveFiles           *prometheus.GaugeVec
	ArchiveTimestamp       *prometheus.GaugeVec
	ArchiveDataAdded       *prometheus.GaugeVec
	ArchiveDataAddedPacked *prometheus.GaugeVec

	// info metrics
	LastSnapshotInfo *prometheus.GaugeVec
}

// NewResticMetrics creates a ResticMetrics object containing all the metrics and returns a pointer to it
func NewResticMetrics() *ResticMetrics {
	return &ResticMetrics{
		LastBackupDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_last_backup_duration_seconds",
			Help: "Duration of the last backup in seconds (unset if the snapshot has no summary data)",
		}, []string{"repository"}),
		LastBackupOriginalSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_last_backup_original_size_bytes",
			Help: "Original (restore) size of the last backup in bytes",
		}, []string{"repository"}),
		LastBackupFiles: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_last_backup_files",
			Help: "Number of files in the last backup",
		}, []string{"repository"}),
		LastBackupTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_last_backup_timestamp",
			Help: "Timestamp of the last backup",
		}, []string{"repository"}),
		LastBackupDataAdded: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_last_backup_data_added_bytes",
			Help: "Uncompressed data added to the repository by the last backup (unset if the snapshot has no summary data)",
		}, []string{"repository"}),
		LastBackupDataAddedPacked: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_last_backup_data_added_packed_bytes",
			Help: "Compressed/packed data added to the repository by the last backup (unset if the snapshot has no summary data)",
		}, []string{"repository"}),

		ArchiveDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_archive_duration_seconds",
			Help: "Duration of the snapshot's backup in seconds (unset if the snapshot has no summary data)",
		}, []string{"repository", "snapshot"}),
		ArchiveOriginalSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_archive_original_size_bytes",
			Help: "Original (restore) size of the snapshot in bytes",
		}, []string{"repository", "snapshot"}),
		ArchiveFiles: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_archive_files",
			Help: "Number of files in the snapshot",
		}, []string{"repository", "snapshot"}),
		ArchiveTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_archive_timestamp",
			Help: "Timestamp of the snapshot's backup",
		}, []string{"repository", "snapshot"}),
		ArchiveDataAdded: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_archive_data_added_bytes",
			Help: "Uncompressed data added to the repository by this snapshot (unset if the snapshot has no summary data)",
		}, []string{"repository", "snapshot"}),
		ArchiveDataAddedPacked: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "restic_archive_data_added_packed_bytes",
			Help: "Compressed/packed data added to the repository by this snapshot (unset if the snapshot has no summary data)",
		}, []string{"repository", "snapshot"}),

		LastSnapshotInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "restic_last_snapshot_info",
				Help: "Information about the last backup snapshot",
			},
			[]string{"repository", "hostname", "id", "paths", "username", "tags"},
		),
	}
}

// Register registers the metrics to the prometheus registry
func (m *ResticMetrics) Register(registry *prometheus.Registry) {
	registry.MustRegister(m.LastBackupDuration)
	registry.MustRegister(m.LastBackupOriginalSize)
	registry.MustRegister(m.LastBackupFiles)
	registry.MustRegister(m.LastBackupTimestamp)
	registry.MustRegister(m.LastBackupDataAdded)
	registry.MustRegister(m.LastBackupDataAddedPacked)

	registry.MustRegister(m.ArchiveDuration)
	registry.MustRegister(m.ArchiveOriginalSize)
	registry.MustRegister(m.ArchiveFiles)
	registry.MustRegister(m.ArchiveTimestamp)
	registry.MustRegister(m.ArchiveDataAdded)
	registry.MustRegister(m.ArchiveDataAddedPacked)

	registry.MustRegister(m.LastSnapshotInfo)
}

// ExporterMetrics holds the metrics describing the exporter's own operation,
// shared across every backup tool it collects from (labeled by "tool").
type ExporterMetrics struct {
	CollectErrors        *prometheus.CounterVec
	LastCollectError     *prometheus.GaugeVec
	LastCollectDuration  *prometheus.GaugeVec
	LastCollectTimestamp *prometheus.GaugeVec
	SystemInfo           *prometheus.GaugeVec
}

// NewExporterMetrics creates an ExporterMetrics object containing all the metrics and returns a pointer to it
func NewExporterMetrics() *ExporterMetrics {
	return &ExporterMetrics{
		CollectErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "exporter_collect_errors",
			Help: "Number of errors encountered while collecting metrics",
		}, []string{"tool", "repository"}),
		LastCollectError: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "exporter_last_collect_error",
			Help: "1 if the last collection failed, 0 if successful",
		}, []string{"tool", "repository"}),
		LastCollectDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "exporter_last_collect_duration_seconds",
			Help: "Duration of the last metrics collection",
		}, []string{"tool", "repository"}),
		LastCollectTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "exporter_last_collect_timestamp",
			Help: "Timestamp of the last metrics collection",
		}, []string{"tool", "repository"}),
		SystemInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "exporter_system_info",
			Help: "Information about a configured backup tool",
		}, []string{"tool", "hostname", "version"}),
	}
}

// Register registers the metrics to the prometheus registry
func (m *ExporterMetrics) Register(registry *prometheus.Registry) {
	registry.MustRegister(m.CollectErrors)
	registry.MustRegister(m.LastCollectError)
	registry.MustRegister(m.LastCollectDuration)
	registry.MustRegister(m.LastCollectTimestamp)
	registry.MustRegister(m.SystemInfo)
}
