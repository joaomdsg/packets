package app

import (
	"sync"

	"golang.org/x/time/rate"
)

// Bundle flood-defense limits (council R85). The auth boundary (R81–R83) makes a
// bundle upload attributable to a producer (== session key), so the host can
// bound BOTH the upload RATE and the aggregate bytes a producer RETAINS. These
// are vars (not consts) so tests can shrink them deterministically.
var (
	// bundleRatePerSec + bundleBurst parameterize a per-producer token bucket on
	// POST /bundle: bundles are infrequent, so a slow refill with a small burst
	// still serves a legitimate producer while throttling a flood.
	bundleRatePerSec = 0.05 // ~one upload per 20s sustained
	bundleBurst      = 4
	// bundleQuotaBytes caps the bytes a single producer may have RETAINED at once
	// (deterministic — bytes accepted, never git's on-disk size). GC-by-resolved
	// (R84) frees it: when a producer's namespace is pruned, its retained count
	// resets to zero.
	bundleQuotaBytes int64 = 128 << 20 // 128 MiB retained per producer
)

// bundleGuard is one producer's flood-defense state: an upload-rate limiter and
// the bytes it currently retains (toward the quota). rate.Limiter is
// goroutine-safe; the retained counter is guarded by mu.
type bundleGuard struct {
	lim      *rate.Limiter
	mu       sync.Mutex
	retained int64
}

var bundleGuards sync.Map // session key → *bundleGuard

func bundleGuardFor(key string) *bundleGuard {
	if g, ok := bundleGuards.Load(key); ok {
		return g.(*bundleGuard)
	}
	g, _ := bundleGuards.LoadOrStore(key, &bundleGuard{
		lim: rate.NewLimiter(rate.Limit(bundleRatePerSec), bundleBurst),
	})
	return g.(*bundleGuard)
}

// allowUpload reports whether this producer may upload now (one token), throttling
// a flood without blocking a legitimate, paced producer.
func (g *bundleGuard) allowUpload() bool { return g.lim.Allow() }

// reserve adds n bytes to the producer's retained total iff that keeps it within
// the quota, returning false (reserving nothing) when it would exceed — so an
// over-quota upload is refused BEFORE the ingest work.
func (g *bundleGuard) reserve(n int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.retained+n > bundleQuotaBytes {
		return false
	}
	g.retained += n
	return true
}

// release returns n reserved bytes (an ingest that the reserve preceded failed),
// so a failed upload never permanently consumes quota.
func (g *bundleGuard) release(n int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.retained -= n; g.retained < 0 {
		g.retained = 0
	}
}

// resetBundleRetained zeroes a producer's retained count — called when its
// namespace is pruned (GC-by-resolved, R84), so reclaimed objects free quota.
func resetBundleRetained(key string) {
	if g, ok := bundleGuards.Load(key); ok {
		gg := g.(*bundleGuard)
		gg.mu.Lock()
		gg.retained = 0
		gg.mu.Unlock()
	}
}
