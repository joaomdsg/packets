package packet_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
)

func TestLaneFloor_risesWithLaneRadiusPerMVPsStrongerRequiredHandshakeRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lane packet.Lane
		want packet.HandshakeStrength
	}{
		{"unmeasured requires nothing — nothing has been measured yet", packet.LaneUnmeasured, packet.StrengthNone},
		{"best-effort requires nothing — narrow blast radius, proportionate scrutiny", packet.LaneBestEffort, packet.StrengthNone},
		{"standard requires at least examples", packet.LaneStandard, packet.StrengthExamples},
		{"strict requires properties", packet.LaneStrict, packet.StrengthProperties},
		{"irreversible requires properties too — unreachable via LaneFor today, but the floor is defined for when it isn't", packet.LaneIrreversible, packet.StrengthProperties},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, packet.LaneFloor(tt.lane))
		})
	}
}

// gatePassed/gatePassedGauntlet build an all-GatePassed Gauntlet so Forwardable
// reports true and only the case under test can drive ReconcileHold's outcome.
func gatePassedGauntlet() packet.Gauntlet {
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

func TestReconcileHold_forcesBlockingWhenHandshakeStrengthIsBelowTheLaneFloor(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		State:             packet.Verified,
		Hold:              packet.HoldNone,
		HoldReason:        "",
		Lane:              packet.LaneStrict,
		HandshakeStrength: packet.StrengthExamples, // strict needs properties
		Gauntlet:          gatePassedGauntlet(),
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.HoldBlocking, got.Hold)
	assert.Equal(t, "handshake below lane floor", got.HoldReason, "this is design/guidelines/voice.md's own canonical example string — verbatim")
}

func TestReconcileHold_escalatesToBlockingWhenAGateFailedNamingItsOwnDetail(t *testing.T) {
	t.Parallel()

	failing := packet.Gate{Status: packet.GateFailed, Detail: "3 survivors of 12"}
	gauntlet := gatePassedGauntlet()
	gauntlet.BuildVetLint = failing

	p := packet.Packet{
		State:             packet.Held,
		Hold:              packet.HoldAdvisory,
		HoldReason:        "open questions · 2",
		Lane:              packet.LaneBestEffort, // floor is StrengthNone, so rule (b) can't fire
		HandshakeStrength: packet.StrengthNone,
		Gauntlet:          gauntlet,
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.HoldBlocking, got.Hold)
	assert.Equal(t, "gate failed · 3 survivors of 12", got.HoldReason, "reuses the failing gate's own Detail, never invented wording")
}

// Pins which gate's Detail wins when more than one gate has failed — the FIRST
// GateFailed found walking IntentFidelity..IndependentCheck in that order,
// never the last, so the reported reason is deterministic.
func TestReconcileHold_walksGatesInG1ThroughG6OrderToFindTheFirstFailure(t *testing.T) {
	t.Parallel()

	gauntlet := gatePassedGauntlet()
	gauntlet.HandshakeTightness = packet.Gate{Status: packet.GateFailed, Detail: "G3 failed first"}
	gauntlet.TestSensitivity = packet.Gate{Status: packet.GateFailed, Detail: "G5 failed too"}

	p := packet.Packet{
		Hold:     packet.HoldNone,
		Lane:     packet.LaneBestEffort,
		Gauntlet: gauntlet,
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, "gate failed · G3 failed first", got.HoldReason)
}

// A GateHeld or GateNotRun gate ahead of the real GateFailed one must be
// skipped by the G1..G6 walk — only GateFailed counts as a failure
// (Forwardable's own rule), so the reported reason must come from the actual
// failing gate, not whichever gate merely isn't GatePassed first.
func TestReconcileHold_gateWalkSkipsHeldAndNotRunGatesToFindTheRealFailure(t *testing.T) {
	t.Parallel()

	gauntlet := gatePassedGauntlet()
	gauntlet.HandshakeTightness = packet.Gate{Status: packet.GateHeld, Detail: "narrowed to 2 of 5 survivors"}
	gauntlet.TestSensitivity = packet.Gate{Status: packet.GateFailed, Detail: "the real failure"}

	p := packet.Packet{
		Hold:     packet.HoldNone,
		Lane:     packet.LaneBestEffort,
		Gauntlet: gauntlet,
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, "gate failed · the real failure", got.HoldReason)
}

