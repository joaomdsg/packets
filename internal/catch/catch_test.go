package catch_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/mutation"
)

// A test-only fix on a STABLE line (operator inventory unchanged) that clears
// the line's survivors is the canonical confirmed catch — the whole economy
// rests on minting exactly this case.
func TestDetect_mintsCatchWhenStableLineSurvivorsCleared(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	assert.Equal(t, catch.Catch, catch.Detect(before, after))
}

// The load-bearing case: a fix that EDITS the anchored line and changes its
// operator alphabet makes the before/after survivor-sets live over different
// alphabets — the transition is ill-typed, so "same mutant killed" is incoherent
// and no catch may be minted.
func TestDetect_refusesCatchWhenFixChangesOperatorInventory(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">"}, Survivors: nil}
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after))
}

// A line that was already constrained (no survivors before) cannot yield a
// catch — there was nothing weak to strengthen.
func TestDetect_refusesCatchWhenLineAlreadyConstrained(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after))
}

// No-op churn (state identical before and after) must never mint a catch.
func TestDetect_refusesCatchForNoOpChurn(t *testing.T) {
	t.Parallel()
	state := catch.LineState{Inventory: []string{">=", "&&"}, Survivors: []string{">="}}
	assert.Equal(t, catch.NoCatch, catch.Detect(state, state))
}

// An operator-free line has no oracle signal at all; that must surface as a
// DISTINCT outcome, never silently collapse to NoCatch (which would
// under-credit operator-free code as "nothing caught").
func TestDetect_reportsNoOracleSignalForOperatorFreeLine(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: nil, Survivors: nil}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	assert.Equal(t, catch.NoOracleSignal, catch.Detect(before, after))
}

// A fix that strictly shrinks the survivor set but does not empty it is a
// partial catch — surfaced distinctly so the reviewer sees the line is now
// better-tested but still not fully constrained.
func TestDetect_reportsPartialCatchWhenSurvivorsShrinkButRemain(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">=", "<="}, Survivors: []string{">=", "<="}}
	after := catch.LineState{Inventory: []string{">=", "<="}, Survivors: []string{">="}}
	assert.Equal(t, catch.PartialCatch, catch.Detect(before, after))
}

// If the survivor set changes to a different operator (a regression introduced
// a new survivor), that is not progress on the original weakness — no catch.
func TestDetect_refusesCatchWhenNewSurvivorAppears(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">=", "&&"}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">=", "&&"}, Survivors: []string{"&&"}}
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after))
}

// A shrunk inventory means the fix EDITED the anchored line (removed an
// operator), so before/after live over different alphabets — ill-typed, not a
// catch — even though the survivor set emptied. Distinguishes the
// inventory-change rule from a genuine same-line strengthening.
func TestDetect_refusesCatchWhenInventoryShrinks(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">=", "&&"}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after))
}

// An operator-free line that stays operator-free has no signal either way;
// NoOracleSignal (driven by the empty pre-fix inventory) takes precedence over
// any "nothing changed" NoCatch reading.
func TestDetect_reportsNoOracleSignalWhenBothRevisionsOperatorFree(t *testing.T) {
	t.Parallel()
	empty := catch.LineState{}
	assert.Equal(t, catch.NoOracleSignal, catch.Detect(empty, empty))
}

const twoOpSrc = "package p\n\nfunc f(a, b, c, d int) bool {\n\treturn a >= b && c == d\n}\n"

// The inventory is an operator SET: a line with the same operator twice
// contributes it once, so the per-revision denominator is stable and not
// inflated by duplicate sites (the deliberate v1 set—not multiset—choice).
func TestLineStateAt_deduplicatesRepeatedOperatorsIntoASet(t *testing.T) {
	t.Parallel()
	src := "package p\n\nfunc f(a, b, c, d int) bool {\n\treturn a >= b && c >= d\n}\n"
	res := mutation.Result{Findings: []mutation.Finding{
		{File: "p.go", Line: 4, Original: ">=", Outcome: mutation.Survived},
		{File: "p.go", Line: 4, Original: ">=", Outcome: mutation.Survived},
	}}
	ls, err := catch.LineStateAt([]byte(src), 4, res)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{">=", "&&"}, ls.Inventory)
	assert.ElementsMatch(t, []string{">="}, ls.Survivors)
}

