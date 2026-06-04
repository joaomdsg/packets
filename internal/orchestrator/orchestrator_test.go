package orchestrator

import (
	"context"
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
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "base")
	return dir
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasFile(out TurnOutcome, path string) bool {
	for _, f := range out.Diff.Files {
		if f.Path == path {
			return true
		}
	}
	return false
}

// A turn that changed the working tree must mint a revision whose SHA, diff,
// and add/delete stats describe the change — this is the revision.created /
// diff.data payload the review surface renders.
func TestChangedTurnMintsARevisionWithDiffAndStats(t *testing.T) {
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")
	write(t, dir, "f.txt", "one\nTWO\nthree\n") // modify line 2

	out, err := SettleTurn(context.Background(), dir, base, "turn")
	if err != nil {
		t.Fatalf("SettleTurn: %v", err)
	}
	if !out.Minted {
		t.Fatalf("a changed turn must mint a revision")
	}
	if head := runGit(t, dir, "rev-parse", "HEAD"); out.SHA != head {
		t.Errorf("SHA %q must equal new HEAD %q", out.SHA, head)
	}
	if !hasFile(out, "f.txt") {
		t.Errorf("diff must include f.txt, got %+v", out.Diff.Files)
	}
	if out.Added != 1 || out.Deleted != 1 {
		t.Errorf("one line modified: Added/Deleted = %d/%d, want 1/1", out.Added, out.Deleted)
	}
	if len(out.Secrets) != 0 {
		t.Errorf("clean change must surface no secrets, got %+v", out.Secrets)
	}
}

// A turn that changed nothing must mint no revision and not error — the no-edit
// guard (iter-8) composed through settle. HEAD must not move.
func TestNoEditTurnMintsNothing(t *testing.T) {
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	out, err := SettleTurn(context.Background(), dir, base, "turn")
	if err != nil {
		t.Fatalf("SettleTurn: %v", err)
	}
	if out.Minted {
		t.Errorf("a no-edit turn must not mint a revision, got SHA=%q", out.SHA)
	}
	if out.SHA != "" {
		t.Errorf("no revision means no SHA, got %q", out.SHA)
	}
	if len(out.Diff.Files) != 0 {
		t.Errorf("no revision means no diff, got %+v", out.Diff.Files)
	}
	if after := runGit(t, dir, "rev-parse", "HEAD"); after != base {
		t.Errorf("HEAD moved on a no-edit turn: %s -> %s", base, after)
	}
}

// A turn that introduces a secret must be BLOCKED: no revision, the secret
// surfaced, HEAD unmoved, no error — the iter-15 guard composed through settle.
func TestSecretTurnIsBlockedAndSurfaced(t *testing.T) {
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")
	write(t, dir, "conf.env", "API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n")

	out, err := SettleTurn(context.Background(), dir, base, "turn")
	if err != nil {
		t.Fatalf("a blocked secret is surfaced, not an error: %v", err)
	}
	if out.Minted {
		t.Errorf("a secret-bearing turn must not mint a revision")
	}
	if len(out.Secrets) == 0 {
		t.Fatalf("the secret must be surfaced in TurnOutcome.Secrets")
	}
	if len(out.Diff.Files) != 0 {
		t.Errorf("a blocked secret means no revision, so no diff must be computed; got %+v", out.Diff.Files)
	}
	if after := runGit(t, dir, "rev-parse", "HEAD"); after != base {
		t.Errorf("HEAD moved despite a blocked secret: %s -> %s", base, after)
	}
}

// The diff must be computed against the caller's baseRev, NOT a fixed HEAD~1.
// With two prior commits and baseRev pinned to the OLDER one, the diff must
// span everything since that older commit — a wrong-base impl (e.g. HEAD~1)
// would miss the intervening change.
func TestDiffIsComputedAgainstTheGivenBaseRevNotHeadParent(t *testing.T) {
	dir := initRepo(t) // base1: f.txt = "one\ntwo\nthree\n"
	base1 := runGit(t, dir, "rev-parse", "HEAD")
	// A second commit changes line 1; this is NOT the base we pass.
	write(t, dir, "f.txt", "ONE\ntwo\nthree\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "base2")
	// The turn changes line 3.
	write(t, dir, "f.txt", "ONE\ntwo\nTHREE\n")

	out, err := SettleTurn(context.Background(), dir, base1, "turn")
	if err != nil {
		t.Fatalf("SettleTurn: %v", err)
	}
	if !out.Minted {
		t.Fatalf("a changed turn must mint a revision")
	}
	// base1..newSHA spans BOTH the line-1 change (commit base2) and the line-3
	// change (the turn): two modified lines. HEAD~1..newSHA would show only one.
	if out.Added != 2 || out.Deleted != 2 {
		t.Errorf("diff vs base1 must span both changes: Added/Deleted = %d/%d, want 2/2", out.Added, out.Deleted)
	}
}

// A real failure from the composed steps (here: a non-existent repo dir makes
// settle's first git call fail) must propagate as an error, never be swallowed
// into a silent "no revision".
func TestUnderlyingGitFailurePropagatesAsError(t *testing.T) {
	_, err := SettleTurn(context.Background(), filepath.Join(t.TempDir(), "does-not-exist"), "deadbeef", "turn")
	if err == nil {
		t.Fatal("expected an error when the repo dir does not exist, got nil")
	}
}

// If settle commits successfully but the DIFF step then fails (e.g. an invalid
// baseRev), the error must propagate — the failure must not be swallowed into a
// minted-looking outcome with an empty diff.
func TestDiffFailureAfterCommitPropagatesAsError(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "f.txt", "one\nTWO\nthree\n") // a real change → settle will commit

	out, err := SettleTurn(context.Background(), dir, "nonexistent-base-rev", "turn")
	if err == nil {
		t.Fatalf("expected an error when diff.Compute fails on a bad baseRev, got nil (out=%+v)", out)
	}
	if out.Minted {
		t.Errorf("a diff failure must not yield a minted outcome, got %+v", out)
	}
}

// Stats are the TOTAL across all files the turn touched, so revision.created
// reports the whole changeset's size.
func TestStatsSumAcrossAllChangedFiles(t *testing.T) {
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")
	write(t, dir, "f.txt", "one\nTWO\nthree\n") // +1 / -1
	write(t, dir, "g.txt", "alpha\nbeta\n")      // +2 / -0 (new file)

	out, err := SettleTurn(context.Background(), dir, base, "turn")
	if err != nil {
		t.Fatalf("SettleTurn: %v", err)
	}
	if !out.Minted {
		t.Fatalf("a changed turn must mint a revision")
	}
	if len(out.Diff.Files) != 2 {
		t.Fatalf("want 2 changed files, got %+v", out.Diff.Files)
	}
	if out.Added != 3 || out.Deleted != 1 {
		t.Errorf("totals across files: Added/Deleted = %d/%d, want 3/1", out.Added, out.Deleted)
	}
}
