# Round 52 — render the per-order verdict (order-diagnostics thread, slice 2) — CONVERGED + BUILT — 2026-06-10

Trigger: R51 persisted the oracle's per-order verdict as a ledger fact
(DispatchView.Verdict), but nothing rendered it yet — the data was captured,
invisible. This slice makes the WHY visible, the natural slice-2 of the
order-diagnostics thread the council chose at R51.

Panelist: a single focused pass (the thread + slice were already set at R51; this
is the render half of the same idea — a small, well-understood continuation).

## The slice

A resolved work-order rendered only "done caught" / "done missed" (with R50's
caught/missed color). But a MISS has kinds — no-catch (the oracle ran, nothing to
catch), lost-via-rename (the anchor moved), no-oracle-signal, anchor-edited — and
the Lead couldn't tell them apart. R51 made the oracle's verdict available per
order; R52 surfaces it.

BUILT (commit 8386a44): the shared renderDispatches helper (card + board) appends a
`board-row__dispatch-why` span carrying the persisted verdict for any DONE order
that has one — "WO#2 beta.go:9 done missed no-catch". Styled dim (--pk-ink-dim) as
calm secondary detail: the outcome word already carries the R50 color; the verdict
is muted reinforcement, never an alarm. An order with no persisted verdict (queued,
or pre-R51 data) renders NO why element — never an empty tag implying the oracle
said something it didn't.

Load-bearing tests:
1. a done+missed order with a persisted verdict "no-catch" renders the verdict text
   AND its own board-row__dispatch-why hook (scoped via bodyOf).
2. an order with no persisted verdict renders no why element (omit path).
The verdict-less existing dispatch tests (board + card round-trip) are unaffected —
they persist no verdict, so the why span never appears. Blue confirmed the
`done && verdict != ""` guard is load-bearing (a verdict shows only on a resolved
order, future-proofing against earlier persistence), no regressions, calm color.
Full-repo gate green.

## New clashes opened / resolved

None. The order-diagnostics thread now reads end-to-end on the surface: spend →
dispatch → watch it resolve caught/missed (R48/R50) → SEE WHY (R51/R52). R53+
options: a drill-in order DETAIL view (a dedicated page per WO# — needs a route +
is it worth it vs the inline why?); OR pivot to the session-menu thread (begin with
the lazy-consumer refactor so runtime-created sessions are reachable). Council to
weigh next tick. Guardrails + reachability gate stand.
