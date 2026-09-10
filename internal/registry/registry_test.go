package registry_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppend_writesOneJSONLinePerEntry(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "registry.jsonl")

	require.NoError(t, registry.Append(path, registry.Entry{
		Timestamp: "2026-01-01T00:00:00Z", Fabric: "acme", Packet: "fix-x", Version: 1,
		Path: "test/x_test.go", Cause: registry.CauseGateSuggested, Commit: "abc123",
	}))
	require.NoError(t, registry.Append(path, registry.Entry{
		Timestamp: "2026-01-01T00:00:01Z", Fabric: "acme", Packet: "fix-x", Version: 1,
		Path: "test/y_test.go", Cause: registry.CauseCIFailure, Commit: "abc123",
	}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2)

	var first registry.Entry
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &first))
	assert.Equal(t, "test/x_test.go", first.Path)
	assert.Equal(t, registry.CauseGateSuggested, first.Cause)
}

func TestReadAllTolerant_skipsUnparseableLinesAndCountsThem(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "registry.jsonl")
	// A torn last line is exactly what a crash mid-append leaves behind.
	require.NoError(t, os.WriteFile(path, []byte(
		`{"cause":"proposal"}`+"\n"+
			`not json`+"\n"+
			`{"cause":"ci_failure"}`+"\n"), 0o600))

	entries, skipped, err := registry.ReadAllTolerant(path)

	require.NoError(t, err)
	assert.Equal(t, 1, skipped)
	assert.Len(t, entries, 2)
	assert.Equal(t, registry.CauseProposal, entries[0].Cause)
	assert.Equal(t, registry.CauseCIFailure, entries[1].Cause)
}
