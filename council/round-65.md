# Round 65 — dispatch → edits → review: tie the work-order loop to the review surface (maintainer steer) — 2026-06-10

Trigger: after the editable review-answer flow shipped, the maintainer steered to a
new thread: "fill a work order, see the agent making edits, then it ties in the
review flow." Connect the DISPATCH economy → the EDITS the work produced → the
REVIEW surface (the Monaco question flow). Full six convened, grounded in the
work-order/dispatch machinery, the diff package, and the review surface.

## Ground truth

- A work order's Target = {BaseRev, FixRev, TipRev, Path, Line}. "Filling" it
  (Spend/FundChosen → drainQueuedOrders → runOneOrder) runs the catch cycle on the
  target. The FIX revision IS "the edits"; there is NO live code-editing agent — the
  edits are the pre-funded base→fix git diff.
- internal/diff.Compute(repoDir, from, to) → Diff{FileDiff{Hunk}} exists (tested),
  but no surface shows the diff.
- runOneOrder DISCARDS res.Findings — the order's review questions are lost; only the
  catch Record + AppendWorkOrderVerdict + status are kept. The connect cycle, by
  contrast, caches res.Findings (setFindings) for /review.
- /review (ReviewCard) shows the session's findings as Monaco + "question:" threads +
  the editable answer flow. Orders render as text spans (renderDispatches), no drill.

## Convergence

- "AGENT MAKING EDITS" = the base→fix DIFF (UX + Refactoring + Systems, unanimous),
  NOT a new editing agent. DATA-HONESTY GUARDRAIL (Systems, binding): the diff is
  STATIC and pre-funded — do NOT theater a "live agent typing"/fake working
  animation; frame it honestly as "the edits this work order made" (show the diff).
- THE LOOP (Game): catch → spend → fund order → watch it fill → REVIEW its questions
  → answer → compound. The missing connector is that the order's findings vanish, so
  there's no quality-check payoff for the work you funded. Capturing them closes it.
  "Watch it fill" beat = the diff revealing / the existing cycle beats (honest), not
  fabricated agent activity.
- PLUMBING (Refactoring + TDD): capture the order's findings (off-ledger, mirror the
  connect-cycle cache) — a per-order findings store; per-order review via
  /review?wo=<id> REUSING ReviewCard + the islands + the answer flow; the diff via a
  Monaco DIFF editor island (the diff DATA payload is the server contract; the editor
  rendering is the client island, browser-verified). diff.Compute already tested.
- FIREWALL (Systems): the order's catch MINTS (existing, on-economy); the order's
  findings + diff + review answers stay OFF the ledger (diagnostic), mirroring the
  connect-cycle findings cache. No new farm/exploit (catch = logged fact, findings =
  unscored diagnostic; the vanishing question is unscoreable).
- COMPUTE (CI/CD): marginal — findings already ride in the cycle result; diff.Compute
  is one git diff, bounded by the existing per-session semaphore. Deterministic
  signals (same revs → same diff/findings); the flaky-fence carries over.

## Slice plan (build over next ticks; tdd-rygba; commit+push; CI; browser-verify renders)

- SLICE 1 — CAPTURE the order's findings + surface its question count. runOneOrder
  stores res.Findings in a per-order cache on liveEntry (off-ledger, like the
  connect-cycle findings); the card's dispatch rows show "N open questions" per
  filled order (the connector: a funded order now shows its review debt). The most
  felt value (Game's "closest win") + server-render-testable.
- SLICE 2 — PER-ORDER REVIEW: /review?wo=<id> reuses ReviewCard to render THAT
  order's findings as "question:" threads + the editable answer flow (re-run scoped
  to the order's revs). The WO# span drills in (href, no JS). vt-testable.
- SLICE 3 — THE EDITS (the diff): diff.Compute(order.Base, order.Fix) → a Monaco DIFF
  editor island on the per-order review ("see the edits this order made" — honestly,
  a static diff). Server-testable diff payload; browser-verified render.
- SLICE 4 (optional, Game) — reveal the diff/beats as the order fills (the "watch it
  work" beat). Bigger; judge after 1–3.

Guardrails: diagnostic-only/off-economy (firewall), data-honesty (the edits are a
static pre-funded diff — never fake "live agent" theater), calm/no fabricated reward,
reachability-grounded, server-tested + client browser-verified. #6 boundary gated.

## Build record — THREAD COMPLETE (slices 1, 2, 3, 2b)

- Slice 1 (19ad88b): runOneOrder CAPTURES the filled order's findings into a per-order
  cache on liveEntry (off-ledger); the card's dispatch rows show "N open questions".
- Slice 2 (0277bf6): per-order review /review?wo=<id> renders THAT order's questions
  as threads; the "N open questions" count drills in. Shared renderQuestionThreads.
- Slice 3 (a323384): "see the edits" — a Monaco DIFF editor of the order's base→fix
  source ("The edits WO#<id> made"), honestly a static diff (orderDiffIsland +
  orderTarget; the {path,base,fix} payload is the server contract).
- Slice 2b (6dd63c5): answer the order's questions IN-PLACE — the editable answer
  pane on the order review, scoped to the order via a $answerwo signal so
  AnswerQuestion re-runs on the ORDER's fix rev and updates the order's findings (a
  kill empties them → the question vanishes, and sticks: the order cache isn't
  cycle-re-populated). Shared renderAnswerForm(anchor, woID); session path = woID 0.

THE VISION IS DELIVERED end-to-end + actionable: catch → spend → fund a work order →
it fills → drill in → SEE the edits it made (diff) → REVIEW its questions → ANSWER
them in place (re-run on the order's revs) → the question vanishes. All off the
economy ledger (the order's catch mints; its questions/diff/answers are diagnostic).
Server contracts unit-tested; the Monaco diff/answer editors are client islands
(browser-verified on :3000).

- Slice 4 (7d685c3): "WATCH IT FILL" — while the runner fills an order, the card shows
  it live ("filling WO#<id> — <beats>"), the cycle beats accruing as the oracle works
  (then vanishing on done). runOneOrder accrues beats into a per-session live-fill
  buffer (startFill/addFillBeat/endFill) instead of discarding them; the card's Stream
  polls the buffer + writes a FillBeats cell to re-render (the runner has no request
  ctx — same poll workaround as the dispatch tally). Off-ledger. The whole loop is now
  FELT end-to-end: catch → spend → fund → WATCH it fill → see the edits (diff) →
  review its questions → answer in place → vanishes.

THREAD FULLY COMPLETE (slices 1, 2, 3, 2b, 4).

## New clashes opened / resolved

None — a clean convergence. "Agent making edits" RESOLVED as the work-order diff (not
a new agent), with a binding honesty caveat against faking live agent activity.
</content>
