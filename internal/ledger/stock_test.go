package ledger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

func catchRec(reason string, selfFlag, wouldShip bool) ledger.CatchRecord {
	r := sampleRecord()
	r.ReasonTag, r.SelfFlagged, r.WouldHaveShipped = reason, selfFlag, wouldShip
	return r
}

func TestConfirmedCatches_countsAndTalliesEveryRealCatch(t *testing.T) {
	t.Parallel()
	s := ledger.ConfirmedCatches([]ledger.CatchRecord{
		catchRec("catch", true, false),
		catchRec("catch", false, true),
		catchRec("catch", true, true),
	})
	assert.Equal(t, 3, s.Count, "every confirmed catch is a held quantity")
	assert.Equal(t, 3, s.ByReason["catch"])
	assert.Equal(t, 2, s.SelfFlagged)
	assert.Equal(t, 2, s.WouldHaveShipped)
}

func TestConfirmedCatches_splitsReinvestedFromConnectMintedSoCompoundingIsLegible(t *testing.T) {
	t.Parallel()
	// A catch minted by a dispatched run carries Producer "wo:<id>"; a connect-cycle
	// mint carries "connect". Reinvested is an ADDITIVE PARTITION of Count (the
	// dispatch-minted share), so the surface can show a spend's catch as distinct
	// from a fresh connect mint — the reinvestment chain made visible.
	s := ledger.ConfirmedCatches([]ledger.CatchRecord{
		{Outcome: catch.Catch, ReasonTag: "catch", Producer: "connect"},
		{Outcome: catch.Catch, ReasonTag: "catch", Producer: "wo:7"},
		{Outcome: catch.Catch, ReasonTag: "catch", Producer: "wo:8"},
		{Outcome: catch.NoCatch, Producer: "wo:9"}, // a non-catch never counts, even tagged wo:
	})
	assert.Equal(t, 3, s.Count, "three real catches")
	assert.Equal(t, 2, s.Reinvested, "two were minted by dispatched runs (wo: prefix); the non-catch wo:9 never counts")
	assert.Equal(t, 1, s.Count-s.Reinvested, "the remainder is connect-minted — an exact partition")
}

func TestConfirmedCatches_aPreProvenanceCatchIsConnectMintedNotReinvested(t *testing.T) {
	t.Parallel()
	// A catch with an empty/absent Producer (a pre-provenance or connect mint) must
	// NOT read as reinvested — the compounding claim can never be silently inflated.
	s := ledger.ConfirmedCatches([]ledger.CatchRecord{
		{Outcome: catch.Catch, ReasonTag: "catch", Producer: ""},
		{Outcome: catch.Catch, ReasonTag: "catch"}, // zero-value Producer
	})
	assert.Equal(t, 2, s.Count)
	assert.Equal(t, 0, s.Reinvested, "no wo: prefix → not reinvested; an empty Producer never inflates the compounding count")
}

func TestConfirmedCatches_ignoresNonCatchRecordsSoTheStockCannotInflate(t *testing.T) {
	t.Parallel()
	s := ledger.ConfirmedCatches([]ledger.CatchRecord{
		catchRec("catch", false, false),
		{Outcome: catch.NoCatch},
		{Outcome: catch.NoOracleSignal},
		{Outcome: catch.PartialCatch},
	})
	assert.Equal(t, 1, s.Count, "only a real Catch counts — a miswired non-catch input can never inflate the stock")
}

func TestConfirmedCatches_emptyLedgerIsAZeroStock(t *testing.T) {
	t.Parallel()
	s := ledger.ConfirmedCatches(nil)
	assert.Equal(t, 0, s.Count)
	assert.Zero(t, s.SelfFlagged)
	assert.Zero(t, s.WouldHaveShipped)
}

func TestConfirmedCatches_isAPureFunctionOfThePersistedRecords(t *testing.T) {
	t.Parallel()
	l := boundLog(t)
	for i := 0; i < 4; i++ {
		r := catchRec("catch", i%2 == 0, false)
		r.Line = 4 + i // distinct identities so 4 appends are 4 catches (the dedup keys on the identity tuple)
		require.NoError(t, l.Append(r))
	}
	require.NoError(t, l.Close())

	reopened := boundLog(t)
	recs, err := reopened.Records()
	require.NoError(t, err)

	s := ledger.ConfirmedCatches(recs)
	assert.Equal(t, 4, s.Count, "re-deriving from the re-bound log yields the same count — a total function of committed facts")
	assert.Equal(t, 2, s.SelfFlagged)
}
