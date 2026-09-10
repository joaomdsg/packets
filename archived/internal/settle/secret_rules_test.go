package settle_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/settle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettle_blocksRevisionOnProviderTokenFormats(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		rule    string
		content string
	}{
		// Bare/comment form (no secret-name=value) so ONLY the dedicated provider
		// rule can catch it, proving each new rule's value over the generic
		// secret-assignment rule (which catches name=value).
		{"github classic PAT", "github-token", "// leaked ghp_" + strings.Repeat("a", 36)},
		{"github fine-grained PAT", "github-token", "// leaked github_pat_" + strings.Repeat("d", 22)},
		{"google api key", "google-api-key", "// leaked AIza" + strings.Repeat("b", 35)},
		{"slack bot token", "slack-token", "// leaked xoxb-" + strings.Repeat("0", 13)},
		{"stripe secret key", "stripe-secret-key", "// leaked sk_live_" + strings.Repeat("c", 24)},
		{"stripe restricted key", "stripe-secret-key", "// leaked rk_live_" + strings.Repeat("c", 24)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			dir := initRepo(t)
			before := runGit(t, dir, "rev-parse", "HEAD")
			require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte(c.content+"\n"), 0o644))

			res, err := settle.Settle(context.Background(), dir, "add token")
			require.NoError(t, err)
			assert.False(t, res.Committed)
			found := false
			for _, h := range res.Secrets {
				if h.Rule == c.rule {
					found = true
				}
			}
			assert.Truef(t, found, "want a %q hit, got %+v", c.rule, res.Secrets)
			assert.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"))
		})
	}
}

func TestSettle_doesNotFalseBlockOnProviderTokenLookalikes(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	content := strings.Join([]string{
		"ghp_short",                  // GitHub prefix but far too short
		"AIza",                       // bare Google prefix, no 35-char body
		"xox-",                       // not xox[baprs]- and no body
		"sk_test_abcdefghijklmnop12", // Stripe TEST key, not _live_
		"x := computeSomething()",    // ordinary code
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte(content), 0o644))

	res, err := settle.Settle(context.Background(), dir, "add lookalikes")
	require.NoError(t, err)
	require.True(t, res.Committed)
	assert.Empty(t, res.Secrets)
}
