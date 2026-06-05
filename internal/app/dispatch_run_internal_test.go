package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

func woDispatchTarget() ledger.Target {
	return ledger.Target{BaseRev: "wo-base", FixRev: "wo-fix", TipRev: "wo-fix", Path: "other.go", Line: 9}
}

func TestLiveCard_spendDispatchesAnOrderThatRunsAndMintsBackADistinctCatch(t *testing.T) {
	// Internal test (package app), NON-parallel (shared globals). The spend-to-earn
	// loop: spend a catch → the order RUNS distinct work → it mints a NEW distinct
	// catch back, so the balance nets -1 then +1 and the economy compounds. The
	// dispatched mint carries Producer="wo:1" (reinvestment provenance).
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	tgt := woDispatchTarget()
	resolveCycle = func(_ context.Context, _, base, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		if base == tgt.BaseRev { // only the dispatched run (on the order's target) mints
			return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{
				Outcome: catch.Catch, Path: tgt.Path, Line: tgt.Line, BeforeRev: tgt.BaseRev, AfterRev: tgt.FixRev, ReasonTag: "catch",
			}}, nil
		}
		return Resolution{}, nil // the connect-cycle mints nothing
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"})) // balance 1 to spend
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, DispatchBacklog: []ledger.Target{tgt},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="1"`)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())

	require.Eventually(t, func() bool {
		c, e := log.DispatchStatusCounts()
		return e == nil && c.Done == 1
	}, 10*time.Second, 10*time.Millisecond, "the dispatched order ran to done")

	bal, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, 1, bal, "spent 1, the run minted 1 distinct catch back — the loop compounds (net even, not a one-way drain)")

	recs, err := log.Records()
	require.NoError(t, err)
	require.Len(t, recs, 2, "the seed catch + the dispatched run's NEW distinct catch")
	var dispatched *ledger.CatchRecord
	for i := range recs {
		if recs[i].Producer == "wo:1" {
			dispatched = &recs[i]
		}
	}
	require.NotNil(t, dispatched, "the dispatched mint carries Producer wo:1 — reinvestment provenance, byte-distinguishable from a connect mint")
}

func TestLiveCard_dispatchingOwnAlreadyCaughtWorkIsAnHonestLossNotAFarm(t *testing.T) {
	// The anti-farm property at the app layer: if the dispatched run reproduces an
	// identity ALREADY in the stock, the identity-dedup gate mints nothing — spend
	// 1, get 0, an honest loss. The economy never inflates from re-running work.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	tgt := woDispatchTarget()
	seededIdentity := ledger.CatchRecord{Outcome: catch.Catch, Path: tgt.Path, Line: tgt.Line, BeforeRev: tgt.BaseRev, AfterRev: tgt.FixRev, ReasonTag: "catch"}
	resolveCycle = func(_ context.Context, _, base, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		if base == tgt.BaseRev {
			r := seededIdentity // the run reproduces an identity already in the stock
			return Resolution{Verdict: string(catch.Catch), Record: &r}, nil
		}
		return Resolution{}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"})) // a credit to spend
	require.NoError(t, seed.Append(seededIdentity))                                                          // the identity the run will re-produce
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, DispatchBacklog: []ledger.Target{tgt},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="2"`)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	require.Eventually(t, func() bool {
		c, e := log.DispatchStatusCounts()
		return e == nil && c.Done == 1
	}, 10*time.Second, 10*time.Millisecond, "the order ran to done even though it minted nothing")

	bal, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, 1, bal, "spent 1, minted 0 (the re-run reproduced a seen identity) — an honest loss, never infinite money")
	recs, err := log.Records()
	require.NoError(t, err)
	require.Len(t, recs, 2, "the dispatched run added NO new catch — the dedup gate held")
}