// A lifecycle-native BLOCKING hold ("run failed", "unknown state · x") must
// pass through byte-for-byte when neither forcing rule fires — this slice
// only ADDS blocking triggers, it never reprocesses or rewords a hold Fold
// already set to blocking.
func TestReconcileHold_leavesAFoldBlockingHoldReasonUntouchedWhenNeitherRuleFires(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		State:      packet.Held,
		Hold:       packet.HoldBlocking,
		HoldReason: "run failed",
		Lane:       packet.LaneBestEffort,
		Gauntlet:   gatePassedGauntlet(),
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.HoldBlocking, got.Hold)
	assert.Equal(t, "run failed", got.HoldReason)
}

func TestReconcileHold_leavesAVerifiedPacketUntouchedWhenNeitherRuleFires(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		State:             packet.Verified,
		Hold:              packet.HoldNone,
		HoldReason:        "",
		Lane:              packet.LaneStandard,
		HandshakeStrength: packet.StrengthExamples, // meets the standard floor exactly
		Gauntlet:          gatePassedGauntlet(),
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.HoldNone, got.Hold)
	assert.Equal(t, "", got.HoldReason)
}

// A Delivered packet (a real ACK) must survive ReconcileHold untouched —
// neither forcing rule may un-deliver it. Escalate-only means escalate,
// never revert a real ACK back to held.
func TestReconcileHold_leavesADeliveredPacketUntouchedWhenNeitherRuleFires(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		State:             packet.Delivered,
		Hold:              packet.HoldNone,
		HoldReason:        "",
		Lane:              packet.LaneStandard,
		HandshakeStrength: packet.StrengthExamples,
		Gauntlet:          gatePassedGauntlet(),
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.Delivered, got.State, "ReconcileHold must never change State, only Hold/HoldReason")
	assert.Equal(t, packet.HoldNone, got.Hold)
	assert.Equal(t, "", got.HoldReason)
}

// A Delivered packet is EXEMPT from both forcing rules, even when their
// triggering conditions are true — surveillance ends at real ACK. Without
// this exemption a Delivered packet could render as blocking-held (a stale
// cached Lane/Gauntlet, or a lane/handshake mismatch nobody re-checked after
// delivery), which reads as self-contradictory in the UI: "delivered" and
// "needs you" at once.
func TestReconcileHold_exemptsADeliveredPacketEvenWhenTheForcingRulesWouldOtherwiseFire(t *testing.T) {
	t.Parallel()

	failing := gatePassedGauntlet()
	failing.BuildVetLint = packet.Gate{Status: packet.GateFailed, Detail: "go vet: 2 issues"}

	p := packet.Packet{
		State:             packet.Delivered,
		Hold:              packet.HoldNone,
		HoldReason:        "",
		Lane:              packet.LaneStrict,   // floor is StrengthProperties
		HandshakeStrength: packet.StrengthNone, // below floor — rule (b) would fire
		Gauntlet:          failing,             // not forwardable — rule (c) would fire too
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.Delivered, got.State)
	assert.Equal(t, packet.HoldNone, got.Hold, "delivered is exempt from lane-floor/gate-failure escalation")
	assert.Equal(t, "", got.HoldReason)
}

// A done+!caught advisory hold from Fold must survive untouched when
// neither the lane-floor nor the gate-failure rule fires — the blocking
// rules here ADD triggers, they never touch Fold's existing lifecycle-hold logic.
func TestReconcileHold_leavesFoldsAdvisoryHoldUntouchedWhenNeitherRuleFires(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		State:             packet.Held,
		Hold:              packet.HoldAdvisory,
		HoldReason:        "gap found · handshake not tightened",
		Lane:              packet.LaneBestEffort,
		HandshakeStrength: packet.StrengthNone,
		Gauntlet:          gatePassedGauntlet(),
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.HoldAdvisory, got.Hold)
	assert.Equal(t, "gap found · handshake not tightened", got.HoldReason)
}

// When BOTH the lane-floor breach and a gate failure are true at once, the
// lane-floor rule wins — it is checked first in the documented precedence, so
// two forcing rules firing together still has one defined winner rather than
// depending on branch evaluation order.
func TestReconcileHold_laneFloorBreachWinsOverAGateFailureWhenBothFire(t *testing.T) {
	t.Parallel()

	gauntlet := gatePassedGauntlet()
	gauntlet.BuildVetLint = packet.Gate{Status: packet.GateFailed, Detail: "build failed"}

	p := packet.Packet{
		Hold:              packet.HoldNone,
		Lane:              packet.LaneStrict,
		HandshakeStrength: packet.StrengthNone, // below strict's properties floor
		Gauntlet:          gauntlet,
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.HoldBlocking, got.Hold)
	assert.Equal(t, "handshake below lane floor", got.HoldReason)
}

