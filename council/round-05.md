# Round 5 — de-risking the right thesis: first economy primitive vs first rendered surface — 2026-06-04

Trigger: first real re-convening since Round 2 (Rounds 3-4 were
evidence-only). Prompted by two RISKS.md meta-findings — (1) "build
order de-risks the WRONG thesis" and (2) "the design's own acceptance
bar would catch none of this" — against 6 green backend packages
(mutation/review/settle/diff/translate/orchestrator), NO end-to-end
pipe, NO UI, NO trust-economy code.

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD,
Refactoring). No new lens.

## New evidence

- Mutation oracle is the keystone, now parallel (maxWorkers=8,
  copy-per-worker), ~30ms/mutant warm, survivors render as question:
  threads. Operator set complete (19 ops). Settle has secret-scan +
  artifact-surfacing + no-edit guard.
- Verified by reading code: review.Thread.Render() returns a string, no
  HTTP/Via/SSE/template anywhere; orchestrator.go:37 diffs against a
  caller-supplied IMMUTABLE baseRev and never reconciles with trunk tip;
  diff.go's TestRenameIsDeleteAddRegardlessOfConfig proves the rename
  cliff is live; QuestionThreadsFromMutations anchors to mutation lines a
  refactor moves wholesale.

## Per panelist

- UX: build is mono-axial — 5 rounds, 6 packages, zero rendered surface.
  Every clash she owns (A Focus-guilt, D scaling, E Bench, H Ledger
  framing, I time-travel) is gated on a screen that doesn't exist. Build
  §17's pipe WITH a real Via/SSE review surface (one card, the question:
  thread on its line, one comment->revision round-trip, no meters). New
  concern: surface has NO defined empty/zero state — the mutation-silent
  case (0 findings, MutantsConsidered>0 = "tested, ship it") is the MOST
  COMMON screen and would read as "broken/nothing happened."
- Game design: rigorous but grading the wrong difficulty — 100% of
  evidence is in the mechanical pipe nobody doubted. You can't FEEL
  anything: no loop, no second agent, no queue to drain. Build §6 slice
  #3 — minimal two-agent Board, queue-to-zero, instrumented day-one for
  idle time and per-review dwell, no meters. Clash D is the load-bearing
  feel question; find the real N ceiling via a human switching between
  two live cards, measuring rework. New concern: the oracle's
  INTERRUPTION RATE per session is an untuned feel-knob — fires-often =
  nag (Clash A relocated to the oracle), fires-rarely = dead weight.
- Systems: sound engineering, miscalibrated economy — five rounds
  polished a SIGNAL GENERATOR; the economy (Focus/Trust/Ship-Quality as
  stocks trading against logged facts) has zero code, zero adversarial
  test. Build confirmed-catch as the FIRST economy primitive: a typed
  append-only Catch{line, survivorSetBefore:nonempty,
  survivorSetAfter:empty, revID, author} per Clash F's survivor-set
  definition, with a degenerate-strategy test suite. New concern: 30ms/
  mutant is a SINGLE-TENANT number — under an N-agent Board, K concurrent
  settles spawn up to 8N concurrent full-suite runs contending for the
  same Treasury+CPU; the economy would price catches against latency
  that degrades superlinearly in the exact regime earned-concurrency
  unlocks. Unmeasured.
- Pragmatic TDD: RYGBA discipline is real (caught restore-swallow +
  vacuous-pass bugs) but pointed at the easy thesis — Rounds 3-5
  hardened a SINGLE-REVISION oracle while the load-bearing thing (does a
  mutation survive across a fix to become a CONFIRMED CATCH) has zero
  code. "A beautiful odometer and no trip counter." Build the
  confirmed-catch oracle as a two-revision differential (survivor-SET
  non-empty->empty on the same anchored line), TDD'd against the
  fix-EDITS-the-anchored-line fixture. New concern: inherits from-base
  re-anchoring (RISKS §28) as a HARD dependency — you cannot define "the
  same line's survivor set across two revisions" without solving
  re-anchoring; the two risks are coupled, neither slice has touched it.
