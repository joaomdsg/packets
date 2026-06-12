package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// The session card is the live control-room: it re-renders over SSE whenever a
// catch confirms, a balance drains, or an order resolves. A sighted Lead sees that
// change; an assistive-tech user only learns of it if the live region is announced.
// The economy is wrapped in a main landmark marked aria-live="polite", so screen
// readers announce state changes without the user hunting for them — the VISION's
// "keyboard-native, accessible" north star, made real. The nav stays its OWN
// landmark (a sibling of main, not nested), so navigation and content are distinct
// regions. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_liveEconomyIsAnnouncedToAssistiveTech(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, `role="main"`, "the economy is the page's main landmark")
	require.Contains(t, body, `aria-live="polite"`, "live SSE state changes are announced to assistive tech")
	require.Contains(t, body, `aria-label="session economy"`, "the main region is named for screen-reader navigation")
	require.Contains(t, body, `aria-label="primary"`, "the nav is a named landmark, distinct from the content")
}

// The fleet board is the cross-session content surface. It exposes a named main
// landmark (distinct from its nav) so an assistive-tech user can jump straight to
// the fleet rather than tab through the chrome. It IS marked aria-live="polite" —
// the board now re-renders over SSE when the fleet changes (OnConnect), so announcing
// updates is honest, not a lie about liveness. NOT parallel (shared globals).
func TestBoardCard_fleetExposesANamedMainLandmark(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	require.Contains(t, body, `role="main"`, "the fleet is the page's main landmark")
	require.Contains(t, body, `aria-label="fleet board"`, "the main region is named for screen-reader navigation")
	require.Contains(t, body, `aria-label="primary"`, "the nav is a named landmark, distinct from the content")
	require.Contains(t, body, `aria-live="polite"`, "the board live-refreshes over SSE, so its main region announces updates")
}
