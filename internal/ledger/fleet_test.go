package ledger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func isolatedFab(t *testing.T) *fabric.Fabric {
	t.Helper()
	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestFleetProjection_foldsEachSessionsEconomySeparately(t *testing.T) {
	t.Parallel()
	f := isolatedFab(t)
	alpha := ledger.Bind(f, "alpha", "i")
	beta := ledger.Bind(f, "beta", "i")

	require.NoError(t, alpha.Append(distinctRecord(0)))
	require.NoError(t, alpha.Append(distinctRecord(1))) // alpha: two distinct catches
	require.NoError(t, beta.Append(distinctRecord(0)))  // beta: one catch
	require.NoError(t, beta.AppendSpend(1, "fund"))      // beta: spent back to zero

	fleet, err := ledger.FleetProjection(context.Background(), f)
	require.NoError(t, err)

	require.Len(t, fleet, 2)
	require.Contains(t, fleet, "alpha")
	require.Contains(t, fleet, "beta") // exactly one projection per session, keyed by it
	assert.Equal(t, 2, fleet["alpha"].Balance(), "alpha's two mints are its own")
	assert.Equal(t, 0, fleet["beta"].Balance(), "beta minted one then spent it")
	assert.Len(t, fleet["beta"].Records(), 1, "beta's single catch is not mixed with alpha's")
}

// The cross-session board must carry each session's producer claim lifecycle —
// pending bets (in-flight) and verified-losses (rejected) — alongside its
// confirmed economy, computed exactly as the per-Log ClaimsInFlight/ClaimsRejected
// do (the two-scores invariant on the fleet surface: a bet never inflates the
// confirmed balance/stock, and a verified-loss is its own resolved count).
func TestFleetBoard_carriesEachSessionsInFlightAndRejected(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := isolatedFab(t)

	// alpha: three distinct bets, one of them verified-and-rejected.
	targets := make([]ledger.Target, 0, 3)
	for i := 1; i <= 3; i++ {
		tgt := ledger.Target{BaseRev: "b", FixRev: "fx", TipRev: "fx", Path: "a.go", Line: i}
		_, err := ledger.PublishClaim(ctx, f, "alpha", "i", ledger.ClaimRecord{Target: tgt})
		require.NoError(t, err)
		targets = append(targets, tgt)
	}
	_, err := ledger.PublishClaimVerdict(ctx, f, "alpha", "i", ledger.ClaimVerdict{Target: targets[0], Rejected: true})
	require.NoError(t, err)

	// beta: one confirmed catch, no claims.
	beta := ledger.Bind(f, "beta", "i")
	require.NoError(t, beta.Append(distinctRecord(0)))

	board, err := ledger.FleetBoard(ctx, f)
	require.NoError(t, err)

	require.Contains(t, board, "alpha")
	assert.Equal(t, 2, board["alpha"].InFlight, "two bets remain pending (one was rejected, leaving flight)")
	assert.Equal(t, 1, board["alpha"].Rejected, "the rejected bet is one verified-loss")
	assert.Equal(t, 0, board["alpha"].Balance(), "pending/rejected bets never inflate the confirmed balance (two-scores)")
	assert.Equal(t, 0, ledger.ConfirmedCatches(board["alpha"].Records()).Count, "alpha confirmed nothing")

	require.Contains(t, board, "beta")
	assert.Equal(t, 1, board["beta"].Balance(), "beta's confirmed catch")
	assert.Equal(t, 0, board["beta"].InFlight, "beta has no pending bets")
	assert.Equal(t, 0, board["beta"].Rejected, "beta has no losses")
}

// A session with BOTH confirmed catches AND a separate still-pending bet must
// carry both counts at once — the board seeds the row from the minted economy
// and then overlays the claim lifecycle on the SAME row. Confirmed balance and
// in-flight bets are independent axes (two-scores), never one folded into the other.
func TestFleetBoard_carriesConfirmedAndPendingTogetherOnOneRow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := isolatedFab(t)
	full := ledger.Bind(f, "full", "i")

	require.NoError(t, full.Append(distinctRecord(0)))
	require.NoError(t, full.Append(distinctRecord(1))) // two confirmed catches → balance 2
	_, err := ledger.PublishClaim(ctx, f, "full", "i", ledger.ClaimRecord{
		Target: ledger.Target{BaseRev: "b", FixRev: "fx", TipRev: "fx", Path: "z.go", Line: 9},
	}) // one separate pending bet
	require.NoError(t, err)

	board, err := ledger.FleetBoard(ctx, f)
	require.NoError(t, err)
	require.Contains(t, board, "full")
	assert.Equal(t, 2, board["full"].Balance(), "the two confirmed catches stand")
	assert.Equal(t, 1, board["full"].InFlight, "the pending bet is its own count, not folded into balance")
	assert.Equal(t, 0, board["full"].Rejected, "no losses")
}

