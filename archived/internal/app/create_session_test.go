package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
)

// The create form's repo picker must be a SERVER-SIDE directory browser, not a
// browser file input: a browser file input can only ever hand back a folder NAME
// (webkitRelativePath is relative by design), never the absolute path the Lead
// actually wants stored. The server has filesystem access, so the form renders a
// Browse control wired to the OpenBrowser action instead. NOT parallel (shared
// globals).
func TestBoardCard_rendersAServerSideDirectoryBrowser(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "/_action/OpenBrowser", "the form renders a server-side Browse control")
	require.NotContains(t, body, "webkitdirectory", "no browser file input — it can't yield an absolute path")
}

// Sessions are created only at boot today — a Lead can't start a new economy from
// the UI. The fleet board must let the Lead CREATE a session and immediately work
// it (the in-process card flow needs no claim consumer), so the board becomes a
// command surface, not a static list. NOT parallel (shared liveReg/liveFabric).
func TestBoardCard_createSessionFromTheUIRegistersAReachableSession(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&app.BoardCard{}).CreateSession).WithSignal("newkey", "experiment").Fire(),
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
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, "/_action/CreateSession", "the board renders the create-session action binding")
	require.Contains(t, body, `data-bind="newkey"`, "with an input bound to the new-session key signal")
}

// A create must never forge an invalid subject token nor silently clobber a live
// economy: an invalid key and a duplicate key are both honest no-ops, not a new or
// overwritten session. NOT parallel (shared globals).
func TestBoardCard_createSessionRejectsInvalidAndDuplicateKeys(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	tc := vt.NewClient(t, server, "/board")
	// An invalid subject token (whitespace) must not register anything.
	require.Equal(t, 200, tc.Action((&app.BoardCard{}).CreateSession).WithSignal("newkey", "bad key").Fire())
	// A duplicate of the seeded default must be a no-op (never clobber a live log).
	require.Equal(t, 200, tc.Action((&app.BoardCard{}).CreateSession).WithSignal("newkey", defaultKey).Fire())

	board := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.NotContains(t, board, "bad key", "an invalid key never becomes a session")
	require.Contains(t, board, `data-key="default"`, "the duplicate no-op leaves the original default intact")
}
