package pipe_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

var goTestCmd = []string{"env", "-u", "GOROOT", "go", "test", "./..."}

// adultGo has `return age >= 18` on line 4 — the anchored line across the cycle.
const adultGo = "package adult\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n"

const weakTest = "package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif !IsAdult(25) {\n\t\tt.Fatal(\"25\")\n\t}\n}\n"

const strongTest = "package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif IsAdult(17) {\n\t\tt.Fatal(\"17 is not an adult\")\n\t}\n\tif !IsAdult(18) {\n\t\tt.Fatal(\"18 is an adult\")\n\t}\n}\n"

func adultAnchor() reanchor.Anchor {
	return reanchor.Anchor{Path: "adult.go", Start: 4, End: 4, LineHash: reanchor.HashLines("\treturn age >= 18")}
}

func beatsContaining(trace []string, sub string) int {
	n := 0
	for _, b := range trace {
		if strings.Contains(b, sub) {
			n++
		}
	}
	return n
}

func firstIndexContaining(trace []string, sub string) int {
	for i, b := range trace {
		if strings.Contains(b, sub) {
			return i
		}
	}
	return -1
}

func TestRunCatchCycle_mintsCatchWhenAgentStrengthensTestOnly(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base: weak test lets >= survive")
	write(t, dir, "adult_test.go", strongTest) // strengthen the test ONLY; adult.go unchanged
	fix := commitAll(t, dir, "fix: strengthen the test")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, res.Outcome)
	assert.Equal(t, pipe.LandClean, res.Land, "rebasing the fix onto an unchanged tip integrates clean")
	assert.Equal(t, "adult.go", res.Path)
	assert.Equal(t, 4, res.Line, "the anchored line is unchanged (Same), so it stays at line 4")

	assert.NotEmpty(t, res.Before.Survivors, "the before-state must be exposed for the presenter and the catch record")
	assert.Empty(t, res.After.Survivors, "the catch cleared the survivor set, and the after-state is exposed")
	assert.NotEmpty(t, res.After.Inventory, "the after inventory (mutants considered) drives the Tested vs blind distinction")

	// Prove the base oracle genuinely ran (and found the line's one mutable
	// operator), so a hardcoded result cannot pass: the anchored line has exactly
	// one operator (`>=`) at base.
	assert.Equal(t, 1, beatsContaining(res.Trace, "oracle ran base: 1 considered"), "the base oracle really ran on the anchored line")

	assert.Equal(t, 1, beatsContaining(res.Trace, "catch:"), "the catch is exactly one discrete beat")
	catchAt := firstIndexContaining(res.Trace, "catch:")
	settledFixAt := firstIndexContaining(res.Trace, "settled fix")
	require.Positive(t, settledFixAt, "the trace records settling the fix revision")
	assert.Greater(t, catchAt, settledFixAt, "the catch beat comes after both settle beats — a human can point to the mint")
}

func TestRunCatchCycle_mintsCatchAtReanchoredLineWhenAnchorMoves(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	// Prepend two comment lines (anchored line content unchanged → Moved to
	// line 6) AND strengthen the test — a test-only fix on a shifted line.
	write(t, dir, "adult.go", "// shifted\n// shifted\n"+adultGo)
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "fix: prepend lines + strengthen the test")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, res.Outcome)
	assert.Equal(t, "adult.go", res.Path)
	assert.Equal(t, 6, res.Line, "the anchor moved down by the two prepended lines")
}

func TestRunCatchCycle_leavesNoWorktreeMetadataInRepo(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "fix: strengthen the test")

	_, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)

	entries, readErr := os.ReadDir(filepath.Join(dir, ".git", "worktrees"))
	if readErr == nil {
		assert.Empty(t, entries, "a settled cycle must leave no worktree metadata in the repo")
	}
}

func TestRunCatchCycle_refusesCatchWhenFixEditsAnchoredLine(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	// Edit the anchored line itself: >= becomes >, a real behavior change.
	write(t, dir, "adult.go", "package adult\n\nfunc IsAdult(age int) bool {\n\treturn age > 18\n}\n")
	fix := commitAll(t, dir, "fix: edit the anchored line")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.NotEqual(t, catch.Catch, res.Outcome, "an edited anchored line must never mint a phantom catch")
	assert.Equal(t, catch.NoOracleSignal, res.Outcome, "the reanchor gate outdates the edited line before Detect runs")
	assert.Equal(t, pipe.ReasonAnchorEdited, res.Reason,
		"the card must say the line was EDITED, never the false 'no mutable operator' — the keystone against a confidently-wrong quiet verdict")
}

func TestRunCatchCycle_renamedAnchorReportsFileRenamedReason(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	runGit(t, dir, "mv", "adult.go", "grown.go") // identical content → detected rename
	fix := commitAll(t, dir, "rename adult.go -> grown.go")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.NoOracleSignal, res.Outcome, "a lost-via-rename anchor mints no catch")
	assert.Equal(t, pipe.ReasonFileRenamed, res.Reason,
		"the quiet verdict is BECAUSE the file was renamed — the surface must not claim the line had no operators")
}

