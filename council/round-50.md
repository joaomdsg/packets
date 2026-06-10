# Round 50 — color the dispatch round-trip outcome (caught vs missed) — CONVERGED + BUILT — 2026-06-10

Trigger: R49 (Spend preview) merged. A whole-surface UX survey this round found the
single-session card flow substantially complete (R43–R49) and flagged that the
maintainer's "menus / full user flows" mandate has a larger untapped thread —
session management, order diagnostics, fleet actions — but those need new data
plumbing or new economy semantics (e.g. order "retry" conflicts with the
no-re-fund-consumed rule) and a maintainer steer; they are NOT thin reachable
slices. The survey's top pick (order-inspection with "why it missed") failed the
reachability test: the ledger stores no per-order verdict to show.

## The one remaining thin, reachable, in-direction slice

R45 colored the verdict + land STATES in the honest palette, but never the dispatch
ROUND-TRIP: caught and missed orders (on both card and board) rendered as
undifferentiated dim mono text. Coloring them is the natural completion of R45's
per-state honesty — reachable (every done order has a caught/missed outcome),
fully server-render testable, calm, and improves a fleet manager's at-a-glance
read of which funded orders paid off.

## Decision — CONVERGED on it (built this round)

BUILT (commit 9cd05ff): renderDispatches (the shared card+board helper) now tags
each RESOLVED order with a `data-outcome` hook (caught/missed); a queued/running
order gets NO hook and stays neutral until it resolves. style.go colors
`.board-row__dispatch[data-outcome="caught"]` with --pk-confirmed and `[="missed"]`
with --pk-lost — the same honest hues R45 uses, never an alarm red/green; strip the
CSS and the " caught"/" missed" text still reads it.

Load-bearing tests:
1. a session with WO#1 caught + WO#2 missed renders both the per-outcome hooks
   (data-outcome="caught"/"missed") AND the stylesheet's coloring selectors.
2. a QUEUED order carries NO outcome hook (Blue-caught gap) — unresolved work is
   never colored as a catch or a loss before it has an outcome.
The existing board outcome-text test still passes (the " caught"/" missed" text is
unchanged; the hook is additive). Full gate green.

## Honest status + steer flag

This round closes the card/board VISUAL + FLOW thread the maintainer opened at R43.
After R50, remaining card/board work is marginal polish OR a bigger thread
(session-management menus, order diagnostics needing per-order verdict persistence,
fleet-scale actions, the management-sim depth) that touches new data/economy
/architecture — beyond pure visual design, and warranting a maintainer steer on
direction before building. The loop will surface this to the maintainer rather than
manufacture marginal slices.

## New clashes opened / resolved

None. Reaffirms the reachability lesson (R49) as a slice-selection gate, and "prove
it for real" / no-fabricated-signal guardrails. R51+ pending a maintainer steer on
the bigger thread (or genuinely valuable reachable slices if they surface).
