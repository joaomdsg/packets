package packet_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
)

func TestWatchKind_stringIsALowercaseHyphenatedMonoWordPerKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind packet.WatchKind
		want string
	}{
		{"strict lane", packet.WatchStrictLane, "strict-lane"},
		{"gate failure", packet.WatchGateFailure, "gate-failure"},
		{"blocking hold", packet.WatchBlockingHold, "blocking-hold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.kind.String())
		})
	}
}

func TestWatchKind_stringFailsSafeOnAnUnrecognizedKind(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unknown", packet.WatchKind(99).String())
}

// gatePassedGauntlet mirrors the shared all-GatePassed Gauntlet helper: an all-GatePassed Gauntlet
// so Forwardable reports true and only the case under test drives the result.
func gatePassedGauntletForWatch() packet.Gauntlet {
	passed := packet.Gate{Status: packet.GatePassed, Detail: "ok"}
	return packet.Gauntlet{
		IntentFidelity:       passed,
		HandshakeConformance: passed,
		HandshakeTightness:   passed,
		BuildVetLint:         passed,
		TestSensitivity:      passed,
		IndependentCheck:     passed,
	}
}

func TestEvaluateWatch_strictLaneFiresOnlyWhenTheLaneIsMeasuredStrict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lane packet.Lane
		want bool
	}{
		{"strict lane fires", packet.LaneStrict, true},
		{"standard lane does not fire", packet.LaneStandard, false},
		{"best-effort lane does not fire", packet.LaneBestEffort, false},
		{"unmeasured lane does not fire", packet.LaneUnmeasured, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := packet.Packet{Lane: tt.lane, Gauntlet: gatePassedGauntletForWatch()}
			assert.Equal(t, tt.want, packet.EvaluateWatch(packet.WatchStrictLane, p))
		})
	}
}

func TestEvaluateWatch_gateFailureFiresOnlyWhenTheGauntletIsNotForwardable(t *testing.T) {
	t.Parallel()

	p := packet.Packet{Gauntlet: gatePassedGauntletForWatch()}
	assert.False(t, packet.EvaluateWatch(packet.WatchGateFailure, p),
		"an all-passed gauntlet is forwardable — the watch must not fire")

	failing := gatePassedGauntletForWatch()
	failing.BuildVetLint = packet.Gate{Status: packet.GateFailed, Detail: "go vet: 2 issues"}
	p2 := packet.Packet{Gauntlet: failing}
	assert.True(t, packet.EvaluateWatch(packet.WatchGateFailure, p2),
		"a hard gate failure makes the gauntlet unforwardable — the watch must fire")
}

func TestEvaluateWatch_blockingHoldFiresOnlyOnAHoldBlockingPacket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hold packet.HoldKind
		want bool
	}{
		{"blocking fires", packet.HoldBlocking, true},
		{"advisory does not fire", packet.HoldAdvisory, false},
		{"none does not fire", packet.HoldNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := packet.Packet{Hold: tt.hold, Gauntlet: gatePassedGauntletForWatch()}
			assert.Equal(t, tt.want, packet.EvaluateWatch(packet.WatchBlockingHold, p))
		})
	}
}

func TestEvaluateWatch_anUnrecognizedKindNeverFires(t *testing.T) {
	t.Parallel()
	p := packet.Packet{Lane: packet.LaneStrict, Hold: packet.HoldBlocking, Gauntlet: gatePassedGauntletForWatch()}
	assert.False(t, packet.EvaluateWatch(packet.WatchKind(99), p),
		"an unrecognized watch kind must never be treated as a match — fail closed, not open")
}

