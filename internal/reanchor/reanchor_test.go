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

	"github.com/joaomdsg/agntpr/internal/reanchor"
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

// A line-number shift that lands on content NOT matching the stored hash must
// outdate, never mis-anchor onto the wrong code — the conservative §28 fallback.
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

func TestReanchor_outdatesAnchorWhenFileDeleted(t *testing.T) {
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
	assert.Equal(t, reanchor.Outdated, got.State)
	assert.Equal(t, "f.txt", got.Path)
}

// A rename must surface as a DISTINCT LostViaRename state carrying the new
// path — never a silent drop and never a phantom re-anchor onto stale lines.
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

// Deltas from several hunks above the anchor must accumulate (§28 sums them);
// a naive impl that shifts by only the first hunk would mis-anchor.
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

// A change far ABOVE the anchor (beyond git's context window, so the hunk's
// old range never reaches it) must re-anchor, not outdate. A change within the
// context window is conservatively outdated (see the overlap test) — §28
// prefers a false-outdated to a mis-anchored comment.
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

// A stale anchor whose shifted range falls beyond the end of the file must
// outdate, not panic on an out-of-range slice (§28 EOF clamping).
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
