package settle_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/agntpr/internal/settle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettle_surfacesStagedBinaryButStillCommits(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 3, 0, 4, 5, 6, 7}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "note.txt"), []byte("just text\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add binary + text")
	require.NoError(t, err)
	require.True(t, res.Committed)

	found := false
	for _, a := range res.Artifacts {
		if a == "data.bin" {
			found = true
		}
		assert.NotEqualf(t, "note.txt", a, "a text file must not be flagged as an artifact, got %+v", res.Artifacts)
	}
	assert.Truef(t, found, "the binary data.bin must be surfaced in Artifacts, got %+v", res.Artifacts)

	runGit(t, dir, "cat-file", "-e", "HEAD:data.bin")
}

func TestSettle_surfacesNoArtifactsOnSecretBlockedTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 0, 3}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "conf.env"), []byte("API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "binary + secret")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.NotEmpty(t, res.Secrets)
	assert.Empty(t, res.Artifacts)
}

func TestSettle_surfacesModifiedBinaryFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 3, 0, 4}, 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "add binary")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 9, 8, 7, 0, 6, 5}, 0o644))

	res, err := settle.Settle(context.Background(), dir, "modify binary")
	require.NoError(t, err)
	require.True(t, res.Committed)
	found := false
	for _, a := range res.Artifacts {
		if a == "data.bin" {
			found = true
		}
	}
	assert.Truef(t, found, "a modified binary must be surfaced as an artifact, got %+v", res.Artifacts)
}

func TestSettle_doesNotSurfaceDeletedBinaryFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 3, 0, 4}, 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "add binary")
	// Delete the binary, adding a text file so there's something to commit.
	require.NoError(t, os.Remove(filepath.Join(dir, "data.bin")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "note.txt"), []byte("bye\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "delete binary")
	require.NoError(t, err)
	require.True(t, res.Committed)
	assert.Empty(t, res.Artifacts)
}

func TestSettle_surfacesBinaryPathWithSpaceIntact(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "my data.bin"), []byte{0, 1, 2, 0, 3}, 0o644))

	res, err := settle.Settle(context.Background(), dir, "add spaced binary")
	require.NoError(t, err)
	found := false
	for _, a := range res.Artifacts {
		if a == "my data.bin" {
			found = true
		}
	}
	assert.Truef(t, found, "a spaced binary path must surface intact, got %+v", res.Artifacts)
}

func TestSettle_surfacesNonASCIIBinaryPathUnquoted(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	name := "café.bin"
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte{0, 1, 2, 0, 3}, 0o644))

	res, err := settle.Settle(context.Background(), dir, "add non-ascii binary")
	require.NoError(t, err)
	found := false
	for _, a := range res.Artifacts {
		if a == name {
			found = true
		}
		assert.Falsef(t, strings.HasPrefix(a, "\""), "path must be surfaced raw, not git-quoted, got %q", a)
	}
	assert.Truef(t, found, "non-ascii binary path must surface raw as %q, got %+v", name, res.Artifacts)
}

func TestSettle_surfacesNoArtifactsOnTextOnlyTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# title\n\nbody\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add doc")
	require.NoError(t, err)
	require.True(t, res.Committed)
	assert.Empty(t, res.Artifacts)
}
