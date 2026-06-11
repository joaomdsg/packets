// Package surface renders the review UI as Via compositions. Its first piece is
// the ReviewCard: one anchored oracle
// verdict shown as a distinct, designed state — including the most common
// screen (a fully-tested line) as an affirmative beat, never empty chrome — and
// streamed in over SSE when the verdict resolves. No economy meters live here;
// the surface is built and validated independently of the economy.
package surface

import (
	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/catch"
)

// Tested is the surface verdict for the most common screen: the oracle ran and
// found the line fully constrained (zero surviving mutants). It is NOT a
// catch.Outcome — it is the single-revision "nothing to catch, already strong"
// state, rendered affirmatively so it never reads as empty.
const Tested = "tested"

// ReviewCard renders one anchored oracle verdict. The verdict arrives from the
// orchestrator through the Post action; an empty verdict means the oracle is
// still running (the in-flight state).
type ReviewCard struct {
	Verdict via.StateTabStr
	Sig     via.SignalStr `via:"verdict"`
}

// Post delivers an oracle verdict to the card — the path the orchestrator uses
// once a settle's oracle run (or catch cycle) completes.
func (c *ReviewCard) Post(ctx *via.Ctx) {
	c.Verdict.Write(ctx, c.Sig.Read(ctx))
}

// View renders the current verdict as exactly one designed state. Each verdict
// gets a distinct data-state marker, headline, and detail; the in-flight and
// no-oracle-signal states deliberately carry no success affirmation.
func (c *ReviewCard) View(ctx *via.CtxR) h.H {
	return RenderVerdict(c.Verdict.Read(ctx))
}

// RenderVerdict renders a verdict token as the card's one designed state. It is
// the shared rendering both the reviewer card and the live wire use, so a
// verdict resolves to the same on-screen state however it is delivered.
func RenderVerdict(verdict string) h.H {
	state, headline, detail := present(verdict)
	return h.Div(
		h.Class("review-card"),
		h.Data("state", state),
		h.P(h.Class("review-card__headline"), h.Text(headline)),
		h.P(h.Class("review-card__detail"), h.Text(detail)),
	)
}

// present maps a verdict to its rendered state token, headline, and detail.
// NoOracleSignal and in-flight intentionally read as neutral/working — never a
// success claim — so "the oracle is silent here" can never be mistaken for
// "verified caught".
func present(verdict string) (state, headline, detail string) {
	switch catch.Outcome(verdict) {
	case catch.Catch:
		return "catch", "Caught", "A previously weak line is now constrained by a test."
	case catch.NoCatch:
		return "no-catch", "No catch", "No survivor-set transition on this line."
	case catch.NoOracleSignal:
		return "no-oracle-signal", "No oracle signal",
			"This line has no mutable operator — the oracle cannot speak to it."
	case catch.PartialCatch:
		return "partial-catch", "Partially caught",
			"Fewer mutants survive, but the line is not yet fully constrained."
	}
	switch verdict {
	case Tested:
		return "tested", "Tested — ship it", "Every mutation on this line was killed."
	case LostViaRename:
		return "lost-via-rename", "Anchor lost: file renamed",
			"The file was renamed, so the oracle cannot follow this line across the change."
	case AnchorEdited:
		return "anchor-edited", "Anchor edited",
			"The anchored line was edited, so the oracle can no longer speak to the original line."
	case AnchorDeleted:
		return "anchor-deleted", "Anchor lost: file gone",
			"The anchored file was deleted — or renamed beyond recognition — so the oracle cannot follow this line."
	default: // empty or unrecognized → the oracle is still working
		return "in-flight", "Oracle running…", "Mutating the changed lines and checking your tests."
	}
}
