package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
)

func boardSession(t *testing.T, key string, seedCatches int, backlog []ledger.Target) *ledger.Log {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), key+".jsonl")
	log, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	for i := 0; i < seedCatches; i++ {
		require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 100 + i, ReasonTag: "catch"}))
	}
	registerSession(key, LiveConfig{BaseRev: "own-b-" + key, FixRev: "own-f", Anchor: anchorForCap(), DispatchBacklog: backlog}, log)
	return log
}

func rowIndex(rows []CardRow, key string) int {
	for i, r := range rows {
		if r.Key == key {
			return i
		}
	}
	return -1
}

func rowFor(t *testing.T, rows []CardRow, key string) CardRow {
	t.Helper()
	i := rowIndex(rows, key)
	require.GreaterOrEqual(t, i, 0, "the board must include a row for "+key)
	return rows[i]
}

func requireBefore(t *testing.T, rows []CardRow, earlier, later string) {
	t.Helper()
	ei, li := rowIndex(rows, earlier), rowIndex(rows, later)
	require.GreaterOrEqual(t, ei, 0, "the board must include a row for "+earlier)
	require.GreaterOrEqual(t, li, 0, "the board must include a row for "+later)
	require.Less(t, ei, li, earlier+" must sort before "+later)
}

func TestBoardRows_ordersByQueuedActivitySoTheLeadSeesWhereTheWorkIsMoving(t *testing.T) {
	// The fleet board ranges liveReg and ranks cards by QUEUED-awaiting-drain — an
	// honest, log-derived "where motion is" signal — so a card with work in flight
	// sorts above an idle one. (Relative order of THIS test's keys; liveReg is a
	// shared global polluted by other tests, so we filter to our own keys.)
	t1, t2 := woTargetN(1), woTargetN(2)
	logB := boardSession(t, "brd-B", 3, []ledger.Target{t1, t2})
	require.NoError(t, logB.AppendDispatch("d", t1, ownTargetOf(LiveConfig{BaseRev: "own-b-brd-B", FixRev: "own-f", Anchor: anchorForCap()})))
	require.NoError(t, logB.AppendDispatch("d", t2, ownTargetOf(LiveConfig{BaseRev: "own-b-brd-B", FixRev: "own-f", Anchor: anchorForCap()})))
	boardSession(t, "brd-A", 1, nil) // a balance, no dispatch → 0 queued

	rows := BoardRows()
	requireBefore(t, rows, "brd-B", "brd-A") // brd-B (2 queued) sorts above brd-A (0 queued) — activity, not hoard size

	b := rowFor(t, rows, "brd-B")
	require.Equal(t, 2, b.Queued, "brd-B has two funded, undrained orders")
	require.Equal(t, 1, b.Balance, "brd-B: 3 catches − 2 dispatched debits")
	require.Equal(t, 3, b.Confirmed)
	require.Equal(t, 0, b.BacklogRemaining, "both backlog targets are funded → none remaining")

	a := rowFor(t, rows, "brd-A")
	require.Equal(t, 0, a.Queued)
	require.Equal(t, 1, a.Balance)
}

func TestBoardRows_reSortsAsWorkDrainsAndIsFundedElsewhere(t *testing.T) {
	t1, t2 := woTargetN(1), woTargetN(2)
	own := ownTargetOf(LiveConfig{BaseRev: "own-b-rsB", FixRev: "own-f", Anchor: anchorForCap()})
	logB := boardSession(t, "rsB", 3, []ledger.Target{t1, t2})
	require.NoError(t, logB.AppendDispatch("d", t1, own))
	require.NoError(t, logB.AppendDispatch("d", t2, own))
	logA := boardSession(t, "rsA", 2, []ledger.Target{t1})

	requireBefore(t, BoardRows(), "rsB", "rsA") // rsB leads while it holds the queued work

	// rsB's orders run to done (no longer queued); rsA funds one.
	require.NoError(t, logB.AppendStatus(1, "done"))
	require.NoError(t, logB.AppendStatus(2, "done"))
	require.NoError(t, logA.AppendDispatch("d", t1, ownTargetOf(LiveConfig{BaseRev: "own-b-rsA", FixRev: "own-f", Anchor: anchorForCap()})))

	requireBefore(t, BoardRows(), "rsA", "rsB") // attention follows the queued work — rsA now leads
}

func TestBoardRows_tieBreaksDeterministicallyByRegistrationOrderNotMapRandomness(t *testing.T) {
	// Equal queued counts must NOT order by sync.Map's nondeterministic Range — the
	// earlier-registered card precedes the later, stably across renders, so the
	// board never flickers (and never fabricates an order from a missing timestamp).
	boardSession(t, "tieEarly", 0, nil)
	boardSession(t, "tieLate", 0, nil)
	for i := 0; i < 5; i++ {
		rows := BoardRows()
		requireBefore(t, rows, "tieEarly", "tieLate") // equal-queued cards hold registration order on every render
	}
}
