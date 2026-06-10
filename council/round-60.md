# Round 60 — next-direction: pick the strongest MODERATE slice (loop relaunch = "keep going") — 2026-06-10

Trigger: R59 surfaced a direction menu to the maintainer and HELD (the high-value
reachable+server-testable space was judged built out R43–R58). The maintainer then
RELAUNCHED the autonomous loop ("orient, convene the council on the next thread,
converge, and build") — which is the "keep going / council picks" steer R59
anticipated. Per R59's standing guidance: the council now picks the STRONGEST
MODERATE reachable + server-render-testable option and builds it (flagged moderate);
DEEP threads (Trust Ledger backend, Monaco, keyboard nav, refactor-as-task-type)
stay OUT until a maintainer steers them.

Panelists present: the full six (UX, Game, Systems, TDD, CI/CD, Refactoring),
re-summoned from README §1, each grounded in board.go / live.go / surface / the vt
test patterns.

The MODERATE menu on the table (from R59):
- (a) accessibility / aria pass — server-testable markup, peripheral pre-users.
- (b) review-thread DELTA-ONLY — only newly-survived mutants; needs a prior-cycle
  findings cache (not built).
- (c) fleet merge-readiness SUMMARY count on the board — small clean extension of
  R58's per-row blocked-land spans ("N of M sessions blocked from landing").
- (d) reanchor-state in the Stock tally — marginal, partly a misread.

## Per panelist

- UX: #1 (c) — thinnest actionable extension of R58's blocked-land surface; honest
  tally (no gauge), reuses BoardRows + the land state hooks. #2 (a).
- Game Designer (dissent): #1 (a) accessibility (kills blank-page anxiety, pure
  hygiene); #2 (c). Flagged delta-only (b) as higher-maintenance (multi-cycle cache).
- Systems: #1 (c) — most honest economy-safe value: a PURE projection off
  BoardRows' real per-session land verdicts, OFF the two-scores economy, no
  fabricated rank/treasury; degenerate-strategy-proof (can't game blocked-count
  without actually fixing a real merge conflict). #2 (a).
- TDD: #1 (c) — strongest test CONSTRAINT: a genuine load-bearing RED (assert the
  blocked-count span renders the correct tally) on the proven R58 land-state vt
  scaffold. Flagged (a) as LOW-constraint (attr-exists tests aren't falsifiable
  without behavior), (b) UNSUPPORTED (needs an unbuilt prior-cycle cache), (d)
  THEATER (redundant disjoint re-assert). #2 (a).
- CI/CD (thread owner): #1 (c) — merge-readiness friction at a glance; CONFIRMED
  reachable purely on the request-scoped /board GET via the in-process land cache
  (NO cross-process /fleet SSE needed — that was R59's blocker for the live half).
  #2 (b).
- Refactoring: #1 (c) — cleanest ADDITIVE build: a pure read of already-cached
  CardRow.Land via one helper paralleling boardLand(), zero churn, no new seam.
  Flagged (a) as structural debt (aria attrs across ~30 spans forces builder churn),
  (b) as new-abstraction (prior-cycle cache + schema). #2 (d).

## Convergence

5 of 6 lenses rank (c) #1; the Game Designer dissents to (a) but ranks (c) #2. That
is a clear majority + reconciled dissent (the accessibility concern is real but
peripheral pre-users and, per TDD, the weakest test constraint of the menu — it is
DEFERRED, not rejected; a future tick can pick it up). CONVERGED on (c).

Reconciled dissent (Game Designer's (a)): deferred as the next moderate candidate;
the merge-readiness summary wins on honesty (Systems), test-constraint (TDD),
reachability-on-/board (CI/CD), and additive-cleanliness (Refactoring).

## Decision — SLICE (this round)

Surface a fleet-level merge-readiness SUMMARY on the board: a calm
`board__land-summary` span reading "N of M sessions blocked from landing", where N =
count of rows whose `boardLand(r.Land).blocked` is true and M = total session rows.
Surfaced ONLY when N ≥ 1 — mirroring the per-row precedent (the per-row land span
shows only when blocked), so a fully-mergeable fleet stays calm (no "0 blocked"
reassurance meter). A pure projection of real per-session integration verdicts,
OFF the two-scores economy, no fabricated rank/treasury, reachable on the
request-scoped /board GET (in-process land cache). data-state hook reuses the R45
honest palette.

Guardrails held: calm (no gauge, silent when all-clear), data-honesty (real land
verdicts, derived count — not a rank), two-scores (diagnostic, off-economy),
reachability (in-process /board GET), server-render-testable (assert the span +
tally through the vt client). #6 live boundary stays gated.

## New clashes opened / resolved

None. A direction checkpoint + a moderate slice; the deep threads remain
maintainer-gated. (a) accessibility is the queued next-moderate candidate.
</content>
</invoke>
