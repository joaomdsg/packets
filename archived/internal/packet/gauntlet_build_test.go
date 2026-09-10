package packet_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A command killed by the context deadline must report NotRun ("timed out"),
// never a fabricated Failed claim — a kill is not proof the command would
// have failed given more time, and this package's whole "never fabricate a
// metric" invariant forbids dressing up "we don't know" as "it failed".
func TestGateForExecError_reportsATimeoutAsNotRunNeverAFabricatedFailure(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // deterministic: the context IS done, no timing race

	gate := packet.GateForExecError(ctx, "go build", "some output that happened to be captured before the kill", errors.New("signal: killed"))

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.Contains(t, gate.Detail, "timed out", "naming which command timed out is honest — the DETAIL must say so")
	assert.NotContains(t, gate.Detail, "some output that happened to be captured before the kill",
		"a timeout's pre-kill output is not evidence of failure — it must never be quoted as if it were")
}

// A real (non-timeout) command failure still reports Failed, naming the
// command and a truncated tail of its output — the existing, honest path.
func TestGateForExecError_reportsARealFailureWhenTheContextIsNotDone(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	gate := packet.GateForExecError(ctx, "go build", "main.go:3: syntax error", errors.New("exit status 1"))

	assert.Equal(t, packet.GateFailed, gate.Status)
	assert.Equal(t, "go build: main.go:3: syntax error", gate.Detail)
}

func runGauntletGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v:\n%s", args, out)
	return strings.TrimSpace(string(out))
}

func initGauntletRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGauntletGit(t, dir, "init", "-q")
	runGauntletGit(t, dir, "config", "user.email", "t@t")
	runGauntletGit(t, dir, "config", "user.name", "t")
	return dir
}

func writeGauntletFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func commitGauntletAll(t *testing.T, dir, msg string) string {
	t.Helper()
	runGauntletGit(t, dir, "add", "-A")
	runGauntletGit(t, dir, "commit", "-qm", msg)
	return runGauntletGit(t, dir, "rev-parse", "HEAD")
}

// cleanGoModule seeds dir with a minimal, valid Go module: a go.mod and one
// package that builds and vets clean.
func cleanGoModule(t *testing.T, dir string) {
	t.Helper()
	writeGauntletFile(t, dir, "go.mod", "module gauntletfixture\n\ngo 1.21\n")
	writeGauntletFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
}

func TestRunBuildVetGate_passesOnACleanBuildingRevision(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	rev := commitGauntletAll(t, dir, "clean")

	gate := packet.RunBuildVetGate(context.Background(), dir, rev)

	assert.Equal(t, packet.GatePassed, gate.Status)
}

func TestRunBuildVetGate_failsWithATruncatedDetailOnASyntaxError(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	commitGauntletAll(t, dir, "clean")

	writeGauntletFile(t, dir, "main.go", "package main\n\nfunc main() { this is not valid go\n")
	rev := commitGauntletAll(t, dir, "break it")

	gate := packet.RunBuildVetGate(context.Background(), dir, rev)

	assert.Equal(t, packet.GateFailed, gate.Status)
	assert.LessOrEqual(t, len(gate.Detail), 200+len("go build: "), "the failing command's output must be truncated, never a full log dump")
	assert.Contains(t, gate.Detail, "go build", "a syntax error fails at build time, before vet ever runs")
}

func TestRunBuildVetGate_failsOnAGoVetIssueDistinctFromABuildFailure(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	commitGauntletAll(t, dir, "clean")

	// Compiles fine (no build error) but go vet flags the Printf/argument mismatch.
	writeGauntletFile(t, dir, "main.go", "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Printf(\"%d\\n\", \"not-a-number\") }\n")
	rev := commitGauntletAll(t, dir, "bad vet")

	gate := packet.RunBuildVetGate(context.Background(), dir, rev)

	assert.Equal(t, packet.GateFailed, gate.Status)
	assert.Contains(t, gate.Detail, "go vet", "a build-clean, vet-dirty revision must name vet as the failing command, not build")
}

