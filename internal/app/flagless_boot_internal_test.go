package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// The server must boot WITHOUT a configured primary anchor (flag-less launch): no
// default session is registered, the fleet board + settings still serve, and "/" is
// a calm landing pointing at the board — never a phantom Oracle-running card with
// nothing to run. NOT parallel (shared liveReg).
func TestNewServer_bootsWithoutADefaultSessionWhenUnconfigured(t *testing.T) {
	resetConsumersForTest() // clear any default registered by a prior test
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{LedgerPath: defLogPath}, via.WithTestServer(&server))
	require.NoError(t, err, "an unconfigured (flag-less) boot is valid")
	t.Cleanup(func() { _ = log.Close() })

	assert.Nil(t, lookupLiveEntry(defaultSessionKey), "no default session is registered when unconfigured")

	home := bodyOf(vt.NewClient(t, server, "/").HTML())
	assert.Contains(t, home, "No session configured", "the / card is a calm no-session landing")
	assert.NotContains(t, home, "Oracle running", "no phantom catch-cycle card when there is no session")
	assert.Contains(t, home, `href="/board"`, "the landing points to the fleet board")

	board := bodyOf(vt.NewClient(t, server, "/board").HTML())
	assert.Contains(t, board, "Create session", "the board still serves on a flag-less boot")
}

// A repo-only boot (a repo but no primary anchor) is a USABLE session: the Lead can
// author prompt orders against the repo and let the harness fill them. It must
// register the default session and render the working card — NOT the "No session
// configured" landing, and NOT a phantom Oracle-running catch-cycle spinner (there
// is no anchor to run a cycle on). NOT parallel (shared liveReg).
func TestNewServer_repoOnlySessionIsUsableNotALanding(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err, "a repo-only boot is valid")
	t.Cleanup(func() { _ = log.Close() })

	assert.NotNil(t, lookupLiveEntry(defaultSessionKey), "a repo-only boot registers a usable default session")

	home := bodyOf(vt.NewClient(t, server, "/").HTML())
	assert.NotContains(t, home, "No session configured", "a repo-only session renders the working card, not the landing")
	assert.Contains(t, home, "session economy", "the working card's economy region renders")
	assert.NotContains(t, home, "Oracle running", "no phantom catch-cycle spinner without an anchor")
}

// A boot WITH a primary anchor still registers the default session and renders the
// economy card (no regression). NOT parallel (shared liveReg).
func TestNewServer_stillRegistersTheDefaultWhenConfigured(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	assert.NotNil(t, lookupLiveEntry(defaultSessionKey), "a configured boot registers the default session")
	home := bodyOf(vt.NewClient(t, server, "/").HTML())
	assert.NotContains(t, home, "No session configured", "a configured session renders the economy card, not the landing")
}
