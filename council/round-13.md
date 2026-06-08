# Round 13 — the felt loop: typed + STREAMED trace, the council's cleanest mandate (6/6) — 2026-06-04

Trigger: the Round-12 #12 wave BUILT and SHIPPED green (integrate-on-tip; typed
Land verdict; Clash C resolved-in-code). Fifth consecutive build-evidence wave.
Build state re-verified: 13 green packages.

Panelists present: all six. No new lens.

## New evidence

- #12 was the SECOND reuse of #11's orthogonal-seam pattern (Reason, then
  Land), now a ratified design rule. A cycle does 3 SERIAL full-suite runs
  (runOracleAt base + runOracleAt fix + integrateOnTip's check.Run) and
  produces ~6 genuinely-timed beats (settled base → oracle base → settled fix →
  oracle fix → catch → land, appended at pipe_cycle.go:73-75/94-96/109/115).
- BUT the felt loop is dead at the seam: CycleResult.Trace is a flat []string
  of fmt.Sprintf lines (no T, no Kind); resolve.go reads Outcome/Reason/Land/
  After and NEVER res.Trace; live.go flushes ONE in-flight→resolved patch via a
  single Stream poll. The human watches a 100ms spinner snap to a verdict over
  seconds of real work. #12 minted a NEW Land beat and lengthened the wait, so
  MORE honest tempo is discarded — the gap widened, it did not close.
- Residuals (de-prioritized by their owners): #13 SET-not-multiset keying
  (catch.go) mis-files 2-same-operator-survivor→kill-1 as NoCatch — a narrow
  lower-edge corner; #11.5 rename-cliff coarseness (no consumer until the loop
  is felt); integrated-cost multiplier (3 suites/cycle) unquantified — #15.

## Per panelist (ALL SIX advocate #14)

- UX: #12 gave a second honest row (Land) but deepened the felt-time problem —
  the only motion is a time-filling spinner. Build #14: type Trace as
  []TraceEvent{T,Kind,Msg}, surface it through Resolution, stream each beat as
  a REAL state change terminating in the verdict. Fixture: staged SSE sequence
  (monotonic T, Kind order), card stays in-flight through intermediate beats,
  terminal frame == PresentVerdict; sibling Outdated fixture asserts the
  oracle-fix beat ABSENT.
