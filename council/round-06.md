# Round 6 — adjudicating the next gate: economy-primitive-first vs surface/integration-first, and the catch's undesigned unhappy path — 2026-06-04

Trigger: reconvening after Round 5 named two camps (render: UX+Game vs
economy: TDD+Systems) and three between-bricks (CI/CD integrate-on-tip,
Refactor adversarial trace) but explicitly did NOT converge — "one more
round to adjudicate render-camp vs economy-camp and rank the three
adversarial bricks." Build state unchanged: 6 green backend packages, no
pipe, no UI, no economy code (re-verified in code this round).

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD,
Refactoring). No new lens.

## New evidence (verified by reading code, not argued)

- thread.go:27-28 `Render()` is `t.Tag + ": " + t.Body` — confirmed
  string concat; no http/template/SSE anywhere in internal/.
- orchestrator.go:37 `SettleTurn(..., baseRev, ...)` with line 49
  `diff.Compute(ctx, repoDir, baseRev, res.SHA)` — baseRev is an
  immutable caller string, never reconciled with trunk tip; no TODO.
- runner.go:72 `const maxWorkers = 8`, copy-per-worker — so K concurrent
  settles → up to 8N concurrent full-suite processes.
- grep for Catch/Trust/Focus/Treasury/Ledger across internal/ returns
  ZERO typed events — meta-finding 1 confirmed in code.

## Per panelist

- UX: Dissents on Round-5's ordering (surface gated AFTER catch). The
  render surface needs ONLY the survivor-set state the CURRENT
  single-revision oracle already emits, so surface and two-revision
  catch are NOT a strict sequence — build the §17 Via/SSE surface (one
  card, one pinned question: thread, one comment->revision round-trip,
  no meters) against today's oracle. New concern: the #1 catch primitive
  has NO render for its most-likely outcome — the round-1 fix where the
  survivor-set is STILL non-empty (partial catch); the binary Catch{}
  schema has no event type for it, repeating §27 one layer down.
- Game design: Came in render-camp but will not be a yes-man to it — the
  catch IS the economy spine AND the game's first honest reward beat, so
  economy-first is correct. Narrow friction: mint the Catch WITH the
  self-flag bit recorded (cheap once typed) so Clash B's
  self-flag-vs-mutation correlation can later run on REAL catches. New
  concern: nobody asked what earning a catch FEELS like — a catch on the
  agent's 3rd self-corrected revision of a never-shipped trivial bug is
  participation-trophy XP; the schema logs the survivor-set transition
  but not the counterfactual "would this have shipped," so the reward is
  uncalibrated (flatter vs grade indistinguishable).
- Systems: Build the confirmed-catch as the FIRST economy primitive —
  typed append-only Catch{anchorLine, survivorSetBefore:nonempty,
  survivorSetAfter:empty, revID, author}, defined ONLY as the line's
  survivor-set transition, NEVER "same mutant killed." Ship it WITH a
  degenerate-strategy suite as the first acceptance entry
  (agent-authors-the-killing-test farming case; fix-edits-anchored-line
  incoherence proof; no-op churn must-not-mint). From-base re-anchoring
  (§28) is a hard sub-dependency. New concern: 30ms/mutant is
  SINGLE-TENANT — 8N concurrent full-suites under a busy Board degrade
  superlinearly in the exact earned-concurrency regime; if catches are
  ever priced, that becomes a "time your settles to a quiet Board"
  exploit. Needs a K-concurrent-settle benchmark before any pricing.
- Pragmatic TDD: Build the two-revision differential (oracle on BOTH
  pre- and post-fix revisions of the same anchored line), TDD'd against
  the three-case fixture with fix-edits-anchored-line as the RED proving
  "same mutant killed" incoherent. Carry from-base re-anchoring as the
  in-scope prerequisite. New concern: the catch silently inherits the
  "0 survivors ambiguous" failure at the catch layer — if the pre-fix
  line had MutantsConsidered==0 (operator-free), a real human fix mints
  NO catch, systematically under-crediting operator-free code; the catch
  must be a THIRD explicit outcome (catch / no-catch / no-oracle-signal),
  not binary.
- CI/CD & Delivery: integrate-on-tip should come BEFORE the catch. A
  mutant killed on a stale base says nothing about survival on a moved
  trunk; BOTH catch revisions are computed against the SAME immutable
  baseRev (orchestrator.go), so the catch inherits a SECOND hidden
  dependency — anchor-survives-rebase — that neither §28 (which
  re-anchors base->cur WITHIN the session branch) nor the #1 brick
  mentions. Build integrate-on-tip (rebase onto tip, checks on the
  INTEGRATED tree, tri-state clean/conflict/checks-red), TDD'd from the
  disjoint-file cross-symbol break (RISKS line 117); assert "integrated
  checks go RED," not "no conflict." New concern: logging the first
  economy primitive on pre-integration coordinates bakes the stale-base
  lie one layer deeper.
