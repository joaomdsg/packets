package app_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
)

// A session card with no way BACK to the fleet strands the Lead. Every page
// carries the nav header, and the card's "fleet" crumb links back to /board.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_rendersNavWithBackToFleet(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "board-nav", "the session card carries the nav header")
	require.Contains(t, body, `href="/board"`, "the card links BACK to the fleet, so the Lead is never stranded")
	// The breadcrumb shows the REAL session key (here the default), not a
	// fabricated label — the Lead always knows which session they're on.
	require.Contains(t, body, "board-nav__key", "the card breadcrumb names the current session")
	require.Contains(t, body, defaultKey, "the breadcrumb shows the real key, not a renamed label")
}

// The fleet board is a DEAD END unless each row drills into its session: the row
// key is a link to that session's card (/?key=<key>). Without it the Lead can see
// the fleet but never reach a session.
func TestBoardCard_rendersNavAndDrillsIntoASession(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	alphaLog, err := app.AddSession("alpha", app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = alphaLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "board-nav", "the board carries the nav header")
	require.Contains(t, body, "board-nav__home", "with a packets home link")
	require.Contains(t, body, `href="/?key=alpha"`, "the alpha row drills into its session card — the board is not a dead end")
	require.Contains(t, body, `href="/?key=default"`, "the pre-seeded default row drills too — every session is reachable")
	// The drill target is a real link element, not loose text.
	require.Contains(t, strings.ToLower(body), `<a href="/?key=alpha"`, "the row key is an anchor")
}

// A session key only has to pass fabric.ValidToken, which forbids '.', whitespace
// and the NATS wildcards — but NOT query metacharacters like '&', '=', '#', '+'.
// Interpolated raw into /?key=<key>, such a key would split or truncate the query
// so the drill link targets the WRONG session (or none). The href must URL-escape
// the key so the link round-trips to the exact session the row names.
func TestBoardCard_drillHrefURLEscapesTheKey(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	const trickyKey = "a&b=c"
	trickyLog, err := app.AddSession(trickyKey, app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = trickyLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	// The query value must be percent-encoded so '&'/'=' can't split the query —
	// the link round-trips to exactly this session, not "a".
	require.Contains(t, body, `href="/?key=`+url.QueryEscape(trickyKey)+`"`,
		"the drill href URL-escapes the key so it targets the exact session")
	// The raw, query-splitting form must NOT appear as the drill href.
	require.NotContains(t, body, `href="/?key=a&b=c"`,
		"the raw key must not leak unescaped into the query string")
}

// The mark + chrome: the nav's home link used to be bare
// text ("packets") — the brand pack locks the mark + stacked lockup as the
// in-app chrome instead. A regression back to plain text would silently drop
// the brand from every page. NOT parallel (shared liveReg/liveFabric).
func TestBoardCard_navHomeLinkIsTheMarkAndStackedLockupNotBareText(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "board-nav__home", "the home link keeps its stable class hook")
	require.Contains(t, body, `href="/board"`, "the home link still navigates to the fleet board")
	require.Contains(t, body, "pk-mark", "the home link mounts the brand mark")
	require.Contains(t, body, "pk-lockup__sub", "the home link mounts the stacked lockup, not the mark alone")
	require.Contains(t, body, `pk-lockup__sub">console<`, "the fleet board's lockup sublabel names its own surface")
	require.NotContains(t, body, `class="board-nav__home">packets<`,
		"the home link must no longer be a bare text-only wordmark")
}

// The session card ("/") carries the same shared nav header as the board —
// it must rebrand identically, not regress to the old bare-text link.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_navHomeLinkIsTheMarkAndStackedLockup(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "pk-mark", "the session card's home link mounts the brand mark")
	require.Contains(t, body, `pk-lockup__sub">console<`, "the session card's lockup sublabel names its own surface")
	require.NotContains(t, body, `class="board-nav__home">packets<`,
		"the home link must no longer be a bare text-only wordmark")
}

// /review and /settings share the same nav header component — each surface's
// lockup sublabel must name ITS surface (Inspect / Settings), not a copy-pasted
// or hardcoded one, so the Lead always knows which surface they're on.
// NOT parallel (shared liveReg/liveFabric).
func TestReviewAndSettings_navLockupSublabelNamesTheirOwnSurface(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	review := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, review, "pk-mark", "/review mounts the brand mark")
	require.Contains(t, review, `pk-lockup__sub">inspect<`, "/review's lockup sublabel is its own surface name")

	settings := bodyOf(vt.NewClient(t, server, "/settings").HTML())
	require.Contains(t, settings, "pk-mark", "/settings mounts the brand mark")
	require.Contains(t, settings, `pk-lockup__sub">settings<`, "/settings' lockup sublabel is its own surface name")
}
