package tokenstore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/tokenstore"
)

func TestStore_loadReturnsEmptyWhenAbsent(t *testing.T) {
	t.Parallel()
	s := tokenstore.New(filepath.Join(t.TempDir(), "anthropic.token"))

	tok, err := s.Load()

	require.NoError(t, err, "a missing token file is not an error — it is the unconfigured state")
	assert.Equal(t, "", tok)
	assert.False(t, s.Configured(), "an absent token reads as unconfigured")
}

func TestStore_roundTripsSavedToken(t *testing.T) {
	t.Parallel()
	s := tokenstore.New(filepath.Join(t.TempDir(), "anthropic.token"))

	require.NoError(t, s.Save("sk-ant-secret"))

	tok, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, "sk-ant-secret", tok, "a saved token loads back verbatim")
	assert.True(t, s.Configured(), "a saved token reads as configured")
}

func TestStore_trimsSurroundingWhitespaceOnSave(t *testing.T) {
	t.Parallel()
	s := tokenstore.New(filepath.Join(t.TempDir(), "anthropic.token"))

	require.NoError(t, s.Save("  sk-ant-padded\n"))

	tok, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, "sk-ant-padded", tok, "a pasted token's stray padding never reaches the harness env")
}

func TestStore_saveOverwritesPreviousToken(t *testing.T) {
	t.Parallel()
	s := tokenstore.New(filepath.Join(t.TempDir(), "anthropic.token"))

	require.NoError(t, s.Save("sk-ant-old"))
	require.NoError(t, s.Save("sk-ant-new"))

	tok, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, "sk-ant-new", tok, "re-saving replaces the token, never appends")
}

func TestStore_savedFileIsOwnerOnly(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "anthropic.token")
	s := tokenstore.New(path)

	require.NoError(t, s.Save("sk-ant-secret"))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
		"a stored API key is readable only by its owner — never group/world readable")
}
