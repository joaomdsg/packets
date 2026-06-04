package settle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Each well-known provider token format a turn introduces must block the
// revision and be surfaced under its own rule name — so a leaked GitHub/Google/
// Slack/Stripe credential never reaches history.
func TestProviderTokenFormatsBlockTheRevision(t *testing.T) {
	cases := []struct {
		name    string
		rule    string
		content string
	}{
		// Bare/comment form (no secret-name=value) so ONLY the dedicated provider
		// rule can catch it — proving each new rule's added value over the
		// existing generic secret-assignment rule (which catches name=value).
		{"github classic PAT", "github-token", "// leaked ghp_" + strings.Repeat("a", 36)},
		{"github fine-grained PAT", "github-token", "// leaked github_pat_" + strings.Repeat("d", 22)},
		{"google api key", "google-api-key", "// leaked AIza" + strings.Repeat("b", 35)},
		{"slack bot token", "slack-token", "// leaked xoxb-" + strings.Repeat("0", 13)},
		{"stripe secret key", "stripe-secret-key", "// leaked sk_live_" + strings.Repeat("c", 24)},
		{"stripe restricted key", "stripe-secret-key", "// leaked rk_live_" + strings.Repeat("c", 24)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := initRepo(t)
			before := runGit(t, dir, "rev-parse", "HEAD")
			if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(c.content+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			res, err := Settle(context.Background(), dir, "add token")
			if err != nil {
				t.Fatalf("a detected secret is a surfaced block, not an error: %v", err)
			}
			if res.Committed {
				t.Errorf("%s must block the commit", c.name)
			}
			found := false
			for _, h := range res.Secrets {
				if h.Rule == c.rule {
					found = true
				}
			}
			if !found {
				t.Errorf("want a %q hit, got %+v", c.rule, res.Secrets)
			}
			if after := runGit(t, dir, "rev-parse", "HEAD"); after != before {
				t.Errorf("HEAD moved despite a blocked secret: %s -> %s", before, after)
			}
		})
	}
}

// The new rules must NOT cry wolf on ordinary code: tokens that are too short
// or the wrong shape (a prefix without the required length, a test-mode key)
// must commit cleanly. This proves the length/shape anchors actually bite.
func TestProviderTokenLookalikesDoNotFalseBlock(t *testing.T) {
	dir := initRepo(t)
	content := strings.Join([]string{
		"ghp_short",                  // GitHub prefix but far too short
		"AIza",                       // bare Google prefix, no 35-char body
		"xox-",                       // not xox[baprs]- and no body
		"sk_test_abcdefghijklmnop12", // Stripe TEST key, not _live_
		"x := computeSomething()",    // ordinary code
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Settle(context.Background(), dir, "add lookalikes")
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if !res.Committed {
		t.Fatalf("lookalikes must not block a commit; got Secrets=%+v", res.Secrets)
	}
	if len(res.Secrets) != 0 {
		t.Errorf("no false-positive hits expected, got %+v", res.Secrets)
	}
}
