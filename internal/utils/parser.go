package utils

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
	CompressedSize   float64 `json:"compressed_size"`
	DeduplicatedSize float64 `json:"deduplicated_size"`
	NFiles           int64   `json:"nfiles"`
	OriginalSize     float64 `json:"original_size"`
}

type InfoOutputRepository struct {
	ID           string   `json:"id"`
	LastModified BorgTime `json:"last_modified"`
	Location     string   `json:"location"`
}

type InfoOutputEncryption struct {
	Mode string `json:"mode"`
}

type InfoOutput struct {
	Archives   []InfoOutputArchive  `json:"archives"`
	Repository InfoOutputRepository `json:"repository"`
	Encryption InfoOutputEncryption `json:"encryption"`
}

type BorgParser struct{}

func (p *BorgParser) ParseInfo(text []byte) (InfoOutput, error) {
	var borgInfoOutput InfoOutput
	if err := json.Unmarshal(text, &borgInfoOutput); err != nil {
		return InfoOutput{}, err
	}
	return borgInfoOutput, nil
}
