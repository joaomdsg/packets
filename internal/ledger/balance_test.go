package ledger_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
)

func openLog(t *testing.T) (*ledger.Log, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	return l, path
}

func TestBalance_isConfirmedCatchesMinusSpends(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(sampleRecord()))
	}
	require.NoError(t, l.AppendSpend(2, "dispatch"))

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "balance is credits (3 confirmed catches) minus debits (one spend of 2)")
}

func TestBalance_replaysIdenticallyFromTheLogAlone(t *testing.T) {
	t.Parallel()
	l, path := openLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(sampleRecord()))
	}
	require.NoError(t, l.AppendSpend(2, "dispatch"))

	reopened, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })
	bal, err := reopened.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "balance is a pure projection of the persisted log — no in-memory counter")
}

func TestAppendSpend_rejectsAnOverBudgetSpendWithoutLoggingIt(t *testing.T) {
	t.Parallel()
	l, path := openLog(t)
	require.NoError(t, l.Append(sampleRecord())) // balance 1

	before, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Error(t, l.AppendSpend(5, "too much"), "a spend exceeding the balance must be refused — you cannot spend what you did not catch")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after, "a rejected over-budget spend must write nothing to the log — byte-identical")
	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "the balance is unchanged by a rejected spend")
}

func TestAppendSpend_rejectsANonPositiveAmount(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(sampleRecord()))
	assert.Error(t, l.AppendSpend(0, "nothing"), "a spend must move a positive amount")
	assert.Error(t, l.AppendSpend(-1, "negative"), "a negative spend would mint credit — forbidden")
}

func TestAppendSpend_allowsSpendingExactlyTheWholeBalanceToZero(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(sampleRecord()))
	}
	require.NoError(t, l.AppendSpend(3, "spend it all"), "spending exactly the balance is allowed — the boundary is >=, not >")

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal)
	assert.Error(t, l.AppendSpend(1, "overdraft"), "a zero balance affords no further spend")
}

func TestAppendSpend_accumulatesSoEachSpendTradesAgainstWhatRemains(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(sampleRecord()))
	}
	require.NoError(t, l.AppendSpend(1, "first"))
	require.NoError(t, l.AppendSpend(1, "second"))

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "spends accumulate — balance reflects every debit so far")
	assert.Error(t, l.AppendSpend(2, "over what remains"), "a later spend trades against the REMAINING balance, not the original")
	require.NoError(t, l.AppendSpend(1, "last"))
	bal, err = l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal)
}

func TestRecords_skipsSpendLinesSoTheConfirmedCatchCountStaysClean(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(sampleRecord()))
	}
	require.NoError(t, l.AppendSpend(1, "dispatch"))

	recs, err := l.Records()
	require.NoError(t, err)
	assert.Len(t, recs, 3, "Records returns confirmed catches only — a spend line never inflates the catch count")
	assert.Equal(t, 3, ledger.ConfirmedCatches(recs).Count)
}

func TestAppend_stillRefusesNonCatchEvenAfterSpendsExist(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.Append(sampleRecord()))
	require.NoError(t, l.AppendSpend(1, "dispatch"))

	bad := sampleRecord()
	bad.Outcome = catch.NoCatch
	require.Error(t, l.Append(bad), "the catch-only farm-denial gate stays intact — debits travel AppendSpend, never Append")
}

func TestAppendSpend_persistsTheReasonAsALoggedAuditFact(t *testing.T) {
	t.Parallel()
	l, path := openLog(t)
	require.NoError(t, l.Append(sampleRecord()))
	require.NoError(t, l.AppendSpend(1, "dispatch-to-lane-7"))

	// Reason is write-only on the public API (Records skips spend lines, Balance
	// reads only Amount), so pin it on disk: the audit reason must round-trip.
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"reason":"dispatch-to-lane-7"`, "a spend's reason is a logged audit fact and must be persisted verbatim")
}

func TestBalance_aHandEditedNegativeSpendCannotMintCredit(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	// AppendSpend rejects amount<=0, but the JSONL is the authoritative replay
	// substrate; a hand-edited negative spend must not drive the balance UP.
	content := `{"outcome":"catch","path":"a.go"}` + "\n" +
		`{"outcome":"catch","path":"b.go"}` + "\n" +
		`{"kind":"spend","amount":-5,"reason":"forged credit"}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 2, bal, "a spend can never mint credit — a non-positive spend amount in the log contributes nothing")
}

func TestBalance_onAPreSpendCatchOnlyLogIsJustTheCatchCount(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	// A log written before spends existed: catch lines with no "kind" field.
	content := `{"outcome":"catch","path":"a.go"}` + "\n" + `{"outcome":"catch","path":"b.go"}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 2, bal, "a pre-spend catch-only log reads as a balance of its catch count — no migration, no kind field needed")
}
