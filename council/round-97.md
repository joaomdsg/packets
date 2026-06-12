# Round 97 — order-loop slice B-iii: a live producer that listens as you type — 2026-06-12

Trigger: the authoring assist (R96) ran only on a button press and its readiness
verdict sat in its own panel, disconnected from the place decision. The Lead asked
for a producer that listens AS the draft is written and helps shape a clear,
verifiable goal. This slice makes the read live and ties its verdict to the place
control.

## The change

The composer's draft textarea gains live debounced re-analysis: a small
progressive-enhancement script (`liveAnalyzeJS`) watches input and, on a typing
pause, triggers the proven `AnalyzeDraft` action by clicking its button — so the
producer keeps pace with the draft WITHOUT a second server seam (the analyze path,
its firewall, and its tests are unchanged). A calm `compose__analyzing` indicator
flips to pending/analyzing while a re-read is in flight; with JS off the manual
button still analyzes. A dataset guard keeps the wiring from re-binding across SSE
re-renders.

`renderComposeWithAnalyze` now takes the cached analysis and, once the producer has
read the draft, reflects its readiness verdict beside place: `ready` (the producer
judged it safe to run unattended, in the balance hue) or `caution` (open questions
flagged, dim). It is a GUIDE, never a gate — placing stays allowed at any
readiness, so the verdict informs the decision without seizing it.

## The tests

`internal/app` (NOT parallel — shared globals): the composer carries the live
debounced re-analysis wiring and the analyzing indicator; the place control
reflects the producer's readiness — `ready` for a ready draft, `caution` for a
not-ready one (table-driven). The existing analyze/place/firewall tests are
unchanged and still green.

## Verdict

Full repo green with `-race`, vet clean. The producer now listens as the Lead
types and its readiness verdict guides the place decision — the live co-author
shape the Lead described, grounded on the proven analyze seam (no new economic or
process surface). Slice B is complete: type → live read → highlights + questions +
readiness → sharpen → place.

## New clashes opened / resolved

Resolved: the authoring read is live (debounced on a typing pause) and its
readiness verdict informs place — closing the R96 "on-demand only" caveat. Open:
the live trigger is a client-side button-click bridge (browser-verified, not
unit-tested, like the other Monaco islands); the debounce interval (900ms) is a
flat constant a future slice could tune; and inline per-question reply threading
(answer a question and have it fold into the draft automatically) remains a
possible refinement over the current edit-and-re-read loop.
