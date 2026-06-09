package ledger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// base is a fixed instant; tests add durations to it so the rate limiter is
// exercised deterministically with an injected clock — never the wall clock.
var base = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// The bucket admits a burst of claims immediately, then refuses once drained:
// a producer can spend its burst at once but not exceed it in a single instant.
func TestTokenBucket_admitsTheBurstThenRefusesWhenDrained(t *testing.T) {
	t.Parallel()
	b := newTokenBucket(12, 0.1, base)
	for i := 0; i < 12; i++ {
		assert.Truef(t, b.allow(base), "claim %d within the burst must be admitted", i+1)
	}
	assert.False(t, b.allow(base), "the 13th claim at the same instant must be refused — the burst is spent")
}

// After draining, tokens refill at the configured rate over time: exactly the
// number of whole tokens elapsed are admitted, no more.
func TestTokenBucket_refillsExactlyTheElapsedTokens(t *testing.T) {
	t.Parallel()
	b := newTokenBucket(12, 0.1, base) // 0.1/s → 1 token per 10s
	for i := 0; i < 12; i++ {
		b.allow(base)
	}
	at := base.Add(30 * time.Second) // 30s × 0.1 = 3 tokens
	assert.True(t, b.allow(at), "1st refilled token")
	assert.True(t, b.allow(at), "2nd refilled token")
	assert.True(t, b.allow(at), "3rd refilled token")
	assert.False(t, b.allow(at), "only 3 tokens refilled in 30s at 0.1/s — the 4th is refused")
}

// A long idle does NOT accumulate unbounded credit: refill is capped at burst,
// so a producer can't bank an hour of silence into a giant flood.
func TestTokenBucket_capsRefillAtBurstAfterLongIdle(t *testing.T) {
	t.Parallel()
	b := newTokenBucket(12, 0.1, base)
	for i := 0; i < 12; i++ {
		b.allow(base)
	}
	at := base.Add(time.Hour) // would be 360 tokens uncapped; must cap at 12
	for i := 0; i < 12; i++ {
		assert.Truef(t, b.allow(at), "refilled token %d (up to the burst)", i+1)
	}
	assert.False(t, b.allow(at), "refill is capped at the burst (12), not the hour of elapsed credit")
}

// A partial token is not enough: refusal holds until a WHOLE token has accrued.
func TestTokenBucket_refusesUntilAWholeTokenHasAccrued(t *testing.T) {
	t.Parallel()
	b := newTokenBucket(12, 0.1, base) // 1 token per 10s
	for i := 0; i < 12; i++ {
		b.allow(base)
	}
	assert.False(t, b.allow(base.Add(5*time.Second)), "0.5 tokens (5s × 0.1) is not a whole token — refused")
	assert.True(t, b.allow(base.Add(10*time.Second)), "by 10s a whole token has accrued — admitted")
}

// Refills accrued in many small steps accumulate float error: ten 0.1-token
// increments sum to 0.9999999999999998, not 1.0. A producer that has genuinely
// waited a full token's worth of time, polled second by second, must STILL be
// admitted — a bare `>= 1` would perpetually starve it. This is exactly what the
// epsilon rescues; the test fails without it (verified by mutation).
func TestTokenBucket_admitsAWholeTokenEarnedDespiteAccumulatedFloatDrift(t *testing.T) {
	t.Parallel()
	b := newTokenBucket(1, 0.1, base) // one token per 10s
	assert.True(t, b.allow(base), "starts full")
	assert.False(t, b.allow(base), "drained")
	// Poll once per second: each call refills 0.1, and ten such increments sum to
	// 0.999…998 in float — short of an exact 1.0 by rounding alone.
	for s := 1; s <= 9; s++ {
		assert.Falsef(t, b.allow(base.Add(time.Duration(s)*time.Second)), "at %ds only a fraction has accrued", s)
	}
	assert.True(t, b.allow(base.Add(10*time.Second)), "by 10s a whole token is earned, even though the incremental sum lands at 0.999…998")
}

// A backwards clock must not corrupt the limiter: a now before the last seen time
// neither refills negatively nor rewinds the bucket's clock, and a later forward
// call still refills correctly from the real elapsed time.
func TestTokenBucket_isRobustToABackwardClock(t *testing.T) {
	t.Parallel()
	b := newTokenBucket(12, 0.1, base)
	for i := 0; i < 12; i++ {
		b.allow(base.Add(time.Minute)) // drain, last advances to base+1m
	}
	assert.False(t, b.allow(base), "a backward clock grants no spurious token and does not rewind last")
	assert.True(t, b.allow(base.Add(70*time.Second)), "10s of real forward progress from last (base+1m) accrues a token")
}
