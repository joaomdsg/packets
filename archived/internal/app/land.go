package app

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/joaomdsg/packets/internal/pipe"
)

// landBlocked reports whether approving (opening a PR) should be refused, and the
// honest reason — mirroring PR etiquette (DESIGN §16): you don't merge with a tree
// that won't integrate (a rebase conflict or red integrated checks) or with unresolved
// review threads. It reports the FIRST blocker so the approve flow can refuse (the
// caller decides whether to honor an override). A clean tree with no open threads is
// clear to land. A PENDING land verdict ("") is allowed — for a direct-branch PR, CI
// runs on the PR itself, so a not-yet-integrated verdict is not a hard block.
func landBlocked(openThreads int, land pipe.LandState) (bool, string) {
	switch land {
	case pipe.LandConflict:
		return true, "held: rebase needed — the fix conflicts with trunk tip"
	case pipe.LandChecksRed:
		return true, "held: checks red — the integrated tree fails its tests"
	}
	if openThreads > 0 {
		return true, strconv.Itoa(openThreads) + " open threads — resolve them first"
	}
	return false, ""
}

// prBranchName derives a stable, valid git ref for a session's pushed branch from its
// key: ref-hostile characters (spaces, '/', '#', …) collapse to '-', and the result is
// namespaced under "packets/" so it never collides with the trunk or a user branch.
// Deterministic per key (the same session lands to the same branch) and distinct
// across keys.
func prBranchName(key string) string {
	var b strings.Builder
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "session"
	}
	return "packets/" + s
}

// prTitleMaxLen bounds the generated PR title — git/GitHub balk at enormous subject
// lines, and a PR title is a one-liner by convention.
const prTitleMaxLen = 72

// prTitleAndBody derives the opened PR's title and body from the task prompt: the
// title is the prompt's first line (truncated to a sane length), and the body carries
// the full task and links back to the originating review session — so the PR is
// legible and traceable, the session becoming its pre-history (DESIGN §16).
func prTitleAndBody(key, prompt string) (title, body string) {
	prompt = strings.TrimSpace(prompt)
	title = prompt
	if i := strings.IndexByte(title, '\n'); i >= 0 {
		title = strings.TrimSpace(title[:i])
	}
	if len(title) > prTitleMaxLen {
		title = title[:prTitleMaxLen]
		// Back off any trailing bytes of a rune the byte-slice split, so the title stays
		// valid UTF-8 within the budget (a stray half-rune is a malformed title).
		for len(title) > 0 && !utf8.ValidString(title) {
			title = title[:len(title)-1]
		}
		title = strings.TrimSpace(title)
	}
	if title == "" {
		title = "packets session " + key
	}
	body = prompt + "\n\n—\nOpened from packets review session `" + key + "`."
	return title, body
}
