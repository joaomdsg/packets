package bridge_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/bridge"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestFleetHandler_ordersRowsByQueuedDescThenKeyAsc(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	alpha := ledger.Bind(f, "alpha", "i")
	beta := ledger.Bind(f, "beta", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	require.NoError(t, alpha.Append(sampleCatch())) // alpha: balance 1, queued 0
	require.NoError(t, beta.Append(sampleCatch()))  // beta: balance 1
	require.NoError(t, beta.AppendDispatch("d",
		ledger.Target{BaseRev: "b2", FixRev: "f2", TipRev: "f2", Path: "other.go", Line: 9},
		ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "adult.go", Line: 4},
	)) // beta: spends 1 → balance 0, one queued work-order

	// beta (queued 1) sorts before alpha (queued 0); the exact frame pins the
	// `data: ` framing, the row shape, the per-session values, and the ordering.
	awaitLine(t, lines, `data: [{"key":"beta","balance":0,"confirmed":1,"reinvested":0,"queued":1,"running":0,"done":0,"caught":0,"misses":0,"in_flight":0,"rejected":0},{"key":"alpha","balance":1,"confirmed":1,"reinvested":0,"queued":0,"running":0,"done":0,"caught":0,"misses":0,"in_flight":0,"rejected":0}]`)
}