// The inventory of a line is its operator alphabet; survivors are only the
// operators on THAT line whose mutant survived — findings on other lines must
// not bleed in.
func TestLineStateAt_derivesInventoryAndSurvivorsForTheAnchoredLine(t *testing.T) {
	t.Parallel()
	res := mutation.Result{
		Findings: []mutation.Finding{
			{File: "p.go", Line: 4, Original: ">=", Outcome: mutation.Survived},
			{File: "p.go", Line: 99, Original: "<", Outcome: mutation.Survived}, // other line
		},
		MutantsConsidered: 3,
	}
	ls, err := catch.LineStateAt([]byte(twoOpSrc), 4, res)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{">=", "&&", "=="}, ls.Inventory)
	assert.ElementsMatch(t, []string{">="}, ls.Survivors)
}

// An Undetermined (timed-out) mutant is not a confirmed survivor; it must not
// count toward the survivor set, or a catch could be minted off a non-verdict.
func TestLineStateAt_excludesUndeterminedFindingsFromSurvivors(t *testing.T) {
	t.Parallel()
	res := mutation.Result{
		Findings: []mutation.Finding{
			{File: "p.go", Line: 4, Original: "==", Outcome: mutation.Undetermined},
		},
		MutantsConsidered: 3,
	}
	ls, err := catch.LineStateAt([]byte(twoOpSrc), 4, res)
	require.NoError(t, err)
	assert.Empty(t, ls.Survivors)
}

func TestLineStateAt_reportsEmptyInventoryForOperatorFreeLine(t *testing.T) {
	t.Parallel()
	src := "package p\n\nfunc f() bool {\n\treturn true\n}\n"
	ls, err := catch.LineStateAt([]byte(src), 4, mutation.Result{})
	require.NoError(t, err)
	assert.Empty(t, ls.Inventory)
}

// Unparseable source cannot yield an operator inventory; LineStateAt must
// surface the parse error rather than silently returning an empty state that a
// caller would misread as "no oracle signal".
func TestLineStateAt_returnsErrorOnUnparseableSource(t *testing.T) {
	t.Parallel()
	_, err := catch.LineStateAt([]byte("package p\n\nfunc f( {"), 3, mutation.Result{})
	require.Error(t, err)
}

var goTestCmd = []string{"env", "-u", "GOROOT", "go", "test", "./..."}

const adultSrc = "package adult\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n"

// End-to-end against the real mutation oracle: a weak suite lets the `>=` mutant
// survive at base; the fix strengthens the test (the anchored line is UNCHANGED)
// so the mutant is killed. The whole chain Run→LineStateAt→Detect must mint a
// Catch — the confirmed-catch primitive proven against the real oracle.
func TestDetect_mintsCatchAcrossRealOracleRevisionsWhenTestStrengthened(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module adultcatch\n\ngo 1.23\n")
	writeFile(t, dir, "adult.go", adultSrc)
	writeFile(t, dir, "adult_test.go",
		"package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif !IsAdult(25) {\n\t\tt.Fatal(\"25\")\n\t}\n}\n")

	opts := mutation.Options{Dir: dir, File: "adult.go", Lines: []mutation.LineRange{{Start: 4, End: 4}}, TestCmd: goTestCmd}
	baseRes, err := mutation.Run(context.Background(), opts)
	require.NoError(t, err)
	before, err := catch.LineStateAt([]byte(adultSrc), 4, baseRes)
	require.NoError(t, err)
	require.NotEmpty(t, before.Survivors, "weak suite must leave a surviving mutant on line 4")

	// Strengthen the test; the anchored source line is unchanged.
	writeFile(t, dir, "adult_test.go",
		"package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif IsAdult(17) {\n\t\tt.Fatal(\"17 is not an adult\")\n\t}\n\tif !IsAdult(18) {\n\t\tt.Fatal(\"18 is an adult\")\n\t}\n}\n")
	fixRes, err := mutation.Run(context.Background(), opts)
	require.NoError(t, err)
	after, err := catch.LineStateAt([]byte(adultSrc), 4, fixRes)
	require.NoError(t, err)

	assert.Equal(t, catch.Catch, catch.Detect(before, after))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}
