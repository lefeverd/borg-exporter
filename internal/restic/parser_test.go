package restic

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseSnapshots(t *testing.T) {
	data, err := os.ReadFile("testdata/restic-snapshots.json")
	require.NoError(t, err)

	parser := Parser{}
	snapshots, err := parser.ParseSnapshots(data)
	require.NoError(t, err)
	require.Len(t, snapshots, 3)

	first := snapshots[0]
	assert.Equal(t, "f1a093e68837807037f6c9c698fee9ec77575d0a0483fb4f486f531c806b0dbd", first.ID)
	assert.Equal(t, "f1a093e6", first.ShortID)
	assert.Equal(t, "fedora", first.Hostname)
	assert.Equal(t, "dvd", first.Username)
	assert.Equal(t, []string{"/data/photos"}, first.Paths)
	assert.Nil(t, first.Tags)
	require.NotNil(t, first.Summary)
	assert.Equal(t, int64(3418), first.Summary.DataAdded)
	assert.Equal(t, int64(2755), first.Summary.DataAddedPacked)
	assert.Equal(t, 749*time.Millisecond, first.Summary.BackupEnd.Sub(first.Summary.BackupStart).Round(time.Millisecond))

	second := snapshots[1]
	assert.Equal(t, []string{"nightly"}, second.Tags)

	// third snapshot predates the "summary" field (older restic) - must not error, just leave it nil.
	third := snapshots[2]
	assert.Nil(t, third.Summary)
	assert.Equal(t, "9a1b2c3d", third.ShortID)
}

func TestParser_ParseSnapshots_InvalidJSON(t *testing.T) {
	parser := Parser{}
	_, err := parser.ParseSnapshots([]byte("not json"))
	assert.Error(t, err)
}

func TestParser_ParseStats(t *testing.T) {
	data, err := os.ReadFile("testdata/restic-stats.json")
	require.NoError(t, err)

	parser := Parser{}
	stats, err := parser.ParseStats(data)
	require.NoError(t, err)
	assert.Equal(t, Stats{TotalSize: 39, TotalFileCount: 8}, stats)
}

func TestParser_ParseStats_InvalidJSON(t *testing.T) {
	parser := Parser{}
	_, err := parser.ParseStats([]byte("not json"))
	assert.Error(t, err)
}
