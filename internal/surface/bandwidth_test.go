package surface_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/surface"
)

func renderBandwidth(t *testing.T, bw int) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, surface.RenderBandwidth(bw).Render(&buf))
	return buf.String()
}

func TestRenderBandwidth_showsEarnedAttentionAsItsOwnRow(t *testing.T) {
	t.Parallel()
	html := renderBandwidth(t, 5)

	assert.Contains(t, html, `data-state="bandwidth"`, "the bandwidth meter is its own row, distinct from balance")
	assert.Contains(t, html, `data-bandwidth="5"`, "it carries the earned bandwidth as a stable marker")
	assert.NotContains(t, html, `data-state="balance"`, "the bandwidth row must not collide with the balance row")
}

func TestRenderBandwidth_readsCalmAtZero(t *testing.T) {
	t.Parallel()
	html := renderBandwidth(t, 0)

	assert.Contains(t, html, `data-bandwidth="0"`, "a zero meter is a calm held quantity, not an alarm")
	assert.NotContains(t, html, "%", "bandwidth is a held quantity, not a percentage gauge")
}
