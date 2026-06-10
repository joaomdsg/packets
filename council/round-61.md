# Round 61 — accessibility landmark pass (R60's queued moderate candidate) — 2026-06-10

Trigger: R60 shipped the strongest moderate slice (fleet merge-readiness summary)
and the full-six council deferred the accessibility / aria pass as the queued next
moderate candidate (the UX north star is literally "keyboard-native, accessible",
so it is VISION-aligned, not padding). The loop continued, so the council acted on
that established decision rather than re-running the identical menu it had just
ranked minutes earlier.

Panelists: the decision was R60's converged outcome; this round SCOPES it under the
three lenses with the strongest opinions on accessibility (carried from R60):
- UX (champion): landmark/semantic structure is the calm-control-room made
  navigable; the live region should announce real state changes.
- TDD (flag): an aria-attr-spray is low-constraint test-theater — the slice must
  assert MEANINGFUL semantics on specific containers (a load-bearing absence today).
- Refactoring (flag): per-span aria across ~30 spans forces builder churn — keep it
  CONTAINER-LEVEL (landmarks + one live region), no per-span churn.

## Decision — SLICE (this round)

A container-level landmark pass (NOT per-span attr-spray), so it dodges both flags:
- nav.go: the shared `<nav>` carries `aria-label="primary"` — a named navigation
  landmark distinct from the content.
- LiveCard.View: the economy (everything below the nav) is wrapped in a `role="main"`
  region marked `aria-live="polite"` + `aria-label="session economy"`. The aria-live
  is the MEANINGFUL bit: the card re-renders over SSE on every catch/balance/dispatch
  change, so assistive tech announces those changes without the user hunting. The nav
  is a SIBLING of main (View now returns `h.Div(navHeader, h.Div(mainParts...))`),
  never nested inside it.
- BoardCard.View: the fleet content is wrapped in `role="main"` + `aria-label="fleet
  board"`, but deliberately NOT aria-live — /board is a request-scoped GET with no
  SSE re-render, so marking it live would lie about its liveness (a data-honesty
  call: the markup must not claim a liveness the surface doesn't have).

Honest-semantics guard in the tests: the board asserts `NotContains aria-live`, so
the static board can never falsely claim the live card's announce behavior.

## Why this is honest + in-guardrails

- Calm: pure semantic structure, no visual/gauge change ("strip the CSS, the truth
  still reads" — here, strip the styling and the landmarks still navigate).
- Data-honesty: aria-live ONLY on the genuinely-live card, never on the static board.
- Two-scores / economy: untouched (pure markup).
- Reachability + server-render-testable: the markup IS the deliverable, asserted
  through the vt client (role/aria substrings genuinely absent before this slice).
- Low churn (Refactoring): container-level only; the View restructure is one inner
  wrapper div per surface, Blue-confirmed non-breaking (data-state/class preserved,
  no duplicate nav, no SSE/mount-id change, CSS has no ancestor-dependent selectors).

## Flagged

MODERATE, by the R60 council's own 5/6 read (peripheral for a tool with no users
yet). Shipped because it is the cleanest VISION-aligned reachable+testable remaining
slice and the loop was directed to keep going. DEEP threads (Trust Ledger backend,
Monaco, keyboard nav, refactor-as-task-type) still need a maintainer steer; #6 live
boundary gated. After this, the genuinely-valuable reachable+testable moderate space
is thin — the next tick should weigh holding for a maintainer steer vs. another
peripheral slice (skeptic gate).

## New clashes opened / resolved

None. A scoped build of R60's queued candidate.
</content>
