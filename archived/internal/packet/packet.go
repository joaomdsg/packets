package packet

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/joaomdsg/packets/internal/ledger"
)

// Packet is the read-model aggregate: one sent packet, made legible for the
// design concept model. Every field derives from a real input (a
// ledger.SendView, the repo Addr, and an injected open-questions count) —
// nothing here is fabricated.
type Packet struct {
	ID            int
	Name          string
	Addr          Addr
	Intent        string
	BaseRev       string
	FixRev        string
	State         Lifecycle
	Hold          HoldKind
	HoldReason    string
	Caught        bool
	Verdict       string
	OpenQuestions int
	// Lane is the packet's measured QoS class. Fold NEVER computes it — Fold
	// is a pure data->data projection over ledger views, while a lane needs a
	// `go list` exec against the repo (Measure, lane_measure.go) — so every
	// folded Packet starts at the honest zero value, LaneUnmeasured, until
	// the app layer measures and attaches one (laneFor).
	Lane Lane
	// Gauntlet is the packet's six-gate pipeline record.
	// Fold NEVER populates it — G3/G4 need a catch outcome and a build/vet
	// exec respectively, neither of which Fold has access to — so every
	// folded Packet starts at the honest zero value (all six gates
	// GateNotRun) until the app layer computes and attaches one per packet at
	// render time (gauntletFor), the same division of
	// labor as Lane above.
	Gauntlet Gauntlet
	// HandshakePath/HandshakeHash are copied straight from the packet's
	// ledger.Target (set at compose time) — zero I/O, a
	// plain data carry. The app layer's gauntletFor uses them to run G2
	// (RunHandshakeGate) and re-verify the handshake hasn't drifted
	// (VerifyHandshake) at render time; empty means no handshake was
	// authored for this packet (a legacy pre-funded packet, or a live packet
	// composed before this concept existed).
	HandshakePath string
	HandshakeHash string
	// HandshakeStrength is copied from the packet's ledger.Target.HandshakeStrength
	// (a plain int there, to avoid an internal/packet<->internal/ledger import
	// cycle — see ledger.go's comment on that field) via an explicit conversion
	// in Fold. The zero value, StrengthNone, is honest for a legacy pre-funded
	// packet or a live packet composed before the handshake concept existed —
	// the same "no handshake" case HandshakePath/HandshakeHash already carry.
	HandshakeStrength HandshakeStrength
}

// Deliverable reports whether this packet has a real ACK: State==Delivered,
// which Fold produces ONLY from a "deployed" send status — the host-
// issued `packets deployed` command's own evidence, never an agent's
// self-report. Every other status is pinned unreachable.
func (p Packet) Deliverable() bool {
	return p.State == Delivered
}

// slugName derives a packet's Name from the packet's own prompt, never
// invented: a lowercase hyphen slug of the first 3 whitespace-separated words,
// keeping only letters/digits within each word. A word that cleans to empty
// (pure punctuation/symbols) is SKIPPED, not fabricated — the slug still uses
// whatever of the first 3 words survives cleaning. Only when EVERY candidate
// word cleans to empty (or the prompt has none) does the name fall back to
// "pkt-<ID>", since there is nothing honest left to slug.
func slugName(prompt string, id int) string {
	words := strings.Fields(prompt)
	if len(words) > 3 {
		words = words[:3]
	}

	var cleaned []string
	for _, w := range words {
		var b strings.Builder
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(unicode.ToLower(r))
			}
		}
		if b.Len() > 0 {
			cleaned = append(cleaned, b.String())
		}
	}

	if len(cleaned) == 0 {
		return fmt.Sprintf("pkt-%d", id)
	}
	return strings.Join(cleaned, "-")
}

// Fold projects ledger SendViews into Packets — the read-model aggregate
// for the design concept model. It is PURE data→data: no I/O, no ledger
// writes, no import of internal/app. openQuestions is the caller's injected
// lookup (the app's findings cache) for how many open review questions a
// given packet left; this package stays ignorant of where that number comes
// from. Packet identity is preserved 1:1 with views (same length, same order,
// matched by ID).
//
// The send status maps onto lifecycle/hold BINDING per this table (fail
// toward attention, never silently Verified, on anything not explicitly
// listed here):
//
//	queued                                  → Composing, no hold
//	running                                 → InFlight, no hold
//	done, Caught, 0 open questions           → Verified, no hold
//	done, !Caught or open questions > 0      → Held, advisory
//	  (open-questions reason wins over the gap reason when both apply)
//	failed                                  → Held, blocking, "run failed"
//	deployed                                → Delivered, no hold (the ONLY
//	                                           path to Delivered — set by the
//	                                           host-issued `packets deployed`
//	                                           command, never self-reported)
//	regressed                               → Held, blocking,
//	                                           "deployment regression"
//	anything else (unknown/future status)   → Held, blocking,
//	                                           "unknown state · <status>"
func Fold(views []ledger.SendView, addr Addr, openQuestions func(orderID int) int) []Packet {
	packets := make([]Packet, len(views))
	for i, v := range views {
		questions := openQuestions(v.ID)
		state, hold, reason := lifecycleFor(v.Status, v.Caught, questions)
		packets[i] = Packet{
			ID:                v.ID,
			Name:              slugName(v.Target.Prompt, v.ID),
			Addr:              addr,
			Intent:            v.Target.Prompt,
			BaseRev:           v.Target.BaseRev,
			FixRev:            v.Target.FixRev,
			Caught:            v.Caught,
			Verdict:           v.Verdict,
			OpenQuestions:     questions,
			State:             state,
			Hold:              hold,
			HoldReason:        reason,
			HandshakePath:     v.Target.HandshakePath,
			HandshakeHash:     v.Target.HandshakeHash,
			HandshakeStrength: HandshakeStrength(v.Target.HandshakeStrength),
		}
	}
	return packets
}

// lifecycleFor encodes the BINDING status→lifecycle mapping documented on
// Fold. It is the single place that decides State/Hold/HoldReason, so the
// table lives in exactly one spot.
func lifecycleFor(status string, caught bool, openQuestions int) (Lifecycle, HoldKind, string) {
	switch status {
	case "queued":
		return Composing, HoldNone, ""
	case "running":
		return InFlight, HoldNone, ""
	case "done":
		if caught && openQuestions == 0 {
			return Verified, HoldNone, ""
		}
		if openQuestions > 0 {
			return Held, HoldAdvisory, fmt.Sprintf("open questions · %d", openQuestions)
		}
		return Held, HoldAdvisory, "gap found · handshake not tightened"
	case "failed":
		return Held, HoldBlocking, "run failed"
	case "deployed":
		return Delivered, HoldNone, ""
	case "regressed":
		return Held, HoldBlocking, "deployment regression"
	default:
		return Held, HoldBlocking, fmt.Sprintf("unknown state · %s", status)
	}
}
