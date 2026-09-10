package state_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleState() *state.State {
	return &state.State{
		Slug:        "fix-login-timeout",
		Version:     1,
		State:       state.Building,
		Attempt:     1,
		RetriesUsed: 0,
		Branch:      "packets/fix-login-timeout-v1",
		CreatedAt:   time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC),
	}
}

func TestSaveLoad_roundTripsAllFields(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	want := sampleState()

	require.NoError(t, state.Save(path, want))
	got, err := state.Load(path)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestSave_leavesNoTmpFileBehindOnSuccess(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")

	require.NoError(t, state.Save(path, sampleState()))

	_, err := os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err))
}

// A crash between the tmp write and the rename must leave the previous
// state.json untouched — the rename is the only step that can make a new
// version visible.
func TestSave_crashBeforeRenameLeavesOldFileIntact(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	original := sampleState()
	require.NoError(t, state.Save(path, original))

	require.NoError(t, os.WriteFile(path+".tmp", []byte(`{"state":"corrupt-mid-write"`), 0o600))

	got, err := state.Load(path)

	require.NoError(t, err)
	assert.Equal(t, original, got)
}

func TestLoad_errorsOnMissingFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	_, err := state.Load(path)

	assert.Error(t, err)
}
