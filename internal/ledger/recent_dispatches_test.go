package ledger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

// woCatch builds a distinct confirmed catch minted by a dispatched work-order
// (Producer "wo:<id>") — the provenance that marks that order CAUGHT.
func woCatch(i, id int) ledger.CatchRecord {
	r := distinctRecord(i)
	r.Producer = "wo:" + itoa(id)
	return r
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func dispatchByID(views []ledger.DispatchView, id int) (ledger.DispatchView, bool) {
	for _, v := range views {
		if v.ID == id {
			return v, true
		}
	}
	return ledger.DispatchView{}, false
}

// A funded order that ran and minted a catch must read CAUGHT — the Lead funded
// it and it paid off; the round-trip is legible end to end.
func TestRecentDispatches_marksAFundedOrderThatMintedAsCaught(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0))) // balance 1, funds one dispatch
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "running"))
	require.NoError(t, l.Append(woCatch(5, 1))) // the work-order's mint
	require.NoError(t, l.AppendStatus(1, "done"))

	views, err := l.RecentDispatches(10)
	require.NoError(t, err)
	v, ok := dispatchByID(views, 1)
	require.True(t, ok, "the funded order appears in recent dispatches")
	assert.Equal(t, distinctTarget(), v.Target, "the order carries the target it ran")
	assert.Equal(t, "done", v.Status)
	assert.True(t, v.Caught, "a dispatched order whose run minted a wo:<id> catch reads CAUGHT")
}

// A funded order that ran and DID NOT mint is a MISSED bet — an honest loss the
// Lead must see, distinct from a catch.
func TestRecentDispatches_marksADoneOrderWithNoCatchAsMissed(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "running"))
	require.NoError(t, l.AppendStatus(1, "done")) // no wo:1 catch

	views, err := l.RecentDispatches(10)
	require.NoError(t, err)
	v, ok := dispatchByID(views, 1)
	require.True(t, ok)
	assert.Equal(t, "done", v.Status)
	assert.False(t, v.Caught, "a done order with no wo:<id> catch is a missed bet, not caught")
}

// A still-queued order reads its queued status and is not yet caught.
func TestRecentDispatches_showsAQueuedOrderAsQueuedNotCaught(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))

	views, err := l.RecentDispatches(10)
	require.NoError(t, err)
	v, ok := dispatchByID(views, 1)
	require.True(t, ok)
	assert.Equal(t, "queued", v.Status, "a funded-but-unrun order is queued")
	assert.False(t, v.Caught)
}

// The view is bounded to the most recent n orders (newest first), so the board
// shows the latest activity without unbounded growth; n<=0 returns all.
func TestRecentDispatches_limitsToTheMostRecentN(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(distinctRecord(i)))
	}
	require.NoError(t, l.AppendDispatch("d1", target(1), ownTarget()))
	require.NoError(t, l.AppendDispatch("d2", target(2), ownTarget()))
	require.NoError(t, l.AppendDispatch("d3", target(3), ownTarget()))

	recent, err := l.RecentDispatches(2)
	require.NoError(t, err)
	require.Len(t, recent, 2, "only the most recent two")
	assert.Equal(t, 3, recent[0].ID, "newest first")
	assert.Equal(t, 2, recent[1].ID)

	all, err := l.RecentDispatches(0)
	require.NoError(t, err)
	require.Len(t, all, 3, "n<=0 returns every dispatch")
}

// A CONNECT-cycle catch (Producer "connect") must NEVER mark a work-order caught:
// only the order's own "wo:<id>" provenance counts. Otherwise an unrelated connect
// mint would falsely credit a dispatched order (a two-scores/provenance leak).
func TestRecentDispatches_aConnectCatchNeverMarksAWorkOrderCaught(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "done"))
	connect := distinctRecord(7)
	connect.Producer = "connect"
	require.NoError(t, l.Append(connect))

	views, err := l.RecentDispatches(10)
	require.NoError(t, err)
	v, ok := dispatchByID(views, 1)
	require.True(t, ok)
	assert.False(t, v.Caught, "a connect-cycle catch does not credit a dispatched order — only wo:<id> does")
}

// target builds a distinct dispatch target keyed by n (≠ ownTarget).
func target(n int) ledger.Target {
	return ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "wo" + itoa(n) + ".go", Line: 9}
}
