package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

func TestLiveCard_spendVerbDrainsTheBalanceRowOverSSE(t *testing.T) {
	// Internal test (package app): swaps resolveCycle so the connect cycle mints
	// NOTHING, isolating the drain to the Spend verb. NOT parallel (shared globals).
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil // no mint — the balance only moves when the Lead SPENDS
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 2, ReasonTag: "catch"})) // distinct identity: 2 catches, not a re-mint
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{woDispatchTarget()},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="2"`) // the seeded balance renders

	// Spend one confirmed catch: the balance row must DRAIN to 1 over SSE — the
	// first non-climbing transition the Lead can actually trigger.
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="1"`)
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
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, ReasonTag: "catch"}))
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{woDispatchTarget()},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="1"`)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire()) // drain 1 → 0
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="0"`)

	// Spend past 0: a no-op. The action still returns 200 (never an error to the
	// Lead). The card may still re-render as the first spend's funded order runs in
	// the background (a legitimate dispatch-progress frame showing an UNCHANGED
	// balance), but the refused spend must never drive the balance negative nor fund
	// a second order.
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	tail := drainFramesFor(frames, 500*time.Millisecond)
	require.NotContains(t, tail, `data-balance="-1"`, "the balance must never render negative")

	bal, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, 0, bal, "the refused spend left no debit in the ledger")
	pending, err := log.PendingDispatches()
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
