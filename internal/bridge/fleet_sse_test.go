package bridge_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/bridge"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestFleetHandler_ordersRowsByQueuedDescThenKeyAsc(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	alpha := ledger.Bind(f, "alpha", "i")
	beta := ledger.Bind(f, "beta", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	require.NoError(t, alpha.Append(sampleCatch())) // alpha: balance 1, queued 0
	require.NoError(t, beta.Append(sampleCatch()))  // beta: balance 1
	require.NoError(t, beta.AppendDispatch("d",
		ledger.Target{BaseRev: "b2", FixRev: "f2", TipRev: "f2", Path: "other.go", Line: 9},
		ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "adult.go", Line: 4},
	)) // beta: spends 1 → balance 0, one queued work-order

	// beta (queued 1) sorts before alpha (queued 0); the exact frame pins the
	// `data: ` framing, the row shape, the per-session values, and the ordering.
	awaitLine(t, lines, `data: [{"key":"beta","balance":0,"catches":1,"orders":1,"queued":1},{"key":"alpha","balance":1,"catches":1,"orders":0,"queued":0}]`)
}

func TestFleetHandler_breaksQueuedTiesBySessionKeyAscending(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	zeta := ledger.Bind(f, "zeta", "i")
	alpha := ledger.Bind(f, "alpha", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	lines := scanLines(resp.Body)

	// Equal queued (both 0): the order must fall back to key ascending, since the
	// in-process registration ordinal is not on the stream.
	require.NoError(t, zeta.Append(sampleCatch()))
	require.NoError(t, alpha.Append(sampleCatch()))

	awaitLine(t, lines, `data: [{"key":"alpha","balance":1,"catches":1,"orders":0,"queued":0},{"key":"zeta","balance":1,"catches":1,"orders":0,"queued":0}]`)
}

func TestFleetHandler_sendsEventStreamContentTypeAndActuallyStreams(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	alpha := ledger.Bind(f, "alpha", "i")
	srv := httptest.NewServer(bridge.FleetHandler(f))
	defer srv.Close()

	resp := connect(t, ctx, srv.URL)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

	// A stub that only sets headers but never streams must not pass.
	lines := scanLines(resp.Body)
	require.NoError(t, alpha.Append(sampleCatch()))
	awaitLine(t, lines, `"key":"alpha"`)
}
