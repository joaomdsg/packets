package settle_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/settle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The handshake is authored independently of the agent's turn — a turn that
// touches it, however it touches it, must never land.
func TestSettle_blocksATurnThatAddsAFileUnderHandshake(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "handshake"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "spec_test.go"), []byte("package handshake\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "sneak in a handshake edit")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	require.NotEmpty(t, res.PathBlocks)
	assert.Equal(t, "handshake/spec_test.go", res.PathBlocks[0].File)
	assert.NotEmpty(t, res.PathBlocks[0].Rule)
	assert.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"))
}

func TestSettle_blocksATurnThatModifiesAnExistingHandshakeFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "handshake"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "spec_test.go"), []byte("package handshake\n\nfunc TestA(t *testing.T){}\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "author the handshake")
	before := runGit(t, dir, "rev-parse", "HEAD")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "spec_test.go"), []byte("package handshake\n\nfunc TestA(t *testing.T){ /* weakened */ }\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "weaken the handshake")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	require.Len(t, res.PathBlocks, 1, "a modified file's path appears in both the +++ and --- headers of one diff — it must be reported once, not twice")
	assert.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"))
}

func TestSettle_blocksATurnThatDeletesAHandshakeFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "handshake"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "spec_test.go"), []byte("package handshake\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "author the handshake")

	require.NoError(t, os.Remove(filepath.Join(dir, "handshake", "spec_test.go")))

	res, err := settle.Settle(context.Background(), dir, "delete the handshake")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.NotEmpty(t, res.PathBlocks)
}

func TestSettle_blocksATurnThatTouchesANestedHandshakePath(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "handshake", "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "sub", "deep_test.go"), []byte("package sub\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add nested handshake file")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	require.NotEmpty(t, res.PathBlocks)
	assert.Equal(t, "handshake/sub/deep_test.go", res.PathBlocks[0].File)
}

// A path that merely shares the "handshake" PREFIX in its name (no directory
// boundary) is a different file entirely — the glob is a directory-prefix
// match, never a substring match.
func TestSettle_doesNotBlockAFileWhoseNameOnlySharesTheHandshakePrefix(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake-notes.md"), []byte("# notes\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add notes")
	require.NoError(t, err)
	require.True(t, res.Committed)
	assert.Empty(t, res.PathBlocks)
}

// An untouched handshake/ directory sitting alongside an edited unrelated
// file must never block — the deny-rule scopes to what THIS turn's diff
// touches, not to the mere existence of a protected path in the tree.
func TestSettle_doesNotBlockAnUnrelatedFileWhenHandshakeDirIsUntouched(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "handshake"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "spec_test.go"), []byte("package handshake\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "author the handshake")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated.go"), []byte("package p\n\nfunc H() int { return 3 }\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "unrelated change")
	require.NoError(t, err)
	require.True(t, res.Committed)
	assert.Empty(t, res.PathBlocks)
}

// A secret hit AND a path-deny hit can both be true of the same turn; either
// alone must block, and the block must not depend on evaluation order.
func TestSettle_blocksOnPathDenyEvenWhenNoSecretIsPresent(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "handshake"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "clean_test.go"), []byte("package handshake\n\nfunc TestClean(t *testing.T){}\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "no secret, still denied")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.Empty(t, res.Secrets, "no secret pattern here — the block must come from the path rule alone")
	assert.NotEmpty(t, res.PathBlocks)
}

// When a turn BOTH leaks a secret and touches the handshake, both hit lists
// must surface — one scan short-circuiting the other would hide a finding
// from the reviewer.
func TestSettle_surfacesBothSecretAndPathBlocksWhenATurnHasBoth(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "handshake"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "spec_test.go"), []byte("package handshake\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "creds.txt"), []byte("aws_key = AKIAIOSFODNN7EXAMPLE\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "leak and touch the handshake")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.NotEmpty(t, res.Secrets, "the secret hit must still surface")
	assert.NotEmpty(t, res.PathBlocks, "the path-deny hit must still surface")
}

// A file literally named "handshake" (no trailing slash — a FILE, not the
// protected directory) must not match the directory-prefix glob.
func TestSettle_doesNotBlockAFileLiterallyNamedHandshake(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake"), []byte("not a directory\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add a file named handshake")
	require.NoError(t, err)
	require.True(t, res.Committed)
	assert.Empty(t, res.PathBlocks)
}
