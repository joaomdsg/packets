# Round 49 — preview what the Spend funds; reject the near-dead "nothing to fund" affordance — CONVERGED + BUILT — 2026-06-10

Trigger: R48 (per-order round-trip on the card) merged. The Lead can spend and
watch past orders resolve. The standing R49 candidate was a "nothing to fund"
honesty affordance (when balance>0 but the backlog is exhausted, the Spend button
is a silent no-op).

## A constraint that reshaped the round

Grounding in the code showed the "nothing to fund" state is NEAR-UNREACHABLE:
fundableBacklog refills from the session's OWN catches (candidatesFromCatches
derives a Line+1 candidate from every catch), and there is a test literally named
`supplyRefillsFromItsOwnCatchesSoSpendNeverSilentlyDeadEnds`. A session with
balance almost always has fundable work. Building UI for that state would be dead
weight. → REJECTED the original candidate.

Panelists: Calm-UI/Pragmatic-TDD + Producer-experience/Full-user-flow.

## The choice (after rejecting "nothing to fund")

- (A) SPEND PREVIEW — name the actual next target on the Spend control ("Spend a
  catch → fund alpha.go:8"), so the Lead knows what the click funds before
  clicking.
- (B) "DISTINCT WORK LEFT" count on the card (fundableBacklog length, the board's
  "N awaiting" counterpart).

## Decision — CLASH, resolved → CONVERGED on A (built this round)

The personas split: Calm-UI picked B (clean, no R47 disruption), Producer-UX
picked A (closes a blind-action gap). Resolved on A: it adds genuinely NEW
decision-informing data — what the next spend buys — turning a blind verb into an
informed choice, the natural complement to R48's PAST round-trip (R48 shows what
resolved; R49 shows what's next). B's count is more a fleet-comparison signal and
partly redundant with the round-trip already on the card. Calm-UI's objection to A
targeted its weakest part (the near-dead no-target guard), not its core value (the
preview); the testability friction is resolved by extracting a pure label helper.

BUILT (commit ca86a38): `spendButtonLabel(cfg, log)` — a pure helper naming the
target nextUnconsumedTarget would pick ("Spend a catch → fund <path>:<line>"),
falling back to the generic phrasing only when no target is fundable. LiveCard.View
labels the (still balance>0-gated) Spend button with it. The preview reads the SAME
nextUnconsumedTarget the Spend action funds → the label is honest (names exactly
what the click buys).

Load-bearing tests:
1. render — a seeded backlog target [preview.go:42] → the card's Spend control
   reads "fund preview.go:42".
2. pure helper — empty log/config → the generic fallback; a fundable target →
   names it exactly. Both branches locked at the function level (the no-target
   branch is near-unreachable through the UI, so it is proven here).
R47's label assertion updated from the old generic string to the target-agnostic
"Spend a catch → fund " framing (the exact target is owned by the preview tests).
Full gate green; Blue confirmed branch coverage, no regressions, and label honesty.

## New clashes opened / resolved

Clash (A vs B) resolved in favor of A as above. R50+ candidates: (B) "distinct
work left" on the card if it proves wanted; keyboard nav (browser-side,
markup-only — land knowing the limit); more first-run/empty-states. Prefer
server-render-testable, reachable slices — and verify a proposed slice's target
state is actually REACHABLE before building (the lesson of this round).
