package ledger

import (
	"math"
	"time"
)

// tokenBucketEpsilon absorbs IEEE-754 drift near whole-token boundaries: a refill
// that should land exactly on an integer (e.g. 10s × 0.1/s = 1 token) can compute
// to 0.9999999999999 for other rate/elapsed pairs, and must still admit. The
// epsilon is far below a fractional token, so a genuine partial token (e.g. 0.5)
// is still refused.
const tokenBucketEpsilon = 1e-9

// tokenBucket is a per-producer admission rate limiter: it starts full (burst
// tokens), refills at ratePerSec up to burst, and admits a claim by consuming one
// token. It is pure given an injected `now` — no wall clock — so admission is
// deterministic and unit-testable. The wiring (B3) holds one bucket per producer
// (session,instance) and ack-drops a claim when allow returns false.
//
// allow is NOT goroutine-safe (it read-modify-writes tokens/last): a bucket must
// be owned by a single consumer; never call allow concurrently on one bucket.
type tokenBucket struct {
	burst  float64
	rate   float64 // tokens per second
	tokens float64
	last   time.Time
}

func newTokenBucket(burst, ratePerSec float64, now time.Time) *tokenBucket {
	return &tokenBucket{burst: burst, rate: ratePerSec, tokens: burst, last: now}
}

// allow refills for the time elapsed since the last call (capped at burst), then
// consumes a token if one is available, returning whether the claim is admitted.
// A backward clock (now before last) refills nothing and does not rewind last, so
// it can neither grant spurious credit nor corrupt later refills.
func (b *tokenBucket) allow(now time.Time) bool {
	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		b.tokens = math.Min(b.burst, b.tokens+elapsed*b.rate)
		b.last = now
	}
	if b.tokens >= 1-tokenBucketEpsilon {
		b.tokens--
		return true
	}
	return false
}
