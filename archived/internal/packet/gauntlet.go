// Package packet's gauntlet mechanic is the pure record shape for the six
// gates a packet runs through: one explicit pipeline
// record per packet, honest about which gates have real data behind them
// today and which are a deliberate absence. This file holds ONLY the pure
// core (types, String(), Forwardable, GateFromCatchOutcome) — no I/O, no
// exec. The one exec seam this slice adds (G4, build/vet) lives in
// gauntlet_build.go; the app layer (internal/app) wires G3/G4 per packet at
// render time, the same compute-on-render, cache pattern as Lane (lane.go).
package packet

import (
	"fmt"

	"github.com/joaomdsg/packets/internal/catch"
)

// GateStatus is one gate's verdict. GateNotRun is the zero value — the
// honest default for a gate with no data behind it yet, never a fabricated
// pass or fail.
type GateStatus int

const (
	// GateNotRun: no data exists for this gate yet (a human residual not yet
	// given, a mechanic not yet wired, or an oracle with nothing to say).
	GateNotRun GateStatus = iota
	// GatePassed: the gate ran and found nothing to hold on.
	GatePassed
	// GateFailed: the gate ran and found a hard problem — this is what
	// Gauntlet.Forwardable checks for.
	GateFailed
	// GateHeld: a mutation gate found a survivor that isn't yet a hard
	// fail — the same nuance as catch.PartialCatch/NoCatch. A real
	// blocking decision from a Held gate is Packet.Hold's job (the lifecycle/hold mapping);
	// this status is only the record of what the gate observed.
	GateHeld
)

// String renders the lowercase, hyphenated mono-voice name used across the
// UI ("not-run", not "NotRun"), failing safe to "not-run" for any
// unrecognized value — an unknown status is never read as a pass.
func (s GateStatus) String() string {
	switch s {
	case GatePassed:
		return "passed"
	case GateFailed:
		return "failed"
	case GateHeld:
		return "held"
	default:
		return "not-run"
	}
}

// Gate is one gauntlet gate's recorded outcome: a status plus a one-clause
// honest note (mono voice — e.g. "not measured — no handshake yet", "3
// survivors of 12", "confirmed by <navKey>"). The zero value is
// {GateNotRun, ""}, the correct default for a gate nobody has run.
type Gate struct {
	Status GateStatus
	Detail string
}

// Gauntlet is the six-gate pipeline record for one packet (G1..G6 in order).
// The zero value leaves every gate GateNotRun — the
// honest default for a packet nothing has gauntleted yet.
type Gauntlet struct {
	IntentFidelity       Gate // G1: the human residual (Inspector affordance).
	HandshakeConformance Gate // G2: run the handshake, line rate.
	HandshakeTightness   Gate // G3: mutation vs SPEC (handshake-scoped).
	BuildVetLint         Gate // G4: deterministic build/vet/lint.
	TestSensitivity      Gate // G5: mutation vs the agent's own tests.
	IndependentCheck     Gate // G6: method diversity (cage re-derivation).
}

// Forwardable reports whether NOTHING in the gauntlet blocks forwarding:
// true unless some gate is GateFailed. GateNotRun and GateHeld do NOT block
// by themselves — a gate that hasn't run yet or merely found a narrowing
// survivor is not the same as a hard failure. A real hold/blocking DECISION
// off a Held or NotRun gate is Packet.Hold's job (the lifecycle/hold
// mapping); Forwardable only reports what this record itself rules out.
func (g Gauntlet) Forwardable() bool {
	for _, gate := range []Gate{
		g.IntentFidelity, g.HandshakeConformance, g.HandshakeTightness,
		g.BuildVetLint, g.TestSensitivity, g.IndependentCheck,
	} {
		if gate.Status == GateFailed {
			return false
		}
	}
	return true
}

// GateFromCatchOutcome derives G3 (handshake tightness) from a catch cycle's
// outcome and its after-revision survivor/inventory counts — pure, no I/O.
// survivors and total are the after-revision LineState's Survivors and
// Inventory sizes (see catch.LineState); the caller supplies them since this
// package never re-runs the mutation oracle. Catch mints a pass (the fix
// genuinely emptied the survivor set); NoCatch and PartialCatch are Held —
// a gap found or narrowed, never a hard fail, matching catch.Outcome's own
// nuance; NoOracleSignal and any unrecognized outcome are NotRun — the line
// had nothing mutable to say, never fabricated into a pass.
func GateFromCatchOutcome(o catch.Outcome, survivors, total int) Gate {
	switch o {
	case catch.Catch:
		return Gate{Status: GatePassed, Detail: fmt.Sprintf("handshake tightened — %d survivors of %d", survivors, total)}
	case catch.NoCatch:
		return Gate{Status: GateHeld, Detail: fmt.Sprintf("%d survivors of %d — gap found", survivors, total)}
	case catch.PartialCatch:
		return Gate{Status: GateHeld, Detail: fmt.Sprintf("narrowed to %d of %d survivors", survivors, total)}
	default: // catch.NoOracleSignal, or any future/unrecognized outcome
		return Gate{Status: GateNotRun, Detail: "no mutable operators on this line"}
	}
}