- Refactoring: Run a real 30+-file rename + extract-module through the
  EXISTING green pipe NOW and assert on the carnage as the first refactor
  acceptance entry — (a) count orphaned threads from the rename
  delete+add cliff, (b) assert mutation fires question: threads on
  behavior-PRESERVING relocated lines, (c) record both as
  expected-failure baselines. Unlike the catch (coupled to unbuilt
  re-anchoring), this needs NO prerequisite, runs on today's tree,
  settles Clash G with evidence, and quantifies the carnage the
  re-anchor work must absorb. New concern: QuestionThreadsFromMutations
  (thread.go:40) turns every non-killed mutant into a thread with no
  behavior-changing-vs-preserving distinction — the keystone oracle
  inverts the refactor stamp-penalty AT THE SOURCE, a code-level
  miscalibration not yet in RISKS.md.

## Clashes touched

- F (identity half still the live gate; now additionally attacked for an
  undesigned UNHAPPY path — three lenses converge that the proposed
  binary repeats §27).
- B (Game/Systems want the self-flag bit captured at mint time; UX wants
  the framing question answered against a real surface).
- C (CI/CD — integrate-on-tip is the experiment that begins to settle
  scoring-on-downstream-truth, and re-argues it should precede the
  catch).
- G (Refactor — now settleable on today's tree, promoted to concurrent).
- D/A/E/H/I (render-camp — still gated on a surface/loop, dissent on
  ordering registered).

## Verdicts updated

- Clash F: remains PARTIALLY RESOLVED, but the #1 DELIVERABLE IS
  REDEFINED — from Round-5's binary Catch{} to a
  TRI-STATE-plus-intermediate primitive: {catch | no-catch |
  no-oracle-signal (pre-fix MutantsConsidered==0) | partial-catch
  (survivor-set still non-empty post-revision)}. The
  survivor-set-transition definition and "never same mutant killed"
  stand and harden; the binary outcome schema does NOT.
- Clash G: still TBD but its settling experiment is promoted from "gated
  after #1" (Round 5) to "run concurrently on today's tree" — it has no
  unbuilt prerequisite.
- Clashes C, D, A, E, H, I: remain TBD; C gains a concrete experiment
  (integrate-on-tip) and a live ordering challenge to the #1 brick.

## New clashes opened

- The catch's UNDESIGNED UNHAPPY PATH (material new finding, three-lens
  convergent): partial-catch (UX), no-oracle-signal third state (TDD),
  missing "would-this-have-shipped" counterfactual (Game). The Round-5
  binary primitive repeats the §27 happy-path-only mistake one layer
  down; #1 must ship tri-state + intermediate outcomes.
- Catch is minted on PRE-INTEGRATION coordinates (CI/CD): a second hidden
  dependency (anchor-survives-rebase) beyond §28 re-anchoring; live
  ordering dispute over whether integrate-on-tip must precede the catch.

## Decisions

1. NEXT BUILD (#1, TDD+Systems convergence, REDEFINED): the
   confirmed-catch oracle as a two-revision differential, survivor-set
   non-empty->empty, NEVER "same mutant killed" — but as a TRI-STATE +
   intermediate primitive {catch | no-catch | no-oracle-signal |
   partial-catch}, NOT the Round-5 binary. TDD'd against the three-case
   fixture (test-only / fix-edits-anchored-line=RED / fix-adds-branch)
   plus the degenerate-strategy cases (agent-authored killing test;
   no-op churn must-not-mint). First adversarial acceptance-suite entry.
2. PREREQUISITE sub-brick of #1: from-base re-anchoring (§28/§14) with
   "lost via rename" as a distinct state. Document the OPEN gap CI/CD
   raised — re-anchoring as scoped does NOT survive an integration
   rebase — on the #1 brick, since #1 is minted on pre-integration
   coordinates.
3. PROMOTED to CONCURRENT with #1 (shares no code, no prerequisite): the
   adversarial refactor trace on today's green tree — assert
   orphaned-thread count + mutation-on-relocated-lines as
   expected-failure baselines. Settles Clash G; quantifies the
   re-anchor carnage.
4. Capture the self-flag bit on every minted Catch while the schema is
   typed (Game+Systems), IF it does not delay #1 — the only path to
   later answering Clash B / flatter-vs-grade on real catches.
5. Then integrate-on-tip (CI/CD) — but its ordering vs #1 is now a LIVE
   disagreement (CI/CD argues precede); revisit next round.
6. Render §17 surface + two-agent loop remain gated after #1, with UX's
   two non-negotiables adopted: a designed empty/zero state for the
   mutation-silent case, and built against the CURRENT single-revision
   oracle's survivor-set state (so not strictly sequenced behind the
   two-revision catch).
7. Add three code-level risks to RISKS.md (oracle latency under fleet
   contention; thread.go:40 miscalibration on behavior-preserving churn;
   orchestrator.go:37 immutable-baseRev gap); run a K-concurrent-settle
   contention benchmark before any catch pricing.

NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass
remains queued per RISKS sequencing step 5). NOT converged — definition
of #1's unit agreed, but ordering (render-camp dissent; CI/CD
integrate-first; Refactor-concurrent) and the newly-opened unhappy-path
scope need one more round.
