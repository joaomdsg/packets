package ledger_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestShouldRecord_persistsOnlyConfirmedCatches(t *testing.T) {
	t.Parallel()
	assert.True(t, ledger.ShouldRecord(catch.Catch), "a real mint is recorded")
	assert.False(t, ledger.ShouldRecord(catch.NoCatch), "no-op churn / no-catch leaves no trace")
	assert.False(t, ledger.ShouldRecord(catch.NoOracleSignal))
	assert.False(t, ledger.ShouldRecord(catch.PartialCatch))
}

func sampleRecord() ledger.CatchRecord {
	return ledger.CatchRecord{
		Outcome:           catch.Catch,
		Path:              "adult.go",
		Line:              4,
		BeforeRev:         "aaaa",
		AfterRev:          "bbbb",
		BeforeInventory:   []string{">="},
		AfterInventory:    []string{">="},
		MutantsConsidered: 1,
		ReasonTag:         "catch",
		SelfFlagged:       true,
		WouldHaveShipped:  true,
	}
}

func distinctRecord(i int) ledger.CatchRecord {
	r := sampleRecord()
	r.Line = 4 + i // a distinct anchored line → a distinct catch identity, so N appends are N catches
	return r
}

func TestLog_appendThenRecordsPreservesEveryMintTimeField(t *testing.T) {
	t.Parallel()
	l := boundLog(t)

	rec := sampleRecord()
	require.NoError(t, l.Append(rec))

	got, err := l.Records()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, rec, got[0], "every mint-time field must survive to replay")
}

func TestLog_appendsAccumulateInOrderWithoutClobbering(t *testing.T) {
	t.Parallel()
	l := boundLog(t)

	first := sampleRecord()
	first.Path = "first.go"
	second := sampleRecord()
	second.Path = "second.go"
	require.NoError(t, l.Append(first))
	require.NoError(t, l.Append(second))

	got, err := l.Records()
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "first.go", got[0].Path)
	assert.Equal(t, "second.go", got[1].Path)
}

func TestLog_refusesToPersistANonCatchRecord(t *testing.T) {
	t.Parallel()
	l := boundLog(t)

	bad := sampleRecord()
	bad.Outcome = catch.NoCatch
	require.Error(t, l.Append(bad), "the log holds only confirmed catches, by construction")

	got, err := l.Records()
	require.NoError(t, err)
	assert.Empty(t, got, "a refused record must leave no trace")
}

func TestLog_surfacesAnErrorOnACorruptedStreamPayload(t *testing.T) {
	t.Parallel()
	l, f := openLog(t)
	require.NoError(t, l.Append(sampleRecord()))

	// A forged, undecodable payload on a catch subject must surface on read, never
	// silently drop a catch — the stream analogue of a corrupted ledger line.
	_, err := f.Publish(context.Background(), fabric.EventSubject(t.Name(), "i", fabric.StatusMinted, "catch"), []byte(`{not json`))
	require.NoError(t, err)

	_, err = l.Records()
	require.Error(t, err, "a corrupted catch payload must surface, not silently drop a catch")
}

func TestLog_emptyLogReadsAsNoRecords(t *testing.T) {
	t.Parallel()
	l := boundLog(t)

	got, err := l.Records()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLog_concurrentAppendAndSpendNeverOverspend(t *testing.T) {
	t.Parallel()
	// The live server drives two writers at once: the catch cycle's Append (a mint)
	// races the Lead's AppendSpend (a debit, action goroutine). Without
	// serialization a TOCTOU read-then-write lets a spend exceed the balance. Seed
	// credits, hammer both writers, and assert the balance equals exact arithmetic
	// and never overshoots below zero.
	l := boundLog(t)

	// A small, exhaustible balance: more spenders than credits. AppendSpend
	// replays the balance, then publishes — a TOCTOU window where concurrent
	// spenders each see "enough" before any commits its debit, overshooting zero.
	const seed = 25
	for i := 0; i < seed; i++ {
		require.NoError(t, l.Append(distinctRecord(i)))
	}

	const spenders = 50
	var wg sync.WaitGroup
	var ok int64
	var okMu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < spenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release all at once to widen the TOCTOU window
			if err := l.AppendSpend(1, "dispatch"); err == nil {
				okMu.Lock()
				ok++
				okMu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	recs, err := l.Records()
	require.NoError(t, err, "every committed event must be well-formed")
	bal, err := l.Balance()
	require.NoError(t, err)
	assert.LessOrEqual(t, int(ok), seed,
		"a TOCTOU on Balance must never let more spends succeed than there was balance to cover")
	assert.Equal(t, len(recs)-int(ok), bal,
		"balance must equal minted catches minus successful spends — no lost or double-counted write")
	assert.GreaterOrEqual(t, bal, 0, "the balance must never overshoot below zero")
}

func TestLog_recordsSurviveAReBindForDurability(t *testing.T) {
	t.Parallel()
	l := boundLog(t)
	require.NoError(t, l.Append(sampleRecord()))
	require.NoError(t, l.Close())

	// A fresh Log on the same session replays the committed catch — the projection
	// holds no in-memory state, so a restart cannot lose (or re-mint) it.
	reopened := boundLog(t)
	got, err := reopened.Records()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "adult.go", got[0].Path)
}
