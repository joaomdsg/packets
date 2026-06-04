package settle

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// A PEM private key introduced by a turn must block the revision: committing it
// would leak the key into git history. The block is surfaced (a SecretHit), not
// an error, and no commit is made.
func TestStagedPrivateKeyBlocksTheRevision(t *testing.T) {
	dir := initRepo(t)
	before := runGit(t, dir, "rev-parse", "HEAD")
	const pem = "-----BEGIN RSA PRIVATE KEY-----\nMIIBOgIBAAExample\n-----END RSA PRIVATE KEY-----\n"
	if err := os.WriteFile(filepath.Join(dir, "key.pem"), []byte(pem), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "add key")
	if err != nil {
		t.Fatalf("a detected secret is a surfaced block, not an error; got %v", err)
	}
	if res.Committed {
		t.Errorf("must not commit a staged private key, got Committed=true")
	}
	if len(res.Secrets) == 0 {
		t.Fatalf("the private key must be surfaced as a SecretHit")
	}
	// Order-independent: some hit must point at key.pem and name a rule.
	var found bool
	for _, h := range res.Secrets {
		if h.File == "key.pem" {
			found = true
			if h.Rule == "" {
				t.Errorf("hit must name the rule that matched")
			}
		}
	}
	if !found {
		t.Errorf("expected a secret hit on key.pem, got %+v", res.Secrets)
	}
	if after := runGit(t, dir, "rev-parse", "HEAD"); after != before {
		t.Errorf("HEAD moved despite a blocked secret: %s -> %s", before, after)
	}
}

// An AWS access key id introduced by a turn must block the revision.
func TestStagedAWSKeyBlocksTheRevision(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "creds.txt"), []byte("aws_key = AKIAIOSFODNN7EXAMPLE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Settle(context.Background(), dir, "add creds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Committed || len(res.Secrets) == 0 {
		t.Fatalf("an AWS key must block the commit and surface a hit; got Committed=%v Secrets=%+v", res.Committed, res.Secrets)
	}
}

// A secret-named env assignment of a long opaque value must block the revision.
func TestStagedSecretAssignmentBlocksTheRevision(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "config.env"), []byte("API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Settle(context.Background(), dir, "add config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Committed || len(res.Secrets) == 0 {
		t.Fatalf("a secret assignment must block the commit; got Committed=%v Secrets=%+v", res.Committed, res.Secrets)
	}
}

// The hit must be anchored to the exact line the secret was ADDED on, so the
// reviewer is pointed at it — not merely told "somewhere in this file". Here
// the secret sits on the third line of a new file, so the hit's Line must be 3.
func TestSecretHitIsAnchoredToTheAddedLineNumber(t *testing.T) {
	dir := initRepo(t)
	content := "# config\ndebug=true\nAPI_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"
	if err := os.WriteFile(filepath.Join(dir, "conf.env"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "add conf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Committed || len(res.Secrets) == 0 {
		t.Fatalf("the secret on line 3 must block the commit; got Committed=%v Secrets=%+v", res.Committed, res.Secrets)
	}
	var line int
	for _, h := range res.Secrets {
		if h.File == "conf.env" {
			line = h.Line
		}
	}
	if line != 3 {
		t.Errorf("secret hit anchored to line %d, want 3 (the line it was added on); Secrets=%+v", line, res.Secrets)
	}
}

// Ordinary source with no secret must commit normally with no secret hits —
// the scanner must not cry wolf on plain code.
func TestCleanSourceCommitsWithNoSecretHits(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\n\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Settle(context.Background(), dir, "add ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Committed {
		t.Fatalf("clean source must commit, got Committed=false (Secrets=%+v)", res.Secrets)
	}
	if len(res.Secrets) != 0 {
		t.Errorf("clean source must yield no secret hits, got %+v", res.Secrets)
	}
}

// The scan must hold under a hostile or merely customized git config. With
// color.diff=always git colorizes `git diff --cached` with ANSI escapes, so an
// added line no longer starts with a bare "+" — and diff.noprefix rewrites the
// "+++ b/<path>" header. If Settle scanned that decorated output, the secret
// line would never be recognized as added and would slip into history. Settle
// must force canonical diff output so the secret is still blocked.
func TestSecretScanHoldsUnderHostileGitConfig(t *testing.T) {
	dir := initRepo(t)
	runGit(t, dir, "config", "color.diff", "always")
	runGit(t, dir, "config", "diff.noprefix", "true")
	before := runGit(t, dir, "rev-parse", "HEAD")

	if err := os.WriteFile(filepath.Join(dir, "leak.env"), []byte("API_KEY=\"ABCDEFGHIJKLMNOP1234\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Settle(context.Background(), dir, "leak")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Committed {
		t.Errorf("secret must be blocked even under color.diff=always/diff.noprefix, got Committed=true")
	}
	if len(res.Secrets) == 0 {
		t.Fatalf("the secret must be surfaced under a hostile git config; got Secrets=%+v", res.Secrets)
	}
	if after := runGit(t, dir, "rev-parse", "HEAD"); after != before {
		t.Errorf("HEAD moved despite a blocked secret under hostile config: %s -> %s", before, after)
	}
}

// Only lines this turn ADDS are scanned. A secret already living in the base
// commit, untouched by this turn, must not be re-flagged — otherwise every
// later turn would be blocked by a pre-existing secret it didn't introduce.
func TestPreexistingSecretNotTouchedThisTurnIsNotFlagged(t *testing.T) {
	dir := initRepo(t)
	// A secret already in history (committed directly, not via Settle).
	if err := os.WriteFile(filepath.Join(dir, "old.txt"), []byte("aws_key = AKIAIOSFODNN7EXAMPLE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", "preexisting secret")

	// This turn touches a DIFFERENT, clean file.
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\n\nfunc Z() int { return 0 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Settle(context.Background(), dir, "unrelated change")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Committed {
		t.Fatalf("an unrelated clean change must commit; a pre-existing secret must not block it (Secrets=%+v)", res.Secrets)
	}
	if len(res.Secrets) != 0 {
		t.Errorf("pre-existing untouched secret must not be flagged, got %+v", res.Secrets)
	}
}
