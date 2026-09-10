package orchestrator_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/orchestrator"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v\n%s", args, out)
	return strings.TrimSpace(string(out))
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\ntwo\nthree\n"), 0o644))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "base")
	return dir
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func hasFile(out orchestrator.TurnOutcome, path string) bool {
	for _, f := range out.Diff.Files {
		if f.Path == path {
			return true
		}
	}
	return false
}

func TestSettleTurn_mintsRevisionWithDiffAndStatsForChangedTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")
	write(t, dir, "f.txt", "one\nTWO\nthree\n")

	out, err := orchestrator.SettleTurn(context.Background(), dir, base, "turn")
	require.NoError(t, err)
	require.True(t, out.Minted, "a changed turn must mint a revision")
	assert.Equal(t, runGit(t, dir, "rev-parse", "HEAD"), out.SHA, "SHA must equal new HEAD")
	assert.True(t, hasFile(out, "f.txt"), "diff must include f.txt, got %+v", out.Diff.Files)
	assert.Equal(t, 1, out.Added, "one line modified")
	assert.Equal(t, 1, out.Deleted, "one line modified")
	assert.Empty(t, out.Secrets, "clean change must surface no secrets")
}

func TestSettleTurn_mintsNothingForNoEditTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	out, err := orchestrator.SettleTurn(context.Background(), dir, base, "turn")
	require.NoError(t, err)
	assert.False(t, out.Minted, "a no-edit turn must not mint a revision")
	assert.Empty(t, out.SHA, "no revision means no SHA")
	assert.Empty(t, out.Diff.Files, "no revision means no diff")
	assert.Equal(t, base, runGit(t, dir, "rev-parse", "HEAD"), "HEAD must not move on a no-edit turn")
}

func TestSettleTurn_blocksAndSurfacesSecretTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")
	write(t, dir, "conf.env", "API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n")

	out, err := orchestrator.SettleTurn(context.Background(), dir, base, "turn")
	require.NoError(t, err, "a blocked secret is surfaced, not an error")
	assert.False(t, out.Minted, "a secret-bearing turn must not mint a revision")
	assert.NotEmpty(t, out.Secrets, "the secret must be surfaced in TurnOutcome.Secrets")
	assert.Empty(t, out.Diff.Files, "a blocked secret means no revision, so no diff")
	assert.Equal(t, base, runGit(t, dir, "rev-parse", "HEAD"), "HEAD must not move despite a blocked secret")
}

func TestSettleTurn_computesDiffAgainstGivenBaseRevNotHeadParent(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base1 := runGit(t, dir, "rev-parse", "HEAD")
	// A second commit changes line 1; this is NOT the base we pass.
	write(t, dir, "f.txt", "ONE\ntwo\nthree\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "base2")
	write(t, dir, "f.txt", "ONE\ntwo\nTHREE\n")

	out, err := orchestrator.SettleTurn(context.Background(), dir, base1, "turn")
	require.NoError(t, err)
	require.True(t, out.Minted, "a changed turn must mint a revision")
	// base1..newSHA spans both the line-1 change (commit base2) and the
	// line-3 change (the turn); HEAD~1..newSHA would show only one.
	assert.Equal(t, 2, out.Added, "diff vs base1 must span both changes")
	assert.Equal(t, 2, out.Deleted, "diff vs base1 must span both changes")
}

func TestSettleTurn_propagatesUnderlyingGitFailure(t *testing.T) {
	t.Parallel()
	_, err := orchestrator.SettleTurn(context.Background(), filepath.Join(t.TempDir(), "does-not-exist"), "deadbeef", "turn")
	assert.Error(t, err, "expected an error when the repo dir does not exist")
}

func TestSettleTurn_propagatesDiffFailureAfterCommit(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.txt", "one\nTWO\nthree\n")

	out, err := orchestrator.SettleTurn(context.Background(), dir, "nonexistent-base-rev", "turn")
	require.Error(t, err, "expected an error when diff.Compute fails on a bad baseRev")
	assert.False(t, out.Minted, "a diff failure must not yield a minted outcome")
}

func TestSettleTurn_sumsStatsAcrossAllChangedFiles(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")
	write(t, dir, "f.txt", "one\nTWO\nthree\n")
	write(t, dir, "g.txt", "alpha\nbeta\n")

	out, err := orchestrator.SettleTurn(context.Background(), dir, base, "turn")
	require.NoError(t, err)
	require.True(t, out.Minted, "a changed turn must mint a revision")
	require.Len(t, out.Diff.Files, 2, "want 2 changed files")
	assert.Equal(t, 3, out.Added, "totals across files")
	assert.Equal(t, 1, out.Deleted, "totals across files")
}

// An errored/timed-out turn must roll the working tree back to the prior revision so its
// partial edits are discarded (DESIGN §1183 / timeout row) — otherwise settle's whole-tree
// `git add -A` would sweep the residue into a LATER turn's mint. DiscardWorkingTree
// restores tracked files, removes untracked ones, and leaves HEAD at the revision.
func TestDiscardWorkingTree_rollsBackPartialEditsToTheRevision(t *testing.T) {
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	write(t, dir, "f.txt", "partial\n") // tracked file modified
	write(t, dir, "new.txt", "junk\n")  // untracked file created

	require.NoError(t, orchestrator.DiscardWorkingTree(context.Background(), dir, base))

	got, err := os.ReadFile(filepath.Join(dir, "f.txt"))
	require.NoError(t, err)
	assert.Equal(t, "one\ntwo\nthree\n", string(got), "the modified tracked file is restored to the revision")
	_, statErr := os.Stat(filepath.Join(dir, "new.txt"))
	assert.True(t, os.IsNotExist(statErr), "the untracked partial file is removed")
	assert.Empty(t, runGit(t, dir, "status", "--porcelain"), "the working tree is clean after the rollback")
	assert.Equal(t, base, runGit(t, dir, "rev-parse", "HEAD"), "HEAD stays at the prior revision")
}

// Rolling back an already-clean tree is a benign no-op (the trailing-errored-turn case).
func TestDiscardWorkingTree_isANoOpOnACleanTree(t *testing.T) {
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")
	require.NoError(t, orchestrator.DiscardWorkingTree(context.Background(), dir, base))
	assert.Equal(t, base, runGit(t, dir, "rev-parse", "HEAD"))
	assert.Empty(t, runGit(t, dir, "status", "--porcelain"))
}
