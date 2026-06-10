package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// R53 lets a Lead create sessions at runtime, so experiment sessions accumulate on
// the fleet board with no way to clear them — the board becomes cluttered. Retiring
// a session removes it from the fleet view, the honest completion of the create
// affordance. NOT parallel (shared liveReg/liveFabric).
func TestBoardCard_retireRemovesASessionFromTheFleet(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	expLog, err := AddSession("experiment", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = expLog.Close() })

	require.Contains(t, bodyOf(vt.NewClient(t, server, "/board").HTML()), `data-key="experiment"`,
		"the created session is on the board before retiring")

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).RetireSession).WithSignal("retirekey", "experiment").Fire())

	require.NotContains(t, bodyOf(vt.NewClient(t, server, "/board").HTML()), `data-key="experiment"`,
		"the retired session is gone from the fleet view")
}

// The seeded default session is the single-card fallback — retiring it would strand
// the "/" route. A retire of the default key must be a no-op. NOT parallel.
func TestBoardCard_retireRefusesTheDefaultSession(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).RetireSession).WithSignal("retirekey", defaultSessionKey).Fire())

	require.Contains(t, bodyOf(vt.NewClient(t, server, "/board").HTML()), `data-key="default"`,
		"the default session is never retired — it is the single-card fallback")
}

// Each NON-default row must render a retire control wired to that row's key; the
// default row must NOT (it is not retirable). NOT parallel.
func TestBoardCard_rendersARetireControlOnlyForNonDefaultRows(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// With only the default session, there is nothing retirable → no retire control.
	require.NotContains(t, bodyOf(vt.NewClient(t, server, "/board").HTML()), "/_action/RetireSession",
		"the default-only board offers no retire control (default is not retirable)")

	expLog, err := AddSession("experiment", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = expLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "/_action/RetireSession", "the non-default row renders a retire control")
	require.Contains(t, body, "experiment", "the retire control targets that row's key (set before the post)")
}
