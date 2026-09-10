package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// boardSession, rowIndex, rowFor, requireBefore are duplicated for the external
// test package (via addFundedSession) for the tests that moved there — kept
// here too since board_activity/board_inflight/board_scout/board_land/
// board_land_summary/board_questions/board_lifecycle stay internal (each needs
// an unexported liveEntry setter or hitRateLabel with no public equivalent).
func boardSession(t *testing.T, key string, seedCatches int, backlog []ledger.Target) *ledger.Log {
	t.Helper()
	log := scratchLog(t)
	for i := 0; i < seedCatches; i++ {
		require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 100 + i, ReasonTag: "catch"}))
	}
	registerSession(key, LiveConfig{BaseRev: "own-b-" + key, FixRev: "own-f", Anchor: anchorForCap(), SendBacklog: backlog}, log)
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

func TestHitRateLabel_isAPureCountRatioOfLoggedBetsNeverAForecast(t *testing.T) {
	t.Parallel()
	// The hit-rate is the one honest progression number: Caught (orders whose run
	// minted a confirmed catch, the exact ScoutingReport count) over Done (resolved
	// dispatched orders). A COUNT ratio of logged events, never an inferred
	// probability — so it redeems against the mint/miss the Lead actually earned.
	require.Equal(t, "hit-rate 1/4", hitRateLabel(CardRow{Caught: 1, Done: 4}))
	require.Equal(t, "hit-rate 3/3", hitRateLabel(CardRow{Caught: 3, Done: 3}), "every bet paid")
	require.Equal(t, "hit-rate 0/0", hitRateLabel(CardRow{Done: 0, Caught: 0}), "no bets resolved yet — a calm 0/0, never NaN or a divide-by-zero")
}

func TestHitRateLabel_caughtNeverExceedsDoneByConstruction(t *testing.T) {
	t.Parallel()
	// Caught comes from ledger.ScoutingReport, which counts only catches on orders
	// that are themselves done — so Caught ≤ Done holds by construction and no clamp
	// is needed (the old Reinvested-stock heuristic needed one because it counted a
	// catch on a not-yet-done order).
	require.Equal(t, "hit-rate 2/2", hitRateLabel(CardRow{Caught: 2, Done: 2}), "all resolved orders caught")
}
