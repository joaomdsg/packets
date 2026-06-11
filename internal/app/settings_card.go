package app

import (
	"os"
	"strings"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/tokenstore"
)

// tokenStore is the server's one Anthropic-key home, set by NewServer beside the
// ledger. The SettingsCard reads its presence and writes through it on save; a nil
// store (a server stood up without one) makes the card report unconfigured and a
// save a silent no-op rather than panic.
var tokenStore *tokenstore.Store

// tokenConfigPath places the key file beside the ledger (one server, one key),
// falling back to a process-local temp file when no ledger path is configured (the
// in-memory test/demo default) so a save still round-trips within the process.
func tokenConfigPath(ledgerPath string) string {
	if ledgerPath == "" {
		return "" // tokenstore.New tolerates this; Save/Load operate on the empty path's temp below
	}
	return ledgerPath + ".token"
}

// loadStoredTokenIntoEnv injects a previously-saved key into ANTHROPIC_API_KEY at
// boot so a server restart keeps the harness runnable WITHOUT a re-entry — but only
// when the env does not already carry one, so an explicitly-exported key (the
// pre-UI workflow) always wins over the stored copy.
func loadStoredTokenIntoEnv() {
	if tokenStore == nil || os.Getenv("ANTHROPIC_API_KEY") != "" {
		return
	}
	if tok, err := tokenStore.Load(); err == nil && tok != "" {
		_ = os.Setenv("ANTHROPIC_API_KEY", tok)
	}
}

// SettingsCard is the setup surface: it reports whether an Anthropic API key is
// configured (presence only — never the value) and carries the one control to set
// it. Saving persists the key owner-only AND injects it into the host env so the
// harness the server spawns inherits it (RunProcess) or passes it through by name
// (RunContainer).
type SettingsCard struct {
	// Token holds the key typed into the (password) input, read by SaveToken on
	// submit. It is write-only from the page's view: the value is never rendered
	// back, so it lives only for the duration of the action POST.
	Token via.SignalStr `via:"token"`
	// Saved is the broadcast trigger written after a successful save so the status
	// row re-renders configured over SSE; it carries no authoritative value (View
	// re-reads the store, the source of truth).
	Saved via.StateTabStr
}

// SaveToken persists the typed key and injects it into the host env. An empty
// submission is a silent no-op — it never clobbers a configured key with a blank,
// so an accidental empty save can't disarm a running setup.
func (c *SettingsCard) SaveToken(ctx *via.Ctx) {
	tok := strings.TrimSpace(c.Token.Read(ctx))
	if tok == "" || tokenStore == nil {
		return
	}
	if err := tokenStore.Save(tok); err != nil {
		return // a disk failure is a no-op to the Lead, never a half-set key
	}
	// The spawned harness inherits the server process env (RunProcess) / passes the
	// name through (RunContainer), so setting it here is what makes the key reach the
	// agent without a restart.
	_ = os.Setenv("ANTHROPIC_API_KEY", tok)
	c.Saved.Write(ctx, "configured")
}

// View renders the setup surface: a calm status line in the configured/unconfigured
// idiom plus a masked input + save button. The token value is NEVER emitted.
func (c *SettingsCard) View(_ *via.CtxR) h.H {
	configured := tokenStore != nil && tokenStore.Configured()
	state, status := "unconfigured", "No Anthropic API key configured — live orders cannot run yet."
	if configured {
		state, status = "configured", "Anthropic API key configured — live orders can run."
	}
	return h.Div(
		navHeader(defaultSessionKey),
		h.Div(
			h.Class("settings"),
			h.Role("main"),
			h.Attr("aria-label", "settings"),
			h.Span(h.Class("settings__status"), h.Data("state", state), h.Text(status)),
			h.Div(
				h.Class("settings__token"),
				h.Input(h.Type("password"), c.Token.Bind(),
					h.Class("settings__token-input"), h.Placeholder("sk-ant-…")),
				h.Button(on.Click(c.SaveToken), h.Class("settings__save"), h.Text("Save key")),
			),
		),
	)
}
