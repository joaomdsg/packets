package app

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// Without the base stylesheet attached to the page <head>, the calm visual
// language never reaches the browser — the whole UX/UI direction is dead on
// arrival. Every rendered page must carry our stylesheet. NOT parallel (shared
// liveReg/liveFabric).
func TestNewServer_attachesTheBaseStylesheetToEveryPage(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	for _, path := range []string{"/", "/board"} {
		body := vt.NewClient(t, server, path).HTML()
		require.Containsf(t, body, "--pk-", "%s must carry OUR design tokens (--pk-*), proving it is the packets stylesheet", path)
		require.Containsf(t, body, ".board-row", "%s's stylesheet must target the real class hooks", path)
		// The <style> must live in the <head> (not stray into the body).
		headEnd := strings.Index(body, "</head>")
		stylePos := strings.Index(body, "<style")
		require.Greaterf(t, stylePos, -1, "%s must carry an inline <style>", path)
		require.Greaterf(t, headEnd, stylePos, "%s's <style> must be inside the <head>", path)
	}

	// Attaching the head must not disturb the body render — the board markup is
	// unchanged.
	board := vt.NewClient(t, server, "/board").HTML()
	require.Contains(t, board, "board-row__stock", "the board body still renders its rows after the head is attached")
	require.NotContains(t, strings.ToLower(board), "progress-bar", "no gauges/progress bars (calm guardrail)")
}

// Every honest verdict + land STATE the card can render must have a per-state
// style rule, so the Lead reads catch-vs-miss-vs-lost at a glance in the calm
// language — not as undifferentiated text. We pin the SELECTOR coverage (every
// real data-state value is targeted), never the colors (taste). If a renderer
// gains a new state, this test fails until the stylesheet styles it too.
func TestBaseStylesheet_stylesEveryVerdictAndLandState(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// The full page (the <style> lives in the head) — we WANT to find the
	// selectors in the stylesheet here, so check the whole HTML.
	page := vt.NewClient(t, server, "/").HTML()
	// Every data-state value the surface renderers emit (verdict + land).
	for _, state := range []string{
		"catch", "no-catch", "partial-catch", "no-oracle-signal",
		"lost-via-rename", "anchor-edited", "tested", "in-flight",
		"land-clean", "land-conflict", "land-checks-red", "land-pending",
	} {
		require.Containsf(t, page, `[data-state="`+state+`"]`,
			"the stylesheet must give state %q its own calm per-state rule (legible at a glance)", state)
	}
	// Calm guardrail: per-state color must use the --pk-* tokens, never a raw
	// alarm red/green hardcode.
	require.NotContains(t, strings.ToLower(page), "#ff0000", "no alarm red")
	require.NotContains(t, strings.ToLower(page), "#00ff00", "no alarm green")
}
