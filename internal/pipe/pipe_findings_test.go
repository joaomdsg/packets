package pipe_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/pipe"
)

// The mutation oracle finds surviving mutants — the honest test gaps a reviewer
// should see as "question:" threads — but the cycle consumed them only to derive
// the survivor SET and then discarded them, so no surface could ever show the
// per-line finding messages. The cycle must carry the fix revision's oracle
// findings up in its result, so the review surface can render them.
func TestRunCatchCycle_carriesTheFixRevisionsSurvivingFindings(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base: weak test lets >= survive")
	// The fix changes an UNRELATED file and leaves adult.go + the WEAK test
	// untouched — so the anchored line stays (reanchor Same) and the `>=` mutant
	// still SURVIVES at the fix: an honest, still-open test gap the reviewer should
	// see as a question.
	write(t, dir, "notes.txt", "unrelated change\n")
	fix := commitAll(t, dir, "fix: unrelated file, adult.go + weak test unchanged")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)

	require.NotEmpty(t, res.Findings, "the cycle exposes the fix revision's oracle findings (they no longer die inside the cycle)")
	var sawSurvivor bool
	for _, f := range res.Findings {
		if f.Outcome == mutation.Survived && f.File == "adult.go" && f.Line == 4 {
			sawSurvivor = true
		}
	}
	assert.True(t, sawSurvivor, "a mutant that survived the fix's tests is carried as a finding — the review question the surface will render")
}

// When the anchor does not survive to the fix (Outdated/LostViaRename), no fix
// oracle runs, so there are no honest fix-revision findings — the cycle carries
// NONE, and the review surface will render no question threads anchored to a line
// the fix may have edited away. This is the refactoring-safety guard.
func TestRunCatchCycle_carriesNoFindingsWhenTheAnchorIsLost(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	// Editing the file around the function makes the reanchor treat the anchored
	// line as Outdated — the fix oracle never runs at it.
	write(t, dir, "adult.go", adultGo+"\n// edited\n")
	fix := commitAll(t, dir, "fix: edit shifts/outdates the anchored line")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	require.Empty(t, res.Findings, "a lost anchor carries no fix-revision findings — no thread anchors to an edited-away line")
}
