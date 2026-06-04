package app_test

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/agntpr/internal/app"
	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
)

func TestLiveServer_streamsAVerdictFromInFlightToCaughtAndLogsIt(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "strengthen the test")

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: fix, TipRev: fix, Anchor: anchor(),
		TestCmd: goTestCmd, LedgerPath: logPath, SelfFlagged: true, WouldHaveShipped: true,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	require.Contains(t, tc.HTML(), `data-state="in-flight"`, "the card starts in-flight before the cycle resolves")

	frames, cancel := tc.SSE()
	defer cancel()
	frame := vt.AwaitFrame(t, frames, 20*time.Second, `data-state="catch"`)
	assert.Contains(t, frame, `data-state="land-clean"`,
		"the served card shows the integration row alongside the verdict: tip==fix integrates clean")

	records, err := log.Records()
	require.NoError(t, err)
	require.Len(t, records, 1, "the watched catch is durably logged")
	assert.Equal(t, catch.Catch, records[0].Outcome)
	assert.True(t, records[0].SelfFlagged)
}

func TestLiveServer_showsTheConfirmedCatchStockFromTheLedger(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "strengthen the test")

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "a.go", ReasonTag: "catch"}))
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "b.go", ReasonTag: "catch"}))
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: fix, TipRev: fix, Anchor: anchor(),
		TestCmd: goTestCmd, LedgerPath: logPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	// The initial connect render reads the pre-seeded ledger — the logged economy
	// is SHOWN. Asserted immediately, before the background cycle could append more.
	html := tc.HTML()
	assert.Contains(t, html, `data-state="stock"`, "the confirmed-catch stock renders as its own row")
	assert.Contains(t, html, "2 confirmed", "the stock shows the two catches already in the ledger")
	assert.Contains(t, html, `data-state="in-flight"`, "the verdict row still renders alongside the stock — rows are orthogonal")
}

func TestLiveServer_survivesACorruptLedgerByShowingAnEmptyStock(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "strengthen the test")

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	require.NoError(t, os.WriteFile(logPath, []byte("{not json\n"), 0o644)) // a corrupt ledger line

	var server *httptest.Server
	_, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: fix, TipRev: fix, Anchor: anchor(),
		TestCmd: goTestCmd, LedgerPath: logPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	html := tc.HTML()
	assert.Contains(t, html, `data-state="stock"`, "a corrupt ledger must not break the card — the stock degrades to empty, not a panic")
	assert.Contains(t, html, "0 confirmed", "an unreadable ledger reads as a calm zero, never a crash")
}

func TestLiveServer_streamsBeatsBeforeTheVerdictResolves(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "strengthen the test")

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: fix, TipRev: fix, Anchor: anchor(),
		TestCmd: goTestCmd, LedgerPath: logPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()

	// The base-oracle beat streams in while the verdict row is STILL in-flight —
	// impossible if the card snapped every beat in one terminal flush. Real oracle
	// work (the fix oracle + integrate still to come) keeps the verdict pending,
	// so this is the staged-flush proof without a flaky wall-clock assertion.
	beatFrame := vt.AwaitFrame(t, frames, 30*time.Second, `data-beat="oracle-base"`)
	assert.Contains(t, beatFrame, `data-state="in-flight"`,
		"a beat reached the card while the verdict was still pending — the loop is streamed, not one snap")

	// Then the verdict resolves on a later frame.
	vt.AwaitFrame(t, frames, 30*time.Second, `data-state="catch"`)
}
