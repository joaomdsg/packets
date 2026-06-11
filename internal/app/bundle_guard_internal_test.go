package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/time/rate"
)

// resetBundleGuardsForTest clears the per-producer guard registry so a test's
// rate/quota state does not leak into the next (the guards are package globals).
func resetBundleGuardsForTest() {
	bundleGuards.Range(func(k, _ any) bool { bundleGuards.Delete(k); return true })
	bundleAcctMu.Lock()
	bundleGlobalRetained = 0
	bundleAcctMu.Unlock()
}

// reserveOK is a test shorthand: the reservation succeeded.
func reserveOK(g *bundleGuard, n int64) bool { ok, _ := g.reserve(n); return ok }

func TestBundleGuard_reserveRefusesOverTheQuotaAndResetFrees(t *testing.T) {
	defer func(q int64) { bundleQuotaBytes = q }(bundleQuotaBytes)
	defer func(c int64) { bundleGlobalCeilingBytes = c }(bundleGlobalCeilingBytes)
	resetBundleGuardsForTest()
	bundleQuotaBytes, bundleGlobalCeilingBytes = 100, 1<<40
	g := &bundleGuard{lim: rate.NewLimiter(rate.Inf, 1)}

	require.True(t, reserveOK(g, 60), "first reservation within quota succeeds")
	require.False(t, reserveOK(g, 60), "a reservation that would exceed the quota is refused, reserving nothing")
	require.True(t, reserveOK(g, 40), "the refused reservation freed nothing, so the remaining 40 still fits")
	require.False(t, reserveOK(g, 1), "now at quota, even one more byte is refused")

	g.release(40)
	require.True(t, reserveOK(g, 40), "release returns bytes to the quota")

	resetBundleRetained("nope") // a missing key is a no-op
	// reset via the registry path: store g and reset its key.
	bundleGuards.Store("k", g)
	defer bundleGuards.Delete("k")
	resetBundleRetained("k")
	require.True(t, reserveOK(g, 100), "after a reset the full quota is available again")
}

// The global ceiling bounds the SUM of retained bytes across DISTINCT producers:
// a second producer well within its own quota is still refused (and flagged as a
// GLOBAL limit, → 503) when the aggregate would exceed the ceiling; a reset frees
// the aggregate. NOT parallel (mutates package globals).
func TestBundleGuard_globalCeilingBoundsTheSumAcrossProducers(t *testing.T) {
	defer func(q int64) { bundleQuotaBytes = q }(bundleQuotaBytes)
	defer func(c int64) { bundleGlobalCeilingBytes = c }(bundleGlobalCeilingBytes)
	resetBundleGuardsForTest()
	bundleQuotaBytes, bundleGlobalCeilingBytes = 1000, 100 // generous per-producer, tight global

	a, b := bundleGuardFor("prodA"), bundleGuardFor("prodB")
	okA, globalA := a.reserve(80)
	require.True(t, okA)
	require.False(t, globalA, "the first producer is within both limits")

	okB, globalB := b.reserve(80)
	require.False(t, okB, "the second producer's 80 bytes would push the aggregate (160) over the 100 ceiling")
	require.True(t, globalB, "the binding limit is the GLOBAL ceiling, not prodB's own quota — caller maps this to 503")

	resetBundleRetained("prodA") // GC frees prodA's bytes from the aggregate
	okB2, _ := b.reserve(80)
	require.True(t, okB2, "freeing prodA's retained bytes makes room for prodB under the ceiling")
}

// A producer that floods POST /bundle past its burst is throttled (429) rather
// than allowed to hammer the ingest path. NOT parallel (mutates package globals).
func TestPostBundle_throttlesAProducerPastItsBurst(t *testing.T) {
	defer func(b int) { bundleBurst = b }(bundleBurst)
	defer func(r float64) { bundleRatePerSec = r }(bundleRatePerSec)
	bundleBurst, bundleRatePerSec = 2, 0.0001 // tiny refill: the burst is all a flood gets in a test window
	resetBundleGuardsForTest()

	server, _ := bundleServer(t)
	bundle, _ := producerCommitBundle(t)

	require.Equal(t, http.StatusAccepted, postBundle(t, server.URL, "", "", bundle), "1st upload within burst")
	require.Equal(t, http.StatusAccepted, postBundle(t, server.URL, "", "", bundle), "2nd upload within burst")
	require.Equal(t, http.StatusTooManyRequests, postBundle(t, server.URL, "", "", bundle),
		"the 3rd rapid upload exceeds the burst and is rate-limited")
}

// A producer cannot retain more than its aggregate byte quota: an upload that
// would exceed it is refused (413) before the ingest work. NOT parallel.
func TestPostBundle_refusesAnUploadOverTheRetainedQuota(t *testing.T) {
	defer func(q int64) { bundleQuotaBytes = q }(bundleQuotaBytes)
	bundleQuotaBytes = 1 // any real bundle exceeds this
	resetBundleGuardsForTest()

	server, _ := bundleServer(t)
	bundle, _ := producerCommitBundle(t)

	require.Equal(t, http.StatusRequestEntityTooLarge, postBundle(t, server.URL, "", "", bundle),
		"an upload past the producer's retained-byte quota is refused before ingest")
	assert.Greater(t, len(bundle), 1, "sanity: the bundle really is larger than the tiny quota")
}

// When the host aggregate is at capacity, a bundle upload is refused with 503
// (host capacity), distinct from the per-producer 413. NOT parallel.
func TestPostBundle_refusesWhenTheGlobalCeilingIsReached(t *testing.T) {
	defer func(q int64) { bundleQuotaBytes = q }(bundleQuotaBytes)
	defer func(c int64) { bundleGlobalCeilingBytes = c }(bundleGlobalCeilingBytes)
	bundleQuotaBytes, bundleGlobalCeilingBytes = 1<<40, 1 // ample per-producer, host at capacity
	resetBundleGuardsForTest()

	server, _ := bundleServer(t)
	bundle, _ := producerCommitBundle(t)

	require.Equal(t, http.StatusServiceUnavailable, postBundle(t, server.URL, "", "", bundle),
		"a bundle that would exceed the global ceiling is refused as host-at-capacity (503), not a producer 413")
}
