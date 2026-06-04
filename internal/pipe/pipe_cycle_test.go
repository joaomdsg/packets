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

// The headline of the whole economy: an agent strengthens the test ONLY (the
// anchored source line is untouched), turning a weakly-tested line into a
// constrained one, and the pipe mints a real Catch from two real settles — the
// first logged economy transaction, surfaced as one discrete, replayable beat.
func TestRunCatchCycle_mintsCatchWhenAgentStrengthensTestOnly(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base: weak test lets >= survive")
	write(t, dir, "adult_test.go", strongTest) // strengthen the test ONLY; adult.go unchanged
	fix := commitAll(t, dir, "fix: strengthen the test")

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, res.Outcome)
	assert.Equal(t, pipe.Unintegrated, res.Land, "the pipe must never report a fake merged")
	assert.Equal(t, "adult.go", res.Path)
	assert.Equal(t, 4, res.Line, "the anchored line is unchanged (Same), so it stays at line 4")

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

// When unrelated lines shift the anchored line between revisions, the cycle
// must re-anchor and run the fix oracle at the MOVED line — minting the catch
// at the new coordinates, not the stale ones. This exercises the pipe's Moved
// branch and proves the re-anchored line drives the fix-revision oracle scope.
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

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, res.Outcome)
	assert.Equal(t, "adult.go", res.Path)
	assert.Equal(t, 6, res.Line, "the anchor moved down by the two prepended lines")
}

// Production hygiene: every cycle materializes throwaway git worktrees in the
// caller's repo; after the cycle the repo must hold no leaked worktree admin
// metadata (.git/worktrees), or a long-lived production repo accumulates stale
// entries cycle after cycle.
func TestRunCatchCycle_leavesNoWorktreeMetadataInRepo(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "fix: strengthen the test")

	_, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)

	entries, readErr := os.ReadDir(filepath.Join(dir, ".git", "worktrees"))
	if readErr == nil {
		assert.Empty(t, entries, "a settled cycle must leave no worktree metadata in the repo")
	}
}

// Cross-layer safety finding: when the fix EDITS the anchored line (a real
// behavior change), the re-anchor gate reports Outdated and the pipe yields
// NoOracleSignal — it never mints a phantom Catch. The reanchor gate dominates
// for edited lines, firing BEFORE catch.Detect's inventory-change rule (which
// is belt-and-suspenders, covered at the catch unit level).
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

	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, adultAnchor(), goTestCmd)
	require.NoError(t, err)
	assert.NotEqual(t, catch.Catch, res.Outcome, "an edited anchored line must never mint a phantom catch")
	assert.Equal(t, catch.NoOracleSignal, res.Outcome, "the reanchor gate outdates the edited line before Detect runs")
}
