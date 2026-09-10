package app

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/pipe"
)

// Landing mirrors PR etiquette (DESIGN §16): you don't open a PR with unresolved
// review threads or a tree that won't integrate. landBlocked reports the FIRST honest
// blocker so the approve flow can refuse (overridably) — a conflict or red checks (the
// tree can't land) or open threads (the review isn't done). A clean tree with no open
// threads is clear to land.
func TestLandBlocked_reportsTheHonestBlocker(t *testing.T) {
	cases := []struct {
		name    string
		threads int
		land    pipe.LandState
		blocked bool
		needle  string
	}{
		{"clean, no threads", 0, pipe.LandClean, false, ""},
		{"rebase conflict blocks", 0, pipe.LandConflict, true, "rebase"},
		{"red checks block", 0, pipe.LandChecksRed, true, "checks"},
		{"open threads block", 3, pipe.LandClean, true, "3"},
		{"pending land is allowed (CI runs on the PR)", 0, pipe.LandState(""), false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			blocked, reason := landBlocked(tc.threads, tc.land)
			assert.Equal(t, tc.blocked, blocked)
			if tc.needle != "" {
				assert.Contains(t, strings.ToLower(reason), strings.ToLower(tc.needle), "the reason names the blocker")
			}
		})
	}
}

// The pushed branch name must be a VALID git ref derived deterministically from the
// session key — spaces and ref-hostile characters collapsed — so two sessions get
// distinct, pushable branches and the same session is stable across lands.
func TestPRBranchName_isAStableValidRef(t *testing.T) {
	assert.Equal(t, prBranchName("feature x"), prBranchName("feature x"), "deterministic for a key")
	b := prBranchName("feat/Auth #2")
	assert.NotContains(t, b, " ", "no spaces in a ref")
	assert.NotContains(t, b, "#", "no ref-hostile characters")
	assert.NotEqual(t, prBranchName("a"), prBranchName("b"), "distinct keys get distinct branches")
	assert.True(t, strings.HasPrefix(b, "packets/"), "branches are namespaced under packets/")
}

// The PR title and body must carry the work: a concise title from the task prompt and
// a body that summarizes it and links back to the review session — so the opened PR is
// legible, not an empty shell.
func TestPRTitleAndBody_carryTheTaskAndLinkBack(t *testing.T) {
	title, body := prTitleAndBody("sessionkey", "Add retry logic to the uploader.\n\nMore detail here.")
	assert.Equal(t, "Add retry logic to the uploader.", title, "the title is the prompt's first line")
	assert.Contains(t, body, "More detail here.", "the body carries the full task")
	assert.Contains(t, body, "sessionkey", "the body links back to the review session")
}

// A very long first line must be truncated into a sane PR title (git/GitHub balk at
// enormous titles), never dumped whole.
func TestPRTitle_truncatesAnOverlongFirstLine(t *testing.T) {
	long := strings.Repeat("x", 200)
	title, _ := prTitleAndBody("k", long)
	assert.LessOrEqual(t, len(title), 72, "the title is truncated to a sane length")
}

// Truncating an overlong title must not split a multi-byte UTF-8 rune — a PR title
// with a stray invalid byte is malformed. The truncation backs off to a rune boundary,
// staying within the byte budget AND valid UTF-8.
func TestPRTitle_truncatesOnARuneBoundary(t *testing.T) {
	long := "a" + strings.Repeat("€", 100) // 1 ASCII + 3-byte runes → byte 72 lands mid-rune
	title, _ := prTitleAndBody("k", long)
	assert.LessOrEqual(t, len(title), prTitleMaxLen, "within the byte budget")
	assert.True(t, utf8.ValidString(title), "no split rune — the title stays valid UTF-8")
}
