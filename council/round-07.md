# Round 7 — CONVERGED: ordering & mint-scope closed around the agreed #1 — 2026-06-04

Trigger: closing round, charged narrowly to resolve the two ordering
disputes Round 6 left live (CI/CD integrate-first; UX render-gating) and
rank the bricks against the now-settled #1 unit — NOT to re-litigate
settled points (survivor-set tri-state, re-anchoring prereq,
refactor-trace concurrency). Build state unchanged: 6 green packages, no
pipe, no UI, no economy code.

Panelists: all six. No new lens.

New evidence (verified in code):

- `grep Catch|Trust|Focus|Treasury|Ledger internal/` → ZERO typed events
  (meta-finding 1 confirmed).
- `mutation/generate.go` — Finding keyed only by `(Line, Original,
  Mutated)` STRINGS; no stable mutant identity across revisions.
- `orchestrator.go:37/49` baseRev immutable, never reconciled with tip;
  `diff.go:47` `--no-renames` hardcoded; `review/thread.go` a `question:`
  on EVERY non-killed mutant; `Render()` is string concat (no http/SSE).

## Per panelist

- UX: 7 rounds, 0 pixels — render-dissent holds; the §17 Via/SSE surface
  needs ONLY today's single-revision survivor-set state, ZERO new
  backend, so NOT sequenced behind the two-revision catch. Build now, all
  FOUR oracle outcomes as distinct states. NEW: no designed in-flight
  state for "oracle still running" — SSE means a live half-rendered card.
  Clash A now buildable (all-killed silent vs "0 survivors" badge).
- Game: economy-first correct — the catch IS the first honest reward
  beat. RATIFY catch as #1, but mint w/ a "would-this-have-shipped"
  counterfactual proxy AT MINT (un-backfillable), else Ledger born
  inflationary → forbidden model-inferred catch-weight (V§13.5).
- Systems: RATIFY catch-first. NEW non-negotiable: survivor-set has NO
  identity key — across a fix changing the line's operator, before/after
  sets span DIFFERENT operator alphabets, so "non-empty→empty" is
  ill-typed; denominator must be a function of the line's CURRENT
  operator inventory per revision, else no-op churn and fix-edits-line
  are the SAME failure mode.
- TDD: "a beautiful odometer, no trip counter" — RATIFY the catch as #1,
  fix-edits-anchored-line the non-negotiable RED. CONCEDES CI/CD ordering
  "ranks just under #1": both revisions compute against the SAME baseRev
  → a 2nd hidden dependency (anchor-survives-rebase). NEW: 2× oracle
  cost; no-oracle-signal MUST be a first-class third outcome.
- CI/CD: BLOCKS "integrate-after" as a correctness dependency — a mutant
  killed on a stale base says nothing about a moved trunk; §28 re-anchors
  WITHIN the branch, not across a rebase. Build integrate-on-tip
  (tri-state {clean|conflict|checks-red}), assert integrated checks RED.
  NEW: O(N²)/8N contention → design the merge-queue (batch+bisect) in.
- Refactoring: RATIFY the refactor trace as concurrent — runs on today's
  tree, no unbuilt prereq, settles Clash G, QUANTIFIES the re-anchor
  carnage #1's sub-brick must absorb (de-risks #1). NEW: extract-module
  invisible to both halves — diff can't link A→B (no rename), mutation
  re-mutates relocated operators as net-new.

Clashes touched: F (identity half, live gate, hardened 3rd round + new
identity-key requirement); G (settleable concurrently); C (integrate-on
-tip experiment + empirically-resolvable ordering); B (self-flag +
would-have-shipped captured at mint); A (concrete render experiment);
D/E/H/I (render-camp, gated on surface/loop, strict-gating dissent
registered).

## Verdicts updated

