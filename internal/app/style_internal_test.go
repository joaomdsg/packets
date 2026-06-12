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

// DRY: .review-answer__submit reuses .pk-btn (which already owns the hairline
// border). The submit rule must NOT re-declare a full `border:` shorthand — it only
// reinforces the accent via `border-color`, letting .pk-btn own the border width/style.
func TestReviewSubmit_doesNotRedeclareTheFullBorderShorthand(t *testing.T) {
	require.Contains(t, packetsStyle, "border-color: var(--pk-accent)",
		".review-answer__submit reinforces the accent via border-color, not a full border shorthand")
	require.NotContains(t, packetsStyle, "border: 1px solid var(--pk-accent)",
		".review-answer__submit must not re-declare the hairline border that .pk-btn already owns")
}

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

// The system layer (PR1) promotes the implicit scale into named --pk-* tokens
// and one shared component layer every surface hooks. We pin the NEW token
// names + the canonical :focus-visible rule so the contract later surfaces
// depend on cannot silently disappear. NOT parallel (shared liveReg/liveFabric).
func TestBaseStylesheet_definesTheSystemLayerTokens(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	page := vt.NewClient(t, server, "/").HTML()

	for _, token := range []string{
		"--pk-radius:", "--pk-radius-sm:", "--pk-border:",
		"--pk-font-sm:", "--pk-font-xs:",
	} {
		require.Containsf(t, page, token,
			"the system layer must define the %q scale token on :root", token)
	}
	// The WCAG 2.4.7 fix: a real focus ring on the shared components.
	require.Contains(t, page, ":focus-visible",
		"the system layer must define the shared :focus-visible focus ring")
	require.Contains(t, page, "outline: 2px solid var(--pk-accent)",
		"the focus ring is the documented bronze accent outline")
}

// Each surface keeps its semantic class and ADDS one shared component class via
// multi-class. We pin both the component selectors in the stylesheet AND the
// multi-class hooks on the rendered surfaces, so the collapse cannot regress a
// surface back to its hand-rolled box. NOT parallel (shared liveReg/liveFabric).
func TestBaseStylesheet_extractsTheSharedComponentLayer(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

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
	for _, hook := range []string{
		"pk-card stock-row", "pk-card balance-row", "pk-card bandwidth-row",
	} {
		require.Containsf(t, page, hook,
			"the session card's %q row composes .pk-card", hook)
	}

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
