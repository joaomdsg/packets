package settle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs a git command in dir and returns trimmed combined output,
// failing the test on error. Tests exercise real git because the settle
// guard's whole job is a git behaviour (commit vs. "nothing to commit").
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

// initRepo creates a fresh git repo with one base commit and returns its path.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "base.go"), []byte("package p\n\nfunc F() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "base")
	return dir
}

// A turn that changed nothing — the agent answered a `question:` with no code
// edit, or edited then reverted — must NOT mint a revision and must NOT error.
// DESIGN §12.2's unconditional `git add -A && git commit` fails here ("nothing
// to commit") and desyncs the state machine; the settle guard must absorb it.
func TestNoEditTurnMintsNoRevisionAndDoesNotError(t *testing.T) {
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")

	res, err := Settle(context.Background(), dir, "turn")
	if err != nil {
		t.Fatalf("Settle on a clean tree must not error, got %v", err)
	}
	if res.Committed {
		t.Errorf("clean tree must not mint a revision, got Committed=true SHA=%q", res.SHA)
	}
	if res.SHA != "" {
		t.Errorf("no revision means no SHA, got %q", res.SHA)
	}
	if after := runGit(t, dir, "rev-parse", "HEAD"); after != before {
		t.Errorf("HEAD moved on a clean tree: %s -> %s", before, after)
	}
}

// A turn that changed the working tree must become a reviewable revision: a new
// commit whose SHA is reported, leaving the tree committed clean.
func TestChangedTurnMintsARevision(t *testing.T) {
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(dir, "base.go"), []byte("package p\n\nfunc F() int { return 2 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "turn")
	if err != nil {
		t.Fatalf("Settle on a dirty tree: %v", err)
	}
	if !res.Committed {
		t.Fatalf("a changed tree must mint a revision, got Committed=false")
	}
	head := runGit(t, dir, "rev-parse", "HEAD")
	if res.SHA != head {
		t.Errorf("reported SHA %q must equal new HEAD %q", res.SHA, head)
	}
	if head == before {
		t.Errorf("HEAD did not advance after a changed turn")
	}
	if status := runGit(t, dir, "status", "--porcelain"); status != "" {
		t.Errorf("working tree must be clean after settle, got:\n%s", status)
	}
	if msg := runGit(t, dir, "log", "-1", "--format=%s"); msg != "turn" {
		t.Errorf("commit message = %q, want %q (the message argument must land in the commit)", msg, "turn")
	}
}

// A turn whose net effect is nothing — it staged a change then reverted the
// working tree back to HEAD's content — leaves `git status --porcelain`
// non-empty ("MM file": staged differs from HEAD, worktree differs from index)
// yet has nothing to commit once everything is staged. The porcelain pre-check
// alone would pass this through to `git commit`, which fails with "nothing to
// commit" and surfaces as an error — the very desync the guard exists to
// absorb. Such a net-revert turn must mint no revision and must not error.
func TestNetRevertedStagedChangeMintsNoRevisionAndDoesNotError(t *testing.T) {
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")

	// Stage a change, then revert the worktree to HEAD's content. Index now
	// holds a change vs HEAD, worktree matches HEAD -> porcelain shows "MM".
	if err := os.WriteFile(filepath.Join(dir, "base.go"), []byte("package p\n\nfunc F() int { return 7 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "base.go")
	if err := os.WriteFile(filepath.Join(dir, "base.go"), []byte("package p\n\nfunc F() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if status := runGit(t, dir, "status", "--porcelain"); status == "" {
		t.Fatalf("test precondition: porcelain should be non-empty (MM), got empty")
	}

	res, err := Settle(context.Background(), dir, "turn")
	if err != nil {
		t.Fatalf("a net-reverted turn must not error, got %v", err)
	}
	if res.Committed {
		t.Errorf("a net-reverted turn must not mint a revision, got Committed=true SHA=%q", res.SHA)
	}
	if res.SHA != "" {
		t.Errorf("no revision means no SHA, got %q", res.SHA)
	}
	if after := runGit(t, dir, "rev-parse", "HEAD"); after != before {
		t.Errorf("HEAD moved on a net-reverted tree: %s -> %s", before, after)
	}
}

// New untracked files a turn creates (e.g. a brand-new source file) must be
// included in the revision — the guard stages all changes, not only edits to
// already-tracked files.
func TestNewUntrackedFileIsCommitted(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "new.go"), []byte("package p\n\nfunc G() int { return 9 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "add new file")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if !res.Committed {
		t.Fatalf("a new untracked file must mint a revision, got Committed=false")
	}
	// `git cat-file -e HEAD:new.go` exits non-zero (failing the test via runGit)
	// if the new file was not committed.
	runGit(t, dir, "cat-file", "-e", "HEAD:new.go")
	if status := runGit(t, dir, "status", "--porcelain"); status != "" {
		t.Errorf("tree must be clean after committing the new file, got:\n%s", status)
	}
}
