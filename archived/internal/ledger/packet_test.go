package ledger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

func ownTarget() ledger.Target {
	return ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "adult.go", Line: 4}
}

func distinctTarget() ledger.Target {
	return ledger.Target{BaseRev: "b2", FixRev: "f2", TipRev: "f2", Path: "other.go", Line: 9}
}

func TestAppendSend_refusesATargetEqualToTheCardsOwnWorkSoItCannotFundAGuaranteedLoss(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0))) // balance 1

	// Funding a dispatch whose target IS the card's own already-caught cycle would
	// run work that can only reproduce an identity already in the stock — the
	// identity-dedup gate would mint nothing, so the spend is a guaranteed loss.
	// Refuse it up front (the distinct-target requirement), writing nothing.
	own := ownTarget()
	require.Error(t, l.AppendSend("dispatch", own, own), "a dispatch must fund DISTINCT work, never the card's own cycle")

	pending, err := l.PendingSends()
	require.NoError(t, err)
	assert.Equal(t, 0, pending, "the refused dispatch funded no work-order")
	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "the refused dispatch debited nothing — nothing was written")
}

func TestAppendSend_fundsAPacketCarryingItsDistinctTargetForTheRunnerToExecute(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))

	tgt := distinctTarget()
	require.NoError(t, l.AppendSend("dispatch", tgt, ownTarget()))

	orders, err := l.Packets()
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, tgt, orders[0].Target, "the funded order carries the target the runner will execute — distinct work, not the card's own")
}

func TestSendStatus_movesQueuedToRunningToDoneViaAppendedLinesNeverMutating(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendSend("dispatch", distinctTarget(), ownTarget()))

	c, err := l.SendStatusCounts()
	require.NoError(t, err)
	assert.Equal(t, ledger.SendCounts{Queued: 1}, c, "a freshly funded order is queued")

	require.NoError(t, l.AppendStatus(1, "running"))
	c, err = l.SendStatusCounts()
	require.NoError(t, err)
	assert.Equal(t, ledger.SendCounts{Running: 1}, c, "the running status is the order's current state — last status line wins per id")

	require.NoError(t, l.AppendStatus(1, "done"))
	c, err = l.SendStatusCounts()
	require.NoError(t, err)
	assert.Equal(t, ledger.SendCounts{Done: 1}, c, "the order reaches done exactly once; the tally MOVED queued→running→done")
}

func TestSendStatus_countsTrackEachPacketIndependentlyWithoutBleedingAcrossIds(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.Append(distinctRecord(1)))
	require.NoError(t, l.AppendSend("d1", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendSend("d2", distinctTarget(), ownTarget()))

	// Advance only order 1. Order 2 must stay queued — the per-id current status
	// must not bleed across ids (a naive global status would mis-count both).
	require.NoError(t, l.AppendStatus(1, "running"))
	c, err := l.SendStatusCounts()
	require.NoError(t, err)
	assert.Equal(t, ledger.SendCounts{Queued: 1, Running: 1}, c, "order 1 is running, order 2 is still queued — counts are per-id, never bled")
}

func TestSendStatus_statusLinesNeverInflateRecordsBalanceOrWorkPackets(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendSend("dispatch", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "running"))
	require.NoError(t, l.AppendStatus(1, "done"))

	recs, err := l.Records()
	require.NoError(t, err)
	assert.Len(t, recs, 1, "a status line is not a catch")
	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "a status line is not a debit — the balance is unchanged by status transitions")
	orders, err := l.Packets()
	require.NoError(t, err)
	assert.Len(t, orders, 1, "a status line tracks an order's state — it never mints or replaces a work-order")
}

func TestQueuedPackets_returnsOnlyTheNotYetRunPacketsInFundingOrder(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.Append(distinctRecord(1)))
	require.NoError(t, l.AppendSend("d1", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendSend("d2", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "done")) // order 1 already ran

	queued, err := l.QueuedPackets()
	require.NoError(t, err)
	require.Len(t, queued, 1, "only the not-yet-run order is queued — a done order is not re-run")
	assert.Equal(t, 2, queued[0].ID, "the remaining queued order is the one the runner picks next")
}

func TestSendStatus_replaysFromThePersistedLogAlone(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendSend("dispatch", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "running"))
	require.NoError(t, l.AppendStatus(1, "done"))
	require.NoError(t, l.Close())

	reopened := boundLog(t)
	c, err := reopened.SendStatusCounts()
	require.NoError(t, err)
	assert.Equal(t, ledger.SendCounts{Done: 1}, c, "status is appended ID-keyed lines (never mutated), so it replays purely from disk")
}
