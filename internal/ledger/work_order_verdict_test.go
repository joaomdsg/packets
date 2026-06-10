package ledger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

// A dispatched order that MISSED leaves only "done + not caught" today — the Lead
// can't see WHY it missed (no catch, no signal, anchor lost…). The oracle computes
// an honest verdict for every order it runs; persisting it per order is the
// foundation of order diagnostics. A persisted per-order verdict must surface on
// that order's dispatch view.
func TestRecentDispatches_carriesThePersistedPerOrderVerdict(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0))) // balance 1, funds one order
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendStatus(1, "done"))
	require.NoError(t, l.AppendWorkOrderVerdict(1, "no-catch")) // the oracle's honest verdict for this order

	views, err := l.RecentDispatches(10)
	require.NoError(t, err)
	v, ok := dispatchByID(views, 1)
	require.True(t, ok)
	assert.Equal(t, "no-catch", v.Verdict, "the order surfaces the oracle verdict persisted for it")
}

// An order with no persisted verdict (e.g. still queued, or pre-diagnostics data)
// reads an EMPTY verdict — never a fabricated default that would imply the oracle
// said something it didn't.
func TestRecentDispatches_anOrderWithoutAPersistedVerdictReadsEmpty(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))

	views, err := l.RecentDispatches(10)
	require.NoError(t, err)
	v, ok := dispatchByID(views, 1)
	require.True(t, ok)
	assert.Equal(t, "", v.Verdict, "no verdict persisted yet → empty, never a fabricated default")
}

// An order can be re-run (the drain loop retries under a cap), appending a verdict
// each run — the order's CURRENT verdict is the last one persisted, mirroring the
// append-only last-writer-wins of its status line.
func TestRecentDispatches_perOrderVerdictIsLastWriterWins(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget()))
	require.NoError(t, l.AppendWorkOrderVerdict(1, "no-oracle-signal")) // first run
	require.NoError(t, l.AppendWorkOrderVerdict(1, "no-catch"))         // a later run resolves differently
	require.NoError(t, l.AppendStatus(1, "done"))

	views, err := l.RecentDispatches(10)
	require.NoError(t, err)
	v, ok := dispatchByID(views, 1)
	require.True(t, ok)
	assert.Equal(t, "no-catch", v.Verdict, "the order's current verdict is the last one persisted")
}

// A per-order verdict is DIAGNOSTIC metadata, never an economic event: it must not
// mint balance nor count as a confirmed catch (the two-scores invariant). It lives
// alongside the work-order/status lines, not the catch/balance ledger.
func TestAppendWorkOrderVerdict_isNeverACatchAndNeverMovesBalance(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))                              // balance 1 (one confirmed catch)
	require.NoError(t, l.AppendDispatch("dispatch", distinctTarget(), ownTarget())) // spend → balance 0
	require.NoError(t, l.AppendWorkOrderVerdict(1, "no-catch"))

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "a per-order verdict never mints balance (two-scores)")

	recs, err := l.Records()
	require.NoError(t, err)
	assert.Equal(t, 1, ledger.ConfirmedCatches(recs).Count, "the verdict fact is not a confirmed catch")
}