// A fresh Composing packet — LaneUnmeasured, an all-GateNotRun Gauntlet — must
// not be force-held by either rule just because nothing has been measured
// yet: LaneUnmeasured's floor is StrengthNone (rule b can't fire) and an
// all-NotRun Gauntlet IS Forwardable()==true (not-run isn't a failure, so
// rule c can't fire either). This holds by the types alone — no special-case
// branch should be needed to make it pass.
// An unrecognized HandshakeStrength (e.g. corrupt/future ledger data outside
// StrengthNone..StrengthProperties) must never be mistaken for a strong
// declaration that satisfies a lane's floor — it fails toward attention,
// ranking as the weakest possible strength, so any lane with a real floor
// (anything above best-effort) still forces a hold rather than silently
// forwarding on data nobody can vouch for.
func TestReconcileHold_treatsAnUnrecognizedHandshakeStrengthAsTheWeakestPossibleOne(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		Hold:              packet.HoldNone,
		Lane:              packet.LaneStandard,
		HandshakeStrength: packet.HandshakeStrength(99),
		Gauntlet:          gatePassedGauntlet(),
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.HoldBlocking, got.Hold)
	assert.Equal(t, "handshake below lane floor", got.HoldReason)
}

// A Verified packet that a LATER lane-floor breach or gate failure forces
// into a hold must ALSO flip to State=Held — never left as Verified+held at
// once. Two independent adversarial reviews confirmed this exact
// self-contradiction was reachable: heldPackets (needs-you rail) filters on
// Hold alone, renderSettledRail/renderHeroStat/drawCalibration filter or
// count on State alone, so an unescalated State let the SAME packet render
// as both "blocking, needs you" and "verified, nothing to see here, safe to
// draw for calibration" simultaneously.
func TestReconcileHold_flipsAnEscalatedVerifiedPacketToHeldSoNoRailShowsItAsSafe(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		State:             packet.Verified,
		Hold:              packet.HoldNone,
		Lane:              packet.LaneStrict,
		HandshakeStrength: packet.StrengthNone, // below strict's properties floor
		Gauntlet:          gatePassedGauntlet(),
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.Held, got.State, "an escalated hold must be reflected in State too, never left Verified")
	assert.Equal(t, packet.HoldBlocking, got.Hold)
	assert.Equal(t, "handshake below lane floor", got.HoldReason)
}

// The same flip applies when a hard GATE failure (not a lane-floor breach)
// is what forces the hold on an otherwise-Verified packet.
func TestReconcileHold_flipsToHeldOnAGateFailureEvenWhenStateWasVerified(t *testing.T) {
	t.Parallel()

	failing := gatePassedGauntlet()
	failing.BuildVetLint = packet.Gate{Status: packet.GateFailed, Detail: "go vet: 2 issues"}

	p := packet.Packet{
		State:    packet.Verified,
		Hold:     packet.HoldNone,
		Lane:     packet.LaneBestEffort, // floor is StrengthNone — rule (b) can't fire
		Gauntlet: failing,
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.Held, got.State)
	assert.Equal(t, packet.HoldBlocking, got.Hold)
	assert.Equal(t, "gate failed · go vet: 2 issues", got.HoldReason)
}

// Delivered stays the ONE exemption — its own rule (checked first) must
// still win even though Verified is no longer a safe harbor.
func TestReconcileHold_deliveredStaysExemptEvenThoughVerifiedNoLongerIs(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		State:             packet.Delivered,
		Hold:              packet.HoldNone,
		Lane:              packet.LaneStrict,
		HandshakeStrength: packet.StrengthNone,
		Gauntlet:          gatePassedGauntlet(),
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.Delivered, got.State, "Delivered's exemption must still hold")
	assert.Equal(t, packet.HoldNone, got.Hold)
}

func TestReconcileHold_leavesAFreshComposingPacketAloneSinceNothingIsMeasuredYet(t *testing.T) {
	t.Parallel()

	p := packet.Packet{
		State: packet.Composing,
		Hold:  packet.HoldNone,
		Lane:  packet.LaneUnmeasured,
		// Gauntlet left at its zero value: every gate GateNotRun.
	}

	got := packet.ReconcileHold(p)

	assert.Equal(t, packet.HoldNone, got.Hold)
	assert.Equal(t, "", got.HoldReason)
}
