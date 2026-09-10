package reanchor_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// numbered returns n lines "1".."n", each newline-terminated, so a line's
// number equals its content — anchors are easy to reason about.
func numbered(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "%d\n", i)
	}
	return b.String()
}

// linesOf joins the 1-based inclusive range [start,end] of content the same
// way the re-anchor implementation extracts a range, so a hash computed here
// matches the implementation's verification hash.
func linesOf(content string, start, end int) string {
	all := strings.Split(content, "\n")
	return strings.Join(all[start-1:end], "\n")
}

// The catch economy mints a catch only when reanchor says the line provably survived
// (Same/Moved). If the anchored line's content is DUPLICATED in the file, a positional
// hash match at the delta-projected line cannot prove WHICH occurrence is the real
// relocation — so Moved would risk anchoring (and confirming a catch) on the wrong line.
// Fail closed to Outdated rather than mis-anchor.
func TestReanchor_outdatesWhenMovedContentIsAmbiguous(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	// "DUP" appears at line 4 AND line 8 — the anchored content is not unique.
	body := "1\n2\n3\nDUP\n5\n6\n7\nDUP\n9\n10\n"
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")
	write(t, dir, "f.txt", "a\nb\n"+body) // prepend 2 → anchor projects to a hash-matching DUP line
	head := commitAll(t, dir, "prepend 2 lines")

	a := reanchor.Anchor{Path: "f.txt", Start: 4, End: 4, LineHash: reanchor.HashLines("DUP")}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Outdated, got.State,
		"duplicated anchor content makes the relocation ambiguous — fail closed, never mis-anchor")
}

// The ambiguity guard must check only the ANCHOR's content uniqueness, not whether the
// file contains any duplicates at all: a uniquely-content anchor still resolves Moved
// even when unrelated duplicate lines exist elsewhere.
func TestReanchor_movesWhenAnchorContentIsUniqueDespiteOtherDuplicates(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	// "UNIQ" appears once (the anchor); "DUP" appears twice but is unrelated.
	body := "1\n2\n3\nUNIQ\n5\n6\nDUP\n8\nDUP\n10\n"
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")
	write(t, dir, "f.txt", "a\nb\n"+body) // prepend 2 → UNIQ shifts from line 4 to line 6
	head := commitAll(t, dir, "prepend 2 lines")

	a := reanchor.Anchor{Path: "f.txt", Start: 4, End: 4, LineHash: reanchor.HashLines("UNIQ")}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Moved, got.State, "a uniquely-content anchor still relocates despite unrelated dups")
	assert.Equal(t, 6, got.Start)
	assert.Equal(t, 6, got.End)
}

// The ambiguity guard must also hold for a MULTI-line anchor: a duplicated BLOCK (not
// just a single line) that projects onto a hash-matching copy is equally untrustworthy.
func TestReanchor_outdatesWhenMovedMultiLineBlockIsAmbiguous(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	// The 2-line block (DUP-A,DUP-B) appears at lines 3-4 AND 7-8.
	body := "1\n2\nDUP-A\nDUP-B\n5\n6\nDUP-A\nDUP-B\n9\n10\n"
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")
	write(t, dir, "f.txt", "a\nb\n"+body) // prepend 2 → block 3-4 projects to 5-6 (a hash-matching copy)
	head := commitAll(t, dir, "prepend 2 lines")

	a := reanchor.Anchor{Path: "f.txt", Start: 3, End: 4, LineHash: reanchor.HashLines("DUP-A\nDUP-B")}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Outdated, got.State,
		"a duplicated multi-line block is an ambiguous relocation — fail closed")
}

func TestReanchor_keepsAnchorWhenFileUnchanged(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(20)
	write(t, dir, "f.txt", body)
	write(t, dir, "other.txt", numbered(5))
	base := commitAll(t, dir, "base")
	// Touch only other.txt; f.txt is untouched.
	write(t, dir, "other.txt", numbered(7))
	head := commitAll(t, dir, "edit other")

	a := reanchor.Anchor{Path: "f.txt", Start: 10, End: 12, LineHash: reanchor.HashLines(linesOf(body, 10, 12))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Same, got.State)
	assert.Equal(t, "f.txt", got.Path)
	assert.Equal(t, 10, got.Start)
	assert.Equal(t, 12, got.End)
}

