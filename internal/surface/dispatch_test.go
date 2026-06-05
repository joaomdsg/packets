package surface_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/surface"
)

func renderDispatch(t *testing.T, c ledger.DispatchCounts) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, surface.RenderDispatch(c).Render(&buf))
	return buf.String()
}

func TestRenderDispatch_showsTheTallyMovingAcrossQueuedRunningDone(t *testing.T) {
	t.Parallel()
	html := renderDispatch(t, ledger.DispatchCounts{Queued: 2, Running: 1, Done: 3})
	assert.Contains(t, html, `data-state="dispatch"`, "the dispatched-work tally is its own row")
	assert.Contains(t, html, `data-dispatch-queued="2"`, "queued is its own stable marker so the tally can MOVE, not just rise")
	assert.Contains(t, html, `data-dispatch-running="1"`, "running is visible — the Lead sees work in flight")
	assert.Contains(t, html, `data-dispatch-done="3"`, "done is visible — the spend-to-earn payoff")
	for _, other := range []string{
		`data-state="balance"`, `data-state="stock"`, `data-state="catch"`,
		`data-state="land-clean"`, `data-state="beats"`, `data-state="in-flight"`,
	} {
		assert.NotContainsf(t, html, other, "the dispatch row must not collide with the %s row", other)
	}
}

func TestRenderDispatch_zeroIsACalmTallyNotEmptyChrome(t *testing.T) {
	t.Parallel()
	html := renderDispatch(t, ledger.DispatchCounts{})
	assert.Contains(t, html, `data-dispatch-queued="0"`, "nothing dispatched yet reads as a calm zero, not an error or blank")
	assert.Contains(t, html, `data-dispatch-running="0"`)
	assert.Contains(t, html, `data-dispatch-done="0"`)
	assert.Contains(t, html, `data-state="dispatch"`, "the row is present at zero so the tally exists before the first spend")
}
