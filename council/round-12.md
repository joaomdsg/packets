# Round 12 — the seam pattern's first reuse: integrate-on-tip makes the catch's base SOUND — 2026-06-04

Trigger: the Round-11 #11 wave BUILT and SHIPPED green (typed orthogonal
pipe.Reason; presenter split; Clash G surface-honesty resolved-in-code; Clash C
seam discipline demonstrated). Fourth consecutive build-evidence wave. Build
state: 13 green packages; ZERO integration primitives (grep
rebase|merge-base|integrate|onto non-test → only the Unintegrated const);
LandState single-valued; RunCatchCycle hardcodes `Land: Unintegrated`
(pipe_cycle.go:103); catch minted on an immutable pre-integration baseRev.

Panelists present: all six. No new lens.

## New evidence

- #11's orthogonal-seam pattern is settled code: `CatchAcross` returns
  `(catch.Outcome, Reason, error)` (pipe.go:45); `pipe.Reason {None|
  NoMutableOperator|AnchorEdited|FileRenamed}` is a typed field on CycleResult
  beside catch.Outcome; ledger mint token untouched (catch.go:20-34 still 4
  economy outcomes, ledger.go:39 ShouldRecord==Catch only). present.go splits
  NoOracleSignal into three TRUE data-states; gate tests green.
- The catch is minted on an UNSOUND base: resolve.go:46 records
  BeforeRev=baseRev (immutable pre-integration); fix oracle runs on fixRev
  directly (pipe_cycle.go:81), never reconciled to trunk tip; LandState is a
  dead single-value const. A fix green in isolation can mint a real Catch on a
  line that conflicts with — or goes checks-red against — trunk tip. "Landed ≠
  Merged" is honestly LABELED but the verdict does not exist.
- Land is INVISIBLE: present.go never reads the Land field, so the one state a
  reviewer must ACT on (rebase/conflict) is the one the calm card cannot show.
- Residuals: four-beat Trace dropped (resolve.go:38 never reads res.Trace) and
  untyped — #14; set-not-multiset keying (catch.go:58-59) — #13; rename-cliff
  coarsening (reanchor.go:141) — #11.5.

## Per panelist

- UX (→ #12): #11 made the card honest on every quiet state, but Land is
  invisible — reviewer sees a verdict on pre-integration coords with zero
  signal trunk moved. Build #12 as a typed Land field surfaced as its OWN
  data-state ROW: clean→quiet/no-chrome, conflict/checks-red→single actionable
  surface; motion only on the real clean→conflict transition.
- Game design (→ #14, lone dissent): the loop has NO felt tempo —
  pipe_cycle.go:64-100 emits a four-beat Trace resolve.go:38 DROPS; live.go
  streams ONE in-flight→resolved patch. Build #14 (typed []TraceEvent{T,Kind,
  Msg} + STREAM each event). SCOPING NOTE (chair-resolvable): #14 must be
  type+STREAM, never type-only.
