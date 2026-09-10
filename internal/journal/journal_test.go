package journal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppend_writesOneJSONObjectPerLine(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.jsonl")

	require.NoError(t, journal.Append(path, journal.Entry{
		Timestamp: "2026-09-09T10:00:00Z",
		Slug:      "fix-login-timeout",
		Version:   1,
		Event:     journal.EventEmit,
		Actor:     journal.ActorHuman,
	}))
	require.NoError(t, journal.Append(path, journal.Entry{
		Timestamp: "2026-09-09T10:01:00Z",
		Slug:      "fix-login-timeout",
		Version:   1,
		Event:     journal.EventKill,
		Actor:     journal.ActorHuman,
	}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	assert.Len(t, lines, 2)
	assert.JSONEq(t, `{
		"ts":"2026-09-09T10:00:00Z","slug":"fix-login-timeout","version":1,
		"node":null,"event":"emit","reason":null,"tokens":0,
		"actor":"human","detail":{}
	}`, lines[0])
}

func TestAppend_alwaysIncludesNullableFieldsExplicitly(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.jsonl")
	node := "build"
	reason := "smell:max_files"

	require.NoError(t, journal.Append(path, journal.Entry{
		Timestamp: "2026-09-09T10:00:00Z",
		Slug:      "fix-login-timeout",
		Node:      &node,
		Event:     journal.EventSmellHalt,
		Reason:    &reason,
		Actor:     journal.ActorHarness,
		Detail:    map[string]any{"files_touched": 12},
	}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"ts":"2026-09-09T10:00:00Z","slug":"fix-login-timeout","version":0,
		"node":"build","event":"smell_halt","reason":"smell:max_files",
		"tokens":0,"actor":"harness","detail":{"files_touched":12}
	}`, strings.TrimRight(string(data), "\n"))
}

func TestReadAllTolerant_skipsUnparseableLinesAndCountsThem(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.jsonl")
	// A torn last line is exactly what a crash mid-append leaves behind.
	require.NoError(t, os.WriteFile(path, []byte(
		`{"event":"emit","actor":"human"}`+"\n"+
			`not json`+"\n"+
			`{"event":"terminate","actor":"harness"}`+"\n"), 0o600))

	entries, skipped, err := journal.ReadAllTolerant(path)

	require.NoError(t, err)
	assert.Equal(t, 1, skipped)
	assert.Len(t, entries, 2)
	assert.Equal(t, journal.EventEmit, entries[0].Event)
	assert.Equal(t, journal.EventTerminate, entries[1].Event)
}

func TestAppend_appendsToExistingFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"event":"pre-existing"}`+"\n"), 0o600))

	require.NoError(t, journal.Append(path, journal.Entry{
		Timestamp: "2026-09-09T10:00:00Z",
		Event:     journal.EventEmit,
		Actor:     journal.ActorHuman,
	}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(data), `{"event":"pre-existing"}`))
}
