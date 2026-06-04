package diff

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
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

// commitAll stages everything and commits, returning the new commit's SHA.
func commitAll(t *testing.T, dir, msg string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", msg)
	return runGit(t, dir, "rev-parse", "HEAD")
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
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

func fileByPath(d Diff, path string) (FileDiff, bool) {
	for _, f := range d.Files {
		if f.Path == path {
			return f, true
		}
	}
	return FileDiff{}, false
}

// A modified file must surface as one FileDiff with its path, the right
// add/delete counts, and a hunk that brackets the changed line — this is the
// substrate the review diff and the re-anchor algorithm both read.
func TestModifiedFileReportsPathCountsAndAHunkAroundTheChange(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(20))
	base := commitAll(t, dir, "base")
	// Change only line 10.
	lines := strings.Split(numbered(20), "\n")
	lines[9] = "ten"
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit line 10")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	f, ok := fileByPath(d, "f.txt")
	if !ok {
		t.Fatalf("f.txt missing from diff: %+v", d.Files)
	}
	if f.Added != 1 || f.Deleted != 1 {
		t.Errorf("one line changed: Added/Deleted = %d/%d, want 1/1", f.Added, f.Deleted)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("want exactly 1 hunk, got %d: %+v", len(f.Hunks), f.Hunks)
	}
	h := f.Hunks[0]
	if !(h.NewStart <= 10 && 10 <= h.NewStart+h.NewLines-1) {
		t.Errorf("hunk %+v must bracket changed new-line 10", h)
	}
	if !(h.OldStart <= 10 && 10 <= h.OldStart+h.OldLines-1) {
		t.Errorf("hunk %+v must bracket old-line 10", h)
	}
}

// The hunk range must be the actual changed region, NOT a fabricated
// whole-file span — otherwise the re-anchor algorithm would treat every line
// as touched. Changing one line deep inside a 40-line file yields git's exact
// 3-context hunk `@@ -17,7 +17,7 @@`; a whole-file impl would give -1,40 and
// fail. (Pins exact non-trivial ranges; assumes git's stable default 3 lines
// of context.)
func TestModifiedHunkIsTheChangedRegionNotTheWholeFile(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(40))
	base := commitAll(t, dir, "base")
	lines := strings.Split(numbered(40), "\n")
	lines[19] = "twenty" // line 20
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit line 20")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	f, ok := fileByPath(d, "f.txt")
	if !ok {
		t.Fatalf("f.txt missing: %+v", d.Files)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("want 1 hunk, got %+v", f.Hunks)
	}
	want := Hunk{OldStart: 17, OldLines: 7, NewStart: 17, NewLines: 7}
	if f.Hunks[0] != want {
		t.Errorf("hunk = %+v, want %+v (the changed region, not the whole file)", f.Hunks[0], want)
	}
}

// A one-line file changed in place makes git emit the count-omitted hunk form
// `@@ -1 +1 @@` (count defaults to 1). The parser must read that as
// OldLines/NewLines == 1, not 0 — exercising the omitted-count branch.
func TestSingleLineFileChangeExercisesOmittedHunkCount(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "f.txt", "alpha\n")
	base := commitAll(t, dir, "base")
	write(t, dir, "f.txt", "omega\n")
	head := commitAll(t, dir, "change the only line")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	f, ok := fileByPath(d, "f.txt")
	if !ok {
		t.Fatalf("f.txt missing: %+v", d.Files)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("want 1 hunk, got %+v", f.Hunks)
	}
	want := Hunk{OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 1}
	if f.Hunks[0] != want {
		t.Errorf("count-omitted hunk = %+v, want %+v (omitted count means 1)", f.Hunks[0], want)
	}
}

// A brand-new file has a context-independent hunk: old side empty (0,0), new
// side starting at line 1. This pins exact hunk ranges without depending on
// git's context width.
func TestNewFileHunkStartsAtLineOneWithEmptyOldSide(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "seed.txt", "x\n")
	base := commitAll(t, dir, "seed")
	write(t, dir, "new.txt", numbered(3))
	head := commitAll(t, dir, "add new.txt")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	f, ok := fileByPath(d, "new.txt")
	if !ok {
		t.Fatalf("new.txt missing: %+v", d.Files)
	}
	if f.Added != 3 || f.Deleted != 0 {
		t.Errorf("new 3-line file: Added/Deleted = %d/%d, want 3/0", f.Added, f.Deleted)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("want 1 hunk, got %+v", f.Hunks)
	}
	h := f.Hunks[0]
	want := Hunk{OldStart: 0, OldLines: 0, NewStart: 1, NewLines: 3}
	if h != want {
		t.Errorf("new-file hunk = %+v, want %+v", h, want)
	}
}

// Deleting lines must be reflected in the counts (Added 0, Deleted N).
func TestDeletedLinesAreCounted(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(6))
	base := commitAll(t, dir, "base")
	// Remove lines 3 and 4.
	lines := strings.Split(numbered(6), "\n")
	kept := append(append([]string{}, lines[:2]...), lines[4:]...)
	write(t, dir, "f.txt", strings.Join(kept, "\n"))
	head := commitAll(t, dir, "delete 3-4")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	f, ok := fileByPath(d, "f.txt")
	if !ok {
		t.Fatalf("f.txt missing: %+v", d.Files)
	}
	if f.Added != 0 || f.Deleted != 2 {
		t.Errorf("two lines deleted: Added/Deleted = %d/%d, want 0/2", f.Added, f.Deleted)
	}
}