func TestReanchor_shiftsAnchorWhenLinesInsertedAbove(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(20)
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")
	// Insert 3 new lines at the very top; the anchored block's content is
	// unchanged but shifts down by 3.
	write(t, dir, "f.txt", "a\nb\nc\n"+body)
	head := commitAll(t, dir, "prepend 3 lines")

	a := reanchor.Anchor{Path: "f.txt", Start: 10, End: 12, LineHash: reanchor.HashLines(linesOf(body, 10, 12))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Moved, got.State)
	assert.Equal(t, "f.txt", got.Path)
	assert.Equal(t, 13, got.Start)
	assert.Equal(t, 15, got.End)
}

func TestReanchor_outdatesAnchorWhenEditedLinesOverlap(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(20)
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")
	// Edit line 11 — inside the anchored [10,12] range.
	lines := strings.Split(body, "\n")
	lines[10] = "eleven"
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit line 11")

	a := reanchor.Anchor{Path: "f.txt", Start: 10, End: 12, LineHash: reanchor.HashLines(linesOf(body, 10, 12))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Outdated, got.State)
	assert.Equal(t, "f.txt", got.Path)
}

func TestReanchor_outdatesWhenMovedContentHashMismatches(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(20)
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")
	write(t, dir, "f.txt", "a\nb\nc\n"+body)
	head := commitAll(t, dir, "prepend 3 lines")

	// Stored hash deliberately does not match the anchored content at base.
	a := reanchor.Anchor{Path: "f.txt", Start: 10, End: 12, LineHash: reanchor.HashLines("not the real content")}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Outdated, got.State)
}

// A removed file resolves to the distinct Deleted state, NOT Outdated: the
// anchored content is GONE (a real deletion — or a rename git could not follow
// because the new file fell below its similarity threshold), which is not the
// same truth as "the line was edited in place." Collapsing it into Outdated made
// the surface assert a false "the line was edited" cause for a vanished file.
func TestReanchor_reportsDeletedWhenFileRemoved(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(20)
	write(t, dir, "f.txt", body)
	write(t, dir, "keep.txt", "x\n")
	base := commitAll(t, dir, "base")
	require.NoError(t, os.Remove(filepath.Join(dir, "f.txt")))
	head := commitAll(t, dir, "delete f.txt")

	a := reanchor.Anchor{Path: "f.txt", Start: 10, End: 12, LineHash: reanchor.HashLines(linesOf(body, 10, 12))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Deleted, got.State)
	assert.Equal(t, "f.txt", got.Path)
}

func TestReanchor_marksLostViaRenameWithNewPath(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(20)
	write(t, dir, "orig.txt", body)
	base := commitAll(t, dir, "base")
	runGit(t, dir, "mv", "orig.txt", "renamed.txt")
	head := commitAll(t, dir, "rename orig->renamed")

	a := reanchor.Anchor{Path: "orig.txt", Start: 10, End: 12, LineHash: reanchor.HashLines(linesOf(body, 10, 12))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.LostViaRename, got.State)
	assert.Equal(t, "renamed.txt", got.Path)
}

// A rename of a NON-ASCII path must still be followed: git's default
// core.quotepath octal-quotes such paths in --name-status output (café.txt →
// "caf\303\251.txt"), so an Anchor.Path with non-ASCII bytes would never match
// the rename record and the anchor would phantom-resolve as Same instead of
// LostViaRename — silently losing a catch on any repo with non-ASCII filenames.
func TestReanchor_followsARenameOfANonASCIIPath(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(20)
	write(t, dir, "café.txt", body)
	base := commitAll(t, dir, "base")
	runGit(t, dir, "mv", "café.txt", "résumé.txt")
	head := commitAll(t, dir, "rename café->résumé")

	a := reanchor.Anchor{Path: "café.txt", Start: 10, End: 12, LineHash: reanchor.HashLines(linesOf(body, 10, 12))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.LostViaRename, got.State, "a non-ASCII rename must be detected, not read as Same")
	assert.Equal(t, "résumé.txt", got.Path, "the new path is the real unquoted non-ASCII name")
}

