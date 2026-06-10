package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// "Landed ≠ Merged": a session's catch can be confirmed while the fix can't yet
// integrate (trunk moved, or checks fail once rebased). The card shows that verdict,
// but a Lead scanning the fleet had no way to see which sessions are BLOCKED from
// merging. The board now surfaces a blocked integration verdict per session. NOT
// parallel (shared liveReg/liveFabric).
func TestBoardCard_surfacesBlockedIntegrationVerdicts(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	blocked, err := AddSession("blocked", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = blocked.Close() })
	e := lookupLiveEntry("blocked")
	require.NotNil(t, e)
	e.setLand("conflict") // trunk moved under the fix — it can't merge

	// A second session blocked a different way — checks fail once rebased onto tip.
	checksRed, err := AddSession("checksred", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = checksRed.Close() })
	lookupLiveEntry("checksred").setLand("checks_red")

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "board-row__land", "the board surfaces the blocked integration verdict")
	require.Contains(t, body, `data-state="land-conflict"`, "a rebase-needed conflict carries its honest-color state hook")
	require.Contains(t, body, `data-state="land-checks-red"`, "a checks-red block carries its own honest-color state hook")
	require.Contains(t, body, "merge blocked", "naming that the session can't land yet")
}

// A clean (or not-yet-resolved) integration verdict surfaces NO land span — the
// board stays calm and a blocked session stands out, rather than flagging every
// mergeable row. NOT parallel (shared globals).
func TestBoardCard_omitsLandWhenCleanOrPending(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	cleanS, err := AddSession("cleanS", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanS.Close() })
	lookupLiveEntry("cleanS").setLand("clean") // integrates cleanly — nothing to flag

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.NotContains(t, body, "board-row__land", "a clean/pending verdict flags nothing — the board stays calm")
}
