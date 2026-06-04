package surface_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/surface"
)

func renderDispatch(t *testing.T, n int) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, surface.RenderDispatch(n).Render(&buf))
	return buf.String()
}

func TestRenderDispatch_showsTheDispatchedTallyAsItsOwnRow(t *testing.T) {
	t.Parallel()
	html := renderDispatch(t, 1)
	assert.Contains(t, html, `data-state="dispatch"`, "the dispatched-work tally is its own row — the thing a spend buys")
	assert.Contains(t, html, `data-dispatch="1"`, "the row carries the count as a stable marker the wire can assert on")
	for _, other := range []string{
		`data-state="balance"`, `data-state="stock"`, `data-state="catch"`,
		`data-state="land-clean"`, `data-state="beats"`, `data-state="in-flight"`,
	} {
		assert.NotContainsf(t, html, other, "the dispatch row must not collide with the %s row — one row never speaks for another", other)
	}
}

func TestRenderDispatch_zeroIsACalmTallyNotEmptyChrome(t *testing.T) {
	t.Parallel()
	html := renderDispatch(t, 0)
	assert.Contains(t, html, `data-dispatch="0"`, "nothing dispatched yet reads as a calm zero, not an error or blank")
	assert.Contains(t, html, `data-state="dispatch"`, "the row is present at zero so the Lead sees the tally exists before the first spend")
}
