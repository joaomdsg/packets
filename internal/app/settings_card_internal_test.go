package app

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// A live order cannot run without an Anthropic API key, but today the key reaches
// the harness only as a host env var set before boot — a Lead working from the UI
// has no way to supply it. The settings surface must report the unconfigured state
// honestly AND render the control to fix it (an input bound to the token signal +
// the save action), else the Lead is stuck. NOT parallel (shared globals + env).
func TestSettingsCard_reportsUnconfiguredAndRendersTheControl(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "") // a stray ambient key must not mask the unconfigured store
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/settings").HTML())
	require.Contains(t, body, `data-state="unconfigured"`, "an empty store reports the unconfigured state")
	require.Contains(t, body, "/_action/SaveToken", "the settings card renders the save-token action binding")
	require.Contains(t, body, `data-bind="token"`, "with an input bound to the token signal")
}

// The setup surface is useless if a Lead can't reach it: the shared nav must carry
// a link to /settings from every page. NOT parallel (shared globals).
func TestNav_linksToTheSettingsSurface(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, `href="/settings"`, "the nav links to the setup surface from every page")
}

// Saving a token from the UI must persist it AND inject it into the host env so the
// harness the server spawns inherits it (RunProcess) or passes it through by name
// (RunContainer) — the whole point of the setup surface. After a save the card
// must report configured. NOT parallel (shared globals + env).
func TestSettingsCard_savingATokenPersistsItAndReportsConfigured(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/settings")
	require.Equal(t, 200, tc.Action((&SettingsCard{}).SaveToken).WithSignal("token", "sk-ant-fromui").Fire(),
		"saving a token is a calm, valid action")

	require.Equal(t, "sk-ant-fromui", os.Getenv("ANTHROPIC_API_KEY"),
		"the saved token is injected into the host env so the spawned harness inherits it")

	body := bodyOf(vt.NewClient(t, server, "/settings").HTML())
	require.Contains(t, body, `data-state="configured"`, "after a save the card reports the configured state")
}

// The token is the one secret the system holds: the settings card reports only its
// PRESENCE, never echoes its value back into the page (which would expose it in the
// DOM, history, and any logging proxy). NOT parallel (shared globals + env).
func TestSettingsCard_neverRendersTheTokenValue(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/settings")
	require.Equal(t, 200, tc.Action((&SettingsCard{}).SaveToken).WithSignal("token", "sk-ant-topsecret").Fire())

	body := vt.NewClient(t, server, "/settings").HTML()
	require.NotContains(t, body, "sk-ant-topsecret", "the stored token value never appears in the rendered page")
}
