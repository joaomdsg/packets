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

// A missed order today reads only "missed" — the Lead can't tell a no-catch (the
// oracle ran, nothing to catch) from a lost-via-rename (the anchor moved) from a
// no-oracle-signal. R51 persists the oracle's per-order verdict; this surfaces it
// so the miss is DIAGNOSABLE on the surface, not just a bare outcome. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_showsThePerOrderVerdictSoAMissIsDiagnosable(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "whyc", "i")

	// One catch funds one order that ran, MISSED, and the oracle's honest verdict
	// (no-catch) was persisted for it.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.AppendWorkOrderVerdict(1, "no-catch"))
	registerSession("whyc", LiveConfig{BaseRev: "own-b-whyc", FixRev: "own-f", Anchor: anchorForCap()}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/?key=whyc").HTML())
	require.Contains(t, body, "no-catch", "the missed order shows the oracle's verdict — the WHY behind the miss")
	require.Contains(t, body, "board-row__dispatch-why", "the verdict carries its own calm hook so it reads as secondary detail")
}

// An order with no persisted verdict (queued, or pre-diagnostics data) renders no
// verdict element — never an empty "why" tag implying the oracle said something.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_omitsTheVerdictWhenNonePersisted(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "noverdictc", "i")

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own)) // queued, no verdict
	registerSession("noverdictc", LiveConfig{BaseRev: "own-b-noverdictc", FixRev: "own-f", Anchor: anchorForCap()}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	// bodyOf scopes past the head — the stylesheet carries .board-row__dispatch-why
	// as a selector; we assert the rendered ORDER has no verdict element.
	body := bodyOf(vt.NewClient(t, server, "/?key=noverdictc").HTML())
	require.Contains(t, body, "WO#1", "the order still renders")
	require.NotContains(t, body, "board-row__dispatch-why",
		"an order with no persisted verdict shows no why element")
}