- Clash F: remains PARTIALLY RESOLVED; #1 unit CONFIRMED a 3rd round
  (tri-state survivor-set transition, never "same mutant killed") and
  HARDENED — Systems' missing IDENTITY KEY (denominator = line's current
  operator inventory per revision + explicit inventory-change rule) added
  to the #1 spec; no-oracle-signal locked as first-class.
- Clash G: still TBD but experiment CONFIRMED concurrent-on-today's-tree
  and de-risking; thread.go source-level miscalibration logged.
- Clash C: gains the integrate-on-tip experiment + a partly-conceded
  ordering argument; settled empirically by the trunk-moved variant.
- Clashes A, D, E, H, I: remain TBD; A gains its first buildable
  experiment.

New clashes opened: none at target level — convergence on #1 held. The
would-have-shipped mint-scope (Game) and strict-gating-of-surface (UX)
are scheduling sub-disputes inside agreed bricks; Systems' identity-key
finding is an additive sharpening of #1.

## Decisions

1. NEXT BUILD (#1, 5-lens convergence, definition hardened a 3rd round):
   tri-state confirmed-catch oracle — two-revision survivor-SET
   non-empty→empty on the same anchored line, NEVER "same mutant killed",
   {catch | no-catch | no-oracle-signal | partial-catch}. TDD'd against
   the three-case fixture (test-only / fix-edits-anchored-line=RED /
   fix-adds-branch) + degenerate suite (agent-authored killing test;
   no-op churn must-not-mint). NEW non-negotiable: define the survivor-set
   IDENTITY KEY as a function of the line's current operator inventory per
   revision, w/ explicit inventory-change rule. First adversarial
   acceptance-suite entry (meta-finding 2).
2. #1 PREREQUISITE sub-brick: from-base re-anchoring (§28/§14), "lost via
   rename" a distinct state. Document the OPEN gap that re-anchoring does
   NOT survive an integration rebase (CI/CD's 2nd hidden dependency).
3. CONCURRENT with #1 (no shared code/prereq, today's green tree): the
   adversarial refactor trace — 30+-file rename + extract-module; assert
   orphaned-thread count + behavior-preserving-suspicion + extract-module-
   invisibility as expected-failure RED baselines. Settles Clash G.
4. CAPTURE AT MINT while the Catch schema is typed (cheap,
   un-backfillable): self-flag bit + would-have-shipped counterfactual
   proxy — data-capture only, guards against inflationary Ledger /
   forbidden catch-weight; adopt IF it does not delay #1's definition.
5. INTEGRATE-ON-TIP brick (CI/CD): rebase onto tip, checks on the
   INTEGRATED tree, tri-state, merge-queue (batch+bisect) designed in.
   Ordering vs #1 LIVE but RESOLVED EMPIRICALLY: build the
   fix-edits-anchored-line fixture first, then a trunk-moved variant; if
   the transition does NOT survive the rebase, integrate-on-tip MUST
   precede catch pricing.
6. RENDER §17 surface + two-agent loop: gated after #1's definition but
   NOT strictly behind the two-revision oracle (needs only today's
   survivor-set state). Adopt all FOUR designed outcome states + a NEW
   in-flight/streaming state; Clash A's silent-vs-badge experiment falls
   out of it.
7. Add FOUR code-level risks to RISKS.md (oracle latency under fleet
   contention; thread.go behavior-preserving-churn miscalibration;
   orchestrator.go immutable-baseRev gap; survivor-set no identity key /
   ill-typed denominator under operator change). Run a K-concurrent-settle
   benchmark before any catch pricing.

CONVERGED on the #1 build (3rd consecutive round, 5/6 lenses incl.
render-camp conceding economy-first). Residual disputes are ordering/
mint-scope settled WHILE building #1 — notably CI/CD's integrate-first,
which the fix-edits-anchored-line + trunk-moved variant resolves
empirically. NO VISION/DESIGN text changed (12-contradiction
reconciliation pass queued per RISKS sequencing step 5). The council loop
terminates here on genuine convergence; the next event is a BUILD (slice
#1), not another deliberation round.
