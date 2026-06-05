package surface

import (
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/pipe"
)

// LostViaRename is the surface verdict for a quiet card whose anchor was lost
// because the file was renamed — distinct from a genuinely operator-free line,
// so the card never falsely claims "no mutable operator" on a rename.
const LostViaRename = "lost_via_rename"

// AnchorEdited is the surface verdict for a quiet card whose anchored line was
// edited between revisions, so the oracle can no longer speak to the original
// line — again distinct from operator-free silence.
const AnchorEdited = "anchor_edited"

// PresentVerdict maps a catch-cycle's state to the verdict token a ReviewCard
// renders, keeping the surface's quiet states distinct. A bare catch.Outcome
// cannot express them — NoOracleSignal alone has three different causes — so
// the pipe→card seam maps here rather than forwarding the enum:
//
//   - while the cycle is still running → in-flight ("");
//   - a NoCatch line the oracle ran and found fully constrained (no survivors)
//     → the affirmative Tested calm-win, NOT the blind no-oracle-signal;
//   - a NoOracleSignal verdict → split by its Reason into three honest tokens
//     (file renamed / anchor edited / genuinely operator-free) so the card can
//     state a TRUE cause instead of one overloaded, often-false claim;
//   - everything else → the catch outcome's own token.
//
// The returned token is always one ReviewCard.present() discriminates, so the
// composition can never resolve to an undefined on-screen state.
func PresentVerdict(running bool, outcome catch.Outcome, reason pipe.Reason, afterConsidered, afterSurvivors int) string {
	if running {
		return "" // the oracle has not reached a verdict yet
	}
	switch outcome {
	case catch.Catch, catch.PartialCatch:
		return string(outcome)
	case catch.NoOracleSignal:
		switch reason {
		case pipe.ReasonFileRenamed:
			return LostViaRename
		case pipe.ReasonAnchorEdited:
			return AnchorEdited
		default: // ReasonNoMutableOperator (or none): the line truly has no operator
			return string(catch.NoOracleSignal)
		}
	case catch.NoCatch:
		if afterConsidered > 0 && afterSurvivors == 0 {
			return Tested // verified-strong: nothing survives the oracle here
		}
		return string(catch.NoCatch)
	default:
		return "" // unrecognized outcome: neutral in-flight, never a borrowed success
	}
}