func TestLiveCard_connectAndDispatchMintsCarryDistinctProducerProvenance(t *testing.T) {
	// Two real producers on one log: the connect-cycle stamps Producer="connect",
	// the dispatched run stamps "wo:<id>". Both mint distinct identities; the field
	// demuxes them on replay (the in-process discharge of the P0 two-producer seq).
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	tgt := woDispatchTarget()
	resolveCycle = func(_ context.Context, _, base, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		if base == tgt.BaseRev {
			return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{
				Outcome: catch.Catch, Path: tgt.Path, Line: tgt.Line, BeforeRev: tgt.BaseRev, AfterRev: tgt.FixRev, ReasonTag: "catch",
			}}, nil
		}
		return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{ // the connect-cycle mints its own distinct catch
			Outcome: catch.Catch, Path: "adult.go", Line: 4, BeforeRev: "b", AfterRev: "f", ReasonTag: "catch",
		}}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"})) // a balance to spend
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, DispatchBacklog: []ledger.Target{tgt},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-state="catch"`) // the connect mint lands

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	require.Eventually(t, func() bool {
		c, e := log.DispatchStatusCounts()
		return e == nil && c.Done == 1
	}, 10*time.Second, 10*time.Millisecond, "the dispatched order ran")

	recs, err := log.Records()
	require.NoError(t, err)
	producers := map[string]int{}
	for _, r := range recs {
		producers[r.Producer]++
	}
	require.Equal(t, 1, producers["connect"], "the connect-cycle mint is tagged connect")
	require.Equal(t, 1, producers["wo:1"], "the dispatched run's mint is tagged wo:1 — the two producers are demuxed on the one log")
}

func TestLiveCard_dispatchedRunDoesNotLeakItsBeatsDiscardGoroutine(t *testing.T) {
	// Each dispatched run spawns a goroutine to discard the cycle's off-ledger beats.
	// resolveCycle (ResolveStreaming) only SENDS on the beats channel, never closes it
	// — the caller owns the close. If the runner forgets to close beats after the
	// cycle returns, that discard goroutine parks forever on the open channel: a leak
	// of one goroutine per dispatched order. Dispatch a batch, drain it to completion,
	// and assert the goroutine count settles back — a permanent rise is the leak.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	tgt := woDispatchTarget()
	resolveCycle = func(_ context.Context, _, base, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, beats chan<- pipe.TraceEvent) (Resolution, error) {
		if beats != nil {
			beats <- pipe.TraceEvent{Kind: "catch"} // exercise the discard goroutine's range
		}
		if base == tgt.BaseRev {
			return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{
				Outcome: catch.Catch, Path: tgt.Path, Line: tgt.Line, BeforeRev: tgt.BaseRev, AfterRev: tgt.FixRev, ReasonTag: "catch",
			}}, nil
		}
		return Resolution{}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	const orders = 8
	for i := 0; i < orders; i++ {
		require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: i + 1, ReasonTag: "catch"})) // balance to fund the orders
	}
	for i := 0; i < orders; i++ {
		require.NoError(t, seed.AppendDispatch("dispatch", tgt, ledger.Target{})) // queue the work the runner will drain
	}
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, DispatchBacklog: []ledger.Target{tgt},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	runtime.GC()
	baseline := runtime.NumGoroutine()
	drainQueuedOrders(defaultSessionKey) // runs all queued orders to completion, synchronously
	require.Eventually(t, func() bool {
		c, e := log.DispatchStatusCounts()
		return e == nil && c.Done == orders
	}, 10*time.Second, 10*time.Millisecond, "every dispatched order ran to done")

	require.Eventually(t, func() bool {
		runtime.GC()
		return runtime.NumGoroutine() <= baseline+1
	}, 5*time.Second, 50*time.Millisecond, "the per-order beats-discard goroutines must exit — a permanent rise of ~%d is the leak", orders)
}

func TestLiveCard_dispatchedOrderProgressIsWatchableQueuedRunningDoneOverSSE(t *testing.T) {
	// The dispatched run happens in a BACKGROUND goroutine (no request ctx), yet the
	// live view must still SHOW its progress. The OnConnect Stream poll surfaces the
	// per-status tally, so the Lead watches queued→running→done over SSE. A blocking
	// fake holds the run in "running" long enough to observe it deterministically.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	tgt := woDispatchTarget()
	release := make(chan struct{})
	resolveCycle = func(_ context.Context, _, base, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		if base == tgt.BaseRev {
			<-release // hold the order in "running" so the SSE poll can observe it
			return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{
				Outcome: catch.Catch, Path: tgt.Path, Line: tgt.Line, BeforeRev: tgt.BaseRev, AfterRev: tgt.FixRev, ReasonTag: "catch",
			}}, nil
		}
		return Resolution{}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, DispatchBacklog: []ledger.Target{tgt},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="1"`)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	vt.AwaitFrame(t, frames, 10*time.Second, `data-dispatch-running="1"`) // work in flight surfaces live
	close(release)
	vt.AwaitFrame(t, frames, 10*time.Second, `data-dispatch-done="1"`) // the payoff surfaces live
}
