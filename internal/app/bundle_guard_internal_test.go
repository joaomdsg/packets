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
}

func TestBundleGuard_reserveRefusesOverTheQuotaAndResetFrees(t *testing.T) {
	defer func(q int64) { bundleQuotaBytes = q }(bundleQuotaBytes)
	bundleQuotaBytes = 100
	g := &bundleGuard{lim: rate.NewLimiter(rate.Inf, 1)}

	require.True(t, g.reserve(60), "first reservation within quota succeeds")
	require.False(t, g.reserve(60), "a reservation that would exceed the quota is refused, reserving nothing")
	require.True(t, g.reserve(40), "the refused reservation freed nothing, so the remaining 40 still fits")
	require.False(t, g.reserve(1), "now at quota, even one more byte is refused")

	g.release(40)
	require.True(t, g.reserve(40), "release returns bytes to the quota")

	resetBundleRetained("nope") // a missing key is a no-op
	// reset via the registry path: store g and reset its key.
	bundleGuards.Store("k", g)
	defer bundleGuards.Delete("k")
	resetBundleRetained("k")
	require.True(t, g.reserve(100), "after a reset the full quota is available again")
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
