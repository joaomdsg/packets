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
	require.Greater(t, b.BacklogRemaining, 0, "both config targets are funded, but from-catch supply keeps fundable candidate work — the faucet refills, no silent dead-end")

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

func TestBoardRows_surfacesADoneOrderThatMintedNothingAsAVisibleMiss(t *testing.T) {
	// The honest loss must be VISIBLE on the board, never a silent discard: a done
	// order that minted no catch (Done counted it, but no "wo:" catch joined the
	// stock) shows as a MISS — the spend was a bet that did not pay, and the Lead
	// can see it. Misses = Done − Reinvested (clamped at 0).
	log := boardSession(t, "missK", 1, []ledger.Target{woTargetN(1)})
	require.NoError(t, log.AppendDispatch("d", woTargetN(1), ownTargetOf(LiveConfig{BaseRev: "own-b-missK", FixRev: "own-f", Anchor: anchorForCap()})))
	require.NoError(t, log.AppendStatus(1, "done")) // the order ran to done but minted NOTHING

	r := rowFor(t, BoardRows(), "missK")
	require.Equal(t, 1, r.Done, "the order reached done")
	require.Equal(t, 0, r.Reinvested, "it minted no catch")
	require.Equal(t, 1, r.Misses, "a done-but-no-mint order is a VISIBLE miss — the honest loss, not a silent discard")
}

func TestHitRateLabel_isAPureCountRatioOfLoggedBetsNeverAForecast(t *testing.T) {
	t.Parallel()
	// The hit-rate is the one honest progression number: Hits (catches a bet
	// minted, = Reinvested) over Bets (resolved dispatched orders, = Done). A COUNT
	// ratio of logged events, never an inferred probability — so it redeems against
	// the mint/miss the Lead actually earned, not a model's forecast.
	require.Equal(t, "hit-rate 1/4", hitRateLabel(CardRow{Reinvested: 1, Done: 4}))
	require.Equal(t, "hit-rate 3/3", hitRateLabel(CardRow{Reinvested: 3, Done: 3}), "every bet paid")
	require.Equal(t, "hit-rate 0/0", hitRateLabel(CardRow{Done: 0, Reinvested: 0}), "no bets resolved yet — a calm 0/0, never NaN or a divide-by-zero")
}

func TestHitRateLabel_neverReadsMoreHitsThanResolvedBets(t *testing.T) {
	t.Parallel()
	// A "wo:" catch is Appended before its order's "done" status line (runOneOrder),
	// so a board read can briefly observe Reinvested > Done. The standing is Hits over
	// Bets — Hits can never exceed Bets, so the displayed numerator is clamped at the
	// denominator, mirroring the Misses = max(0, Done−Reinvested) guard in BoardRows.
	// Without the clamp this leaks a nonsense ratio like "hit-rate 1/0".
	require.Equal(t, "hit-rate 0/0", hitRateLabel(CardRow{Reinvested: 1, Done: 0}), "a catch logged before its done line must not read Hits > Bets")
	require.Equal(t, "hit-rate 2/2", hitRateLabel(CardRow{Reinvested: 3, Done: 2}), "the numerator is clamped to the resolved-bet count")
}
