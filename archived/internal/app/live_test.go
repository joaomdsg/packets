package app_test

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
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
	viaApp, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: fix, TipRev: fix, Anchor: anchor(),
		TestCmd: goTestCmd, LedgerPath: logPath, SelfFlagged: true, WouldHaveShipped: true,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	require.Contains(t, tc.HTML(), `data-state="in-flight"`, "the card starts in-flight before the cycle resolves")

	frames, cancel := tc.SSE()
	defer cancel()
	// The integration row streams in the SAME resolved render as the catch
	// verdict — both needles in ONE AwaitFrame call, since it accumulates
	// chunks until every needle matches. Two SEPARATE calls are unsafe here:
	// vt frames are fixed-size read chunks, not SSE event boundaries, so
	// growing the console card can shift BOTH needles into the same chunk —
	// the first call would match on "catch" and return, discarding the rest
	// of that already-consumed chunk (and "land-clean" along with it) before
	// the second call ever sees it.
	vt.AwaitFrame(t, frames, 20*time.Second, `data-state="catch"`, `data-state="land-clean"`)

	records, err := log.Records()
	require.NoError(t, err)
	require.Len(t, records, 1, "the watched catch is durably logged")
	assert.Equal(t, catch.Catch, records[0].Outcome)
	assert.True(t, records[0].SelfFlagged)
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
	viaApp, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: fix, TipRev: fix, Anchor: anchor(),
		TestCmd: goTestCmd, LedgerPath: logPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
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
