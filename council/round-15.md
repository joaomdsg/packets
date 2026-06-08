# Round 15 — cost must be a COUNTED invariant before it is a measured ceiling: the integrated-cost benchmark — 2026-06-04

Trigger: the Round-14 #16 smallest brick BUILT and SHIPPED green (first economy
STOCK rendered — surface.RenderStock, a read-only ledger.ConfirmedCatches
projection). Seventh consecutive build-evidence wave. 14 green packages; the
watchable single-card wire SHOWS the economy (stock/beats/verdict/land rows).

Panelists: all six. No new lens.

New evidence (verified by reading code; cost hazard worse than "3 serial
suites" shorthand):

- A catch cycle (pipe_cycle.go) = runOracleAt(base) + runOracleAt(fix when
  Same/Moved) + integrateOnTip; each oracle run fires `runTests` ONCE PER
  MUTANT (runner.go:243) across maxWorkers=8. Cycle ≈ (M_base+M_fix+1)
  full-suite execs, M of them CPU-saturating.
- `LiveCard.OnConnect` (live.go:93 → ResolveStreaming) fires the WHOLE cycle
  UNCONDITIONALLY per SSE connect with ZERO dedup — no sync.Once/cache/queue/
  Semaphore in live.go. N tabs = N×(M_base+M_fix+1) concurrently; the ≈8N
  moving-tip regime is named in integrateOnTip prose (pipe_cycle.go:163-166),
  unenforced.
- `BenchmarkRunManySites` times ONE oracle run on a 30-site fixture, asserts NO
  invariant — a vanity number, neither the CYCLE nor any K-concurrent regime.
- `liveState` is a package var keyed by NOTHING (live.go:37, set once in
  NewServer); every tab reads the SAME cfg+log. A second agent forces a
  per-session keying rewrite — the real big-bang. live_test.go drives ONE
  connect; NONE pins two-concurrent-connect behavior.

## Per panelist (ALL SIX advocate #15)

- UX: #16 gave the economy its first pixel but a Board without a cost ceiling
  is a thundering herd designed blind. #15 is invisible AS A SCREEN but is the
  gate that turns "how many cards breathe at once" into a measured K_max.
- Game design: stock accumulates but it's one workbench, no shop; Clash D
  unanswerable with one card. Wants the Board but won't build a tycoon shop on
  an unmeasured tick budget — #15 gives the per-action cost.
- Systems: the catch is SHOWN but not SPENT. Pricing (#17) is unblocked on the
  stock axis but has no cost denominator; pricing without the cost-to-mint is
  the cheat the lens prevents. #15 exposes the 3N→8N multiplier as a number.
- Pragmatic TDD: stock bought ZERO cost-safety. Honest first brick is NOT a
  wall-clock bench — it's an atomic suite-exec COUNTER through runTests + a test
  asserting RunCatchCycle fires exactly (M_base+M_fix+1) times. Counted
  invariant before measured ceiling, before any Board rewrite.
- CI/CD & Delivery: per-cycle ~(2M+1) execs, per connect, no queue/cap;
  integrateOnTip names the 8N regime as unenforced prose. #15 is his standing
  #1; building card 2 first ships the 8N multiplier the code warns about.
- Refactoring: second agent is a big-bang rewrite of the untested liveState
  global. Refuses to rewrite an untested global. #15's K-connect harness IS the
  characterization fixture — measure-before-refactor and test-before-refactor
  collapse into one build.

Clashes touched: A (pre-empted in the small by #16's retrospective meters-OFF
stock; full adjudication needs a live meter on a loop); H (calm retrospective
not a dread-gauge — unresolvable until a Board exists); D (still UN-adjudicable
— one card, one package-var liveState; #15 makes a CALM Board designable but
does not touch D — D moves only at the Board brick). No §3 clash flips.

Verdicts updated: none flip; #15 is the gating prerequisite making the Board
(settles D and A-on-a-loop) safe — converts the 8N prose warning into a
measured, regression-guarded ceiling.

New clashes opened: NONE. 6/6 on #15, zero new target-level clashes. The two
divergences (TDD: counted-invariant-before-wall-clock; Refactoring: K-connect
harness is the rewrite's characterization snapshot) are build-order refinements
INSIDE the agreed brick, both ADOPTED into sub-brick ordering.

## Decisions

1. NEXT BUILD (#15, 6/6 — an INSTRUMENTED-INVARIANT brick, NOT a vanity timer):
   the K-concurrent integrated-cost benchmark — counted suite-execs first,
   characterization second, wall-clock ceiling third.
2. PREREQUISITE sub-bricks IN ORDER (each via tdd-rygba): [SB1, RED today]
   thread an atomic suite-exec counter through the cost path (hook at the
   `mutation.runTests` chokepoint, surfaced as a test seam); a test runs
   RunCatchCycle on a deterministic fixture (known M_base/M_fix) asserting
   suite-execs == M_base+M_fix+1 EXACTLY (no sleeps); [SB2, RED today] fire K=3
   LiveCard.OnConnect cycles concurrently via via.WithTestServer, gated on a
   SYNC BARRIER, assert suite-execs == K×(M_base+M_fix+1) AND capture the
   Refactoring characterization snapshot (all K observe the SAME liveState
   cfg/log, all K appends land in the one Log); [SB3] BenchmarkConcurrentCycle
   K∈{1,2,4,8}, b.ReportMetric on suite-execs, p50/p95 wall-clock/cycle,
   ratio-vs-K=1, emitting the K_max where per-cycle time crosses a budget.
3. ACCEPTANCE FIXTURES: [counted invariant]
   TestRunCatchCycle_firesExactlySuiteExecsPerCycle (== M_base+M_fix+1; RED
   today); [uncapped fan-out + characterization]
   TestLiveCard_perConnectFanOutIsUncapped (K=3 behind barrier → counter ==
   K×cycle AND all K read identical cfg/log AND all K records in the one
   ledger); [ceiling] BenchmarkConcurrentCycle (stable printed K_max + ns/cycle
   the Board cap and #17 pricing cite; optionally fail loud if K=1 exceeds a
   declared budget).
4. RANKED ROADMAP: [#15 THIS ROUND] instrumented-invariant integrated-cost
   benchmark; [#16-Board-brick-1] smallest honest Board slice — re-key
   liveState from package var to per-session state behind a SINGLE BOUNDED
   QUEUE enforcing K_max, behavior-preserving, NOT an N-agent big-bang; [#17]
   pricing/conversion (first SPENT scarce resource against the cost-to-mint
   denominator); [#16-Board-later] full multi-card shop adjudicating Clash D +
   A-on-a-loop; [#13] multiset corner; [#11.5] rename-cliff fidelity.
5. BLOCKERS: (a) integrated cost unquantified + uncapped — #15 is the gate; (b)
   a concurrent Board ships the 8N multiplier until #15 sets K_max; (c) second
   agent rewrites an untested global — #15's harness is its characterization
   snapshot; (d) #17 pricing denominator-free until #15. NO VISION/DESIGN text
   changed (12-contradiction reconciliation queued per RISKS step 5).

CONVERGED (11th consecutive round, 6/6): all six advocate #15, scoped (TDD's
sharpening, unanimous) as an instrumented-invariant brick — a counted suite-exec
invariant BEFORE a measured ceiling — with the Refactoring lens's K-concurrent
harness as the characterization snapshot the eventual per-session liveState
rewrite must preserve, folded into SB2. No new target-level clash. The verified
cost model makes #15 the gate both the Board (#16 later, settles D) and pricing
(#17) are blind without. Next event is a BUILD — SB1 → SB2 → SB3 — not another
round.