- Systems (→ #12): #11 added a verdict dimension without diluting the unit of
  account — ratified the binding rule. But the mint is priced against an
  UNSOUND base (resolve.go:46, zero rebase primitives); no pricing (#17) can be
  trusted until the base is the tree that actually integrates. Secondary:
  catch.go:58-59 toSet collapses same-op survivor sites = #13.
- Pragmatic TDD (→ #12): a Land field must reuse the orthogonal-seam template
  (never an Outcome overload). The load-bearing RED is NOT no-conflict and NOT
  clean — it is clean-MERGE-but-checks-red (trunk renamed a symbol the fix
  calls → merges clean, rebased suite FAILS). Mandatory degenerate guard:
  trunk advanced DISJOINTLY → LandClean (proves the verdict CONSTRAINS).
- CI/CD & Delivery (→ #12): "Landed ≠ Merged" is labeled but the verdict
  doesn't exist; pricing sits on a fictional base. Build integrate-on-tip
  {Clean|Conflict|ChecksRed} by rebasing fixRev onto tip and re-running checks
  on the rebased tree, as an orthogonal CycleResult.Land field, built as ONE
  serialized merge-queue lane (not N concurrent rebases — the O(N²)/8N regime).
  Clash C stays open as #12's experiment.
- Refactoring (→ #12): Land is a dead const; catch minted on an immutable
  baseRev never reconciled. My #11.5 rename-cliff is honest-but-coarse and
  RECURS on the rebased tree #12 introduces — #12 subsumes its urgency; #11.5
  becomes fast-follow. Assert each Land case keeps catch.Outcome IDENTICAL to
  the pre-integration run (orthogonality).

## Clashes touched

- C: RESOLUTION RATIFIED via #12 — the typed Land verdict on the rebased tree
  closes "the mint is unconstrained by integration truth", witnessed by the
  clean-rebase-but-checks-red RED; STILL OPEN until that code ships green.
- G: surface half resolved by #11; #12 extends the SAME seam to integration
  (present.go gains a Land row) — completing the honest-card half.
- D/E/H/I: render/Board-gated, unchanged; reached at #16/#17.

## Verdicts updated

- Clash C: RESOLUTION PATH RATIFIED. #12 integrate-on-tip is experiment AND
  fix; closing RED is clean-rebase-but-checks-red, with a disjoint-trunk
  LandClean degenerate guard. Flips to resolved-in-code on the green #12 build.
  catch.Outcome/ledger token stays byte-identical across the orthogonal Land
  seam (the invariant under test).

## New clashes opened

NONE at target level. Two notes, both chair-resolved in-band: (1) Game's #14
scope (type-only mis-scoped → ADOPTED as binding scope-lock: #14 is "typed
Trace + STREAM", gate asserts staged SSE sequence with monotonic T); (2)
CI/CD's one-lane-vs-per-card-rebase → folded into sub-brick 12e.

## Decisions

1. NEXT BUILD (#12, 5/6 converge; Game's #14 dissent is build-order, chair-
   resolved): integrate-on-tip — a typed orthogonal `Land` verdict {LandClean |
   LandConflict | LandChecksRed} on CycleResult, computed by rebasing fixRev
   onto trunk tip and re-running testCmd on the REBASED tree (reusing
   runOracleAt's worktree+checks path incl. Background-ctx cleanup), threaded
   like #11's Reason (NEVER folded into catch.Outcome), AND surfaced as its own
   data-state row. ONE serialized merge-queue lane. Via tdd-rygba.
2. PREREQUISITE sub-bricks IN ORDER: [12a] trunk-tip rebase primitive in
   internal/pipe (rebase fixRev onto tip in throwaway worktree, detect conflict
   from rebase exit); [12b] typed Land enum replacing the Unintegrated const,
   populated from 12a, catch.Outcome/ledger UNCHANGED; [12c] integrated-checks
   re-run on the rebased tree (clean+green→LandClean, clean+red→LandChecksRed,
   conflict short-circuits to LandConflict before checks); [12d] surface the
   Land row (present.go Land branch, separate data-state, LandClean→no chrome,
   Conflict/ChecksRed→one actionable row; RenderVerdict shared); [12e]
   serialize the lane (lock the no-fan-out invariant in code/comment).
3. ACCEPTANCE FIXTURES (real git repos, each asserting catch.Outcome IDENTICAL
   to the pre-integration run): [closing RED]
   TestRunCatchCycle_cleanRebaseButChecksRedYieldsChecksRed (clean-merge but
   rebased suite fails → LandChecksRed);
   TestRunCatchCycle_landsConflictWhenFixDivergesFromTip (conflicting hunk →
   LandConflict, short-circuits before checks); [degenerate guard]
   TestRunCatchCycle_landsCleanOnNonConflictingTip (disjoint file → LandClean);
   TestResolve_cleanLandRendersNoIntegrationChrome;
   TestReviewCard_rendersConflictLandStateAsActionableWithoutClaimingClean.
4. RANKED ROADMAP: [#12 THIS ROUND] integrate-on-tip (sound base, all pricing/
   trust gates on it); [#11.5 fast-follow AFTER #12] rename-cliff (recurs on
   the rebased worktree — do NOT preempt #12); [#13] multiset survivors; [#14
   type+STREAM, near fast-follow] typed Trace + stream the four beats (first
   felt tempo); [#16] first economy stock + two-agent Board (meters ON); [#17]
   catch PRICING against an integrated base (gated on #12 + #15).
5. BLOCKERS: (a) mint's base unsound until #12; (b) Land invisible until 12d;
   (c) rebase MUST be one serialized lane, never N concurrent (12e); (d) #14's
   four beats are produced and dropped — the felt loop waits on #12 so it
   streams beats that mean something. NO VISION/DESIGN text changed
   (12-contradiction reconciliation queued per RISKS sequencing step 5).

## Convergence

CONVERGED (8th consecutive round): 5/6 (UX, Systems, TDD, CI/CD, Refactoring)
advocate #12 integrate-on-tip as #1 from independent vantage points on the SAME
structural reason — the catch is minted on an unsound pre-integration base, and
#11's orthogonal-seam pattern is the proven template Land must reuse. Sole
dissent (Game: #14 typed-trace-first) is build-order, chair-resolved against
#12 on the no-fake-XP principle (honest beats culminating in a verdict minted
on a base that may never integrate makes the loop FEEL honest while staying
economically fictional); #14 follows #12 closely, type+STREAM lock preserved.
No new target-level clash. Next event is a BUILD — 12a→12e in order, gated on
the five fixtures with clean-rebase-but-checks-red as the closing RED.
