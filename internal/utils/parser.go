package utils

import (
	"encoding/json"
	"time"
)

type BorgParserInterface interface {
	ParseInfo(text []byte) (InfoOutput, error)
}

type InfoOutput struct {
	Archives []struct {
		Comment  string    `json:"comment"`
		Duration float64   `json:"duration"`
		End      time.Time `json:"end"`
		Hostname string    `json:"hostname"`
		ID       string    `json:"id"`
		Name     string    `json:"name"`
		Start    time.Time `json:"start"`
		Stats    struct {
			CompressedSize   float64 `json:"compressed_size"`
			DeduplicatedSize float64 `json:"deduplicated_size"`
			NFiles           int64   `json:"nfiles"`
			OriginalSize     float64 `json:"original_size"`
		} `json:"stats"`
		Username string `json:"username"`
	} `json:"archives"`
	Repository struct {
		ID           string    `json:"id"`
		LastModified time.Time `json:"last_modified"`
		Location     string    `json:"location"`
	} `json:"repository"`
}

type BorgParser struct{}

func (p *BorgParser) ParseInfo(text []byte) (InfoOutput, error) {
	var borgInfoOutput InfoOutput
	if err := json.Unmarshal(text, &borgInfoOutput); err != nil {
		return InfoOutput{}, err
	}
	return borgInfoOutput, nil
}
