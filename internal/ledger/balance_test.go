package ledger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestBalance_isConfirmedCatchesMinusSpends(t *testing.T) {
	t.Parallel()
	l := boundLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(distinctRecord(i)))
	}
	require.NoError(t, l.AppendSpend(2, "dispatch"))

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "balance is credits (3 confirmed catches) minus debits (one spend of 2)")
}

func TestBalance_replaysIdenticallyFromTheStreamAlone(t *testing.T) {
	t.Parallel()
	l := boundLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(distinctRecord(i)))
	}
	require.NoError(t, l.AppendSpend(2, "dispatch"))

	// A FRESH Log bound to the same session sees the same balance — the projection
	// holds no in-memory counter, it replays the committed stream.
	reopened := boundLog(t)
	bal, err := reopened.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "balance is a pure projection of the committed stream — no in-memory counter")
}

func TestAppendSpend_rejectsAnOverBudgetSpendWithoutLoggingIt(t *testing.T) {
	t.Parallel()
	l, f := openLog(t)
	require.NoError(t, l.Append(sampleRecord())) // balance 1

	before := eventCount(t, f, t.Name())
	require.Error(t, l.AppendSpend(5, "too much"), "a spend exceeding the balance must be refused — you cannot spend what you did not catch")
	after := eventCount(t, f, t.Name())

	assert.Equal(t, before, after, "a rejected over-budget spend must publish nothing — no new stream event")
	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "the balance is unchanged by a rejected spend")
}

func TestAppendSpend_rejectsANonPositiveAmount(t *testing.T) {
	t.Parallel()
	l := boundLog(t)
	require.NoError(t, l.Append(sampleRecord()))
	assert.Error(t, l.AppendSpend(0, "nothing"), "a spend must move a positive amount")
	assert.Error(t, l.AppendSpend(-1, "negative"), "a negative spend would mint credit — forbidden")
}

func TestAppendSpend_allowsSpendingExactlyTheWholeBalanceToZero(t *testing.T) {
	t.Parallel()
	l := boundLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(distinctRecord(i)))
	}
	require.NoError(t, l.AppendSpend(3, "spend it all"), "spending exactly the balance is allowed — the boundary is >=, not >")

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal)
	assert.Error(t, l.AppendSpend(1, "overdraft"), "a zero balance affords no further spend")
}

func TestAppendSpend_accumulatesSoEachSpendTradesAgainstWhatRemains(t *testing.T) {
	t.Parallel()
	l := boundLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(distinctRecord(i)))
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
	l := boundLog(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Append(distinctRecord(i)))
	}
	require.NoError(t, l.AppendSpend(1, "dispatch"))

	recs, err := l.Records()
	require.NoError(t, err)
	assert.Len(t, recs, 3, "Records returns confirmed catches only — a spend event never inflates the catch count")
	assert.Equal(t, 3, ledger.ConfirmedCatches(recs).Count)
}

func TestAppend_stillRefusesNonCatchEvenAfterSpendsExist(t *testing.T) {
	t.Parallel()
	l := boundLog(t)
	require.NoError(t, l.Append(sampleRecord()))
	require.NoError(t, l.AppendSpend(1, "dispatch"))

	bad := sampleRecord()
	bad.Outcome = catch.NoCatch
	require.Error(t, l.Append(bad), "the catch-only farm-denial gate stays intact — debits travel AppendSpend, never Append")
}

func TestAppendSpend_persistsTheReasonAsALoggedAuditFact(t *testing.T) {
	t.Parallel()
	l, f := openLog(t)
	require.NoError(t, l.Append(sampleRecord()))
	require.NoError(t, l.AppendSpend(1, "dispatch-to-lane-7"))

	// Reason is write-only on the projecting API (Records skips spends, Balance
	// reads only Amount), so replay the spend subject: the audit reason must
	// round-trip on the stream verbatim.
	events, err := f.ReplaySubject(context.Background(), fabric.EventSubject(t.Name(), "i", fabric.StatusMinted, "spend"))
	require.NoError(t, err)
	require.Len(t, events, 1)
	spend, err := ledger.DecodeSpend(events[0].Data)
	require.NoError(t, err)
	assert.Equal(t, "dispatch-to-lane-7", spend.Reason, "a spend's reason is a logged audit fact and must round-trip")
}

func TestBalance_aForgedNegativeSpendCannotMintCredit(t *testing.T) {
	t.Parallel()
	l, f := openLog(t)
	require.NoError(t, l.Append(distinctRecord(0)))
	require.NoError(t, l.Append(distinctRecord(1))) // balance 2

	// AppendSpend rejects amount<=0, but the stream is the authoritative substrate;
	// a FORGED negative spend published straight to the subject must not drive the
	// balance UP — the projection's amount>0 guard holds against hand-forged data.
	_, err := ledger.PublishSpend(context.Background(), f, t.Name(), "i", ledger.SpendRecord{Kind: "spend", Amount: -5, Reason: "forged credit"})
	require.NoError(t, err)

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 2, bal, "a spend can never mint credit — a non-positive amount on the stream contributes nothing")
}
