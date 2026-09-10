package app_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"
)

// Without the base stylesheet attached to the page <head>, the calm visual
// language never reaches the browser — the whole UX/UI direction is dead on
// arrival. Every rendered page must carry our stylesheet. NOT parallel (shared
// liveReg/liveFabric).
func TestNewServer_attachesTheBaseStylesheetToEveryPage(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	for _, path := range []string{"/", "/board"} {
		body := vt.NewClient(t, server, path).HTML()
		require.Containsf(t, body, "--signal:", "%s must carry OUR design tokens (the state grammar), proving it is the packets stylesheet", path)
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
	server, _ := bootDefaultServer(t, defaultBootCfg)

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
	// Calm guardrail: per-state color must use the design tokens, never a raw
	// alarm red/green hardcode.
	require.NotContains(t, strings.ToLower(page), "#ff0000", "no alarm red")
	require.NotContains(t, strings.ToLower(page), "#00ff00", "no alarm green")
}

// The system layer promotes the brand-pack scale into named tokens (the
// brand-pack token port) and one shared component layer every surface hooks. We pin the
// token names + the canonical :focus-visible rule so the contract later
// surfaces depend on cannot silently disappear. NOT parallel (shared
// liveReg/liveFabric).
func TestBaseStylesheet_definesTheSystemLayerTokens(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	page := vt.NewClient(t, server, "/").HTML()

	for _, token := range []string{
		"--r-btn:", "--r-glyph:", "--hairline:",
		"--fs-small:", "--fs-micro:",
	} {
		require.Containsf(t, page, token,
			"the system layer must define the %q scale token on :root", token)
	}
	// The WCAG 2.4.7 fix: a real focus ring on the shared components.
	require.Contains(t, page, ":focus-visible",
		"the system layer must define the shared :focus-visible focus ring")
	require.Contains(t, page, "outline: 2px solid var(--signal)",
		"the focus ring is the documented signal-cyan accent outline")
}

// Each surface keeps its semantic class and ADDS one shared component class via
// multi-class. We pin both the component selectors in the stylesheet AND the
// multi-class hooks on the rendered surfaces, so the collapse cannot regress a
// surface back to its hand-rolled box. NOT parallel (shared liveReg/liveFabric).
func TestBaseStylesheet_extractsTheSharedComponentLayer(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	page := vt.NewClient(t, server, "/").HTML()
	board := vt.NewClient(t, server, "/board").HTML()

	for _, sel := range []string{
		".pk-btn", ".pk-btn--quiet", ".pk-input", ".pk-chip", ".pk-section-label",
		".pk-card",
	} {
		require.Containsf(t, page, sel,
			"the system layer must define the shared %q component selector", sel)
	}

	// The padded box rows compose the shared .pk-card; the semantic class keeps
	// only hue/state/layout. We pin the multi-class hooks so the collapse cannot
	// regress a row back to its hand-rolled box.
	require.Contains(t, board, `class="pk-card board-row"`,
		"each fleet row composes .pk-card")

	// The board's create input + button compose the shared classes.
	require.Contains(t, board, `class="pk-input board-create__key"`,
		"the create-key input composes .pk-input")
	require.Contains(t, board, "pk-btn",
		"the board surfaces a .pk-btn control")
	// The retire control is a quiet variant.
	require.Contains(t, board, "pk-btn--quiet",
		"the board's retire control composes .pk-btn--quiet")
	// The uppercase labels are the shared section-label component.
	require.Contains(t, board, "pk-section-label",
		"the board's uppercase labels compose .pk-section-label")
}
