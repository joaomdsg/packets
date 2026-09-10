package surface_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/surface"
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

// The raw Kind values ("oracle-base", "land", …) are fine as the data-beat
// attribute (an internal hook, asserted separately as its own row)
// but must never leak into the VISIBLE text — "oracle" and "land" are retired
// from every surface, and a beat row streaming live is real dynamic content a
// static fixture sweep never exercises.
func TestRenderBeats_visibleTextNeverLeaksRetiredVocabulary(t *testing.T) {
	t.Parallel()
	html := renderBeats(t, "settle-base,oracle-base,settle-fix,oracle-fix,catch,land")
	// The data-beat hooks keep the raw kinds — that's the internal identifier,
	// never rendered prose.
	for _, kind := range []string{"settle-base", "oracle-base", "settle-fix", "oracle-fix", "catch", "land"} {
		assert.Contains(t, html, `data-beat="`+kind+`"`)
	}
	// Strip every tag so only the visible text nodes remain, then confirm no
	// retired word appears in THAT — matching how a screen reader (or a human)
	// actually experiences the row.
	visible := stripTagsForTest(html)
	lower := strings.ToLower(visible)
	assert.NotContains(t, lower, "oracle", "the visible beat text must never render the retired word \"oracle\"")
	assert.NotContains(t, lower, "land", "the visible beat text must never render the retired word \"land\"")
	// The fix must REPLACE the retired words with real vocabulary-clean labels,
	// not just delete text — a beat row that silently drops labels to dodge the
	// banned-word check would be a worse regression than the leak itself.
	assert.Contains(t, visible, "settled base")
	assert.Contains(t, visible, "settled fix")
	assert.Contains(t, visible, "catch", "\"catch\" carries no retired word and must survive unchanged")
	assert.Contains(t, lower, "gate", "the retired \"oracle\" beats still name what actually ran — the gate — never a blank label")
	assert.Contains(t, lower, "forward", "the retired \"land\" beat still names the real outcome — forwarding — never a blank label")
}

func stripTagsForTest(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}