- CI/CD & Delivery: everything built sits UPSTREAM of the integration
  seam — settle mints a revision, orchestrator diffs a FIXED baseRev,
  nothing rebases or re-reads trunk tip. The downstream-truth surface
  (Merged!=Landed, stale-base collisions, fan-out safety) has zero code;
  §27's happy-path trace passes precisely because it never integrates
  anything. Build the integrate-on-tip brick (rebase session branch onto
  trunk tip, run checks on the INTEGRATED tree, tri-state
  clean/conflict/checks-red), TDD'd from the disjoint-file cross-symbol
  break (RISKS line 117). New concern: orchestrator.go takes baseRev as
  an immutable caller string and never reconciles with tip — the
  stale-base assumption is now baked into CODE with no TODO.
- Refactoring: build is sound but scope-misselected — six P0 bricks all
  serve the one work-type the diff-first model already handles (small
  local operator-bearing diffs); zero code touches refactoring, the
  work-type the model is structurally hostile to (Clash G). Run a real
  30+-file rename + extract-module through diff->mutation->thread as an
  ADVERSARIAL trace and assert on the carnage (orphaned threads from the
  rename cliff; mutation re-litigating untouched behavior). New concern:
  QuestionThreadsFromMutations treats behavior-preserving refactor churn
  as MAXIMUM-suspicion surface — the keystone oracle is actively
  miscalibrated for the refactor task-type (inverse of the §29.6
  stamp-penalty inversion, on the oracle side). Not previously captured.

## Clashes touched

- F (re-opened — its IDENTITY half is unresolved, only latency was
  resolved in R3-4).
- B (UX reframes the framing baseline as a RENDER question, not logic).
- D (Game + CI/CD both nominate it as load-bearing but propose opposite
  experiments — human-dwell vs integrated-build-RED).
- G (Refactor — now settleable against the current green tree).
- A/E/H/I (UX + Game: all gated on a rendered/instrumented loop that
  doesn't exist).

## Verdicts updated

- Clash F: downgraded from "MOSTLY RESOLVED" to PARTIALLY RESOLVED —
  latency/feasibility (R3-4) stands, but the catch-IDENTITY definition
  (survivor-set transition vs "same mutant killed") has zero code and is
  the live gate for the entire trust economy (RISKS §29.3 HIGH). Coupled
  to the unbuilt from-base re-anchoring (§28).
- Clash D, G, A, E, H, I: remain TBD, but their settling experiments are
  now named as concrete adversarial traces / a rendered slice rather
  than arguments.

## New clashes opened

- Render-camp vs economy-camp on what the next gate IS: feel/scaling
  (UX+Game, needs a visible loop) vs economy-integrity (TDD+Systems,
  needs the logged primitive first). CI/CD + Refactor sit between:
  adversarial traces that need neither a human nor an economy.
- Three new CODE-level risks (none yet in RISKS.md as code observations):
  oracle latency under fleet contention (Systems);
  QuestionThreadsFromMutations miscalibration on behavior-preserving
  churn (Refactor); orchestrator immutable-baseRev stale-base gap
  (CI/CD).
- Two render-only concerns that only surface once a screen exists: the
  undefined empty/zero state (UX) and the untuned oracle interruption
  rate (Game).

## Decisions

1. NEXT BUILD (ranked #1, two-lens convergence TDD+Systems): the
   confirmed-catch oracle — a logged, append-only, two-revision
   survivor-set non-empty->empty Catch event on the anchored line, NEVER
   "same mutant killed", TDD'd against the three-case fixture (test-only
   fix / fix-edits-anchored-line / fix-adds-branch); the
   fix-edits-anchored-line case is the killer that proves "same mutant
   killed" incoherent. First acceptance-suite entry per meta-finding 2.
2. PREREQUISITE sub-brick of #1: from-base re-anchoring (RISKS §28), with
   "lost via rename" surfaced as a distinct state.
3. Then the integrate-on-tip brick (CI/CD) and the adversarial refactor
   trace (Refactor) as the next two adversarial-suite entries — both
   costed against #1 next round.
4. The rendered §17 surface (UX) + two-agent instrumented loop (Game)
   are gated AFTER the catch primitive; the §17 slice MUST ship a
   designed empty/zero state.
5. Add the three new code-level risks to RISKS.md; benchmark oracle
   latency under K-concurrent-settle contention.

NO VISION/DESIGN text changed this round (the doc reconciliation pass
for the 12 contradictions remains queued per RISKS sequencing step 5).
Not converged — one more round to adjudicate render-camp vs economy-camp
and rank the three adversarial bricks.
