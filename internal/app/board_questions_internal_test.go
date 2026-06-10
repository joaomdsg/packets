package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/mutation"
)

// A session can pass its verdict green while the oracle left surviving mutants —
// test debt buried in that session's /review. The fleet board must surface the
// open-question COUNT per session so a Lead spots which sessions carry debt at a
// glance, across the fleet, without opening each one. NOT parallel (shared globals).
func TestBoardCard_showsPerSessionOpenQuestionCount(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	dbgLog, err := AddSession("withdebt", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = dbgLog.Close() })
	e := lookupLiveEntry("withdebt")
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{
		{File: "a.go", Line: 4, Outcome: mutation.Survived, Message: "mutated >= to >"},
		{File: "a.go", Line: 9, Outcome: mutation.Survived, Message: "mutated + to -"},
	})

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "board-row__questions", "the board surfaces a per-session open-question count")
	require.Contains(t, body, "2 open questions", "showing how much test debt that session carries")
	require.Contains(t, body, `href="/review?key=withdebt"`, "the count links into that session's /review surface")
}

// Sessions with no open questions (the oracle killed everything, or no cycle ran)
// carry no questions span — the board stays calm and a debt-carrying session stands
// out, rather than every clean row showing "0". NOT parallel (shared globals).
func TestBoardCard_omitsTheQuestionCountForCleanSessions(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// The default session has no cached findings → no questions span on the board.
	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.NotContains(t, body, "board-row__questions", "a clean fleet shows no question counts (gated off at 0)")
}
