package pipe_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/pipe"
)

// Answering a review question means: write a test that KILLS the surviving mutant,
// and the question disappears. RerunWithTestOverlay re-runs the oracle at the fix
// revision with the reviewer's test injected into the worktree, scoped to the
// anchored line — so the surface can tell whether the answer actually constrains
// the line. A killing test removes the finding; a weak one leaves it open. This is
// the diagnostic re-run the editable-answering flow is built on (no economy effect).
func TestRerunWithTestOverlay_reflectsWhetherTheReviewersTestKillsTheMutant(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	rev := commitAll(t, dir, "weak test lets the >= mutant survive")

	// Baseline / a non-killing answer: the weak test (only asserts 25) leaves the
	// `>=`→`>` mutant on line 4 alive — the question stays OPEN.
	open, err := pipe.RerunWithTestOverlay(
		context.Background(), dir, rev, "adult.go", 4, goTestCmd,
		map[string]string{"adult_test.go": weakTest},
	)
	require.NoError(t, err)
	require.NotEmpty(t, open, "a test that doesn't constrain line 4 leaves the mutant alive — the question stays open")

	// A killing answer: the strong test asserts the 18 boundary, so the `>=`→`>`
	// mutant fails it — the mutant dies and the finding disappears (answered).
	answered, err := pipe.RerunWithTestOverlay(
		context.Background(), dir, rev, "adult.go", 4, goTestCmd,
		map[string]string{"adult_test.go": strongTest},
	)
	require.NoError(t, err)
	require.Empty(t, answered, "the reviewer's boundary test kills the mutant — the question is answered")
}

// The overlay carries reviewer-influenced file paths, so an entry that tries to
// escape the worktree (e.g. "../../x") must be REJECTED before any write — never
// allowed to clobber a file outside the throwaway worktree. NOT parallel-sensitive
// (own temp repo), but kept simple.
func TestRerunWithTestOverlay_rejectsAnOverlayPathThatEscapesTheWorktree(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	rev := commitAll(t, dir, "base")

	_, err := pipe.RerunWithTestOverlay(
		context.Background(), dir, rev, "adult.go", 4, goTestCmd,
		map[string]string{"../../ESCAPED.txt": "pwned"},
	)
	require.Error(t, err, "an overlay path escaping the worktree is rejected before any write")
}
