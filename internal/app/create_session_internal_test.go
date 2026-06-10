package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// Sessions are created only at boot today — a Lead can't start a new economy from
// the UI. The fleet board must let the Lead CREATE a session and immediately work
// it (the in-process card flow needs no claim consumer), so the board becomes a
// command surface, not a static list. NOT parallel (shared liveReg/liveFabric).
func TestBoardCard_createSessionFromTheUIRegistersAReachableSession(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).WithSignal("newkey", "experiment").Fire(),
		"creating a session is a calm, valid action")

	// The created session appears on the fleet board…
	board := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, board, `data-key="experiment"`, "the created session appears as a board row")
	// …and is immediately reachable as its own card (the card flow needs no consumer).
	card := bodyOf(vt.NewClient(t, server, "/?key=experiment").HTML())
	require.Contains(t, card, "board-nav__key", "the created session renders as a card")
	require.Contains(t, card, "experiment", "the card breadcrumb names the created session")
}

// The board must render the create control itself — an input bound to the
// new-session signal and a button wired to the CreateSession action — else the
// Lead has no way to invoke it. NOT parallel (shared globals).
func TestBoardCard_rendersACreateSessionControl(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "/_action/CreateSession", "the board renders the create-session action binding")
	require.Contains(t, body, `data-bind="newkey"`, "with an input bound to the new-session key signal")
}

// A create must never forge an invalid subject token nor silently clobber a live
// economy: an invalid key and a duplicate key are both honest no-ops, not a new or
// overwritten session. NOT parallel (shared globals).
func TestBoardCard_createSessionRejectsInvalidAndDuplicateKeys(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/board")
	// An invalid subject token (whitespace) must not register anything.
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).WithSignal("newkey", "bad key").Fire())
	// A duplicate of the seeded default must be a no-op (never clobber a live log).
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).WithSignal("newkey", defaultSessionKey).Fire())

	board := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.NotContains(t, board, "bad key", "an invalid key never becomes a session")
	require.Contains(t, board, `data-key="default"`, "the duplicate no-op leaves the original default intact")
}