- Game design: ~5-6 genuine beats over seconds, all discarded; the human feels
  none of the integration drama #12 made real. Build #14 type+STREAM (binding
  scope-lock, NEVER type-only): emit each beat on a channel, drain it in the
  existing via.Stream 100ms poll into a new c.Beats cell. Streaming infra
  ALREADY EXISTS (#12) — #14 adds the per-beat write. Gate: beats arrive as
  SEPARATE flushes over wall-clock — must FAIL a type-only refactor.
- Systems: the mint is sound (catch.Outcome byte-identical, Land orthogonal).
  #13's multiset lossiness is REAL but narrow. #14 is #1 because every
  downstream economy primitive — meters (#16), pricing (#17), the K-concurrent
  benchmark (#15) — needs a TIMED typed event stream as substrate; the
  integrated-cost multiplier is currently invisible. #13 unblocks nothing.
- Pragmatic TDD: the current green (single terminal-frame await,
  live_test.go:42) proves the transition fires, NOT that tempo is delivered — a
  type-only Trace refactor would green VACUOUSLY. The gate that makes #14 real:
  collect the ORDERED SSE frame list, assert ≥4 staged beats before the verdict,
  strictly-monotonic T, terminal verdict == PresentVerdict. A typed Trace that
  still snaps FAILS it.
- CI/CD & Delivery: #12 confirmed the cost worry — 3 serial full suites/cycle,
  re-run per SSE connect; the single lane is synchronous, the integrated-cost
  multiplier unquantified (#17 has no cost denominator). #14 type+STREAM the
  existing beats with the terminal beat carrying Land; #14's timed stream is
  the measurement substrate the #15 benchmark needs.
- Refactoring: the felt loop is dead — no tempo to refactor against. #14 is a
  clean characterization-then-retype over the 6 append sites + streaming
  through live.go's Stream loop. #1 NOW: the first felt tempo and the only
  thing that unblocks pacing/meters (#16). #11.5/#13 are accuracy refinements
  with no consumer until the loop is felt — non-urgent.

## Clashes touched

NONE at target level. #14 closes a missing-CAPABILITY gap (felt tempo), not a
doc contradiction (catalogue holds at 12). The Game Designer's two-round
type+STREAM scope-lock is adopted verbatim by all six as the gate definition —
convergence, not a clash. #13 (mint-denominator fidelity) and #11.5
(rename-similarity cliff) are sequenced as fast-follows, not contested as
alternative #1s. D/A/E/H/I remain TBD (Board/meters, #16).

## Verdicts updated

None of the §3 clashes move (F/G/C resolved-in-code; D/A/E/H/I gated on #16).
#14 is a capability build, not a clash settler — its value is unblocking the
downstream economy primitives that DO settle A/D/H.

## New clashes opened

NONE. 6/6 on #14, zero new target-level clashes — exceeds the Round-7 bar.

## Decisions

1. NEXT BUILD (#14, 6/6 — the cleanest mandate of the council): typed +
   STREAMED Trace. Type CycleResult.Trace as []TraceEvent{T,Kind,Msg}, surface
   it through Resolution, stream each beat to the LiveCard as its own SSE patch
   AS the pipe transition lands, terminating in a verdict frame matching
   surface.PresentVerdict. BINDING SCOPE-LOCK: type+STREAM, gate on the staged
   SSE sequence, NEVER type-only — a typed Trace that resolves in one terminal
   snap FAILS the gate.
2. PREREQUISITE sub-bricks IN ORDER: [SB1] characterize-then-retype — lock the
   existing 6-beat ordering with a characterization test on CycleResult.Trace,
   THEN retype from []string to []TraceEvent{T,Kind,Msg}, stamping T at the
   REAL transition with Kind ∈ {settle-base, oracle-base, settle-fix,
   oracle-fix, catch, land}; [SB2] emit each beat on a nil-safe chan<-
   TraceEvent at the real pipe boundary (nil channel preserves the batch API;
   oracle-fix beat conditionally absent on Outdated/LostViaRename); [SB3] carry
   Trace through Resolution, PresentVerdict UNCHANGED, Land orthogonal; [SB4]
   live.go drains the beats channel per-beat into a new c.Beats via.StateTabStr
   cell inside the existing via.Stream poll, terminal frame writes Verdict+Land
   as today; [SB5] surface.RenderBeats renders the streamed Kinds as a 3rd
   orthogonal row (one row never speaks for another).
3. ACCEPTANCE FIXTURES: [A, load-bearing] extend live_test.go — capture the
   FULL ordered tc.SSE() frame list, assert ≥4 staged beats in pipe Kind-order
   with strictly-monotonic T before the verdict frame, card stays in-flight,
   terminal verdict == PresentVerdict + data-state=land-clean (a type-only
   Trace flushing all beats at the end FAILS); [B, separate-flush proof] a
   gated/slow cycle so beats arrive as SEPARATE flushes over wall-clock; [C,
   real-path] an Outdated-anchor fixture asserts the oracle-fix beat ABSENT;
   [D, beats⊥verdict] reuse the clean-rebase-but-checks-red pair so earlier
   beats show a green catch while the terminal land beat is LandChecksRed.
4. RANKED ROADMAP: [#14 THIS ROUND] typed+STREAMED Trace (first felt tempo; the
   substrate every downstream primitive needs); [#13 fast-follow] multiset
   survivor accounting (RED ready: 2 same-operator survivors → kill 1 →
   currently NoCatch); [#15] integrated-cost benchmark (#14's timed stream is
   the measurement substrate); [#16] meters/Board pacing (needs #14's timed
   stream); [#11.5] rename-cliff fidelity (non-urgent); [#17] pricing against
   integrated cost (blocked on #15).
5. BLOCKERS: (a) the felt loop is unobservable+untuneable until #14 — #16's
   meters/Board have nothing to pace against; (b) the integrated-cost
   multiplier is invisible until #14's timed stream exposes it; (c) the gate
   must assert SEPARATE staged flushes + terminal-verdict match, or #14 greens
   vacuously. NO VISION/DESIGN text changed (12-contradiction reconciliation
   queued per RISKS sequencing step 5).

## Convergence

CONVERGED (9th consecutive round, 6/6 — the cleanest mandate of the protocol):
all six land on #14 typed+STREAMED Trace as #1 NOW, with the identical
mechanism and the Game Designer's two-round type+STREAM scope-lock now
unanimous as the gate definition. The convergence is causal: #12 added a Land
beat and lengthened the wait, so MORE honest tempo is discarded — the
felt-loop gap widened. #13/#11.5 are fast-follow accuracy refinements
explicitly sequenced behind #14 by their owners (build-order, not dissent).
Next event is a BUILD — SB1 (characterize-then-retype) → SB5 (the beat row),
gated on the staged-SSE-sequence fixtures with the separate-flush proof as the
load-bearing RED.
