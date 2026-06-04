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

func TestLog_appendThenRecordsPreservesEveryMintTimeField(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	rec := sampleRecord()
	require.NoError(t, l.Append(rec))

	got, err := l.Records()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, rec, got[0], "every mint-time field must survive to replay")
}

func TestLog_appendsAccumulateInOrderWithoutClobbering(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

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
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	bad := sampleRecord()
	bad.Outcome = catch.NoCatch
	require.Error(t, l.Append(bad), "the log holds only confirmed catches, by construction")

	got, err := l.Records()
	require.NoError(t, err)
	assert.Empty(t, got, "a refused record must leave no trace")
}

func TestLog_skipsBlankLinesWhenReadingAHandEditedLog(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	content := `{"outcome":"catch","path":"a.go"}` + "\n\n" + `{"outcome":"catch","path":"b.go"}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	got, err := l.Records()
	require.NoError(t, err)
	require.Len(t, got, 2, "a blank line must be skipped, not break the read")
	assert.Equal(t, "a.go", got[0].Path)
	assert.Equal(t, "b.go", got[1].Path)
}

func TestLog_surfacesAnErrorOnACorruptedRecordLine(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"outcome":"catch"}`+"\n"+`{not json`+"\n"), 0o644))
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	_, err = l.Records()
	require.Error(t, err, "a corrupted ledger line must surface, not silently drop a catch")
}

func TestLog_emptyLogReadsAsNoRecords(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	got, err := l.Records()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLog_recordsSurviveReopenForDurability(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(path)
	require.NoError(t, err)
	require.NoError(t, l.Append(sampleRecord()))
	require.NoError(t, l.Close())

	reopened, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })
	got, err := reopened.Records()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "adult.go", got[0].Path)
}