// A session that has ONLY submitted claims (no minted events yet) must still
// appear on the board with its in-flight count — otherwise a producer's brand-new
// bets would be invisible until the host's first mint for that session.
func TestFleetBoard_includesAClaimOnlySessionWithNoMint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := isolatedFab(t)

	_, err := ledger.PublishClaim(ctx, f, "newcomer", "i", ledger.ClaimRecord{
		Target: ledger.Target{BaseRev: "b", FixRev: "fx", TipRev: "fx", Path: "a.go", Line: 1},
	})
	require.NoError(t, err)

	board, err := ledger.FleetBoard(ctx, f)
	require.NoError(t, err)
	require.Contains(t, board, "newcomer", "a session with only pending bets must still appear on the board")
	assert.Equal(t, 1, board["newcomer"].InFlight)
	assert.Equal(t, 0, board["newcomer"].Balance(), "it has minted nothing")
}

// A confirmed target is in NEITHER in-flight NOR rejected — it is a catch. The
// same identity, claimed then minted, must move cleanly into the confirmed
// economy and out of every bet bucket (no double-booking on the fleet surface).
func TestFleetBoard_aConfirmedTargetIsNeitherInFlightNorRejected(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := isolatedFab(t)
	s := ledger.Bind(f, "s", "i")

	_, err := ledger.PublishClaim(ctx, f, "s", "i", claimAt(4))
	require.NoError(t, err)
	rec, err := confirmFromClaim(claimAt(4))
	require.NoError(t, err)
	require.NoError(t, s.Append(*rec))

	board, err := ledger.FleetBoard(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, 0, board["s"].InFlight, "a minted target is no longer a pending bet")
	assert.Equal(t, 0, board["s"].Rejected, "a confirmed catch is never a loss")
	assert.Equal(t, 1, board["s"].Balance(), "it is the confirmed catch")
}

func TestFleetProjection_isEmptyWhenNoSessionHasMinted(t *testing.T) {
	t.Parallel()
	f := isolatedFab(t)
	fleet, err := ledger.FleetProjection(context.Background(), f)
	require.NoError(t, err)
	assert.Empty(t, fleet)
}

// FleetProjection wildcards the instance/kind tokens (`*.*.minted.>`) while
// ReplayProjection pins them (`session.instance.minted.*`); both must fold the
// SAME state for a single session, or the cross-process board would silently
// disagree with the per-session read path.
func TestFleetProjection_foldsTheSameStateAsReplayProjection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := isolatedFab(t)

	tgt := func(line int) ledger.Target {
		return ledger.Target{BaseRev: "base", FixRev: "fix", TipRev: "fix", Path: "a.go", Line: line}
	}
	fixture := []evt{
		{kind: "catch", catch: ledger.CatchRecord{Outcome: catch.Catch, Path: "a.go", Line: 4, BeforeRev: "base", AfterRev: "fix", ReasonTag: "boundary", Producer: "connect"}},
		{kind: "spend", spend: ledger.SpendRecord{Kind: "spend", Amount: 1, Reason: "d1"}},
		{kind: "workorder", order: ledger.WorkOrderRecord{Kind: "workorder", ID: 1, Producer: "in-process", Status: "queued", Reason: "d1", Target: tgt(5)}},
		{kind: "wostatus", stat: ledger.StatusRecord{Kind: "wostatus", ID: 1, Status: "running"}},
		{kind: "workorder", order: ledger.WorkOrderRecord{Kind: "workorder", ID: 2, Producer: "in-process", Status: "queued", Reason: "d2", Target: tgt(6)}},
		{kind: "wostatus", stat: ledger.StatusRecord{Kind: "wostatus", ID: 1, Status: "done"}},
	}
	for _, e := range fixture {
		e.publish(t, ctx, f, "solo", "i1")
	}

	fleet, err := ledger.FleetProjection(ctx, f)
	require.NoError(t, err)
	require.Contains(t, fleet, "solo")
	fromFleet := fleet["solo"]

	fromReplay, err := ledger.ReplayProjection(ctx, f, "solo", "i1")
	require.NoError(t, err)

	assert.Equal(t, fromReplay.Balance(), fromFleet.Balance())
	assert.Equal(t, fromReplay.Records(), fromFleet.Records())
	assert.Equal(t, fromReplay.WorkOrders(), fromFleet.WorkOrders())
	assert.Equal(t, fromReplay.DispatchStatusCounts(), fromFleet.DispatchStatusCounts())
	assert.Equal(t, fromReplay.QueuedWorkOrders(), fromFleet.QueuedWorkOrders())
}
