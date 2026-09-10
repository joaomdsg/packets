package pipe_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	return dir
}

func commitAll(t *testing.T, dir, msg string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", msg)
	return runGit(t, dir, "rev-parse", "HEAD")
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

// gte is a source file with a `>=` operator on line 4 (1-based), used as the
// anchored line across the scenarios below.
const gte = "package p\n\nfunc f(a, b int) bool {\n\treturn a >= b\n}\n"

// anchorLine4 builds an anchor on line 4 of gte whose hash matches its content.
func anchorLine4(path string) reanchor.Anchor {
	return reanchor.Anchor{Path: path, Start: 4, End: 4, LineHash: reanchor.HashLines("\treturn a >= b")}
}

func TestCatchAcross_refusesPhantomCatchOnNeutralRename(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "orig.go", gte)
	base := commitAll(t, dir, "base")
	runGit(t, dir, "mv", "orig.go", "moved.go") // content identical → detected rename
	head := commitAll(t, dir, "rename orig->moved")

	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}

	// A direct Detect would mint a phantom Catch on this very data:
	require.Equal(t, catch.Catch, catch.Detect(before, after))

	got, reason, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("orig.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.NoOracleSignal, got)
	assert.Equal(t, pipe.ReasonFileRenamed, reason, "a renamed anchor is quiet BECAUSE the file was renamed, not for lack of operators")
}

func TestCatchAcross_mintsCatchWhenAnchorSurvivesAsMoved(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	base := commitAll(t, dir, "base")
	write(t, dir, "f.go", "// added\n// added\n"+gte) // prepend 2 lines → line 4 shifts to 6
	head := commitAll(t, dir, "prepend lines above the anchor")

	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}

	got, reason, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, got)
	assert.Equal(t, pipe.ReasonNone, reason, "a minted catch carries its meaning in the Outcome, not a quiet reason")
}

func TestCatchAcross_refusesCatchWhenEditOverlapsAnchor(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	base := commitAll(t, dir, "base")
	write(t, dir, "f.go", "package p\n\nfunc f(a, b int) bool {\n\treturn a > b\n}\n") // edit line 4
	head := commitAll(t, dir, "edit the anchored line")

	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}

	got, reason, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.NoOracleSignal, got)
	assert.Equal(t, pipe.ReasonAnchorEdited, reason, "an edited anchor is quiet BECAUSE the line changed, not for lack of operators")
}

func TestCatchAcross_reportsAnchorDeletedWhenFileRemoved(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	write(t, dir, "keep.go", "package p\n\nvar Keep = 1\n")
	base := commitAll(t, dir, "base")
	require.NoError(t, os.Remove(filepath.Join(dir, "f.go")))
	head := commitAll(t, dir, "delete the anchored file")

	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}

	got, reason, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.NoOracleSignal, got, "a vanished file mints nothing — the after-state is ignored")
	assert.Equal(t, pipe.ReasonAnchorDeleted, reason, "a deleted file is quiet because it is GONE, not because the line was edited in place")
}

func TestCatchAcross_delegatesToDetectWhenFileUnchanged(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	write(t, dir, "other.go", "package p\n\nvar X = 1\n")
	base := commitAll(t, dir, "base")
	write(t, dir, "other.go", "package p\n\nvar X = 2\n") // touch only other.go
	head := commitAll(t, dir, "edit other.go")

	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}

	got, _, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.Catch, got)
}

// When the anchor survives but the after-oracle run did not FINISH (its mutants timed
// out), the quiet verdict must be attributed to the incomplete oracle — not to "no
// mutable operator" (the line plainly has one). Mislabeling would tell the Lead the
// oracle had nothing to say when really it never finished saying it.
func TestCatchAcross_attributesAQuietVerdictToTheIncompleteOracle(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	write(t, dir, "other.go", "package p\n\nvar X = 1\n")
	base := commitAll(t, dir, "base")
	write(t, dir, "other.go", "package p\n\nvar X = 2\n") // touch only other.go → anchor survives Same
	head := commitAll(t, dir, "edit other.go")

	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil, Undetermined: []string{">="}}

	got, reason, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.NoOracleSignal, got, "an incomplete after-run mints no catch")
	assert.Equal(t, pipe.ReasonOracleIncomplete, reason,
		"the quiet verdict is the oracle not finishing, NOT a missing mutable operator")
}

// An incomplete BEFORE run also makes the verdict quiet for the "oracle didn't finish"
// reason — not "no mutable operator" — so the attribution tracks completeness on either
// side, mirroring Detect's either-side fail-closed guard.
func TestCatchAcross_attributesAnIncompleteBeforeRunToTheOracle(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	write(t, dir, "other.go", "package p\n\nvar X = 1\n")
	base := commitAll(t, dir, "base")
	write(t, dir, "other.go", "package p\n\nvar X = 2\n") // touch only other.go → anchor survives Same
	head := commitAll(t, dir, "edit other.go")

	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}, Undetermined: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil, Undetermined: nil}

	got, reason, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.NoOracleSignal, got, "an incomplete before-run mints no catch")
	assert.Equal(t, pipe.ReasonOracleIncomplete, reason,
		"the quiet verdict is the oracle not finishing on the before side, not a missing operator")
}

// A surviving anchor on a line with NO mutable operators is genuinely no-mutable-operator
// — and must be attributed as such, NOT as an incomplete oracle, so the two distinct
// "quiet" truths never collapse.
func TestCatchAcross_attributesAQuietVerdictToNoMutableOperator(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	write(t, dir, "other.go", "package p\n\nvar X = 1\n")
	base := commitAll(t, dir, "base")
	write(t, dir, "other.go", "package p\n\nvar X = 2\n")
	head := commitAll(t, dir, "edit other.go")

	// Empty inventory = the line had no mutable operator; after-undetermined is irrelevant.
	before := catch.LineState{Inventory: nil, Survivors: nil}
	after := catch.LineState{Inventory: nil, Survivors: nil, Undetermined: []string{">="}}

	got, reason, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.NoOracleSignal, got)
	assert.Equal(t, pipe.ReasonNoMutableOperator, reason,
		"a line with no mutable operator is no-mutable-operator, not oracle-incomplete")
}

func TestCatchAcross_delegatesNoCatchWhenNothingWasWeak(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	write(t, dir, "other.go", "package p\n\nvar X = 1\n")
	base := commitAll(t, dir, "base")
	write(t, dir, "other.go", "package p\n\nvar X = 2\n")
	head := commitAll(t, dir, "edit other.go")

	before := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}

	got, _, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), base, head, before, after)
	require.NoError(t, err)
	assert.Equal(t, catch.NoCatch, got)
}

func TestCatchAcross_returnsErrorOnUnknownRevision(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.go", gte)
	head := commitAll(t, dir, "base")

	_, _, err := pipe.CatchAcross(context.Background(), dir, anchorLine4("f.go"), "deadbeefdeadbeef", head,
		catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}, catch.LineState{})
	require.Error(t, err)
}
