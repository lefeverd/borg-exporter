package restic

import (
	"encoding/json"
	"time"
)

// ParserInterface parses the JSON output of restic commands.
type ParserInterface interface {
	ParseSnapshots(data []byte) ([]Snapshot, error)
	ParseStats(data []byte) (Stats, error)
}

// Snapshot is a single entry from `restic snapshots --json`.
type Snapshot struct {
	Time     time.Time `json:"time"`
	Tree     string    `json:"tree"`
	Paths    []string  `json:"paths"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	Tags     []string  `json:"tags,omitempty"`
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`

	// Summary is only present on snapshots taken by a restic version recent
	// enough to record it - older snapshots (or older restic) leave this nil.
	Summary *SnapshotSummary `json:"summary,omitempty"`
}

// SnapshotSummary is the "summary" block of a snapshot, when present.
type SnapshotSummary struct {
	BackupStart     time.Time `json:"backup_start"`
	BackupEnd       time.Time `json:"backup_end"`
	DataAdded       int64     `json:"data_added"`
	DataAddedPacked int64     `json:"data_added_packed"`
}

// Stats is the output of `restic stats <id> --mode restore-size --json`.
type Stats struct {
	TotalSize      int64 `json:"total_size"`
	TotalFileCount int64 `json:"total_file_count"`
}

// Parser parses restic CLI JSON output.
type Parser struct{}

func (p *Parser) ParseSnapshots(data []byte) ([]Snapshot, error) {
	var snapshots []Snapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (p *Parser) ParseStats(data []byte) (Stats, error) {
	var stats Stats
	if err := json.Unmarshal(data, &stats); err != nil {
		return Stats{}, err
	}
	return stats, nil
}
