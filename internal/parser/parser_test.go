package parser

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestBorgParser_ParseInfo(t *testing.T) {
	tests := []struct {
		name           string
		testFile       string
		wantErr        error
		wantInfoOutput InfoOutput
	}{
		{
			name:     "Parse valid output",
			testFile: "testdata/borg-info.json",
			wantErr:  nil,
			wantInfoOutput: InfoOutput{
				Archives: []InfoOutputArchive{
					{
						Comment:  "",
						Duration: 4540.154685,
						End:      mustParseBorgTime(t, "2024-10-28T21:52:44.000000"),
						Hostname: "my-hostname",
						ID:       "a0ef59abfd45d22460a586053e7266e24b9989d00d44aae8442d3d8e6fe92cbf",
						Name:     "my-hostname-2024-10-28T20:37:03.464475",
						Start:    mustParseBorgTime(t, "2024-10-28T20:37:04.000000"),
						Stats: InfoOutputArchiveStats{
							CompressedSize:   687831993433,
							DeduplicatedSize: 53654483387,
							NFiles:           13079758,
							OriginalSize:     1341294469810,
						},
						Username: "root",
					},
				},
				Cache: InfoOutputCache{
					Path: "/root/.cache/borg/03a461422fd3be21cbf5235e8d40c2ecbe28b1e4c295ae2ac456563ca62c94af",
					Stats: InfoOutputCacheStats{
						TotalChunks:                139398821,
						TotalCompressedSize:        4055905565460,
						TotalSize:                  7047547605252,
						TotalUniqueChunks:          1675085,
						DeduplicatedCompressedSize: 304339691351,
						DeduplicatedSize:           454963879225,
					},
				},
				Repository: InfoOutputRepository{
					ID:           "c58db5835b4fbd34ac8c747897674d46c58db5835b4fbd34ac8c747897674d46",
					LastModified: mustParseBorgTime(t, "2024-10-28T22:00:45.000000"),
					Location:     "ssh://backup-host/backups/backup-name",
				},
				Encryption: InfoOutputEncryption{
					Mode: "none",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := BorgParser{}
			data, err := os.ReadFile(tt.testFile)
			if err != nil {
				t.Fatal(err)
			}
			infoOutput, err := parser.ParseInfo(data)
			assert.Equal(t, tt.wantErr, err)
			assert.EqualValues(t, tt.wantInfoOutput, infoOutput)
		})
	}
}

func mustParseBorgTime(t *testing.T, s string) BorgTime {
	t.Helper()
	result, err := ParseBorgTime(s)
	if err != nil {
		t.Fatalf("Failed to parse time %q: %v", s, err)
	}
	return result
}
