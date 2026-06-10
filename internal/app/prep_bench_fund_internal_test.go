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

// The prep bench's payoff: the Lead CHOOSES which fundable target the next Spend
// funds, instead of a blind auto-FIFO pick. Funding a chosen bench item must
// dispatch THAT target — even when it is not the FIFO head — turning dispatch into
// a real management-sim decision. NOT parallel (shared globals).
func TestLiveCard_fundChosenDispatchesThePickedBenchTarget(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil // no mint — isolate the dispatch to the Lead's choice
	}

	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{
			{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, // FIFO head
			{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9},  // the chosen one
		},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"})) // balance to fund

	tc := vt.NewClient(t, server, "/")
	// Fund the SECOND bench target (not the FIFO head) by its path:line key.
	require.Equal(t, 200, tc.Action((&LiveCard{}).FundChosen).WithSignal("fundtarget", "beta.go:9").Fire())

	require.Eventually(t, func() bool {
		ds, e := log.RecentDispatches(10)
		if e != nil {
			return false
		}
		for _, d := range ds {
			if d.Target.Path == "beta.go" && d.Target.Line == 9 {
				return true
			}
		}
		return false
	}, 10*time.Second, 10*time.Millisecond, "funding a chosen bench item dispatches THAT target, not the FIFO head")

	// The FIFO head was NOT funded — only the chosen target.
	ds, err := log.RecentDispatches(10)
	require.NoError(t, err)
	for _, d := range ds {
		require.False(t, d.Target.Path == "alpha.go", "the un-chosen FIFO head was not dispatched")
	}
}

// Funding a target that is NOT in the fundable set (unknown, consumed, or the
// card's own cycle) is a no-op — the choice is constrained to real fundable work,
// so the Lead can't fund arbitrary or already-handled targets. NOT parallel.
func TestLiveCard_fundChosenRefusesATargetNotOnTheBench(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil
	}

	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&LiveCard{}).FundChosen).WithSignal("fundtarget", "nowhere.go:99").Fire())

	// Nothing was dispatched — the off-bench target was refused.
	require.Never(t, func() bool {
		ds, e := log.RecentDispatches(10)
		return e == nil && len(ds) > 0
	}, 1*time.Second, 50*time.Millisecond, "an off-bench target funds nothing")
}
