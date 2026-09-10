package packet_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
)

// writeHandshakeFixture seeds dir with a minimal Go module plus a handshake
// package whose one test either passes or fails, per want.
func writeHandshakeFixture(t *testing.T, dir string, passing bool) {
	t.Helper()
	writeGauntletFile(t, dir, "go.mod", "module gauntlethandshakefixture\n\ngo 1.21\n")
	writeGauntletFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	body := "t.Fatal(\"the handshake's own contract failed\")"
	if passing {
		body = "_ = t"
	}
	writeGauntletFile(t, dir, "handshake/spec_test.go",
		"package handshake\n\nimport \"testing\"\n\nfunc TestSpec(t *testing.T) { "+body+" }\n")
}

func TestRunHandshakeGate_isNotRunWhenNoHandshakeWasAuthored(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir)
	rev := commitGauntletAll(t, dir, "clean")

	gate := packet.RunHandshakeGate(context.Background(), dir, rev, "")

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.Equal(t, "no handshake authored", gate.Detail)
}

func TestRunHandshakeGate_passesWhenTheAuthoredHandshakeTestsPass(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	writeHandshakeFixture(t, dir, true)
	rev := commitGauntletAll(t, dir, "clean")

	gate := packet.RunHandshakeGate(context.Background(), dir, rev, "handshake")

	assert.Equal(t, packet.GatePassed, gate.Status)
}

func TestRunHandshakeGate_failsWithATruncatedDetailWhenTheHandshakeTestsFail(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	writeHandshakeFixture(t, dir, false)
	rev := commitGauntletAll(t, dir, "failing handshake")

	gate := packet.RunHandshakeGate(context.Background(), dir, rev, "handshake")

	assert.Equal(t, packet.GateFailed, gate.Status)
	assert.LessOrEqual(t, len(gate.Detail), 200+len("go test: "), "the failing run's output must be truncated, never a full log dump")
	assert.Contains(t, gate.Detail, "handshake's own contract failed")
}

// The handshake dir is missing at this fix revision entirely — a REAL
// finding (the human authored one, so its absence is not "unmeasured"),
// never a silent not-run.
func TestRunHandshakeGate_failsWhenTheHandshakePackageIsMissingAtThisRevision(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	cleanGoModule(t, dir) // no handshake/ dir at all
	rev := commitGauntletAll(t, dir, "no handshake dir")

	gate := packet.RunHandshakeGate(context.Background(), dir, rev, "handshake")

	assert.Equal(t, packet.GateFailed, gate.Status)
	assert.Equal(t, "handshake test package not found at this revision", gate.Detail)
}

// A handshake package that fails to COMPILE (not just fails at runtime) is
// the same honest finding as a missing directory — never a not-run.
func TestRunHandshakeGate_failsWhenTheHandshakePackageDoesNotCompile(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	writeGauntletFile(t, dir, "go.mod", "module gauntlethandshakebroken\n\ngo 1.21\n")
	writeGauntletFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	writeGauntletFile(t, dir, "handshake/spec_test.go", "package handshake\n\nfunc TestSpec(t *testing.T\n")
	rev := commitGauntletAll(t, dir, "broken handshake")

	gate := packet.RunHandshakeGate(context.Background(), dir, rev, "handshake")

	assert.Equal(t, packet.GateFailed, gate.Status)
	assert.Equal(t, "handshake test package not found at this revision", gate.Detail)
}

// RunHandshakeGate runs the revision the ORDER's fix rev names, not
// repoDir's live working tree, and not HEAD when fixRev is older — same
// worktree-materialize discipline as RunBuildVetGate.
func TestRunHandshakeGate_gatesTheGivenRevisionNotHeadOrTheDirtyWorkingTree(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	writeHandshakeFixture(t, dir, true)
	cleanRev := commitGauntletAll(t, dir, "clean handshake")

	writeHandshakeFixture(t, dir, false) // HEAD now fails
	commitGauntletAll(t, dir, "break the handshake")

	gate := packet.RunHandshakeGate(context.Background(), dir, cleanRev, "handshake")

	assert.Equal(t, packet.GatePassed, gate.Status, "the requested older revision's handshake passes even though HEAD's is broken")
}

