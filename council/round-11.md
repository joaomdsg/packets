# Round 11 — the round the thesis caught the project lying: the served card resolves to a confidently-wrong terminal on rename — 2026-06-04

Trigger: the Round-10 wire (#10) BUILT — third consecutive post-build wave,
charged to confirm/refine the next build against REAL code. Roadmap named
#11 = "the missing TERMINAL lost-via-rename card state (a renamed file spins
'Oracle running…' forever)". Build state: 13 green packages; R10 landed
cmd/agntpr + internal/app.LiveServer (first prod ListenAndServe/SSE),
surface.PresentVerdict, ledger.CatchRecord.

Panelists present: all six. No new lens.

## New evidence — THE VERIFIED CORRECTION

All six independently opened present.go/card.go/pipe.go/pipe_cycle.go; the
#11 forever-spinner premise was WRONG.

- A LostViaRename anchor flows RunCatchCycle → CatchAcross fail-closes to
  `catch.NoOracleSignal` (pipe.go:33-36) → PresentVerdict(running=false) →
  the card RESOLVES (does NOT spin) to `data-state="no-oracle-signal"` /
  "This line has no mutable operator — the oracle cannot speak to it."
  (card.go:66-68). On a rename that is a CONFIDENT FALSEHOOD — a false fact
  with the authority of a settled verdict, the exact confidently-wrong
  terminal the confirmed-catch economy exists to prevent.
- NoOracleSignal is TRIPLE-OVERLOADED: `catch.Detect` returns it for genuine
  empty-inventory (catch.go:52-54); `CatchAcross` fail-closes BOTH
  reanchor.Outdated AND reanchor.LostViaRename to it (pipe.go:33-36). Three
  truths, one token.
- `reanchor.State` (Same/Moved/Outdated/LostViaRename, reanchor.go:24-36) is
  computed at pipe_cycle.go:67, branched on, stringified into the flat Trace
  (line 88), then DROPPED: CycleResult (pipe_cycle.go:31-44) carries Outcome
  but no State/Reason. The presenter is structurally blind to a distinction
  the layer below had in hand (Clash C made concrete).

## Per panelist

- UX: #10 first told the truth WHY a card is quiet, but card now LIES on
  rename. #1: split the triple-overload — thread reanchor.State (or typed
  SignalCause) through CycleResult → PresentVerdict so three quiet states
  render three TRUE details. Copy/data-plumbing fix, not a new spinner.
- Game design: card lies on its BEST-feeling resolution; "honest hooks, no
  lying" is north star. #1: carry reanchor.State onto CycleResult, split the
  verdict — #11 done RIGHT. DEFERS his typed Trace (#14): fixing the lie
  comes before tuning a lying loop.
- Systems: plumb reanchor.State as typed LostReason{none|no-operator|edited|
  renamed} from ra.State and split the presenter — higher-leverage than
  #11-as-scoped (its spinner doesn't exist); real defect is truth-in-labeling
  on a verdict the human ACTS on. Secondary: set-not-multiset keying
  (catch.go:39) un-credits 2-survivor→1-kill as NoCatch.
- Pragmatic TDD: the confidently-wrong terminal is a test that GREENS for the
  wrong reason. #1: typed Reason on CycleResult — a PRECONDITION for #11, not
  separate. KEYSTONE RED: the edited-anchor (Outdated) fixture — forbids
  greening on the NoOracleSignal token alone.
- CI/CD & Delivery: #10 shipped `Land: Unintegrated` (pipe_cycle.go:25),
  "Landed ≠ Merged". His nextBuild is #12 integrate-on-tip. Concedes the
  rename correction PROVES the lossy seam: #12 MUST widen the seam (typed
  Land), not bolt a 4th overload — #11's seam-widening is #12's prerequisite.
- Refactoring: RE-OPENS Clash G (refactor carnage) as live-at-the-SURFACE —
  resolved at the oracle seam but a rename is reported as operator-free.
  #1: #11 RESHAPED — thread reanchor.State, render "Anchor lost: file
  renamed". Validating fixture: an httptest SSE renamed-file case.

## Clashes touched

- C (lossy CycleResult.Outcome seam): MOVED latent → actively-binding
  (ra.State known at pipe_cycle.go:67, dropped) and ELEVATED to a design
  rule — every verdict dimension is an orthogonal typed field, never a new
  meaning on an existing Outcome token.
- G (refactor carnage): RE-OPENED at the surface; closing gate is #11's
  renamed-file surface fixture.
- A (calm-win-vs-guilt): unaffected — happy path is honest.
- D/E/H/I (render-camp): reached at #16/#17, unchanged.

## Verdicts updated

