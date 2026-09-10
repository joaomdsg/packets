package settle_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/settle"
)

// A .gitattributes entry coercing a path to binary for diff (e.g. `*.env -diff`) makes
// `git diff` render a PLAINTEXT secret file as "Binary files differ" — zero added lines —
// so the added-line scanner would miss the secret and let it into history. The settle
// gate must force a textual diff so an attribute-coerced file is still scanned (RISKS.md
// CRITICAL: a secret never enters history at any settle).
func TestSettle_scansSecretInAttributeCoercedBinaryFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.env -diff\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "attrs")
	before := runGit(t, dir, "rev-parse", "HEAD")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "leak.env"), []byte("API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "leak")
	require.NoError(t, err)
	assert.False(t, res.Committed, "a coerced-binary secret must still block the commit")
	require.NotEmpty(t, res.Secrets, "the secret must be caught despite the -diff attribute")
	assert.True(t, hasHit(res.Secrets, "leak.env", "secret-assignment"),
		"the hit must anchor to the coerced file; got %v", res.Secrets)
	assert.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"), "no revision minted when a secret is blocked")
}

// The pre-push land gate (ScanCommitRange) has the same exposure: a secret in an
// attribute-coerced binary file must still be caught before the bytes leave the machine.
func TestScanCommitRange_catchesSecretInAttributeCoercedBinaryFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.env -diff\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "attrs")
	base := runGit(t, dir, "rev-parse", "HEAD")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "leak.env"), []byte("API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "smuggle a key in a coerced file")
	rev := runGit(t, dir, "commit-tree", "HEAD^{tree}", "-p", base, "-m", "squash")

	hits, err := settle.ScanCommitRange(context.Background(), dir, base, rev)
	require.NoError(t, err)
	assert.True(t, hasHit(hits, "leak.env", "secret-assignment"),
		"the pre-push gate must catch a coerced-binary secret; got %v", hits)
}
