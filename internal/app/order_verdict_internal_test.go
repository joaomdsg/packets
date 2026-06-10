package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// When the runner executes a funded order, the oracle computes an honest verdict —
// today that verdict is DISCARDED for a miss (only a catch is recorded), so the
// Lead can never learn WHY an order missed. The runner must persist the oracle's
// per-order verdict so it survives on the order's dispatch view. NOT parallel
// (shared globals: resolveCycle, liveReg).
func TestLiveCard_persistsTheOracleVerdictForEachRunOrder(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	// Every cycle (connect + each dispatched order) resolves to an honest no-catch
	// verdict and mints NOTHING (Record nil) — so the only thing under test is that
	// the runner persists the verdict for the order it ran.
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Verdict: "no-catch"}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	// One confirmed catch → balance 1, enough to fund one dispatched order.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire()) // funds + runs one order in the background

	// The order runs to done and the runner persists the oracle's verdict for it.
	require.Eventually(t, func() bool {
		views, e := log.RecentDispatches(1)
		return e == nil && len(views) == 1 && views[0].Status == "done" && views[0].Verdict == "no-catch"
	}, 10*time.Second, 10*time.Millisecond,
		"the runner persists the oracle's honest per-order verdict, so a miss carries its WHY")
}
