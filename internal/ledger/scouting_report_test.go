package ledger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The outward scouting report reads "this lane ships clean — N/M first-pass": of
// the orders that completed, how many caught on the first pass. It is the honest,
// counts-only foundation of the Trust Ledger — redeemed against logged facts
// (order status + the wo:<id> catch provenance), never a model guess.
func TestScoutingReport_reportsFirstPassCatchRateOverCompletedOrders(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(distinctRecord(i))) // balance 3 → funds 3 dispatches
	}
	require.NoError(t, l.AppendDispatch("d1", target(1), ownTarget()))
	require.NoError(t, l.AppendDispatch("d2", target(2), ownTarget()))
	require.NoError(t, l.AppendDispatch("d3", target(3), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "done"))
	require.NoError(t, l.AppendStatus(2, "done"))
	require.NoError(t, l.AppendStatus(3, "done"))
	require.NoError(t, l.Append(woCatch(10, 1))) // order 1 caught
	require.NoError(t, l.Append(woCatch(11, 2))) // order 2 caught; order 3 done-but-missed

	got, err := l.ScoutingReport()
	require.NoError(t, err)
	assert.Equal(t, 3, got.Completed, "all three orders ran to done")
	assert.Equal(t, 2, got.Caught, "two of the three minted a wo:<id> catch")
	assert.InDelta(t, 2.0/3.0, got.FirstPassRate(), 0.001, "first-pass rate is caught/completed")
}

// Only completed (done) orders count toward the rate — a still-queued or running
// order hasn't had its first pass yet, so it must not dilute the denominator.
func TestScoutingReport_countsOnlyCompletedOrders(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.Append(distinctRecord(1)))
	require.NoError(t, l.AppendDispatch("d1", target(1), ownTarget()))
	require.NoError(t, l.AppendDispatch("d2", target(2), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "done"))
	require.NoError(t, l.Append(woCatch(10, 1)))
	require.NoError(t, l.AppendStatus(2, "running")) // still in flight — not a completed pass

	got, err := l.ScoutingReport()
	require.NoError(t, err)
	assert.Equal(t, 1, got.Completed, "only the done order is a completed pass")
	assert.Equal(t, 1, got.Caught)
}

// A FAILED order is an infra failure (the harness crashed), not a missed catch —
// it must not count against the lane's first-pass rate.
func TestScoutingReport_excludesFailedOrdersFromTheRate(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.Append(distinctRecord(1)))
	require.NoError(t, l.AppendDispatch("d1", target(1), ownTarget()))
	require.NoError(t, l.AppendDispatch("d2", target(2), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "done"))
	require.NoError(t, l.Append(woCatch(10, 1)))
	require.NoError(t, l.AppendStatus(2, "failed")) // harness crash — not a missed catch

	got, err := l.ScoutingReport()
	require.NoError(t, err)
	assert.Equal(t, 1, got.Completed, "a failed order is not a completed pass")
	assert.Equal(t, 1, got.Caught)
}

// Caught is gated on completion: a wo:<id> catch on an order still running must
// not count (you can't have first-pass-caught a pass that hasn't completed), so
// Caught can never exceed Completed.
func TestScoutingReport_doesNotCountACatchOnAnUncompletedOrder(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendDispatch("d1", target(1), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "running"))
	require.NoError(t, l.Append(woCatch(10, 1))) // a catch, but the order isn't done

	got, err := l.ScoutingReport()
	require.NoError(t, err)
	assert.Equal(t, 0, got.Completed, "the order hasn't completed")
	assert.Equal(t, 0, got.Caught, "a catch on an uncompleted order does not count — Caught never exceeds Completed")
}

// A fresh session has no completed orders: the report is empty and the rate is 0,
// which the render must read as NO SIGNAL (not a 0% lane), gating on Completed>0.
func TestScoutingReport_isEmptyForASessionWithNoCompletedOrders(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	got, err := l.ScoutingReport()
	require.NoError(t, err)
	assert.Equal(t, 0, got.Completed, "no orders → nothing completed")
	assert.Equal(t, 0, got.Caught)
	assert.Equal(t, 0.0, got.FirstPassRate(), "no signal yet — the render gates on Completed>0")
}
