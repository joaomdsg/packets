package settle_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/settle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests exercise real git because the settle guard's whole job is a git
// behaviour (commit vs. "nothing to commit").
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v:\n%s", args, out)
	return strings.TrimSpace(string(out))
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.go"), []byte("package p\n\nfunc F() int { return 1 }\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "base")
	return dir
}

func TestSettle_mintsNoRevisionForNoEditTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")

	res, err := settle.Settle(context.Background(), dir, "turn")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.Empty(t, res.SHA)
	assert.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"))
}

func TestSettle_mintsRevisionForChangedTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.go"), []byte("package p\n\nfunc F() int { return 2 }\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "turn")
	require.NoError(t, err)
	require.True(t, res.Committed)
	head := runGit(t, dir, "rev-parse", "HEAD")
	assert.Equal(t, head, res.SHA)
	assert.NotEqual(t, before, head)
	assert.Empty(t, runGit(t, dir, "status", "--porcelain"))
	assert.Equal(t, "turn", runGit(t, dir, "log", "-1", "--format=%s"))
}

func TestSettle_mintsNoRevisionForNetRevertedStagedChange(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")

	// Stage a change, then revert the worktree to HEAD's content. Index now
	// holds a change vs HEAD, worktree matches HEAD -> porcelain shows "MM"
	// (nothing to commit once everything is staged).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.go"), []byte("package p\n\nfunc F() int { return 7 }\n"), 0o644))
	runGit(t, dir, "add", "base.go")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.go"), []byte("package p\n\nfunc F() int { return 1 }\n"), 0o644))
	require.NotEmpty(t, runGit(t, dir, "status", "--porcelain"))

	res, err := settle.Settle(context.Background(), dir, "turn")
	require.NoError(t, err)
	assert.False(t, res.Committed)
	assert.Empty(t, res.SHA)
	assert.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"))
}

func TestSettle_committsNewUntrackedFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.go"), []byte("package p\n\nfunc G() int { return 9 }\n"), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add new file")
	require.NoError(t, err)
	require.True(t, res.Committed)
	runGit(t, dir, "cat-file", "-e", "HEAD:new.go")
	assert.Empty(t, runGit(t, dir, "status", "--porcelain"))
}