func TestRunCatchCycle_operatorFreeLineReportsNoMutableOperatorReason(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module idpipe\n\ngo 1.23\n")
	write(t, dir, "id.go", "package idp\n\nfunc Id(n int) int {\n\treturn n\n}\n") // line 4 `return n` has no mutable operator
	write(t, dir, "id_test.go", "package idp\n\nimport \"testing\"\n\nfunc TestId(t *testing.T) {\n\tif Id(3) != 3 {\n\t\tt.Fatal(\"3\")\n\t}\n}\n")
	base := commitAll(t, dir, "base")
	write(t, dir, "id_test.go", "package idp\n\nimport \"testing\"\n\nfunc TestId(t *testing.T) {\n\tif Id(3) != 3 {\n\t\tt.Fatal(\"3\")\n\t}\n\tif Id(0) != 0 {\n\t\tt.Fatal(\"0\")\n\t}\n}\n")
	fix := commitAll(t, dir, "strengthen the test; id.go unchanged (Same)")

	anchor := reanchor.Anchor{Path: "id.go", Start: 4, End: 4, LineHash: reanchor.HashLines("\treturn n")}
	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, anchor, goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.NoOracleSignal, res.Outcome, "a line with no mutable operator yields no oracle signal")
	assert.Equal(t, pipe.ReasonNoMutableOperator, res.Reason,
		"here 'no mutable operator' is the TRUE reason — the split must not regress the one honest case")
}

func TestRunCatchCycle_emptyTestCmdErrorsInsteadOfPanicking(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module idpipe\n\ngo 1.23\n")
	write(t, dir, "id.go", "package idp\n\nfunc Id(n int) int {\n\treturn n\n}\n") // line 4 `return n`: no mutable operator
	base := commitAll(t, dir, "base")
	// Anchor an operator-free line: the oracle generates 0 mutants and so never
	// calls the suite — its empty-testCmd guard is never tripped. The anchor
	// survives (Same), so the cycle reaches integrateOnTip with the empty testCmd
	// untouched. A clean rebase then indexes testCmd[0]/testCmd[1:]: without a
	// guard that panics instead of failing closed like the oracle does.
	write(t, dir, "other.go", "package idp\n\nvar X = 2\n")
	fix := commitAll(t, dir, "touch a disjoint file; id.go unchanged")

	anchor := reanchor.Anchor{Path: "id.go", Start: 4, End: 4, LineHash: reanchor.HashLines("\treturn n")}
	_, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, anchor, nil)
	require.Error(t, err, "an empty testCmd must fail closed with an error, never panic indexing testCmd")
}

func TestRunCatchCycle_emptyTipErrorsInsteadOfMislabelingConflict(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest) // test-only fix; anchor stays Same so the cycle reaches integrateOnTip
	fix := commitAll(t, dir, "fix: strengthen the test")

	// An empty tipRev reaches `git rebase ""`, which exits non-zero with "invalid
	// upstream". integrateOnTip maps ANY rebase failure to LandConflict, so an
	// omitted tip would silently render "Trunk moved — rebase needed" — a
	// dishonest verdict for what is a caller/config error. It must fail closed
	// with an error instead, like the empty-testCmd guard.
	_, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, "", adultAnchor(), goTestCmd)
	require.Error(t, err, "an empty tipRev must fail closed, never be mislabeled as a trunk-moved LandConflict")
}

func TestRunCatchCycle_landsCleanOnNonConflictingTip(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	write(t, dir, "other.go", "package adult\n\nvar Other = 1\n")
	base := commitAll(t, dir, "base")

	runGit(t, dir, "checkout", "-q", "-b", "fixbranch")
	write(t, dir, "adult_test.go", strongTest) // test-only fix → real Catch, adult.go untouched
	fix := commitAll(t, dir, "fix: strengthen the test")

	runGit(t, dir, "checkout", "-q", "-") // back to base branch
	write(t, dir, "other.go", "package adult\n\nvar Other = 2\n") // trunk advances on a DISJOINT file
	tip := commitAll(t, dir, "trunk: edit a disjoint file")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, tip, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, res.Outcome, "the fix mints a real catch (orthogonal to integration)")
	assert.Equal(t, pipe.LandClean, res.Land,
		"a fix that rebases cleanly onto a moved tip with a green integrated suite lands clean")
}

func TestRunCatchCycle_landsConflictWhenFixDivergesFromTip(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")

	runGit(t, dir, "checkout", "-q", "-b", "fixbranch")
	write(t, dir, "adult_test.go", strongTest) // fix rewrites the whole test file; adult.go untouched → Catch
	fix := commitAll(t, dir, "fix: strengthen the test")

	runGit(t, dir, "checkout", "-q", "-")
	// trunk rewrites the SAME test file differently → rebase conflicts on adult_test.go
	write(t, dir, "adult_test.go", "package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif !IsAdult(99) {\n\t\tt.Fatal(\"trunk's own assertion\")\n\t}\n}\n")
	tip := commitAll(t, dir, "trunk: rewrite the same test file")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, tip, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, res.Outcome, "the catch is minted on the base; integration is orthogonal")
	assert.Equal(t, pipe.LandConflict, res.Land,
		"a fix that textually conflicts with the moved tip cannot integrate — conflict short-circuits before checks")
}

func TestRunCatchCycle_cleanRebaseButChecksRedYieldsChecksRed(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")

	runGit(t, dir, "checkout", "-q", "-b", "fixbranch")
	write(t, dir, "adult_test.go", strongTest) // green in isolation → mints a Catch
	fix := commitAll(t, dir, "fix: strengthen the test")

	runGit(t, dir, "checkout", "-q", "-")
	// trunk adds a NEW test file (no textual conflict with the fix) that the
	// integrated tree fails — the green pre-integration catch goes red on tip.
	write(t, dir, "invariant_test.go", "package adult\n\nimport \"testing\"\n\nfunc TestTrunkInvariant(t *testing.T) {\n\tif IsAdult(18) {\n\t\tt.Fatal(\"trunk invariant: 18 must not count as adult\")\n\t}\n}\n")
	tip := commitAll(t, dir, "trunk: add an invariant the fix's behavior violates")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, tip, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, res.Outcome, "the catch was real pre-integration; the mint token is unchanged")
	assert.Equal(t, pipe.LandChecksRed, res.Land,
		"clean rebase but the integrated suite fails — a green pre-integration catch is a red post-integration regression")
}
