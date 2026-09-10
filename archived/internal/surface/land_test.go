package surface_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/surface"
)

func renderLand(t *testing.T, land pipe.LandState) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, surface.RenderLand(land).Render(&buf))
	return buf.String()
}

func TestRenderLand_cleanCarriesNoActionableChrome(t *testing.T) {
	t.Parallel()
	html := renderLand(t, pipe.LandClean)
	assert.Contains(t, html, `data-state="land-clean"`)
	assert.NotContains(t, strings.ToLower(html), "rebase", "a clean integration has nothing for the reviewer to act on")
	assert.NotContains(t, strings.ToLower(html), "conflict")
}

func TestRenderLand_conflictIsActionableWithoutClaimingClean(t *testing.T) {
	t.Parallel()
	html := renderLand(t, pipe.LandConflict)
	assert.Contains(t, html, `data-state="land-conflict"`)
	assert.Contains(t, strings.ToLower(html), "rebase", "a conflict tells the reviewer trunk moved and a rebase is needed")
	assert.NotContains(t, html, `data-state="land-clean"`, "a conflict must never render as clean")
}

func TestRenderLand_checksRedNamesThePostIntegrationRegression(t *testing.T) {
	t.Parallel()
	html := renderLand(t, pipe.LandChecksRed)
	assert.Contains(t, html, `data-state="land-checks-red"`)
	assert.Contains(t, strings.ToLower(html), "trunk", "checks-red explains the fix is green pre-integration but red on trunk tip")
	assert.NotContains(t, html, `data-state="land-clean"`)
}

func TestRenderLand_pendingBeforeIntegrationMakesNoClaim(t *testing.T) {
	t.Parallel()
	html := renderLand(t, pipe.LandState("")) // the live card's land row before the cycle resolves
	assert.NotContains(t, html, `data-state="land-clean"`, "an unresolved integration must not claim it integrates cleanly")
	assert.NotContains(t, html, `data-state="land-conflict"`)
	assert.NotContains(t, strings.ToLower(html), "rebase")
}

func TestRenderLand_eachStateRendersADistinctMarkerDisjointFromVerdicts(t *testing.T) {
	t.Parallel()
	markers := map[pipe.LandState]string{
		pipe.LandClean:     `data-state="land-clean"`,
		pipe.LandConflict:  `data-state="land-conflict"`,
		pipe.LandChecksRed: `data-state="land-checks-red"`,
	}
	seen := map[string]bool{}
	for land, marker := range markers {
		html := renderLand(t, land)
		assert.Containsf(t, html, marker, "%s must render %s", land, marker)
		seen[marker] = true

		for _, verdictState := range []string{"catch", "no-catch", "tested", "no-oracle-signal", "lost-via-rename", "anchor-edited", "in-flight"} {
			assert.NotContainsf(t, html, `data-state="`+verdictState+`"`,
				"the land row %s must not collide with the oracle verdict state %s", land, verdictState)
		}
	}
	assert.Len(t, seen, 3, "the three land states render three distinct rows")
}
