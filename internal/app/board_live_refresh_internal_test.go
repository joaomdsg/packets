package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// The fleet board must LIVE-REFRESH: a Lead watching the board sees a session another
// tab creates appear without a manual reload. Today /board is a request-scoped GET
// with no SSE re-render, so the board goes stale the moment the fleet changes. A
// board client's SSE stream must pick up a session created by a DIFFERENT client.
// NOT parallel (shared liveReg).
func TestBoardCard_liveRefreshesWhenTheFleetChanges(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	watcher := vt.NewClient(t, server, "/board")
	frames, cancel := watcher.SSE()
	defer cancel()

	// A different client creates a session — the watcher never reloads, so only a
	// live SSE re-render can surface the new row.
	creator := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, creator.Action((&BoardCard{}).CreateSession).
		WithSignal("newkey", "liverefresh").WithSignal("newrepo", ".").Fire())

	frame := vt.AwaitFrame(t, frames, 10*time.Second, `data-key="liverefresh"`)
	require.Contains(t, frame, `data-key="liverefresh"`, "the board live-refreshes the new session onto the watcher's stream")
}

// Live-refresh is not just membership: a moving count on an existing session (here a
// new confirmed catch) must reach the watcher's board too — the cross-session "watch
// the shop" value. NOT parallel (shared liveReg).
func TestBoardCard_liveRefreshesWhenASessionsCountsMove(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	watcher := vt.NewClient(t, server, "/board")
	frames, cancel := watcher.SSE()
	defer cancel()

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))

	frame := vt.AwaitFrame(t, frames, 10*time.Second, "1 confirmed")
	require.Contains(t, frame, "1 confirmed", "a moving count live-refreshes onto the watcher's board")
}

// An idle board must NOT flood the stream: when the fleet is unchanged, the polling
// tick writes nothing, so no re-render frame is pushed (the fingerprint skip branch).
// NOT parallel (shared liveReg).
func TestBoardCard_idleBoardDoesNotFloodFrames(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	watcher := vt.NewClient(t, server, "/board")
	frames, cancel := watcher.SSE()
	defer cancel()

	// Span several poll ticks with no fleet change; only the SSE handshake may arrive,
	// never a board re-render frame.
	deadline := time.After(4 * boardRefreshInterval)
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				return // stream closed without a re-render — also fine
			}
			require.NotContains(t, f, "data-key", "an unchanged fleet must not push a re-render frame")
		case <-deadline:
			return
		}
	}
}
