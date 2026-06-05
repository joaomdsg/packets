package ledger_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/ledger"
)

func TestAppendDispatch_fundsExactlyOneWorkOrderPerDebitConserved(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.Append(distinctRecord(1)))

	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "two dispatches each debit one catch — the balance drains exactly like two spends")

	pending, err := l.PendingDispatches()
	require.NoError(t, err)
	assert.Equal(t, 2, pending, "one debit funds exactly one work-order — conserved, debits==orders")

	orders, err := l.WorkOrders()
	require.NoError(t, err)
	require.Len(t, orders, 2)
	assert.NotEqual(t, orders[0].ID, orders[1].ID, "work-order ids are distinct and monotonic")
	assert.Less(t, orders[0].ID, orders[1].ID, "ids increase in funding order")
	for _, o := range orders {
		assert.NotEmpty(t, o.Producer, "each order carries a producer (pre-paid for the cross-process fan-out the P0 log-schema needs)")
		assert.Equal(t, "queued", o.Status, "this round funds the order queued — it does not run")
	}
}

func TestAppendDispatch_workOrdersReplayFromThePersistedLogAlone(t *testing.T) {
	t.Parallel()
	l, path := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.Append(distinctRecord(1)))
	require.NoError(t, l.AppendDispatch("dispatch-a", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendDispatch("dispatch-b", distinctTarget(), ownTarget()))
	require.NoError(t, l.Close())

	reopened, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })

	bal, err := reopened.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "the debits replay from disk — the projection holds no in-memory counter, and a work-order line never inflates the balance")

	pending, err := reopened.PendingDispatches()
	require.NoError(t, err)
	assert.Equal(t, 2, pending, "both funded work-orders survive a reopen — pure projection")

	orders, err := reopened.WorkOrders()
	require.NoError(t, err)
	require.Len(t, orders, 2)
	assert.Less(t, orders[0].ID, orders[1].ID, "monotonic ids survive the reopen — they are derived from the persisted log, not a process-local counter")
	assert.Equal(t, "dispatch-a", orders[0].Reason, "the funding reason is a persisted audit fact")
	assert.Equal(t, "dispatch-b", orders[1].Reason)
	for _, o := range orders {
		assert.NotEmpty(t, o.Producer, "the producer survives the reopen — the field the cross-process fan-out will key on")
	}
}

func TestPendingDispatches_doesNotPolluteTheConfirmedCatchCount(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(sampleRecord()))
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))

	recs, err := l.Records()
	require.NoError(t, err)
	assert.Len(t, recs, 1, "Records stays catch-only — the work-order line is skipped like a spend line")
}

func TestAppendDispatch_overBudgetFundsNoOrderAndWritesNothing(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	require.Error(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()), "a dispatch with nothing to fund is refused — you cannot dispatch what you did not catch")

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "the refused dispatch left no debit")
	pending, err := l.PendingDispatches()
	require.NoError(t, err)
	assert.Equal(t, 0, pending, "the refused dispatch funded no work-order — nothing was written")
}

func TestAppendDispatch_isAtomicUnderRaceNeverTearsOrOverFunds(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(sampleRecord())) // a balance of exactly 1: more dispatchers than credit

	const dispatchers = 8
	var ok int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < dispatchers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := l.AppendDispatch("dispatch", distinctTarget(), ownTarget()); err == nil {
				atomic.AddInt64(&ok, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), atomic.LoadInt64(&ok), "exactly one dispatcher wins the lone catch — the read-then-write is atomic, no over-fund")
	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "the balance never overshoots below zero")
	pending, err := l.PendingDispatches()
	require.NoError(t, err)
	assert.Equal(t, 1, pending, "exactly one work-order was funded — the debit and the order never tear apart")
}
