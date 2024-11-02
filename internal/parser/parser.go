package parser

import (
	"encoding/json"
	"strings"
	"time"
)

var BorgTimeLayout = "2006-01-02T15:04:05.000000"

type BorgParserInterface interface {
	ParseInfo(text []byte) (InfoOutput, error)
}

// BorgTime is a custom time type for borg, which uses ISO 8601
type BorgTime struct {
	time.Time
}

// UnmarshalJSON implements JSON unmarshalling for the borg timestamp format
func (bt *BorgTime) UnmarshalJSON(data []byte) error {
	// Remove quotes from string
	s := strings.Trim(string(data), `"`)
	// Parse the time
	t, err := ParseBorgTime(s)
	if err != nil {
		return err
	}
	*bt = t
	return nil
}

func ParseBorgTime(s string) (BorgTime, error) {
	t, err := time.Parse(BorgTimeLayout, s)
	if err != nil {
		return BorgTime{}, err
	}
	bt := BorgTime{}
	bt.Time = t
	return bt, nil
}

// InfoOutput represents the rood node of the json
type InfoOutput struct {
	Archives   []InfoOutputArchive  `json:"archives"`
	Cache      InfoOutputCache      `json:"cache"`
	Repository InfoOutputRepository `json:"repository"`
	Encryption InfoOutputEncryption `json:"encryption"`
}

type InfoOutputArchive struct {
	Comment  string                 `json:"comment"`
	Duration float64                `json:"duration"`
	End      BorgTime               `json:"end"`
	Hostname string                 `json:"hostname"`
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Start    BorgTime               `json:"start"`
	Stats    InfoOutputArchiveStats `json:"stats"`
	Username string                 `json:"username"`
}

type InfoOutputArchiveStats struct {
	CompressedSize   int64 `json:"compressed_size"`
	DeduplicatedSize int64 `json:"deduplicated_size"`
	NFiles           int64 `json:"nfiles"`
	OriginalSize     int64 `json:"original_size"`
}

type InfoOutputCache struct {
	Path  string               `json:"path"`
	Stats InfoOutputCacheStats `json:"stats"`
}

type InfoOutputCacheStats struct {
	TotalChunks                int64 `json:"total_chunks"`
	TotalCompressedSize        int64 `json:"total_csize"`
	TotalSize                  int64 `json:"total_size"`
	TotalUniqueChunks          int64 `json:"total_unique_chunks"`
	DeduplicatedCompressedSize int64 `json:"unique_csize"`
	DeduplicatedSize           int64 `json:"unique_size"`
}

type InfoOutputRepository struct {
	ID           string   `json:"id"`
	LastModified BorgTime `json:"last_modified"`
	Location     string   `json:"location"`
}

type InfoOutputEncryption struct {
	Mode string `json:"mode"`
}

type BorgParser struct{}

func (p *BorgParser) ParseInfo(text []byte) (InfoOutput, error) {
	var borgInfoOutput InfoOutput
	if err := json.Unmarshal(text, &borgInfoOutput); err != nil {
		return InfoOutput{}, err
	}
	return borgInfoOutput, nil
}
