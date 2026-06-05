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

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

func awaitFrameContaining(t *testing.T, frames <-chan string, d time.Duration, must ...string) {
	t.Helper()
	deadline := time.After(d)
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				t.Fatalf("frame stream closed before a frame carried all of %v", must)
			}
			all := true
			for _, m := range must {
				if !strings.Contains(f, m) {
					all = false
					break
				}
			}
			if all {
				return
			}
		case <-deadline:
			t.Fatalf("no single frame carried all of %v within %s", must, d)
		}
	}
}

func TestLiveCard_spendFundsAWorkOrderAndTheDispatchRowRisesAsTheBalanceDrains(t *testing.T) {
	// Internal test (package app): swaps resolveCycle so connect mints NOTHING,
	// isolating the consequence to the Spend verb. NOT parallel (shared globals).
	// The property: a spend BUYS something visible — in ONE render the balance row
	// drains to 0 AND the dispatch row rises to 1 (the funded work-order).
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
		DispatchTarget: woDispatchTarget(),
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	awaitFrameContaining(t, frames, 10*time.Second, `data-balance="1"`, `data-dispatch-queued="0"`)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	// The spend drains the balance to 0 and funds an order; the order then RUNS
	// (its fake target mints nothing) to done, surfaced live by the dispatch poll.
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="0"`)
	vt.AwaitFrame(t, frames, 10*time.Second, `data-dispatch-done="1"`)

	pending, err := log.PendingDispatches()
	require.NoError(t, err)
	require.Equal(t, 1, pending, "the spend funded exactly one work-order in this session's ledger")

	// Balance is now 0. A further Spend is over-budget: AppendDispatch must refuse,
	// so it funds NO second work-order — the consequence honors the over-budget
	// guard exactly as the balance drain does.
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	tail := drainFramesFor(frames, 500*time.Millisecond)
	require.NotContains(t, tail, `data-dispatch-done="2"`, "an over-budget spend must fund no second work-order")
	stillOne, err := log.PendingDispatches()
	require.NoError(t, err)
	require.Equal(t, 1, stillOne, "the refused dispatch left the work-order count unchanged")
}

func TestLiveCard_spendDispatchesOnlyIntoItsOwnSessionNotAnother(t *testing.T) {
	// Internal test (package app): two keyed sessions; a spend on A must fund a
	// work-order ONLY in A — B's dispatched tally never moves (isolated economies,
	// carried through the consequence, not just the balance). NOT parallel.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{Outcome: catch.Catch, ReasonTag: "catch"}}, nil
	}

	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.jsonl")
	bPath := filepath.Join(dir, "b.jsonl")
	logA, err := ledger.Open(aPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = logA.Close() })
	logB, err := ledger.Open(bPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = logB.Close() })

	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(dir, "default.jsonl"),
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	registerSession("dspA", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, LedgerPath: aPath, DispatchTarget: woDispatchTarget()}, logA)
	registerSession("dspB", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, LedgerPath: bPath, DispatchTarget: woDispatchTarget()}, logB)

	ca := vt.NewClient(t, server, "/?key=dspA")
	fa, cancelA := ca.SSE()
	defer cancelA()
	vt.AwaitFrame(t, fa, 10*time.Second, `data-state="catch"`)

	cb := vt.NewClient(t, server, "/?key=dspB")
	fb, cancelB := cb.SSE()
	defer cancelB()
	vt.AwaitFrame(t, fb, 10*time.Second, `data-state="catch"`)

	require.Equal(t, 200, ca.Action((&LiveCard{Key: "dspA"}).Spend).Fire())
	require.Eventually(t, func() bool {
		p, e := logA.PendingDispatches()
		return e == nil && p == 1
	}, 10*time.Second, 5*time.Millisecond, "the spend funded a work-order in session A")

	pB, err := logB.PendingDispatches()
	require.NoError(t, err)
	require.Equal(t, 0, pB, "session B funded NO work-order — a dispatch on A never touches B")
}