- Clash C → ACTIVELY-BINDING (was latent). Design rule adopted: Reason (this
  round) and Land (#12, stubbed pipe_cycle.go:21-25) are orthogonal typed
  fields on CycleResult, never overloads.
- Clash G → RE-OPENED at the surface (was resolved-in-code). Fail-closed-at-
  oracle is correct AND insufficient. Closing gate: #11's httptest SSE
  renamed-file fixture asserting the card does NOT claim "no mutable
  operator".

## New clashes opened

NONE at target level. Three proposed, all self-declared "none": TDD's is a
reframe of #11's RED; CI/CD's sharpens Clash C; Refactoring's re-opens G at a
finer layer. Re-opening at a finer layer is not a new target-level clash;
convergence stands.

## Decisions

1. NEXT BUILD (#11, RESHAPED, 6/6 on the seam-fix; 0/6 ratify #11-as-scoped):
   split the NoOracleSignal triple-overload at the pipe→surface seam. Add a
   typed `Reason` (orthogonal to Outcome) to CycleResult distinguishing
   NoMutableOperator (catch.go:52-54) / AnchorEdited (reanchor.Outdated) /
   FileRenamed (reanchor.LostViaRename); replace card.go:66-68 with a branch
   rendering each cause TRUE — rename case must NOT say "no mutable operator".
   Outcome + ledger token unchanged. Via tdd-rygba.
2. PREREQUISITE sub-brick A, BUILD FIRST: widen the seam type. Carry the cause
   out of `CatchAcross` itself (pipe.go), not inferred by the caller. Populate
   CycleResult.Reason in RunCatchCycle from ra.State AND the genuine-no-
   operator case. RED-then-GREEN at internal/pipe.
3. PREREQUISITE sub-brick B, BUILD SECOND: presenter split. PresentVerdict
   (present.go) gains Reason, maps each no-signal cause to a distinct verdict
   token; ReviewCard.present (card.go) renders true details. SIBLING tokens —
   do NOT add a fourth meaning onto NoOracleSignal. #12's Land MUST reuse this
   widened-seam pattern.
4. ADVERSARIAL ACCEPTANCE (real git fixtures): (a)
   TestRunCatchCycle_renamedAnchor_reasonFileRenamed — git mv above
   --find-renames threshold → Outcome==NoOracleSignal AND Reason==FileRenamed;
   (b) TestRunCatchCycle_editedAnchor_reasonAnchorEdited — KEYSTONE RED,
   Reason==AnchorEdited; (c)
   TestRunCatchCycle_literalOnlyLine_reasonNoMutableOperator; (d) surface/
   httptest SSE renamed-file case — distinct data-state, rename-naming detail,
   NOT-contains "no mutable operator" (Clash G's closing gate).
5. RANKED ROADMAP to a USABLE PROTOTYPE: [#11 THIS ROUND] split the
   NoOracleSignal triple-overload (typed Reason + presenter split; edited-
   anchor keystone RED); [#11.5 fast-follow] tighten reanchor rename detection
   OR make FileRenamed copy admit threshold-uncertainty; [#12 integrate-on-tip]
   replace Unintegrated const with {clean|conflict|checks-red} Land, REUSING
   #11's widened-seam pattern; [#13 multiset survivor accounting] catch.go
   SET→multiset (2-survivor→1-kill mints PartialCatch); [#14 typed Trace]
   []TraceEvent{T,Kind,Msg}; [#15 K-concurrent benchmark]; [#16 first economy
   stock + two-agent Board — meters ON]; [#17 catch PRICING, gated on #12+#15].
6. BLOCKERS: (a) the served card asserts a confident falsehood on every
   rename/edited-anchor — highest-priority defect; (b) CycleResult is blind to
   reanchor.State until Reason is a field; (c) seam-widening discipline must
   land in #11 so #12's Land doesn't re-introduce the overload.
7. RISKS.md: log the surface confidently-wrong-terminal; set-not-multiset
   under-crediting + rename-similarity-cliff stand; integrated-cost multiplier
   and the four R5-7 code-level risks stand. NO VISION/DESIGN text changed
   (12-contradiction reconciliation queued per RISKS sequencing step 5).

## Convergence

CONVERGED (7th consecutive round): 6/6 independently found the SAME defect — a
renamed file resolves to a calm, terminal, confidently-WRONG "no mutable
operator" verdict — and the SAME #1: thread reanchor.State through CycleResult
as a typed orthogonal Reason, split the triple-overload at the presenter. 0/6
ratify #11 AS-SCOPED (forever-spinner) — reshape, not clash. Sole dissent is
build-ORDER (CI/CD: #12 first), chair-resolved for #11 by CI/CD's own argument
— #11's seam-widening is #12's prerequisite. Clash C → actively-binding rule;
Clash G re-opened at surface with #11's renamed-file fixture as closing gate.
Next event is a BUILD — sub-brick A then B, edited-anchor as keystone RED.
