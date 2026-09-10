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

// awaitFrameContaining waits until the streamed SSE output has carried ALL of the
// given substrings. It ACCUMULATES across frames: vt.Client.SSE reads the body in
// fixed 4096-byte buffers and emits each read as a "frame", so a single rendered
// card larger than 4096 bytes is split across reads — a harness artifact, not an SSE
// event boundary. Requiring all substrings in one read would make the assertion
// fragile to card growth (it broke when the act-now runner control was added), so we
// assert the rendered stream contains them, which is the real property.
func awaitFrameContaining(t *testing.T, frames <-chan string, d time.Duration, must ...string) {
	t.Helper()
	deadline := time.After(d)
	var acc strings.Builder
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				t.Fatalf("frame stream closed before the stream carried all of %v", must)
			}
			acc.WriteString(f)
			all := true
			for _, m := range must {
				if !strings.Contains(acc.String(), m) {
					all = false
					break
				}
			}
			if all {
				return
			}
		case <-deadline:
			t.Fatalf("the stream did not carry all of %v within %s", must, d)
		}
	}
}

func TestLiveCard_spendFundsAWorkPacketWhoseRoundTripSurfacesOverSSE(t *testing.T) {
	// Internal test (package app): swaps resolveCycle so connect mints NOTHING,
	// isolating the consequence to the Spend verb. NOT parallel (shared globals).
	// The property: a spend BUYS something visible — the funded work-order
	// surfaces on the card and settles to done over SSE (the retired meter rows
	// no longer render, so the dispatch LIST is the visible round-trip).
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
	// The seeded balance renders the spend affordance (present only with a
	// catch to spend) — the pre-spend synchronization point.
	awaitFrameContaining(t, frames, 10*time.Second, "/_action/Spend")

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	// The spend funds an order; the order then RUNS (its fake target mints
	// nothing) to done, surfaced live by the dispatch poll on the order list.
	vt.AwaitFrame(t, frames, 10*time.Second, "PKT#1 other.go:9 done")

	pending, err := log.PendingSends()
	require.NoError(t, err)
	require.Equal(t, 1, pending, "the spend funded exactly one work-order in this session's ledger")

	// Balance is now 0. A further Spend is over-budget: AppendSend must refuse,
	// so it funds NO second work-order — the consequence honors the over-budget
	// guard exactly as the balance drain does.
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	tail := drainFramesFor(frames, 500*time.Millisecond)
	require.NotContains(t, tail, "PKT#2", "an over-budget spend must fund no second work-order")
	stillOne, err := log.PendingSends()
	require.NoError(t, err)
	require.Equal(t, 1, stillOne, "the refused dispatch left the work-order count unchanged")
}

func TestLiveCard_spendSendsOnlyIntoItsOwnSessionNotAnother(t *testing.T) {
	// Internal test (package app): two keyed sessions; a spend on A must fund a
	// work-order ONLY in A — B's dispatched tally never moves (isolated economies,
	// carried through the consequence, not just the balance). NOT parallel.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{Outcome: catch.Catch, ReasonTag: "catch"}}, nil
	}

	dir := t.TempDir()
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(dir, "default.jsonl"),
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	logA := ledger.Bind(liveFabric, "dspA", LedgerInstance)
	logB := ledger.Bind(liveFabric, "dspB", LedgerInstance)
	registerSession("dspA", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, SendBacklog: []ledger.Target{sendTarget()}}, logA)
	registerSession("dspB", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, SendBacklog: []ledger.Target{sendTarget()}}, logB)

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
		p, e := logA.PendingSends()
		return e == nil && p == 1
	}, 10*time.Second, 5*time.Millisecond, "the spend funded a work-order in session A")

	pB, err := logB.PendingSends()
	require.NoError(t, err)
	require.Equal(t, 0, pB, "session B funded NO work-order — a dispatch on A never touches B")
}
