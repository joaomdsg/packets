package surface_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/surface"
)

func renderStock(t *testing.T, s ledger.Stock) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, surface.RenderStock(s).Render(&buf))
	return buf.String()
}

func TestRenderStock_emptyIsACalmZeroNotALiveGauge(t *testing.T) {
	t.Parallel()
	html := renderStock(t, ledger.Stock{})
	assert.Contains(t, html, `data-state="stock"`, "the stock is its own row")
	assert.Contains(t, html, "0", "an empty ledger reads as a calm zero")
	// Retrospective, never a live gauge: no meter/percentage/guilt affordance.
	assert.NotContains(t, html, `data-state="meter"`, "the stock is a retrospective tally, never a live meter")
	assert.NotContains(t, html, "%", "no percentage gauge — nothing for the Lead to feel guilty about")
}

func TestRenderStock_showsTheConfirmedCountAndTallies(t *testing.T) {
	t.Parallel()
	html := renderStock(t, ledger.Stock{Count: 2, ByReason: map[string]int{"catch": 2}, SelfFlagged: 1, WouldHaveShipped: 1})
	assert.Contains(t, html, "2 confirmed", "the held quantity of confirmed catches is shown")
	assert.Contains(t, html, `data-state="stock"`)
	assert.Contains(t, html, "catch: 2", "the per-reason tally is shown so the count is auditable by reason")
	assert.Contains(t, html, "self-flagged 1", "the self-flag tally (a mint-time fact) is shown")
	assert.Contains(t, html, "would-have-shipped 1", "the would-have-shipped tally (a mint-time fact) is shown")
}

func TestRenderStock_showsTheReinvestedShareSoCompoundingIsVisible(t *testing.T) {
	t.Parallel()
	// The reinvested share (catches a spend bought by dispatching distinct work) is
	// its own span, byte-disjoint from the flat count — so the Lead SEES that some
	// catches were born of spends (the reinvestment chain), not two equal bumps.
	html := renderStock(t, ledger.Stock{Count: 3, Reinvested: 2, ByReason: map[string]int{"catch": 3}})
	assert.Contains(t, html, "3 confirmed", "the total held quantity")
	assert.Contains(t, html, `class="stock__reinvested"`, "the reinvested share is its own span, distinct from the flat count")
	assert.Contains(t, html, "reinvested 2", "two of the three catches were minted by dispatched runs")
}

func TestRenderStock_zeroReinvestedRendersCalmlyWithoutAPhantomBadge(t *testing.T) {
	t.Parallel()
	// A connect-only run (no spends yet) must read calm: the reinvested span is
	// present at 0, never an error or a phantom "you should be reinvesting" nudge.
	html := renderStock(t, ledger.Stock{Count: 1, Reinvested: 0, ByReason: map[string]int{"catch": 1}})
	assert.Contains(t, html, "reinvested 0", "a connect-only stock shows a calm reinvested 0, no phantom badge")
}

func TestRenderStock_rowIsDisjointFromEveryVerdictAndLandAndBeatState(t *testing.T) {
	t.Parallel()
	html := renderStock(t, ledger.Stock{Count: 1, ByReason: map[string]int{"catch": 1}})
	for _, otherState := range []string{
		"catch", "no-catch", "tested", "no-oracle-signal", "lost-via-rename",
		"anchor-edited", "in-flight", "beats", "land-clean", "land-conflict", "land-checks-red",
	} {
		assert.NotContainsf(t, html, `data-state="`+otherState+`"`,
			"the stock row must not collide with the %s row — one row never speaks for another", otherState)
	}
	assert.Equal(t, 1, strings.Count(html, `data-state="stock"`), "the stock renders exactly one stock-row marker, never duplicated")
}