func TestRunHandshakeGate_isNotRunWithNoFixRevisionRatherThanGuessing(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	writeHandshakeFixture(t, dir, true)
	commitGauntletAll(t, dir, "clean")

	gate := packet.RunHandshakeGate(context.Background(), dir, "", "handshake")

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.Equal(t, "no revision to build yet", gate.Detail)
}

func TestRunHandshakeGate_isNotRunWhenRepoDirIsNotAGitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // no `git init` — a bare directory

	gate := packet.RunHandshakeGate(context.Background(), dir, "deadbeef", "handshake")

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.Equal(t, "no revision to build yet", gate.Detail)
}

// Not t.Parallel(): t.Setenv forbids running in parallel with other tests
// that might also mutate the environment.
func TestRunHandshakeGate_namesAScratchDirFailureDistinctlyFromNoRevision(t *testing.T) {
	dir := initGauntletRepo(t)
	writeHandshakeFixture(t, dir, true)
	rev := commitGauntletAll(t, dir, "clean")

	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "does-not-exist"))

	gate := packet.RunHandshakeGate(context.Background(), dir, rev, "handshake")

	assert.Equal(t, packet.GateNotRun, gate.Status)
	assert.NotEqual(t, "no revision to build yet", gate.Detail,
		"a scratch-dir failure is a different honest cause than 'nothing to build' — collapsing them would mislead about WHY nothing ran")
}

func TestRunHandshakeGate_isNotRunWhenTheFixRevisionDoesNotResolve(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	writeHandshakeFixture(t, dir, true)
	commitGauntletAll(t, dir, "clean")

	gate := packet.RunHandshakeGate(context.Background(), dir, "0000000000000000000000000000000000dead", "handshake")

	assert.Equal(t, packet.GateNotRun, gate.Status, "an unresolvable revision is an honest absence, never a fabricated pass or fail")
	assert.NotEmpty(t, gate.Detail)
}

// A genuine test failure whose OWN message happens to contain the literal
// text Go uses to mark a build/setup failure must still be classified as a
// real failure (the generic truncated tail), never misclassified as a
// missing/uncompileable package — proving the detection keys off go test's
// actual package-level outcome, not a naive substring search over output
// that could collide with a test's own failure text.
func TestRunHandshakeGate_doesNotMisclassifyARealFailureWhoseMessageContainsTheSetupFailedMarkerText(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	writeGauntletFile(t, dir, "go.mod", "module gauntlethandshakecollision\n\ngo 1.21\n")
	writeGauntletFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	writeGauntletFile(t, dir, "handshake/spec_test.go",
		"package handshake\n\nimport \"testing\"\n\nfunc TestSpec(t *testing.T) { t.Fatal(\"looks like [setup failed] but is not\") }\n")
	rev := commitGauntletAll(t, dir, "real failure with lookalike text")

	gate := packet.RunHandshakeGate(context.Background(), dir, rev, "handshake")

	assert.Equal(t, packet.GateFailed, gate.Status)
	assert.Contains(t, gate.Detail, "looks like", "the real failure's own message must surface, not the fixed missing-package string")
	assert.NotEqual(t, "handshake test package not found at this revision", gate.Detail)
}

func TestRunHandshakeGate_leavesNoWorktreeBehindAfterRunning(t *testing.T) {
	t.Parallel()
	dir := initGauntletRepo(t)
	writeHandshakeFixture(t, dir, true)
	rev := commitGauntletAll(t, dir, "clean")
	before := worktreeCount(t, dir)

	packet.RunHandshakeGate(context.Background(), dir, rev, "handshake")

	assert.Equal(t, before, worktreeCount(t, dir))
}
