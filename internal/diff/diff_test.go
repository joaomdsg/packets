package diff_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v:\n%s", args, out)
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

// numbered returns n lines "1".."n" (each followed by newline) for building
// files whose line numbers are easy to reason about.
func numbered(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "%d\n", i)
	}
	return b.String()
}

func fileByPath(d diff.Diff, path string) (diff.FileDiff, bool) {
	for _, f := range d.Files {
		if f.Path == path {
			return f, true
		}
	}
	return diff.FileDiff{}, false
}

func TestCompute_reportsPathCountsAndHunkBracketingChange(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(20))
	base := commitAll(t, dir, "base")
	lines := strings.Split(numbered(20), "\n")
	lines[9] = "ten"
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit line 10")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	f, ok := fileByPath(d, "f.txt")
	require.True(t, ok, "f.txt missing from diff: %+v", d.Files)
	assert.Equal(t, 1, f.Added)
	assert.Equal(t, 1, f.Deleted)
	require.Len(t, f.Hunks, 1)
	h := f.Hunks[0]
	assert.True(t, h.NewStart <= 10 && 10 <= h.NewStart+h.NewLines-1, "hunk %+v must bracket changed new-line 10", h)
	assert.True(t, h.OldStart <= 10 && 10 <= h.OldStart+h.OldLines-1, "hunk %+v must bracket old-line 10", h)
}

func TestCompute_reportsExactChangedRegionNotWholeFile(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(40))
	base := commitAll(t, dir, "base")
	lines := strings.Split(numbered(40), "\n")
	lines[19] = "twenty"
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit line 20")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	f, ok := fileByPath(d, "f.txt")
	require.True(t, ok, "f.txt missing: %+v", d.Files)
	require.Len(t, f.Hunks, 1)
	// Assumes git's stable default 3 lines of context: one edit deep in a
	// 40-line file yields @@ -17,7 +17,7 @@, not a whole-file -1,40 span.
	want := diff.Hunk{OldStart: 17, OldLines: 7, NewStart: 17, NewLines: 7}
	assert.Equal(t, want, f.Hunks[0])
}

func TestCompute_parsesOmittedHunkCountAsOne(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.txt", "alpha\n")
	base := commitAll(t, dir, "base")
	write(t, dir, "f.txt", "omega\n")
	head := commitAll(t, dir, "change the only line")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	f, ok := fileByPath(d, "f.txt")
	require.True(t, ok, "f.txt missing: %+v", d.Files)
	require.Len(t, f.Hunks, 1)
	// A one-line in-place change makes git emit the count-omitted form
	// @@ -1 +1 @@; the omitted count must read as 1, not 0.
	want := diff.Hunk{OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 1}
	assert.Equal(t, want, f.Hunks[0])
}

func TestCompute_startsNewFileHunkAtLineOneWithEmptyOldSide(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "seed.txt", "x\n")
	base := commitAll(t, dir, "seed")
	write(t, dir, "new.txt", numbered(3))
	head := commitAll(t, dir, "add new.txt")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	f, ok := fileByPath(d, "new.txt")
	require.True(t, ok, "new.txt missing: %+v", d.Files)
	assert.Equal(t, 3, f.Added)
	assert.Equal(t, 0, f.Deleted)
	require.Len(t, f.Hunks, 1)
	want := diff.Hunk{OldStart: 0, OldLines: 0, NewStart: 1, NewLines: 3}
	assert.Equal(t, want, f.Hunks[0])
}

func TestCompute_countsDeletedLines(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(6))
	base := commitAll(t, dir, "base")
	lines := strings.Split(numbered(6), "\n")
	kept := append(append([]string{}, lines[:2]...), lines[4:]...)
	write(t, dir, "f.txt", strings.Join(kept, "\n"))
	head := commitAll(t, dir, "delete 3-4")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	f, ok := fileByPath(d, "f.txt")
	require.True(t, ok, "f.txt missing: %+v", d.Files)
	assert.Equal(t, 0, f.Added)
	assert.Equal(t, 2, f.Deleted)
}

func TestCompute_separatesEachFileIntoOwnFileDiff(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "a.txt", numbered(10))
	base := commitAll(t, dir, "base")
	la := strings.Split(numbered(10), "\n")
	la[4] = "five"
	write(t, dir, "a.txt", strings.Join(la, "\n"))
	write(t, dir, "c.txt", numbered(2))
	head := commitAll(t, dir, "edit a, add c")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	require.Len(t, d.Files, 2)
	a, ok := fileByPath(d, "a.txt")
	require.True(t, ok, "a.txt missing: %+v", d.Files)
	assert.Equal(t, 1, a.Added)
	assert.Equal(t, 1, a.Deleted)
	c, ok := fileByPath(d, "c.txt")
	require.True(t, ok, "c.txt missing: %+v", d.Files)
	assert.Equal(t, 2, c.Added)
	assert.Equal(t, 0, c.Deleted)
}

func TestCompute_reportsRenameAsDeleteAddRegardlessOfConfig(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	// diff.renames=true proves config can't collapse the rename: Compute
	// pins delete+add with --no-renames so the re-anchor algorithm always
	// sees content-bearing delete/add hunks.
	runGit(t, dir, "config", "diff.renames", "true")
	write(t, dir, "orig.txt", numbered(3))
	base := commitAll(t, dir, "base")
	runGit(t, dir, "mv", "orig.txt", "renamed.txt")
	head := commitAll(t, dir, "rename")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	require.Len(t, d.Files, 2)
	del, ok := fileByPath(d, "orig.txt")
	require.True(t, ok, "old path orig.txt missing (rename collapsed?): %+v", d.Files)
	assert.Equal(t, 0, del.Added)
	assert.Equal(t, 3, del.Deleted)
	add, ok := fileByPath(d, "renamed.txt")
	require.True(t, ok, "new path renamed.txt missing: %+v", d.Files)
	assert.Equal(t, 3, add.Added)
	assert.Equal(t, 0, add.Deleted)
}

func TestCompute_yieldsNoFilesForIdenticalRevisions(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(3))
	head := commitAll(t, dir, "base")

	d, err := diff.Compute(context.Background(), dir, head, head)
	require.NoError(t, err)
	assert.Empty(t, d.Files)
}

func TestCompute_attributesBinaryChangeToPathWithNoHunks(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 3, 4}, 0o644))
	base := commitAll(t, dir, "base")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte{9, 8, 7, 0, 6, 5}, 0o644))
	head := commitAll(t, dir, "change binary")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	f, ok := fileByPath(d, "data.bin")
	require.True(t, ok, "binary file must still be attributed to its path (from the diff --git header): %+v", d.Files)
	assert.Empty(t, f.Hunks)
	assert.Equal(t, 0, f.Added)
	assert.Equal(t, 0, f.Deleted)
}

func TestCompute_parsesMultipleHunksInOneFileSeparately(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(40))
	base := commitAll(t, dir, "base")
	lines := strings.Split(numbered(40), "\n")
	lines[4] = "five"
	lines[34] = "thirty-five"
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit 5 and 35")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	f, ok := fileByPath(d, "f.txt")
	require.True(t, ok, "f.txt missing: %+v", d.Files)
	require.Len(t, f.Hunks, 2)
	brackets := func(h diff.Hunk, line int) bool { return h.NewStart <= line && line <= h.NewStart+h.NewLines-1 }
	assert.True(t, brackets(f.Hunks[0], 5), "first hunk %+v must bracket line 5", f.Hunks[0])
	assert.True(t, brackets(f.Hunks[1], 35), "second hunk %+v must bracket line 35", f.Hunks[1])
}
