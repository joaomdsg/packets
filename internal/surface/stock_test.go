package surface_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/surface"
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
