package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/packet"
)

// A calibration draw must never invent a sample when
// nothing has forwarded on its own — the dashed empty state stays honest.
func TestDrawCalibration_hasNothingToDrawFromAnEmptyAutoForwardedSet(t *testing.T) {
	t.Parallel()
	packets := []packet.Packet{{ID: 1, State: packet.Held}, {ID: 2, State: packet.InFlight}}

	id, ok := drawCalibration(packets, 0)

	assert.False(t, ok, "no Verified packet exists — nothing honest to draw")
	assert.Equal(t, 0, id)
}

// Attention economics needs a STABLE sample, not a flicker: re-rendering
// (the 100ms poll) must never swap the sample while it still qualifies.
func TestDrawCalibration_keepsTheSameDrawStableAcrossRendersWhilePreviousStillQualifies(t *testing.T) {
	t.Parallel()
	packets := []packet.Packet{
		{ID: 1, State: packet.Verified},
		{ID: 2, State: packet.Verified},
		{ID: 3, State: packet.Verified},
	}

	for i := 0; i < 20; i++ {
		id, ok := drawCalibration(packets, 2)
		assert.True(t, ok)
		assert.Equal(t, 2, id, "the previous draw stays fixed while it is still in the auto-forwarded set")
	}
}

// The stability rule must hold the CACHED draw specifically — not merely
// "the first candidate" — so a previous draw that isn't first in the slice
// still wins over a fresh random pick.
func TestDrawCalibration_keepsANonFirstPreviousDrawStable(t *testing.T) {
	t.Parallel()
	packets := []packet.Packet{
		{ID: 1, State: packet.Verified},
		{ID: 2, State: packet.Verified},
		{ID: 3, State: packet.Verified},
	}

	for i := 0; i < 20; i++ {
		id, ok := drawCalibration(packets, 3)
		assert.True(t, ok)
		assert.Equal(t, 3, id, "the cached draw wins even when it is not the first qualifying candidate")
	}
}

// A packet that WAS Verified and cached as the draw can later be demoted
// (e.g. a new hold surfaces) — its id still exists in the full packets slice
// but no longer qualifies, so it must be treated as aged-out exactly like an
// id that vanished entirely, never kept just because it's still present.
func TestDrawCalibration_treatsADemotedPreviousDrawAsAgedOut(t *testing.T) {
	t.Parallel()
	packets := []packet.Packet{
		{ID: 1, State: packet.Held}, // previously verified, now held
		{ID: 2, State: packet.Verified},
	}

	id, ok := drawCalibration(packets, 1)

	assert.True(t, ok)
	assert.Equal(t, 2, id, "a demoted previous draw no longer qualifies — the sample must come from what IS currently verified")
}

// A nil packets slice (no session data at all, distinct from a populated but
// all-non-Verified slice) must be handled exactly like the empty case.
func TestDrawCalibration_hasNothingToDrawFromANilPacketsSlice(t *testing.T) {
	t.Parallel()
	id, ok := drawCalibration(nil, 0)

	assert.False(t, ok)
	assert.Equal(t, 0, id)
}

// Once the cached draw ages out of the auto-forwarded set (its packet no
// longer qualifies), a fresh sample must come from what CURRENTLY qualifies.
func TestDrawCalibration_redrawsFromTheQualifyingSetWhenThePreviousDrawAgesOut(t *testing.T) {
	t.Parallel()
	packets := []packet.Packet{
		{ID: 1, State: packet.Verified},
		{ID: 2, State: packet.Verified},
	}

	id, ok := drawCalibration(packets, 99) // 99 never existed in the set

	assert.True(t, ok)
	assert.Contains(t, []int{1, 2}, id, "a fresh draw must come from the CURRENT auto-forwarded set")
}

// With exactly one candidate the draw has only one honest answer — no
// randomness should be able to produce anything else.
func TestDrawCalibration_isDeterministicWithASingleCandidate(t *testing.T) {
	t.Parallel()
	packets := []packet.Packet{{ID: 7, State: packet.Verified}}

	for i := 0; i < 20; i++ {
		id, ok := drawCalibration(packets, 0)
		assert.True(t, ok)
		assert.Equal(t, 7, id, "with exactly one candidate, the draw has only one honest answer")
	}
}

// "Auto-forwarded" means State==Verified specifically — a
// packet that reached Verified without ever being held. Any other lifecycle
// state must never be eligible for the sample.
func TestDrawCalibration_onlyConsidersAutoForwardedVerifiedPackets(t *testing.T) {
	t.Parallel()
	packets := []packet.Packet{
		{ID: 1, State: packet.Held},
		{ID: 2, State: packet.InFlight},
		{ID: 3, State: packet.Composing},
		{ID: 4, State: packet.Verified},
	}

	id, ok := drawCalibration(packets, 0)

	assert.True(t, ok)
	assert.Equal(t, 4, id, "only a packet that reached Verified without ever being held is eligible")
}

// InFlight and Composing packets are NOT held, so a filter that merely
// excludes Held (rather than requiring == Verified) would wrongly admit
// them — pin the filter down precisely.
func TestDrawCalibration_excludesInFlightAndComposingEvenThoughNeitherIsHeld(t *testing.T) {
	t.Parallel()
	packets := []packet.Packet{
		{ID: 1, State: packet.InFlight},
		{ID: 2, State: packet.Composing},
	}

	id, ok := drawCalibration(packets, 0)

	assert.False(t, ok, "neither in-flight nor composing has reached Verified — nothing honest to draw")
	assert.Equal(t, 0, id)
}
