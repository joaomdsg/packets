# Round 56 — the REVIEW-THREAD SURFACE thread, full 6-member council — RATIFIED — 2026-06-10

Trigger: the maintainer authorized the council to creatively steer to feature-
completion without further input, and corrected an under-convening: this round
re-summons the FULL six panelists (§1) — UX Designer, Game Designer, Systems/Economy
Designer, Pragmatic TDD Expert, CI/CD & Delivery Expert, Refactoring Expert — not a
partial 2-voice shortcut.

## The thread

Surface the mutation oracle's surviving/undetermined mutants as actionable
"question:" review comments — the TDD Expert's registered bold swing ("surviving
mutants become `question:` threads — 'green is a lie here'"). internal/review/
thread.go models it (Thread + QuestionThreadsFromMutations) but is UNWIRED; the
per-line finding messages die at the pipe layer (catch.LineStateAt consumes
res.Findings to derive the survivor SET then discards them; pipe.CycleResult +
app.Resolution carry no findings).

## Decision — RATIFIED by all six, with refinements + two settled clashes

Completion path (each slice via tdd-rygba, ship to main, watch CI):
1. PLUMB findings up — add Findings to pipe.CycleResult (the FIX revision's oracle
   findings) → app.Resolution.
2. RENDER question-threads — GATED (only when survivors > 0), CALM (a card badge/
   count, not full inline clutter on the already-dense card), full anchored threads
   on a server-rendered /review route (NOT Monaco). Filter to non-killed at the
   convert layer.
3. PERSIST as a NEW diagnostic ledger kind, OFF the two-scores economy.
4. Reviewer intent tags + the mastery refinements (delta-only, vanish-on-killed).

### Clashes settled
- WHICH revision's findings (TDD said base, Refactoring said fix) → FIX. The fix's
  still-surviving mutants are the OPEN questions, stamped at the current/reanchored
  coordinates (anchor-correct). A baseRev survivor the fix KILLED is not a question —
  it is the catch itself. fixRev findings are nil when the anchor was Lost/Outdated
  (no fix oracle ran), so threads correctly suppress on a lost anchor.
- WHERE questions render (UX: not on the dense card) → a gated card badge linking to
  a dedicated server-rendered /review surface; Game Designer concurs (surface with
  the verdict beat, delta-only, resolution = the mutant dying → a real mastery loop).

### Unanimous guards
- Systems/Economy: a persisted QuestionRecord is a NEW diagnostic kind, skipped by
  Log.Records(), never minting/scoring/spending (two-scores intact, like R51's
  woverdict). No quantity-farming, no answer-reward.
- CI/CD: questions are DIAGNOSTIC-ONLY, NON-GATING (the catch still ships); carrying
  findings is free (already computed), but measure ledger/SSE payload at slice 3.
- Refactoring: render only on a surviving anchor (Same/Moved) — natural since fixRev
  findings are nil otherwise; the finding coords are the oracle-execution (reanchored)
  revision's, so they are honest under line-shifting refactors.
- TDD: fix the QuestionThreadsFromMutations filter (convert only non-killed) in the
  convert layer (slice 2); the Survived-vs-Undetermined distinct-tag decision is
  deferred to the render/tags slice.

## Slice 1 (building now)

pipe.CycleResult gains `Findings []mutation.Finding` (the fix oracle's findings, set
in the ra.Same||ra.Moved branch; nil when the anchor is lost); app.Resolution gains
`Findings`, mapped in ResolveStreaming. Load-bearing test (pipe integration, real
oracle fixture): a cycle whose FIX leaves a surviving `>=` mutant carries it in
CycleResult.Findings — the findings no longer die in the cycle.

## Slice 2 (built, commit 79a2151)

app.Resolution gained Findings (a pass-through of CycleResult.Findings). LiveCard
gained a Questions cell; OnConnect writes the open-question COUNT (len of the fix
oracle's non-killed findings) after the connect cycle; View renders a GATED, calm
".review-questions" badge ("N open question(s) — the oracle found unkilled mutants
the tests didn't catch") only when count > 0. Per the UX Designer's guidance the
card shows the COUNT ONLY (a humble summary), not the full threads. Tests (SSE):
survivors → the badge with its count + class; a clean verdict → no badge.

Re-sequencing note: a []Finding can't ride a scalar via state cell, so the card
carries only the count. The full anchored threads need the findings RETRIEVABLE,
which requires PERSISTENCE — so slice 3 (persist findings as a diagnostic ledger
fact, off the two-scores economy per the Systems guard) must precede the /review
full-thread surface (slice 3b/4), inverting the council's original render-then-
persist order.

## Slice 3 (built, commit 7b471ff)

The /review surface — the full anchored "question:" threads the card's badge only
counts. ReviewCard (mounted at /review) renders sessionOpenThreads(key): each
finding as a .review-thread anchored File:Line + "question: <message>", or a calm
empty state. The card badge is now a link to it. Threads come from a per-session
in-memory findings cache on the liveEntry (written by the connect cycle,
mutex-guarded wholesale-replace; read by the /review GET) — EPHEMERAL diagnostic
state, recomputed each connect, deliberately OFF the economy ledger (the Systems
guard: a question never mints/scores/spends). Because it is the LATEST cycle's
findings, answering a question makes it vanish next cycle (the Game Designer's
mastery loop) with no scored transaction. Unknown keys fall back to default
(consistent with the card/stream/claim surfaces). Blue: concurrency-safe (slice
replaced wholesale, read under the lock), no dead code, no badge regression.

The review-thread surface thread is now FEATURE-USABLE: a verdict's green that
hides surviving mutants shows a gated count on the card → click → /review reads the
full open questions. Remaining (optional polish): reviewer intent tags
(question→blocking/suggestion), and ledger PERSISTENCE if cross-restart history is
wanted (deferred — the open-questions view needs only the latest cycle).

## New clashes opened / resolved

Clashes 1 (base vs fix) + 2 (where to render) resolved above. None left open.
