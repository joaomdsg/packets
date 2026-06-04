package surface_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/surface"
)

func renderBalance(t *testing.T, balance int) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, surface.RenderBalance(balance).Render(&buf))
	return buf.String()
}

func TestRenderBalance_showsTheHeldBalanceAsItsOwnRow(t *testing.T) {
	t.Parallel()
	html := renderBalance(t, 3)
	assert.Contains(t, html, `data-state="balance"`, "the spendable balance is its own row")
	assert.Contains(t, html, `data-balance="3"`, "the row carries the balance as a stable marker the wire can assert on")
	for _, other := range []string{
		`data-state="stock"`, `data-state="catch"`, `data-state="tested"`,
		`data-state="land-clean"`, `data-state="beats"`, `data-state="in-flight"`,
	} {
		assert.NotContainsf(t, html, other, "the balance row must not collide with the %s row", other)
	}
}

func TestRenderBalance_zeroIsACalmEmptyNotAGuiltGauge(t *testing.T) {
	t.Parallel()
	html := renderBalance(t, 0)
	assert.Contains(t, html, `data-balance="0"`, "a spent-down balance reads as a calm zero")
	assert.NotContains(t, html, "%", "the balance is a held quantity, not a percentage gauge")
}