// An unrecognized kind must stay false even when the packet's OWN facts would
// satisfy a recognized kind's predicate (here, a failed gate would fire
// WatchGateFailure) — proving the fail-closed default isn't just "vacuously
// true because nothing in the packet would match anyway".
func TestEvaluateWatch_anUnrecognizedKindStaysFalseEvenWhenTheFactsWouldMatchARealKind(t *testing.T) {
	t.Parallel()
	failing := gatePassedGauntletForWatch()
	failing.BuildVetLint = packet.Gate{Status: packet.GateFailed, Detail: "go vet: 2 issues"}
	p := packet.Packet{Gauntlet: failing}

	assert.True(t, packet.EvaluateWatch(packet.WatchGateFailure, p), "sanity: the real kind fires on these facts")
	assert.False(t, packet.EvaluateWatch(packet.WatchKind(99), p),
		"an unrecognized kind must not piggyback on facts that would satisfy a different, real kind")
}

func useful(v bool) *bool { return &v }

func TestPrecision_reportsNoHistoryWhenNoFireOfThisKindHasBeenMarked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		fires []packet.WatchFire
	}{
		{"no fires at all", nil},
		{"fires exist but none marked", []packet.WatchFire{
			{Kind: packet.WatchStrictLane, PacketID: 1, Useful: nil},
			{Kind: packet.WatchStrictLane, PacketID: 2, Useful: nil},
		}},
		{"marked fires exist but for a DIFFERENT kind", []packet.WatchFire{
			{Kind: packet.WatchGateFailure, PacketID: 1, Useful: useful(true)},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			score, sampled, ok := packet.Precision(tt.fires, packet.WatchStrictLane)
			assert.False(t, ok, "zero marked fires of this kind must report ok=false, never a fabricated score")
			assert.Equal(t, 0, sampled)
			assert.Zero(t, score)
		})
	}
}

func TestPrecision_scoresOnlyMarkedFiresOfTheRequestedKind(t *testing.T) {
	t.Parallel()

	fires := []packet.WatchFire{
		{Kind: packet.WatchStrictLane, PacketID: 1, Useful: useful(true)},
		{Kind: packet.WatchStrictLane, PacketID: 2, Useful: useful(true)},
		{Kind: packet.WatchStrictLane, PacketID: 3, Useful: useful(false)},
		{Kind: packet.WatchStrictLane, PacketID: 4, Useful: nil},            // unmarked — excluded from the sample
		{Kind: packet.WatchGateFailure, PacketID: 5, Useful: useful(false)}, // different kind — excluded
	}

	score, sampled, ok := packet.Precision(fires, packet.WatchStrictLane)

	assert.True(t, ok)
	assert.Equal(t, 3, sampled, "only the 3 MARKED strict-lane fires count as the sample")
	assert.InDelta(t, 2.0/3.0, score, 0.0001)
}

func TestPrecision_perfectAndZeroScoresAreExactAtTheirEdges(t *testing.T) {
	t.Parallel()

	allUseful := []packet.WatchFire{
		{Kind: packet.WatchBlockingHold, PacketID: 1, Useful: useful(true)},
		{Kind: packet.WatchBlockingHold, PacketID: 2, Useful: useful(true)},
	}
	score, sampled, ok := packet.Precision(allUseful, packet.WatchBlockingHold)
	assert.True(t, ok)
	assert.Equal(t, 2, sampled)
	assert.Equal(t, 1.0, score)

	allNoise := []packet.WatchFire{
		{Kind: packet.WatchBlockingHold, PacketID: 1, Useful: useful(false)},
	}
	score, sampled, ok = packet.Precision(allNoise, packet.WatchBlockingHold)
	assert.True(t, ok)
	assert.Equal(t, 1, sampled)
	assert.Equal(t, 0.0, score)
}

func TestIsNoisy_requiresBothARealSampleSizeAndBelowHalfUsefulness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		score   float64
		sampled int
		want    bool
	}{
		{"below-half score but too small a sample to judge yet", 0.0, 4, false},
		{"below-half score with a real sample is noisy", 0.4, 5, true},
		{"exactly half is not below half — not noisy", 0.5, 5, false},
		{"majority-useful with a real sample is not noisy", 0.6, 5, false},
		{"zero sample is never noisy — there's nothing to judge", 0.0, 0, false},
		{"large sample, low score is noisy", 0.1, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, packet.IsNoisy(tt.score, tt.sampled))
		})
	}
}
