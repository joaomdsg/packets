package app

import (
	"sync"

	"golang.org/x/time/rate"
)

// Bundle flood-defense limits (council R85/R86). The auth boundary (R81–R83)
// makes a bundle upload attributable to a producer (== session key), so the host
// can bound the upload RATE, the bytes a single producer RETAINS, AND the
// aggregate bytes retained across ALL producers. These are vars (not consts) so
// tests can shrink them deterministically.
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
	// bundleGlobalCeilingBytes caps the SUM of retained bytes across all producers
	// (R86): the per-producer quota bounds one flooder, this bounds many producers
	// collectively filling the host store. An upload that would push the aggregate
	// over the ceiling is refused even when the producer is within its own quota.
	bundleGlobalCeilingBytes int64 = 1 << 30 // 1 GiB across all producers
)

// bundleAcctMu serializes ALL retained-byte accounting (each guard's retained and
// the global aggregate), so the per-producer quota and the global ceiling stay
// mutually consistent. Uploads are infrequent, so one accounting mutex is ample
// and avoids any per-guard/global lock-ordering hazard.
var (
	bundleAcctMu        sync.Mutex
	bundleGlobalRetained int64
)

// bundleGuard is one producer's flood-defense state: an upload-rate limiter
// (rate.Limiter is goroutine-safe on its own) and its retained-byte count, which
// is accounted under bundleAcctMu alongside the global aggregate.
type bundleGuard struct {
	lim      *rate.Limiter
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
// BOTH its own quota AND the global ceiling, reserving nothing when either would
// be exceeded — so an over-limit upload is refused BEFORE the ingest work. The
// second return value is true when the GLOBAL ceiling (not the per-producer
// quota) was the binding limit, so the caller can distinguish 503 from 413.
func (g *bundleGuard) reserve(n int64) (ok, globalLimited bool) {
	bundleAcctMu.Lock()
	defer bundleAcctMu.Unlock()
	if g.retained+n > bundleQuotaBytes {
		return false, false
	}
	if bundleGlobalRetained+n > bundleGlobalCeilingBytes {
		return false, true
	}
	g.retained += n
	bundleGlobalRetained += n
	return true, false
}

// release returns n reserved bytes (an ingest that the reserve preceded failed),
// so a failed upload never permanently consumes quota or the global ceiling.
func (g *bundleGuard) release(n int64) {
	bundleAcctMu.Lock()
	defer bundleAcctMu.Unlock()
	subRetained(g, n)
}

// resetBundleRetained zeroes a producer's retained count and removes it from the
// global aggregate — called when its namespace is pruned (GC-by-resolved, R84),
// so reclaimed objects free both the per-producer quota and the global ceiling.
func resetBundleRetained(key string) {
	if g, ok := bundleGuards.Load(key); ok {
		gg := g.(*bundleGuard)
		bundleAcctMu.Lock()
		subRetained(gg, gg.retained)
		bundleAcctMu.Unlock()
	}
}

// subRetained drops n from both g.retained and the global aggregate, clamping at
// zero. Callers hold bundleAcctMu.
func subRetained(g *bundleGuard, n int64) {
	if g.retained -= n; g.retained < 0 {
		g.retained = 0
	}
	if bundleGlobalRetained -= n; bundleGlobalRetained < 0 {
		bundleGlobalRetained = 0
	}
}
