package app

import (
	"context"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// FLOW C: /review is a drill-in with no way BACK to the session card it came from —
// the Lead reviews questions then dead-ends. A back-affordance links to the
// originating session card (/?key=<key>), reusing the breadcrumb idiom. NOT parallel
// (shared liveReg/liveFabric).
func TestReviewCard_linksBackToTheOriginatingSessionCard(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=default").HTML())
	require.Contains(t, body, `href="/?key=default"`,
		"the review surface links BACK to the originating session card, never dead-ends")
}

// FLOW C: /settings is a drill-in too — saving a key then having no way back to the
// card strands the Lead. It links back to the session card (/?key=<key>).
func TestSettingsCard_linksBackToTheSessionCard(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/settings").HTML())
	require.Contains(t, body, `href="/?key=`+url.QueryEscape(defaultSessionKey)+`"`,
		"the settings surface links BACK to the session card, never dead-ends")
}

// A11y: the drill-return back-affordance is a navigation control and must sit inside
// a <nav> landmark, not be stranded as bare body text — so an assistive-tech user can
// reach it via landmark navigation. The crumb wraps in its own labelled nav. NOT
// parallel (shared liveReg/liveFabric).
func TestDrillReturnCrumb_sitsInsideANavigationLandmark(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	for _, path := range []string{"/review?key=default", "/settings"} {
		body := bodyOf(vt.NewClient(t, server, path).HTML())
		require.Contains(t, body, `aria-label="return"`,
			"%s wraps the drill-return crumb in a labelled nav landmark", path)
		idx := strings.Index(body, `aria-label="return"`)
		seg := body[idx:]
		require.Contains(t, seg, "drill-return",
			"%s nav landmark contains the back-affordance crumb", path)
	}
}

// FLOW C: per-order /review?wo= and session /review are asymmetric — the Lead can't
// move between the two without dead-ending. Each exposes a link to the other so
// review navigation is symmetric. NOT parallel (shared liveReg + the resolveCycle
// seam).
func TestReviewCard_perOrderAndSessionReviewLinkToEachOther(t *testing.T) {
	resetConsumersForTest()
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Findings: []mutation.Finding{
			{File: "alpha.go", Line: 7, Outcome: mutation.Survived, Message: "mutated >= to >"},
		}}, nil
	}
	restoreReader := reviewFileReader
	t.Cleanup(func() { reviewFileReader = restoreReader })
	reviewFileReader = func(_ context.Context, _, _, _ string) (string, error) {
		return "package main\n", nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "sym", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	registerSession("sym", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	drainQueuedOrders("sym")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	// The per-order review links to the SESSION review (drop the wo scope) so the
	// Lead can move from a funded order's test-debt up to the session's open
	// questions without dead-ending.
	orderBody := bodyOf(vt.NewClient(t, server, "/review?key=sym&wo=1").HTML())
	require.Contains(t, orderBody, `href="/review?key=sym"`,
		"the per-order review links UP to the session review — not a dead end")

	// The reverse edge (session → a per-order review) is the card's dispatch rows,
	// which already expose /review?...&wo=<id>; the session review links back to the
	// originating card so that reverse path stays reachable (Flow C return loop).
	sessionBody := bodyOf(vt.NewClient(t, server, "/review?key=sym").HTML())
	require.Contains(t, sessionBody, `href="/?key=sym"`,
		"the session review links back to the card, where per-order reviews are reachable")
}
