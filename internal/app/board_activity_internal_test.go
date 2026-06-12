package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// The fleet board should show what each agent is DOING right now (its live activity
// beat), not just its economy counts — the cross-session "watch the shop" ticker.
// The beat is already captured per session (activitySnapshot); the board surfaces
// it. NOT parallel (shared liveReg).
func TestBoardRows_carryEachSessionsLiveActivityBeat(t *testing.T) {
	boardSession(t, "act-rows", 0, nil)
	e := lookupLiveEntry("act-rows")
	require.NotNil(t, e)
	e.addActivityBeat("editing auth.go")

	var got string
	for _, r := range BoardRows() {
		if r.Key == "act-rows" {
			got = r.Activity
		}
	}
	require.Equal(t, "editing auth.go", got, "the board row carries the session's live activity beat")
}

func TestBoardCard_showsAnAgentsLiveActivityOnItsRow(t *testing.T) {
	// NOT parallel (shared liveReg).
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	boardSession(t, "act-render", 0, nil)
	lookupLiveEntry("act-render").addActivityBeat("running go test")

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "board-row__activity-beat", "a live activity beat renders as its own hook on the row")
	require.Contains(t, body, "running go test", "the board shows what the agent is doing right now")
}

func TestBoardCard_omitsTheActivityBeatWhenTheAgentIsIdle(t *testing.T) {
	// NOT parallel (shared liveReg). An idle session (no live fill) shows no beat —
	// no dead "·" with nothing after it.
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	boardSession(t, "act-idle", 0, nil) // registered, never given an activity beat

	rows := BoardRows()
	for _, r := range rows {
		if r.Key == "act-idle" {
			require.Empty(t, r.Activity, "an idle session carries no activity beat")
		}
	}
}
