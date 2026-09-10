package packet_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandshakeStrength_stringIsALowercaseMonoWordPerName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		strength packet.HandshakeStrength
		want     string
	}{
		{"none", packet.StrengthNone, "none"},
		{"examples", packet.StrengthExamples, "examples"},
		{"properties", packet.StrengthProperties, "properties"},
		{"unknown value fails safe", packet.HandshakeStrength(99), "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.strength.String())
		})
	}
}

func TestHandshakePath_isTheHandshakeDirectoryUnderTheRepo(t *testing.T) {
	t.Parallel()
	assert.Equal(t, filepath.Join("/repo", "handshake"), packet.HandshakePath("/repo"))
}

func TestWriteHandshake_writesAHashedFileUnderTheProtectedDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const content = "package handshake\n\nfunc TestSpec(t *testing.T) {}\n"

	h, err := packet.WriteHandshake(dir, "spec_test", content, packet.StrengthExamples)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "handshake", "spec_test.go"), h.Path)
	assert.Equal(t, filepath.Dir(h.Path), packet.HandshakePath(dir), "the written file must land under the one protected directory")
	assert.Equal(t, reanchor.HashLines(content), h.Hash, "the hash must be the codebase's one content-hash primitive, not a second scheme")
	assert.Equal(t, packet.StrengthExamples, h.Strength)
	got, err := os.ReadFile(h.Path)
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
}

// Strength is self-declared by the human authoring the handshake — never
// scored or defaulted — so a non-default strength must round-trip exactly.
func TestWriteHandshake_recordsTheAuthorsDeclaredStrengthExactly(t *testing.T) {
	t.Parallel()
	h, err := packet.WriteHandshake(t.TempDir(), "spec_test", "package handshake\n", packet.StrengthProperties)

	require.NoError(t, err)
	assert.Equal(t, packet.StrengthProperties, h.Strength)
}

// Re-authoring (the human strengthening their handshake before dispatch)
// must overwrite the prior content and hash, not append or refuse.
func TestWriteHandshake_reauthoringOverwritesThePriorContentAndHash(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	first, err := packet.WriteHandshake(dir, "spec_test", "package handshake\n// v1\n", packet.StrengthExamples)
	require.NoError(t, err)

	second, err := packet.WriteHandshake(dir, "spec_test", "package handshake\n// v2\n", packet.StrengthProperties)

	require.NoError(t, err)
	assert.NotEqual(t, first.Hash, second.Hash)
	got, err := os.ReadFile(second.Path)
	require.NoError(t, err)
	assert.Equal(t, "package handshake\n// v2\n", string(got), "the second write must replace, not append to, the first")
}

func TestWriteHandshake_refusesEmptyContentRatherThanRecordAnEmptyHash(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	_, err := packet.WriteHandshake(dir, "spec_test", "", packet.StrengthExamples)

	assert.Error(t, err)
	_, statErr := os.Stat(filepath.Join(dir, "handshake", "spec_test.go"))
	assert.True(t, os.IsNotExist(statErr), "a refused write must leave no file behind")
}

func TestWriteHandshake_refusesAnUnsafeName(t *testing.T) {
	t.Parallel()
	tests := []string{
		"../escape",
		"nested/path",
		"Has Spaces",
		"UPPER",
		"",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := packet.WriteHandshake(t.TempDir(), name, "package handshake\n", packet.StrengthExamples)
			assert.Error(t, err)
		})
	}
}

func TestVerifyHandshake_reportsOkWhenTheFileMatchesItsRecordedHash(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h, err := packet.WriteHandshake(dir, "spec_test", "package handshake\n", packet.StrengthExamples)
	require.NoError(t, err)

	ok, err := packet.VerifyHandshake(h)

	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyHandshake_reportsAnHonestMismatchWhenTheFileChangedRatherThanErroring(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h, err := packet.WriteHandshake(dir, "spec_test", "package handshake\n", packet.StrengthExamples)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(h.Path, []byte("package handshake\n\n// weakened\n"), 0o644))

	ok, err := packet.VerifyHandshake(h)

	require.NoError(t, err, "a content mismatch is an honest false, never an error")
	assert.False(t, ok)
}

// A name with no letters at all (still inside [a-z0-9_-]+) must still be
// accepted — the regex boundary is the character class, not "must contain a
// letter".
func TestWriteHandshake_acceptsANameWithNoLetters(t *testing.T) {
	t.Parallel()
	h, err := packet.WriteHandshake(t.TempDir(), "123-_456", "package handshake\n", packet.StrengthExamples)
	require.NoError(t, err)
	assert.Contains(t, h.Path, "123-_456.go")
}

// A pre-existing FILE (not directory) at the handshake path is a real, if
// unusual, authoring-time failure — WriteHandshake must report it rather
// than panic or silently succeed.
func TestWriteHandshake_errorsWhenTheHandshakeDirectoryPathIsBlockedByAFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(packet.HandshakePath(dir), []byte("not a directory"), 0o644))

	_, err := packet.WriteHandshake(dir, "spec_test", "package handshake\n", packet.StrengthExamples)

	assert.Error(t, err)
}

// A pre-existing DIRECTORY at the exact file path WriteHandshake means to
// write is the write-side counterpart of the above.
func TestWriteHandshake_errorsWhenTheTargetFilePathIsBlockedByADirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(packet.HandshakePath(dir), "spec_test.go"), 0o755))

	_, err := packet.WriteHandshake(dir, "spec_test", "package handshake\n", packet.StrengthExamples)

	assert.Error(t, err)
}

func TestVerifyHandshake_errorsWhenTheFileIsGone(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h, err := packet.WriteHandshake(dir, "spec_test", "package handshake\n", packet.StrengthExamples)
	require.NoError(t, err)
	require.NoError(t, os.Remove(h.Path))

	ok, err := packet.VerifyHandshake(h)

	assert.Error(t, err)
	assert.False(t, ok)
}
