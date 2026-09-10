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

func writeCommit(t *testing.T, dir, file, content, msg string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", msg)
}

func TestScanHistory_findsASecretBuriedInAnOldCommit(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)

	// A secret enters in one commit, then later commits bury it under clean work
	// (it is gone from HEAD's working tree but lives forever in history).
	writeCommit(t, dir, "leak.txt", "aws_key = AKIAIOSFODNN7EXAMPLE\n", "oops")
	require.NoError(t, os.Remove(filepath.Join(dir, "leak.txt")))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "remove the file")
	writeCommit(t, dir, "clean.go", "package p\n", "more work")

	hits, err := settle.ScanHistory(context.Background(), dir)
	require.NoError(t, err)
	assert.True(t, hasHit(hits, "leak.txt", "aws-access-key-id"),
		"a secret anywhere in history must be detected, even if gone from HEAD; got %v", hits)
}

func hasHit(hits []settle.SecretHit, file, rule string) bool {
	for _, h := range hits {
		if h.File == file && h.Rule == rule {
			return true
		}
	}
	return false
}

func TestScanHistory_isCleanForAHistoryWithNoSecrets(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	writeCommit(t, dir, "more.go", "package p\n\nfunc G() int { return 2 }\n", "clean work")

	hits, err := settle.ScanHistory(context.Background(), dir)
	require.NoError(t, err)
	assert.Empty(t, hits, "a history with no secrets must scan clean")
}

// Security regression guard: the history scan must catch a secret even in a file a
// .gitattributes entry coerces to "binary" for diff. Without the pinned --text the
// `git log -p` for such a file would emit "Binary files differ" (no added lines) and the
// scanner would miss the secret living in history. (Parallel to the Settle/ScanCommitRange
// coercion guards.)
func TestScanHistory_catchesASecretInAnAttributeCoercedBinaryFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.env -diff\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "attrs")
	writeCommit(t, dir, "leak.env", "API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n", "smuggle a key in a coerced file")

	hits, err := settle.ScanHistory(context.Background(), dir)
	require.NoError(t, err)
	assert.True(t, hasHit(hits, "leak.env", "secret-assignment"),
		"the history scan must catch a coerced-binary secret; got %v", hits)
}
