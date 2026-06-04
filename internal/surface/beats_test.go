package surface_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/surface"
)

func renderBeats(t *testing.T, beats string) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, surface.RenderBeats(beats).Render(&buf))
	return buf.String()
}

func TestRenderBeats_listsStreamedKindsAsItsOwnRow(t *testing.T) {
	t.Parallel()
	html := renderBeats(t, "settle-base,oracle-base,catch")
	assert.Contains(t, html, `data-state="beats"`, "the streamed beats are their own row")
	assert.Contains(t, html, `data-beat="oracle-base"`, "each streamed beat gets its own marker, unambiguous from any verdict text")
	assert.Contains(t, html, `data-beat="settle-base"`)
	for _, verdictState := range []string{`data-state="catch"`, `data-state="land-clean"`, `data-state="in-flight"`, `data-state="tested"`} {
		assert.NotContainsf(t, html, verdictState, "the beat row must not collide with the %s verdict/land state — one row never speaks for another", verdictState)
	}
}

func TestRenderBeats_emptyBeforeAnyBeatShowsNoTempo(t *testing.T) {
	t.Parallel()
	html := renderBeats(t, "")
	assert.NotContains(t, strings.ToLower(html), "oracle", "no beats have streamed yet → the row shows no tempo")
	assert.NotContains(t, html, "settle-base")
}
