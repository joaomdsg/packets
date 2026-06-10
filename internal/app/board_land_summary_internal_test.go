package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// Per-row blocked spans (R58) tell a Lead WHICH sessions can't land, but scanning a
// large fleet to count them is work. A fleet-level merge-readiness summary answers
// "how much of the fleet is blocked from merging?" at a glance — a calm roll-up of
// the same honest per-session land verdicts, off the economy. NOT parallel (shared
// liveReg/liveFabric).
func TestBoardCard_summarisesHowMuchOfTheFleetIsBlockedFromLanding(t *testing.T) {
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
	lookupLiveEntry("blocked").setLand("conflict") // trunk moved — can't merge

	checksRed, err := AddSession("checksred", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = checksRed.Close() })
	lookupLiveEntry("checksred").setLand("checks_red") // checks fail rebased

	// The default session (created by NewServer) is pending — NOT blocked — so M counts
	// it but N does not: 2 of 3 sessions are blocked from landing.
	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "board__land-summary", "the fleet surfaces a merge-readiness roll-up")
	require.Contains(t, body, "2 of 3 sessions blocked from landing", "an honest count of blocked vs total sessions")
}

// The count is an honest projection of the actual fleet, not a fixed string: a
// single blocked session among a pending default reads "1 of 2", proving N and M
// track the real rows. NOT parallel (shared globals).
func TestBoardCard_landSummaryCountTracksTheRealFleet(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	one, err := AddSession("oneblocked", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = one.Close() })
	lookupLiveEntry("oneblocked").setLand("conflict")

	// Default (pending) + oneblocked (conflict) = 1 of 2 blocked.
	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "1 of 2 sessions blocked from landing", "the count tracks the real blocked/total, not a fixed string")
}

// When every session can land (clean or not-yet-resolved), the summary is silent —
// the board stays calm and a future block stands out, rather than nagging a fully
// mergeable fleet with a "0 blocked" meter. NOT parallel (shared globals).
func TestBoardCard_omitsLandSummaryWhenNothingIsBlocked(t *testing.T) {
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
	lookupLiveEntry("cleanS").setLand("clean") // integrates cleanly

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.NotContains(t, body, "board__land-summary", "a fully-mergeable fleet shows no summary — the board stays calm")
}