func TestRunBuildVetGate_buildsTheGivenRevisionEvenWhenItIsOlderThanHead(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	cleanRev := commitGauntletAll(t, dir, "clean")

	writeGauntletFile(t, dir, "main.go", "package main\n\nfunc main() { this is not valid go\n")
	commitGauntletAll(t, dir, "break it") // HEAD is now broken

	gate := packet.RunBuildVetGate(context.Background(), dir, cleanRev)

	assert.Equal(t, packet.GatePassed, gate.Status, "the requested older revision is clean even though HEAD is broken — the gate must not silently build HEAD instead")
}

func TestRunBuildVetGate_buildsTheCommittedRevisionNotADirtyWorkingTree(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	commitGauntletAll(t, dir, "clean")

	writeGauntletFile(t, dir, "main.go", "package main\n\nfunc main() { this is not valid go\n")
	brokenRev := commitGauntletAll(t, dir, "break it")

	// Dirty the working tree back to something that builds, WITHOUT committing —
	// the requested revision's own committed content is still broken.
	cleanGoModule(t, dir)

	gate := packet.RunBuildVetGate(context.Background(), dir, brokenRev)

	assert.Equal(t, packet.GateFailed, gate.Status, "the gate must build the WORKTREE checkout of brokenRev, not repoDir's live (dirty) working tree")
}

func TestRunBuildVetGate_isNotRunWhenTheRevisionDoesNotResolve(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	commitGauntletAll(t, dir, "clean")

	gate := packet.RunBuildVetGate(context.Background(), dir, "0000000000000000000000000000000000dead")

	assert.Equal(t, packet.GateNotRun, gate.Status, "an unresolvable revision is an honest absence, never a fabricated pass or fail")
	assert.NotEmpty(t, gate.Detail)
}

func TestRunBuildVetGate_stillRunsAndPassesOnANonGoOnlyRevision(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	commitGauntletAll(t, dir, "clean")

	writeGauntletFile(t, dir, "README.md", "# fixture\n")
	rev := commitGauntletAll(t, dir, "docs only")

	gate := packet.RunBuildVetGate(context.Background(), dir, rev)

	assert.Equal(t, packet.GatePassed, gate.Status, "a doc-only revision still has a Go module that builds and vets clean")
}

func TestRunBuildVetGate_isNotRunWithNoFixRevisionRatherThanGuessing(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	commitGauntletAll(t, dir, "clean")

	gate := packet.RunBuildVetGate(context.Background(), dir, "")

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.Equal(t, "no revision to build yet", gate.Detail)
}

func TestRunBuildVetGate_isNotRunWhenRepoDirIsNotAGitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // no `git init` — a bare directory

	gate := packet.RunBuildVetGate(context.Background(), dir, "deadbeef")

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.Equal(t, "no revision to build yet", gate.Detail)
}

func TestRunBuildVetGate_isNotRunWhenRepoDirDoesNotExistAtAll(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	gate := packet.RunBuildVetGate(context.Background(), missing, "deadbeef")

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.Equal(t, "no revision to build yet", gate.Detail)
}

// Not t.Parallel(): t.Setenv forbids running in parallel with other tests
// that might also mutate the environment.
func TestRunBuildVetGate_namesAScratchDirFailureDistinctlyFromNoRevision(t *testing.T) {
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	rev := commitGauntletAll(t, dir, "clean")

	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "does-not-exist"))

	gate := packet.RunBuildVetGate(context.Background(), dir, rev)

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.NotEqual(t, "no revision to build yet", gate.Detail,
		"a scratch-dir failure is a different honest cause than 'nothing to build' — collapsing them would mislead about WHY nothing ran")
}

// worktreeCount returns how many worktrees git currently tracks for dir's repo
// (the "worktree " lines `git worktree list --porcelain` emits, one per
// worktree including the main one) — used to pin that RunBuildVetGate never
// leaks a worktree behind after it returns.
func worktreeCount(t *testing.T, dir string) int {
	t.Helper()
	out := runGauntletGit(t, dir, "worktree", "list", "--porcelain")
	n := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			n++
		}
	}
	return n
}

func TestRunBuildVetGate_leavesNoWorktreeBehindAfterRunning(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	rev := commitGauntletAll(t, dir, "clean")
	before := worktreeCount(t, dir)

	packet.RunBuildVetGate(context.Background(), dir, rev)

	assert.Equal(t, before, worktreeCount(t, dir), "the throwaway worktree materialized to build/vet the revision must be removed before returning")
}
