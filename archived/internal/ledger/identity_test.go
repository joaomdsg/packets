package ledger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestAppend_refusesADuplicateCatchIdentitySoARerunCannotMintTwice(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	first := ledger.CatchRecord{
		Outcome: catch.Catch, Path: "adult.go", Line: 4,
		BeforeRev: "aaaa", AfterRev: "bbbb", ReasonTag: "catch",
	}
	require.NoError(t, l.Append(first), "the first mint of a catch identity is accepted")

	// Re-running the SAME (base,fix,anchor) reproduces a byte-identical identity
	// tuple — the latent double-mint #16e closes. The second Append must be a
	// no-op refusal, never a second credit (spend 1, get 0 = honest loss).
	dup := first
	require.Error(t, l.Append(dup), "a second Append of an identity already in the log is refused")

	recs, err := l.Records()
	require.NoError(t, err)
	assert.Len(t, recs, 1, "the duplicate minted nothing — the confirmed-catch count is unchanged")
	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "the duplicate added no credit to the balance")
}

func TestAppend_acceptsADistinctCatchIdentitySoGenuineWorkStillCompounds(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	require.NoError(t, l.Append(ledger.CatchRecord{
		Outcome: catch.Catch, Path: "adult.go", Line: 4,
		BeforeRev: "aaaa", AfterRev: "bbbb", ReasonTag: "catch",
	}))
	// A genuinely DISTINCT target (different line → different identity tuple) is a
	// new catch and must mint — the dedup keys on identity, never on the act of
	// appending, so honest distinct work still compounds.
	require.NoError(t, l.Append(ledger.CatchRecord{
		Outcome: catch.Catch, Path: "adult.go", Line: 9,
		BeforeRev: "aaaa", AfterRev: "bbbb", ReasonTag: "catch",
	}))

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 2, bal, "two DISTINCT catch identities are two credits")
}

func TestAppend_dedupSurvivesAReopenSoARestartCannotReopenTheFarm(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	rec := ledger.CatchRecord{
		Outcome: catch.Catch, Path: "adult.go", Line: 4,
		BeforeRev: "aaaa", AfterRev: "bbbb", ReasonTag: "catch",
	}
	require.NoError(t, l.Append(rec))
	require.NoError(t, l.Close())

	// The dedup MUST project from the committed stream, not an in-memory set — else
	// a server restart (re-bind) lets the same catch be minted again.
	reopened := boundLog(t)
	require.Error(t, reopened.Append(rec), "the same identity is refused after a re-bind — the gate replays from the stream")

	bal, err := reopened.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "the restart did not reopen the farm")
}

func TestAppend_identityKeyIsTheFullTupleNotJustPathAndLine(t *testing.T) {
	t.Parallel()
	base := ledger.CatchRecord{
		Outcome: catch.Catch, Path: "adult.go", Line: 4,
		BeforeRev: "aaaa", AfterRev: "bbbb", ReasonTag: "catch",
	}
	// Each variant differs from base in exactly ONE identity component, so each is
	// a DISTINCT catch the gate must accept — a key of only Path+Line would wrongly
	// refuse these and let a farm dodge by varying the revs/reason instead.
	variants := map[string]ledger.CatchRecord{
		"differentBeforeRev": {Outcome: catch.Catch, Path: "adult.go", Line: 4, BeforeRev: "cccc", AfterRev: "bbbb", ReasonTag: "catch"},
		"differentAfterRev":  {Outcome: catch.Catch, Path: "adult.go", Line: 4, BeforeRev: "aaaa", AfterRev: "dddd", ReasonTag: "catch"},
		"differentReasonTag": {Outcome: catch.Catch, Path: "adult.go", Line: 4, BeforeRev: "aaaa", AfterRev: "bbbb", ReasonTag: "branch_condition"},
		"differentPath":      {Outcome: catch.Catch, Path: "child.go", Line: 4, BeforeRev: "aaaa", AfterRev: "bbbb", ReasonTag: "catch"},
	}
	for name, v := range variants {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			l, _ := openLog(t)
			require.NoError(t, l.Append(base))
			require.NoError(t, l.Append(v), "a record differing in one identity component is a distinct catch and must mint")
			bal, err := l.Balance()
			require.NoError(t, err)
			assert.Equal(t, 2, bal)
		})
	}
}
