package packet_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
)

func TestGateStatus_stringIsALowercaseMonoWordPerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status packet.GateStatus
		want   string
	}{
		{"not-run", packet.GateNotRun, "not-run"},
		{"passed", packet.GatePassed, "passed"},
		{"failed", packet.GateFailed, "failed"},
		{"held", packet.GateHeld, "held"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestGateStatus_stringFailsSafeOnAnOutOfRangeValue(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "not-run", packet.GateStatus(99).String())
}

func TestGauntlet_zeroValueLeavesEveryGateNotRun(t *testing.T) {
	t.Parallel()

	var g packet.Gauntlet
	for _, gate := range []packet.Gate{
		g.IntentFidelity, g.HandshakeConformance, g.HandshakeTightness,
		g.BuildVetLint, g.TestSensitivity, g.IndependentCheck,
	} {
		assert.Equal(t, packet.GateNotRun, gate.Status, "an unmeasured gate must never default to any other status")
	}
}

func TestGauntlet_forwardableTruthTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		g    packet.Gauntlet
		want bool
	}{
		{
			name: "all six gates unmeasured is still forwardable",
			g:    packet.Gauntlet{},
			want: true,
		},
		{
			name: "every gate passed is forwardable",
			g: packet.Gauntlet{
				IntentFidelity:       packet.Gate{Status: packet.GatePassed},
				HandshakeConformance: packet.Gate{Status: packet.GatePassed},
				HandshakeTightness:   packet.Gate{Status: packet.GatePassed},
				BuildVetLint:         packet.Gate{Status: packet.GatePassed},
				TestSensitivity:      packet.Gate{Status: packet.GatePassed},
				IndependentCheck:     packet.Gate{Status: packet.GatePassed},
			},
			want: true,
		},
		{
			name: "a held gate does not block forwarding by itself",
			g:    packet.Gauntlet{HandshakeTightness: packet.Gate{Status: packet.GateHeld}},
			want: true,
		},
		{
			name: "a failed intent-fidelity gate alone blocks forwarding",
			g:    packet.Gauntlet{IntentFidelity: packet.Gate{Status: packet.GateFailed}},
			want: false,
		},
		{
			name: "a failed handshake-conformance gate alone blocks forwarding",
			g:    packet.Gauntlet{HandshakeConformance: packet.Gate{Status: packet.GateFailed}},
			want: false,
		},
		{
			name: "a failed handshake-tightness gate alone blocks forwarding",
			g:    packet.Gauntlet{HandshakeTightness: packet.Gate{Status: packet.GateFailed}},
			want: false,
		},
		{
			name: "a single failed build-vet-lint gate blocks forwarding",
			g:    packet.Gauntlet{BuildVetLint: packet.Gate{Status: packet.GateFailed}},
			want: false,
		},
		{
			name: "a failed test-sensitivity gate alone blocks forwarding",
			g:    packet.Gauntlet{TestSensitivity: packet.Gate{Status: packet.GateFailed}},
			want: false,
		},
		{
			name: "a failed gate blocks even alongside passed and held gates",
			g: packet.Gauntlet{
				IntentFidelity:     packet.Gate{Status: packet.GatePassed},
				HandshakeTightness: packet.Gate{Status: packet.GateHeld},
				IndependentCheck:   packet.Gate{Status: packet.GateFailed},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.g.Forwardable())
		})
	}
}

func TestGateFromCatchOutcome_mintsAPassedGateOnACatch(t *testing.T) {
	t.Parallel()

	g := packet.GateFromCatchOutcome(catch.Catch, 0, 4)
	assert.Equal(t, packet.GatePassed, g.Status)
	assert.Equal(t, "handshake tightened — 0 survivors of 4", g.Detail)
}

func TestGateFromCatchOutcome_holdsOnANoCatchNamingTheGap(t *testing.T) {
	t.Parallel()

	g := packet.GateFromCatchOutcome(catch.NoCatch, 3, 5)
	assert.Equal(t, packet.GateHeld, g.Status)
	assert.Equal(t, "3 survivors of 5 — gap found", g.Detail)
}

func TestGateFromCatchOutcome_holdsOnAPartialCatchNamingTheNarrowing(t *testing.T) {
	t.Parallel()

	g := packet.GateFromCatchOutcome(catch.PartialCatch, 1, 3)
	assert.Equal(t, packet.GateHeld, g.Status)
	assert.Equal(t, "narrowed to 1 of 3 survivors", g.Detail)
}

func TestGateFromCatchOutcome_isNotRunOnNoOracleSignalRatherThanAFalsePass(t *testing.T) {
	t.Parallel()

	g := packet.GateFromCatchOutcome(catch.NoOracleSignal, 0, 0)
	assert.Equal(t, packet.GateNotRun, g.Status, "no mutable operators means the oracle said nothing — never a fabricated pass")
	assert.Equal(t, "no mutable operators on this line", g.Detail)
}

func TestGateFromCatchOutcome_failsSafeToNotRunOnAnUnrecognizedOutcome(t *testing.T) {
	t.Parallel()

	g := packet.GateFromCatchOutcome(catch.Outcome("bogus"), 0, 0)
	assert.Equal(t, packet.GateNotRun, g.Status, "an unrecognized outcome must never be read as a pass")
}