// Several files in one diff must each become their own FileDiff with no
// cross-file bleed of hunks or counts.
func TestMultipleFilesEachBecomeASeparateFileDiff(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "a.txt", numbered(10))
	base := commitAll(t, dir, "base")
	// Modify a.txt and add c.txt.
	la := strings.Split(numbered(10), "\n")
	la[4] = "five"
	write(t, dir, "a.txt", strings.Join(la, "\n"))
	write(t, dir, "c.txt", numbered(2))
	head := commitAll(t, dir, "edit a, add c")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(d.Files) != 2 {
		t.Fatalf("want 2 files, got %d: %+v", len(d.Files), d.Files)
	}
	a, ok := fileByPath(d, "a.txt")
	if !ok {
		t.Fatalf("a.txt missing: %+v", d.Files)
	}
	if a.Added != 1 || a.Deleted != 1 {
		t.Errorf("a.txt counts = %d/%d, want 1/1", a.Added, a.Deleted)
	}
	c, ok := fileByPath(d, "c.txt")
	if !ok {
		t.Fatalf("c.txt missing: %+v", d.Files)
	}
	if c.Added != 2 || c.Deleted != 0 {
		t.Errorf("c.txt counts = %d/%d, want 2/0", c.Added, c.Deleted)
	}
}

// A rename must surface deterministically as a delete (old path) + an add (new
// path) — two FileDiffs — regardless of the repo/user `diff.renames` config.
// Compute pins this with --no-renames; without it, a config-enabled rename
// detection collapses the rename into a single 0/0-count FileDiff, breaking the
// re-anchor algorithm (which expects content-bearing delete/add hunks). The
// repo here sets diff.renames=true to prove config can't change the result.
func TestRenameIsDeleteAddRegardlessOfConfig(t *testing.T) {
	dir := initRepo(t)
	runGit(t, dir, "config", "diff.renames", "true")
	write(t, dir, "orig.txt", numbered(3))
	base := commitAll(t, dir, "base")
	runGit(t, dir, "mv", "orig.txt", "renamed.txt")
	head := commitAll(t, dir, "rename")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(d.Files) != 2 {
		t.Fatalf("rename must be delete+add = 2 files, got %d: %+v", len(d.Files), d.Files)
	}
	del, ok := fileByPath(d, "orig.txt")
	if !ok {
		t.Fatalf("old path orig.txt missing (rename collapsed?): %+v", d.Files)
	}
	if del.Added != 0 || del.Deleted != 3 {
		t.Errorf("old path counts = %d/%d, want 0/3", del.Added, del.Deleted)
	}
	add, ok := fileByPath(d, "renamed.txt")
	if !ok {
		t.Fatalf("new path renamed.txt missing: %+v", d.Files)
	}
	if add.Added != 3 || add.Deleted != 0 {
		t.Errorf("new path counts = %d/%d, want 3/0", add.Added, add.Deleted)
	}
}

// Diffing a revision against itself yields no files and no error — the common
// "nothing changed between these two revisions" case.
func TestIdenticalRevisionsYieldNoFiles(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(3))
	head := commitAll(t, dir, "base")

	d, err := Compute(context.Background(), dir, head, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(d.Files) != 0 {
		t.Errorf("identical revisions must yield no files, got %+v", d.Files)
	}
}

// A binary file change has no `+++ b/<path>` header and no textual hunks, so
// its path must come from the `diff --git a/.. b/..` header instead — the file
// must still be attributed (not dropped or left path-less), with no hunks and
// zero line counts.
func TestBinaryFileChangeIsAttributedToItsPathWithNoHunks(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0, 1, 2, 3, 4}, 0o644); err != nil {
		t.Fatal(err)
	}
	base := commitAll(t, dir, "base")
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{9, 8, 7, 0, 6, 5}, 0o644); err != nil {
		t.Fatal(err)
	}
	head := commitAll(t, dir, "change binary")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	f, ok := fileByPath(d, "data.bin")
	if !ok {
		t.Fatalf("binary file must still be attributed to its path (from the diff --git header): %+v", d.Files)
	}
	if len(f.Hunks) != 0 {
		t.Errorf("a binary diff has no textual hunks, got %+v", f.Hunks)
	}
	if f.Added != 0 || f.Deleted != 0 {
		t.Errorf("a binary diff has no +/- content lines, got %d/%d", f.Added, f.Deleted)
	}
}

// Two well-separated edits in one file must produce two distinct hunks, each
// with its own ranges — the re-anchor algorithm maps threads through these.
func TestMultipleHunksInOneFileAreParsedSeparately(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "f.txt", numbered(40))
	base := commitAll(t, dir, "base")
	lines := strings.Split(numbered(40), "\n")
	lines[4] = "five"   // line 5
	lines[34] = "thirty-five" // line 35
	write(t, dir, "f.txt", strings.Join(lines, "\n"))
	head := commitAll(t, dir, "edit 5 and 35")

	d, err := Compute(context.Background(), dir, base, head)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	f, ok := fileByPath(d, "f.txt")
	if !ok {
		t.Fatalf("f.txt missing: %+v", d.Files)
	}
	if len(f.Hunks) != 2 {
		t.Fatalf("two separated edits must give 2 hunks, got %d: %+v", len(f.Hunks), f.Hunks)
	}
	brackets := func(h Hunk, line int) bool { return h.NewStart <= line && line <= h.NewStart+h.NewLines-1 }
	if !brackets(f.Hunks[0], 5) {
		t.Errorf("first hunk %+v must bracket line 5", f.Hunks[0])
	}
	if !brackets(f.Hunks[1], 35) {
		t.Errorf("second hunk %+v must bracket line 35", f.Hunks[1])
	}
}
