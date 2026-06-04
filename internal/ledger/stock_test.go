package ledger_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
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
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(path)
	require.NoError(t, err)
	for i := 0; i < 4; i++ {
		require.NoError(t, l.Append(catchRec("catch", i%2 == 0, false)))
	}
	require.NoError(t, l.Close())

	reopened, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })
	recs, err := reopened.Records()
	require.NoError(t, err)

	s := ledger.ConfirmedCatches(recs)
	assert.Equal(t, 4, s.Count, "re-deriving from the re-opened log yields the same count — a total function of persisted facts")
	assert.Equal(t, 2, s.SelfFlagged)
}
