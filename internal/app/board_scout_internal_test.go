package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// The board's first-pass hit-rate must credit a CAUGHT order only when THAT order
// is the one that completed — a catch minted on a still-running order must never
// inflate a different done-but-missed order's standing. (The old Reinvested-stock
// heuristic clamped hits=min(Reinvested,Done), misattributing such a catch; the
// exact ScoutingReport gates Caught on the same order being done.) NOT parallel
// (shared liveReg).
func TestBoardRows_doesNotCreditADoneOrderForACatchOnADifferentRunningOrder(t *testing.T) {
	log := boardSession(t, "miscredit", 2, nil) // balance 2 funds two dispatches
	own := ownTargetOf(LiveConfig{BaseRev: "own-b-miscredit", FixRev: "own-f", Anchor: anchorForCap()})
	require.NoError(t, log.AppendDispatch("d1", woTargetN(1), own))
	require.NoError(t, log.AppendDispatch("d2", woTargetN(2), own))
	require.NoError(t, log.AppendStatus(1, "done"))    // order 1 ran to done, minted NOTHING
	require.NoError(t, log.AppendStatus(2, "running")) // order 2 still in flight
	// A catch on the RUNNING order 2 — it must not credit the done order 1.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Producer: "wo:2", ReasonTag: "catch", Line: 7}))

	r := rowFor(t, BoardRows(), "miscredit")
	require.Equal(t, 1, r.Done, "one order reached done")
	require.Equal(t, 0, r.Caught, "the done order minted no catch; the running order's catch must not credit it")
	require.Equal(t, 1, r.Misses, "the done order is an honest miss")
	require.Equal(t, "hit-rate 0/1", hitRateLabel(r), "a catch on a different running order must not inflate the hit-rate")
}

// The steady-state happy path: a done order whose OWN run minted a catch counts as
// a first-pass hit — so the board reads "hit-rate 1/1", no miss. (Guards against a
// broken ScoutingReport that always reports 0, which would still pass the
// misattribution test above.) NOT parallel (shared liveReg).
func TestBoardRows_countsADoneOrdersOwnCatchAsAFirstPassHit(t *testing.T) {
	log := boardSession(t, "hitlane", 1, nil) // balance 1 funds one dispatch
	own := ownTargetOf(LiveConfig{BaseRev: "own-b-hitlane", FixRev: "own-f", Anchor: anchorForCap()})
	require.NoError(t, log.AppendDispatch("d1", woTargetN(1), own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Producer: "wo:1", ReasonTag: "catch", Line: 7})) // THIS order's catch

	r := rowFor(t, BoardRows(), "hitlane")
	require.Equal(t, 1, r.Caught, "the order's own catch is a first-pass hit")
	require.Equal(t, 0, r.Misses, "a caught order is not a miss")
	require.Equal(t, "hit-rate 1/1", hitRateLabel(r))
}
