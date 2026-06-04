package surface

import "github.com/joaomdsg/agntpr/internal/catch"

// PresentVerdict maps a catch-cycle's state to the verdict token a ReviewCard
// renders, keeping the surface's three quiet states distinct. A bare
// catch.Outcome cannot express two of them, so the pipe→card seam maps here
// rather than forwarding the enum:
//
//   - while the cycle is still running → in-flight ("");
//   - a NoCatch line the oracle ran and found fully constrained (no survivors)
//     → the affirmative Tested calm-win, NOT the blind no-oracle-signal;
//   - everything else → the catch outcome's own token.
//
// The returned token is always one ReviewCard.present() discriminates, so the
// composition can never resolve to an undefined on-screen state.
func PresentVerdict(running bool, outcome catch.Outcome, afterConsidered, afterSurvivors int) string {
	if running {
		return "" // the oracle has not reached a verdict yet
	}
	switch outcome {
	case catch.Catch, catch.PartialCatch, catch.NoOracleSignal:
		return string(outcome)
	case catch.NoCatch:
		if afterConsidered > 0 && afterSurvivors == 0 {
			return Tested // verified-strong: nothing survives the oracle here
		}
		return string(catch.NoCatch)
	default:
		return "" // unrecognized outcome: neutral in-flight, never a borrowed success
	}
}
