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
