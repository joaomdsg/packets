package packet_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/packet"
)

// A repo with fewer settled packets than the threshold hasn't earned
// convergence yet — the honest default is "still learning," not a fabricated
// pass.
func TestConverged_isFalseBelowTheThreshold(t *testing.T) {
	t.Parallel()
	packets := make([]packet.Packet, packet.LearningThreshold-1)
	for i := range packets {
		packets[i] = packet.Packet{ID: i + 1, State: packet.Verified}
	}
	assert.False(t, packet.Converged(packets))
}

// Reaching exactly the threshold's worth of real settled history is enough —
// the bar is a minimum sample, not a strictly-greater one.
func TestConverged_isTrueAtExactlyTheThreshold(t *testing.T) {
	t.Parallel()
	packets := make([]packet.Packet, packet.LearningThreshold)
	for i := range packets {
		packets[i] = packet.Packet{ID: i + 1, State: packet.Verified}
	}
	assert.True(t, packet.Converged(packets))
}

// Composing/in-flight packets haven't produced any real judgment yet — they
// must never count toward convergence, or a repo could "converge" purely by
// queuing work nobody has looked at.
func TestConverged_neverCountsComposingOrInFlightPackets(t *testing.T) {
	t.Parallel()
	packets := make([]packet.Packet, packet.LearningThreshold)
	for i := range packets {
		state := packet.Composing
		if i%2 == 1 {
			state = packet.InFlight
		}
		packets[i] = packet.Packet{ID: i + 1, State: state}
	}
	assert.False(t, packet.Converged(packets), "unsettled packets carry no real judgment and must not count")
}

// A mix of real settled history and unsettled noise must count only the
// settled share — one short of the threshold stays "learning" even when
// padded with plenty of queued/running packets nobody has judged.
func TestConverged_ignoresUnsettledPacketsMixedInWithRealSettledHistory(t *testing.T) {
	t.Parallel()
	packets := make([]packet.Packet, 0, packet.LearningThreshold+3)
	for i := 0; i < packet.LearningThreshold-1; i++ {
		packets = append(packets, packet.Packet{ID: i + 1, State: packet.Verified})
	}
	packets = append(packets,
		packet.Packet{ID: 100, State: packet.Composing},
		packet.Packet{ID: 101, State: packet.InFlight},
		packet.Packet{ID: 102, State: packet.Composing},
	)
	assert.False(t, packet.Converged(packets), "one short of the threshold's real settled history stays learning, no matter how much unsettled work is queued")
}

// SettledCount mirrors the console's own "settled" definition exactly
// (verified, held, OR delivered) — a held-blocking packet is still real
// judgment the gauntlet produced, not noise to discard. Each qualifying
// state is isolated so a subtly wrong OR condition (e.g. missing Delivered)
// can't hide behind a count that merely happens to add up.
func TestSettledCount_countsVerifiedHeldAndDeliveredPackets(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		state packet.Lifecycle
	}{
		{"verified", packet.Verified},
		{"held", packet.Held},
		{"delivered", packet.Delivered},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 1, packet.SettledCount([]packet.Packet{{ID: 1, State: tc.state}}),
				"%s must count as settled on its own, independent of any other qualifying state", tc.name)
		})
	}

	packets := []packet.Packet{
		{ID: 1, State: packet.Verified},
		{ID: 2, State: packet.Held},
		{ID: 3, State: packet.Delivered},
		{ID: 4, State: packet.Composing},
		{ID: 5, State: packet.InFlight},
	}
	assert.Equal(t, 3, packet.SettledCount(packets), "composing/in-flight packets carry no real judgment and must not count")
}

// An empty session must report the honest zero, never a fabricated count.
func TestSettledCount_isZeroOnAFreshSessionWithNoPackets(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, packet.SettledCount(nil))
	assert.False(t, packet.Converged(nil))
}
