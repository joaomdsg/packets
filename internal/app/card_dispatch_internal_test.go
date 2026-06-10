package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// After a Spend funds a work-order, the Lead on the session card needs to watch
// THAT order resolve caught-or-missed — the payoff of the spend. Today the card
// shows only aggregate dispatch counts (queued/running/done); the per-order
// round-trip lives only on the fleet board, forcing a context-switch off the card
// the Lead is acting on. The live card must surface this session's recent
// work-orders with their caught/missed outcome, closing spend → dispatch → watch
// it resolve on one surface. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_showsThisSessionsDispatchRoundTripOutcomes(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "cardc", "i")

	// Fund two work-orders (two catches → balance 2), run both: WO#1 mints (caught),
	// WO#2 does not (missed) — the same round-trip the board already surfaces.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendDispatch("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.AppendStatus(2, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Producer: "wo:1"}))
	registerSession("cardc", LiveConfig{BaseRev: "own-b-cardc", FixRev: "own-f", Anchor: anchorForCap()}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := vt.NewClient(t, server, "/?key=cardc").HTML()
	require.Contains(t, body, "WO#1", "the card shows the caught order by id")
	require.Contains(t, body, "alpha.go:7", "with its target line")
	require.Contains(t, body, "caught", "WO#1 minted → caught, shown on the card")
	require.Contains(t, body, "WO#2", "the card shows the missed order too")
	require.Contains(t, body, "missed", "WO#2 ran but minted nothing → missed, shown on the card")
	// Each resolved order carries a per-outcome hook so the calm palette can color
	// caught vs missed — the round-trip outcome legible at a glance, not
	// undifferentiated dim text (extends R45's per-state honesty to dispatches).
	require.Contains(t, body, `data-outcome="caught"`, "the caught order carries a per-outcome hook")
	require.Contains(t, body, `data-outcome="missed"`, "the missed order carries a per-outcome hook")
	// The stylesheet colors both outcomes in the honest palette (selectors live in
	// the head), never an alarm red/green.
	require.Contains(t, body, `[data-outcome="caught"]`, "the stylesheet colors a caught order")
	require.Contains(t, body, `[data-outcome="missed"]`, "the stylesheet colors a missed order")
}

// A session that has funded no work-orders must NOT render an empty dispatch
// cluster — an empty round-trip block is visual noise that implies activity where
// there is none. The cluster is omitted entirely until there is an order to show.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_omitsTheDispatchClusterWhenNoOrdersFunded(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// bodyOf scopes past the <head>: the stylesheet contains "board-row__dispatches"
	// as a CSS selector, so a whole-page check would always match.
	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.NotContains(t, body, "board-row__dispatches",
		"a session with no funded orders renders no dispatch cluster, not an empty block")
}

// A queued/running order has NOT resolved, so it must carry NO outcome hook — it
// stays neutral, never colored caught or missed before it has an outcome. Without
// this guard a bug could paint unresolved work as a confirmed catch or a loss.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_aQueuedOrderCarriesNoOutcomeHook(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "queuedc", "i")

	// One catch funds one order, left QUEUED (no done status, no outcome yet).
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	registerSession("queuedc", LiveConfig{BaseRev: "own-b-queuedc", FixRev: "own-f", Anchor: anchorForCap()}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	// bodyOf scopes past the head — the stylesheet's [data-outcome=...] selectors
	// live there; we assert the rendered queued ORDER carries no outcome attribute.
	body := bodyOf(vt.NewClient(t, server, "/?key=queuedc").HTML())
	require.Contains(t, body, "WO#1", "the queued order is shown")
	require.Contains(t, body, "queued", "with its unresolved status")
	require.NotContains(t, body, "data-outcome=",
		"a queued order carries no outcome hook — it is never colored before it resolves")
}
