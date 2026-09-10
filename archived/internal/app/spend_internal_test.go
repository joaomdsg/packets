package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

func TestLiveCard_spendVerbDrainsTheBalanceAndTheFundedPacketSurfacesOverSSE(t *testing.T) {
	// Internal test (package app): swaps resolveCycle so the connect cycle mints
	// NOTHING, isolating the drain to the Spend verb. NOT parallel (shared globals).
	// The balance row is retired from the UI, so the drain is
	// asserted on the ledger and the SSE-visible consequence is the funded order.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil // no mint — the balance only moves when the Lead SPENDS
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		SendBacklog: []ledger.Target{sendTarget()},
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 2, ReasonTag: "catch"})) // distinct identity: 2 catches, not a re-mint

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, "/_action/Spend") // the seeded balance renders the spend affordance

	// Spend one confirmed catch: the funded work-order must surface over SSE —
	// the first non-climbing transition the Lead can actually trigger — and the
	// ledger's balance drains by exactly one.
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	vt.AwaitFrame(t, frames, 10*time.Second, "PKT#1")
	bal, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, 1, bal, "the spend debited exactly one catch")
}

func TestLiveCard_overBudgetSpendIsASilentNoOpNotASpuriousFrame(t *testing.T) {
	// Internal test (package app): swaps resolveCycle to mint NOTHING so the only
	// balance movement is the Lead's Spend. NOT parallel (shared globals). Drains
	// the lone catch to 0, then spends PAST 0: the ledger refuses it and Spend
	// must return without writing the Balance cell, so no spurious re-render frame
	// reaches the live stream and the row never shows a negative balance.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		SendBacklog: []ledger.Target{sendTarget()},
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, ReasonTag: "catch"}))

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, "/_action/Spend") // the seeded catch renders the spend affordance

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire()) // drain 1 → 0
	vt.AwaitFrame(t, frames, 10*time.Second, "PKT#1")            // the funded order surfaces — the drain landed

	// Spend past 0: a no-op. The action still returns 200 (never an error to the
	// Lead). The card may still re-render as the first spend's funded order runs in
	// the background (a legitimate dispatch-progress frame), but the refused spend
	// must never fund a second order.
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	tail := drainFramesFor(frames, 500*time.Millisecond)
	require.NotContains(t, tail, "PKT#2", "the refused spend must surface no second work-order")

	bal, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, 0, bal, "the refused spend left no debit in the ledger")
	pending, err := log.PendingSends()
	require.NoError(t, err)
	require.Equal(t, 1, pending, "the over-budget spend funded NO second order — only the first spend's order exists")
}

// drainFramesFor collects every SSE frame that arrives within d, then returns
// the concatenation. Used to assert the ABSENCE of an expected-not-to-happen
// frame (a positive AwaitFrame can't prove a non-event).
func drainFramesFor(frames <-chan string, d time.Duration) string {
	deadline := time.After(d)
	var b strings.Builder
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				return b.String()
			}
			b.WriteString(f)
		case <-deadline:
			return b.String()
		}
	}
}
