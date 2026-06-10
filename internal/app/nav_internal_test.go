package app

import (
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// A session card with no way BACK to the fleet strands the Lead. Every page
// carries the nav header, and the card's "fleet" crumb links back to /board.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_rendersNavWithBackToFleet(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "board-nav", "the session card carries the nav header")
	require.Contains(t, body, `href="/board"`, "the card links BACK to the fleet, so the Lead is never stranded")
	// The breadcrumb shows the REAL session key (here the default), not a
	// fabricated label — the Lead always knows which session they're on.
	require.Contains(t, body, "board-nav__key", "the card breadcrumb names the current session")
	require.Contains(t, body, defaultSessionKey, "the breadcrumb shows the real key, not a renamed label")
}

// The fleet board is a DEAD END unless each row drills into its session: the row
// key is a link to that session's card (/?key=<key>). Without it the Lead can see
// the fleet but never reach a session.
func TestBoardCard_rendersNavAndDrillsIntoASession(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	alphaLog, err := AddSession("alpha", LiveConfig{
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
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	const trickyKey = "a&b=c"
	trickyLog, err := AddSession(trickyKey, LiveConfig{
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