func TestFleetHandler_breaksQueuedTiesBySessionKeyAscending(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	zeta := ledger.Bind(f, "zeta", "i")
	alpha := ledger.Bind(f, "alpha", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	// Equal queued (both 0): the order must fall back to key ascending, since the
	// in-process registration ordinal is not on the stream.
	require.NoError(t, zeta.Append(sampleCatch()))
	require.NoError(t, alpha.Append(sampleCatch()))

	awaitLine(t, lines, `data: [{"key":"alpha","balance":1,"confirmed":1,"reinvested":0,"queued":0,"running":0,"done":0,"caught":0,"misses":0,"in_flight":0,"rejected":0},{"key":"zeta","balance":1,"confirmed":1,"reinvested":0,"queued":0,"running":0,"done":0,"caught":0,"misses":0,"in_flight":0,"rejected":0}]`)
}

func TestFleetHandler_rowsCarryReinvestedDoneAndMisses(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	solo := ledger.Bind(f, "solo", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	own := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "adult.go", Line: 4}
	require.NoError(t, solo.Append(sampleCatch())) // connect catch: confirmed 1, balance 1
	require.NoError(t, solo.AppendDispatch("d1",
		ledger.Target{BaseRev: "b2", FixRev: "f2", TipRev: "f2", Path: "other.go", Line: 9}, own)) // spend → balance 0, order 1 queued
	require.NoError(t, solo.AppendStatus(1, "running"))
	require.NoError(t, solo.AppendStatus(1, "done")) // done 1

	wo := sampleCatch()
	wo.Line = 5
	wo.Producer = "wo:1"
	require.NoError(t, solo.Append(wo)) // dispatch-minted catch: confirmed 2, reinvested 1, balance 1

	require.NoError(t, solo.AppendDispatch("d2",
		ledger.Target{BaseRev: "b3", FixRev: "f3", TipRev: "f3", Path: "other.go", Line: 10}, own)) // spend → balance 0, order 2 queued
	require.NoError(t, solo.AppendStatus(2, "running"))
	require.NoError(t, solo.AppendStatus(2, "done")) // done 2; misses = done(2) − reinvested(1) = 1

	awaitLine(t, lines, `data: [{"key":"solo","balance":0,"confirmed":2,"reinvested":1,"queued":0,"running":0,"done":2,"caught":1,"misses":1,"in_flight":0,"rejected":0}]`)
}

// The fleet frame's first-pass count must credit a CAUGHT order only when THAT
// order is done — a wo:<id> catch minted on a still-running order must never mark a
// different done-but-missed order as caught. (The old done−reinvested heuristic
// would: it counts the running order's catch and hides the real miss.) This is the
// /fleet-stream twin of the /board fix; both read the exact ledger.ScoutingReport.
func TestFleetHandler_doesNotCreditADoneOrderForACatchOnARunningOrder(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	solo := ledger.Bind(f, "solo", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	own := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "adult.go", Line: 4}
	require.NoError(t, solo.Append(sampleCatch())) // balance 1 → funds order 1
	require.NoError(t, solo.AppendDispatch("d1",
		ledger.Target{BaseRev: "b2", FixRev: "f2", TipRev: "f2", Path: "other.go", Line: 9}, own))
	require.NoError(t, solo.AppendStatus(1, "running"))
	require.NoError(t, solo.AppendStatus(1, "done")) // order 1 done, minted NOTHING

	credit := sampleCatch()
	credit.Line = 50
	require.NoError(t, solo.Append(credit)) // balance 1 → funds order 2
	require.NoError(t, solo.AppendDispatch("d2",
		ledger.Target{BaseRev: "b3", FixRev: "f3", TipRev: "f3", Path: "other.go", Line: 10}, own))
	require.NoError(t, solo.AppendStatus(2, "running")) // order 2 still RUNNING

	wo := sampleCatch()
	wo.Line = 60
	wo.Producer = "wo:2"
	require.NoError(t, solo.Append(wo)) // a catch on the RUNNING order 2 — must not credit order 1

	// done 1 (order 1), caught 0 (order 1 missed; order 2's catch is on a running
	// order), misses 1 — NOT misses 0 as the old done−reinvested heuristic gave.
	awaitLine(t, lines, `"running":1,"done":1,"caught":0,"misses":1`)
}

func TestFleetHandler_rowReportsAnInFlightRunningOrder(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	solo := ledger.Bind(f, "solo", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	own := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "adult.go", Line: 4}
	require.NoError(t, solo.Append(sampleCatch())) // balance 1
	require.NoError(t, solo.AppendDispatch("d",
		ledger.Target{BaseRev: "b2", FixRev: "f2", TipRev: "f2", Path: "other.go", Line: 9}, own)) // order 1 queued
	require.NoError(t, solo.AppendStatus(1, "running")) // moves off queued → running, not done

	awaitLine(t, lines, `data: [{"key":"solo","balance":0,"confirmed":1,"reinvested":0,"queued":0,"running":1,"done":0,"caught":0,"misses":0,"in_flight":0,"rejected":0}]`)
}

// The fleet stream must react to the producer claim LIFECYCLE, not only mints:
// a submitted claim drives a frame showing the bet in flight, and the host's
// rejection verdict drives a frame moving it to verified-lost — live, off the
// same stream, with no mint involved. Before C3b2b the feed only woke on minted
// events, so a producer's bets were invisible until an unrelated mint fired.
func TestFleetHandler_streamsTheClaimLifecycleNotOnlyMints(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	tgt := ledger.Target{BaseRev: "b", FixRev: "fx", TipRev: "fx", Path: "a.go", Line: 4}
	// A submitted claim — no mint — must surface as one bet in flight.
	_, err := ledger.PublishClaim(ctx, f, "prod", "i", ledger.ClaimRecord{Target: tgt})
	require.NoError(t, err)
	awaitLine(t, lines, `"key":"prod","balance":0,"confirmed":0,"reinvested":0,"queued":0,"running":0,"done":0,"caught":0,"misses":0,"in_flight":1,"rejected":0`)

	// The host's rejection verdict — still no mint — must move it to verified-lost.
	_, err = ledger.PublishClaimVerdict(ctx, f, "prod", "i", ledger.ClaimVerdict{Target: tgt, Rejected: true})
	require.NoError(t, err)
	awaitLine(t, lines, `"in_flight":0,"rejected":1`)
}

func TestFleetHandler_sendsEventStreamContentTypeAndActuallyStreams(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	alpha := ledger.Bind(f, "alpha", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

	// A stub that only sets headers but never streams must not pass.
	lines := scanLines(resp.Body)
	require.NoError(t, alpha.Append(sampleCatch()))
	awaitLine(t, lines, `"key":"alpha"`)
}
