package settle_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/agntpr/internal/settle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettle_blocksRevisionOnStagedPrivateKey(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")
	const pem = "-----BEGIN RSA PRIVATE KEY-----\nMIIBOgIBAAExample\n-----END RSA PRIVATE KEY-----\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "key.pem"), []byte(pem), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add key")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	require.NotEmpty(t, res.Secrets)
	var found bool
	for _, h := range res.Secrets {
		if h.File == "key.pem" {
			found = true
			assert.NotEmpty(t, h.Rule)
		}
	}
	assert.Truef(t, found, "expected a secret hit on key.pem, got %+v", res.Secrets)
	assert.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"))
}

func TestSettle_blocksRevisionOnStagedAWSKey(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "creds.txt"), []byte("aws_key = AKIAIOSFODNN7EXAMPLE\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add creds")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.NotEmpty(t, res.Secrets)
}

func TestSettle_blocksRevisionOnStagedSecretAssignment(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.env"), []byte("API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add config")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.NotEmpty(t, res.Secrets)
}

func TestSettle_anchorsSecretHitToAddedLineNumber(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	// The secret sits on the third line of a new file, so the hit's Line must be 3.
	content := "# config\ndebug=true\nAPI_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "conf.env"), []byte(content), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add conf")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	require.NotEmpty(t, res.Secrets)
	var line int
	for _, h := range res.Secrets {
		if h.File == "conf.env" {
			line = h.Line
		}
	}
	assert.Equalf(t, 3, line, "Secrets=%+v", res.Secrets)
}

func TestSettle_committsCleanSourceWithNoSecretHits(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\n\nfunc Add(a, b int) int { return a + b }\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add ok")
	require.NoError(t, err)
	require.True(t, res.Committed)
	assert.Empty(t, res.Secrets)
}

func TestSettle_scansSecretsUnderHostileGitConfig(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	// color.diff=always colorizes the diff with ANSI escapes (added lines no
	// longer start with a bare "+") and diff.noprefix rewrites the "+++ b/<path>"
	// header. Settle must force canonical diff output or the secret slips through.
	runGit(t, dir, "config", "color.diff", "always")
	runGit(t, dir, "config", "diff.noprefix", "true")
	before := runGit(t, dir, "rev-parse", "HEAD")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "leak.env"), []byte("API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "leak")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.NotEmpty(t, res.Secrets)
	assert.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"))
}

func TestSettle_doesNotFlagPreexistingSecretNotTouchedThisTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	// A secret already in history (committed directly, not via Settle) must not
	// be re-flagged, or every later turn would be blocked by it.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "old.txt"), []byte("aws_key = AKIAIOSFODNN7EXAMPLE\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "preexisting secret")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\n\nfunc Z() int { return 0 }\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "unrelated change")
	require.NoError(t, err)
	require.True(t, res.Committed)
	assert.Empty(t, res.Secrets)
}