// A filename containing a literal TAB is C-quoted by git even with
// core.quotepath=false ("tab\tname.txt"), so a name-status parse that splits on
// TAB mis-reads the record and the anchored file reads as Same — silently
// dropping a catch. A -z (NUL-delimited, raw paths) parse must classify the
// edit honestly as Outdated.
func TestReanchor_outdatesAnchorOnAFileWhoseNameContainsATab(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	name := "tab\tname.txt"
	body := numbered(20)
	write(t, dir, name, body)
	base := commitAll(t, dir, "base")
	// Edit line 10 (overlapping the anchor) so the honest verdict is Outdated, not Same.
	lines := strings.Split(body, "\n")
	lines[9] = "EDITED"
	write(t, dir, name, strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit tab-named file")

	a := reanchor.Anchor{Path: name, Start: 10, End: 10, LineHash: reanchor.HashLines(linesOf(body, 10, 10))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Outdated, got.State, "a tab-named edited file must not phantom-resolve as Same")
}

// The capstone of the -z path fix across BOTH fileStatus and diff.Compute: a
// MOVED verdict on a tab-named file. Lines inserted above the anchor shift it;
// resolving Moved (not Outdated) requires diff.Compute to surface the file's
// hunks under its real path so the delta is computed — which the patch-header
// parse could not do for a quoted name.
func TestReanchor_movesAnchorOnAFileWhoseNameContainsATab(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	name := "tab\tname.txt"
	body := numbered(20)
	write(t, dir, name, body)
	base := commitAll(t, dir, "base")
	// Prepend 3 lines so the anchored content shifts down by 3 but is unchanged.
	write(t, dir, name, "a\nb\nc\n"+body)
	head := commitAll(t, dir, "insert 3 lines above")

	a := reanchor.Anchor{Path: name, Start: 10, End: 12, LineHash: reanchor.HashLines(linesOf(body, 10, 12))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Moved, got.State, "a tab-named file with content shifted must resolve Moved, not Outdated")
	assert.Equal(t, 13, got.Start, "anchor start shifts by the 3 inserted lines")
	assert.Equal(t, 15, got.End)
}

func TestReanchor_accumulatesDeltaFromMultipleHunksAbove(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(40)
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")

	all := strings.Split(body, "\n") // "1".."40", then ""
	var nl []string
	nl = append(nl, "a", "b")      // +2 prepended at the top
	nl = append(nl, all[0:20]...)  // lines "1".."20"
	nl = append(nl, "x", "y", "z") // +3 inserted after line 20
	nl = append(nl, all[20:]...)   // lines "21".."40", ""
	write(t, dir, "f.txt", strings.Join(nl, "\n"))
	head := commitAll(t, dir, "two insertions above the anchor")

	// Anchor [30,32] sits below both inserts → shifts by +5.
	a := reanchor.Anchor{Path: "f.txt", Start: 30, End: 32, LineHash: reanchor.HashLines(linesOf(body, 30, 32))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Moved, got.State)
	assert.Equal(t, 35, got.Start)
	assert.Equal(t, 37, got.End)
}

func TestReanchor_reanchorsWhenChangeIsFarAboveAnchor(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(40)
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")
	// Replace line 3 in place (net delta 0); line 3 is far above [20,22], well
	// beyond git's 3-line context, so the hunk cannot reach the anchor.
	lines := strings.Split(body, "\n")
	lines[2] = "three"
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit line 3 only")

	a := reanchor.Anchor{Path: "f.txt", Start: 20, End: 22, LineHash: reanchor.HashLines(linesOf(body, 20, 22))}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Moved, got.State)
	assert.Equal(t, 20, got.Start)
	assert.Equal(t, 22, got.End)
}

func TestReanchor_outdatesAnchorBeyondEndOfFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	body := numbered(20)
	write(t, dir, "f.txt", body)
	base := commitAll(t, dir, "base")
	// Modify a far-above line so the file is in the changed set (reaches the
	// shift path), but the anchor points past EOF.
	lines := strings.Split(body, "\n")
	lines[2] = "three"
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit line 3")

	a := reanchor.Anchor{Path: "f.txt", Start: 100, End: 102, LineHash: reanchor.HashLines("whatever")}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	assert.Equal(t, reanchor.Outdated, got.State)
}

func TestReanchor_returnsErrorOnUnknownRevision(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(5))
	head := commitAll(t, dir, "base")

	a := reanchor.Anchor{Path: "f.txt", Start: 1, End: 2, LineHash: reanchor.HashLines("1\n2")}
	_, err := reanchor.Reanchor(context.Background(), dir, a, "deadbeefdeadbeef", head)
	require.Error(t, err)
}

func TestHashLines_isStableAndContentSensitive(t *testing.T) {
	t.Parallel()
	h1 := reanchor.HashLines("alpha\nbeta")
	h2 := reanchor.HashLines("alpha\nbeta")
	h3 := reanchor.HashLines("alpha\nbetax")
	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
	assert.NotEmpty(t, h1)
}
