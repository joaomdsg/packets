// Package packet's hold mechanic is the composition point for the
// forward/hold concept: the RULES that turn a measured Lane and a computed
// Gauntlet into an escalated hold, layered on top of the lifecycle-hold
// baseline Fold already sets (packet.go). This file is pure — no I/O, no
// exec — because Lane and Gauntlet are populated OUTSIDE Fold, in the app
// layer (laneFor/gauntletFor), so ReconcileHold must be
// a SEPARATE function the app calls after attaching both, never folded into
// Fold itself.
package packet

// LaneFloor is the minimum HandshakeStrength a lane's blast radius REQUIRES
// (radius buys... a stronger required handshake).
// LaneIrreversible is unreachable from LaneFor today (see lane.go), but its
// floor is defined now so ReconcileHold is already correct the day
// irreversibility becomes measurable.
func LaneFloor(l Lane) HandshakeStrength {
	switch l {
	case LaneStandard:
		return StrengthExamples
	case LaneStrict, LaneIrreversible:
		return StrengthProperties
	default: // LaneUnmeasured, LaneBestEffort
		return StrengthNone
	}
}

// handshakeStrengthRank names each HandshakeStrength's ordinal position
// explicitly, so belowFloor's comparison depends on THIS table rather than
// on HandshakeStrength and Lane happening to share compatible iota values —
// a future reordering or insertion into either enum cannot silently change
// what "below floor" means without also updating this rank. A key absent
// from this table (an unrecognized/corrupt HandshakeStrength) ranks as the
// Go zero value, 0 — the same rank as StrengthNone, so an invalid strength
// fails toward attention: it can never be mistaken for a stronger, floor-
// satisfying declaration, only ever the weakest one.
var handshakeStrengthRank = map[HandshakeStrength]int{
	StrengthNone:       0,
	StrengthExamples:   1,
	StrengthProperties: 2,
}

// belowFloor reports whether strength fails to meet floor, comparing each
// side's own named rank (handshakeStrengthRank) rather than raw int
// identity.
func belowFloor(strength HandshakeStrength, floor HandshakeStrength) bool {
	return handshakeStrengthRank[strength] < handshakeStrengthRank[floor]
}

// firstFailedGate walks g's six gates in G1..G6 order and returns the Detail
// of the first GateFailed one found, ok=false if none failed. The fixed walk
// order makes the reported reason deterministic when more than one gate has
// failed — always the earliest gate in the pipeline, never whichever branch
// a range over a struct might visit last.
func firstFailedGate(g Gauntlet) (detail string, ok bool) {
	for _, gate := range []Gate{
		g.IntentFidelity, g.HandshakeConformance, g.HandshakeTightness,
		g.BuildVetLint, g.TestSensitivity, g.IndependentCheck,
	} {
		if gate.Status == GateFailed {
			return gate.Detail, true
		}
	}
	return "", false
}

// ReconcileHold is the composition point for the forward/hold concept: called
// by the app AFTER Lane and Gauntlet are attached to p (Fold alone cannot do
// this — see the package doc), it escalates p's lifecycle-hold baseline toward
// blocking when a lane-floor breach or a hard gate failure demands it, and
// otherwise leaves p exactly as Fold set it. Precedence is fixed and
// escalate-only — a later rule never de-escalates what an earlier one (or
// Fold itself) found:
//
//  1. Baseline: p.Hold/p.HoldReason exactly as Fold left them.
//  2. Lane-floor breach: p.HandshakeStrength below what p.Lane requires
//     forces HoldBlocking with the exact design-voice phrase "handshake
//     below lane floor" (design/guidelines/voice.md's own canonical
//     example). Checked FIRST, so if a gate has also failed, the lane-floor
//     reason still wins — a missing/weak handshake is the more fundamental
//     problem (the gate that should have run under the required strength
//     may not even be trustworthy).
//  3. Hard gate failure: !p.Gauntlet.Forwardable() forces HoldBlocking, with
//     the reason naming the first failed gate's OWN Detail (never invented
//     wording) — reused verbatim rather than duplicated.
//  4. Otherwise: p is returned unchanged. A Composing/InFlight packet with
//     an unmeasured lane and an all-not-run gauntlet holds here naturally
//     (LaneUnmeasured's floor is StrengthNone, so rule 2 never fires; an
//     all-not-run Gauntlet IS Forwardable()==true, so rule 3 never fires) —
//     no special-case branch is needed for that to behave correctly.
//  0. Delivered is EXEMPT from both forcing rules (checked before either) —
//     a real ACK ends standing surveillance. Without this, a stale cached
//     Lane/Gauntlet (both are render-time exec-derived caches, never
//     re-measured after delivery) could force a Delivered packet's Hold to
//     blocking, rendering as self-contradictory in the UI: "delivered" and
//     "needs you" at once.
//
// Whenever rule 2 or 3 fires, State is ALSO forced to Held (never left at
// whatever Fold set it to). This closes an adversarially-found gap: Lane and
// Gauntlet are render-time caches that can populate LONG after a packet
// already reached State==Verified (e.g. a human opens the Inspector on a
// verified packet days later, measuring a lane-floor breach nobody had
// computed before) — leaving State untouched let the SAME packet render as
// "verified, safe to draw for calibration" in the settled rail/hero
// stat/calibration draw AND "blocking, needs you" in the needs-you rail at
// once. Held is the honest state once a forcing rule has fired, full stop.
func ReconcileHold(p Packet) Packet {
	if p.State == Delivered {
		return p
	}
	if belowFloor(p.HandshakeStrength, LaneFloor(p.Lane)) {
		p.State = Held
		p.Hold = HoldBlocking
		p.HoldReason = "handshake below lane floor"
		return p
	}
	if !p.Gauntlet.Forwardable() {
		if detail, ok := firstFailedGate(p.Gauntlet); ok {
			p.State = Held
			p.Hold = HoldBlocking
			p.HoldReason = "gate failed · " + detail
		}
		return p
	}
	return p
}
