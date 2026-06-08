# Round 22 — the closed toy gets fuel: a work-source backlog so the loop compounds more than once (+ harden the runner) — 2026-06-05

Trigger: the Round-21 #16e wave BUILT and SHIPPED green (the spend-to-earn
loop closes: identity-dedup anti-farm gate RED-first; a funded order RUNS
in-process as the first second producer; two-producer Producer demux;
queued→running→done watchable over SSE; the audit caught + fixed a runner
beats-discard goroutine leak). Fourteenth consecutive build-evidence wave.

Panelists: all six. No new lens. CONVERGED 5/6 on WORK-SOURCE with the
runner-termination guard bundled as a non-negotiable rider; a build-ORDER
dissent (harden-first vs work-source-first vs the UX provenance-split)
chair-resolved by SCOPE — NOT a target clash. #16f deferred with unanimous
assent.

Shared diagnosis: #16e closed the loop in DATA but every lens converged on
the same revealed truth (verified in live.go): the loop is a CLOSED TOY
fuelled by a SINGLE hand-configured cfg.DispatchTarget (live.go:38). The
anti-farm identity-dedup correctly mints 0 on the SECOND dispatch of that one
target (silent no-op, live.go:205), making "compounds" true exactly ONCE and
structurally DEFLATIONARY thereafter (a full suite fan-out burned per spend
for zero). The trust-economy bricks (calibration/the-bet/Focus/tiers) have no
live multi-cycle economy to calibrate against until fixed. Secondary live
liability: the dispatch runner can spin a CPU-bound suite loop FOREVER on a
single permanent status-write error (closed handle) under a held runMu —
uncapped #15-multiplier burn (drainQueuedOrders for{} at live.go:243-249,
discarded AppendStatus errors at 253/271).

## Per panelist

- UX (→ stock-provenance split, the 6th pick): #16e closed the loop in data
  not in SIGHT — RenderStock shows a flat aggregate, so a dispatch-minted
  ("wo:<id>") catch is indistinguishable from a connect-minted one. #1: extend
  ledger.Stock with Reinvested (Producer prefix "wo:") + a RenderStock span.
  [Chair: ranked #2 — meaningful only once a backlog gives a SECOND distinct
  mint to display.]
- Game design (→ work-source): the loop is a closed toy; one target → one
  productive dispatch, then every spend is a dedup'd honest loss. #1: a
  ledger-backed backlog of ≥2 distinct Targets; Spend pulls the next
  UNCONSUMED head (FIFO); exhaustion REFUSES with explicit "no work" (honest
  scarcity). Fold the runner-harden in as a cheap rider.
- Systems / Economy (→ work-source): a bounded FAUCET — cfg.DispatchBacklog
  []Target consumed head-first, projected purely from the log; total mint ties
  to DISTINCT targets consumed, never to dispatch ACTS; exhaustion is a legible
  refusal, not a silent 0-mint.
- Pragmatic TDD (→ harden-first, build-ORDER dissent): runOneOrder must stop
  discarding AppendStatus errors + break on failure; drainQueuedOrders must
  never re-run an id twice. Don't widen the loop before it's safe. Rejected
  #16f. [Chair: honored by SCOPE — bundled as a rider with its own RED.]
- CI/CD & Delivery (→ harden-first, build-ORDER dissent): same termination
  harden; on a status-write error break the drain loop + record the order
  failed/abandoned via an append-only terminal line. Rejected #16f-first.
- Refactoring (→ work-source): a NextTarget seam deriving the next fundable
  distinct Target from the existing log, NOT #16f; the work-order's
  self-describing Target+Producer is already the seam a backlog AND the triage
  graph want (schema-free reuse, NATS unearned).

## Clashes / risks touched

Resolves the #16e KNOWN LIMITATION "no real WORK SOURCE / compounding depends
on a configured target." CLOSES DISPATCH-RUNNER NON-TERMINATION via the
bundled rider (break-on-status-error + per-order max-attempts). BOUNDS the #15
cost multiplier (a finite drawn-down backlog caps real fan-outs at
len(backlog); the attempt-cap removes infinite re-run — strictly safer).
Forces the terminal failed status to live as an append-only line (no silent
CRUD mutation). SEEDS leverage-needs-a-dependency-graph: a per-card backlog of
funded Targets is the node-with-target set the triage queue was blocked on.
Does NOT touch the SECURITY TRIO (shim/egress/secret-scrub) — all gate #16f
together, correctly deferred.

## Verdicts updated

None flip; work-source turns one-shot compounding into a draw-down-able supply
(a second honest distinct mint + real scarcity), and the rider makes the
just-shipped runner safe to fuel.

## New clashes opened

NONE. No lens claims the security trio is discharged or disputes that
work-source is a legitimate honest in-process target. The only contested axis
is build-ORDER (work-source-first vs harden-first vs UX provenance-split),
dissolved by bundling the harden into the work-source slice; #16f-first is
explicitly rejected by every lens.

## Decisions

1. NEXT BUILD (work-source + runner-termination rider, CONVERGED 5/6):
   replace the singleton LiveConfig.DispatchTarget with an ordered backlog of
   ≥2 DISTINCT ledger.Targets that Spend consumes head-first (consumption
   projected purely from the JSONL — no in-memory head pointer, survives
   reopen); an exhausted backlog REFUSES Spend with an explicit "backlog
   exhausted / no distinct work" reason (balance unchanged, NO queued order
   appended); BUNDLED RIDER: runOneOrder stops discarding AppendStatus errors
   and breaks on a status-write failure, and drainQueuedOrders enforces a
   per-order max-attempts cap so a permanently-failing status can never re-run
   the suite fan-out unboundedly.

2. ACCEPTANCE FIXTURES (hard RED, fully in-process, no security gating): seed
   a session backlog [T1,T2] (distinct identities) + balance 2 → Spend#1 funds
   T1 → runs → mints C1; Spend#2 funds T2 (NOT T1) → runs → mints DISTINCT C2
   (total mint ties to distinct targets consumed); Spend#3 on the exhausted
   backlog is REFUSED with the explicit reason, balance unchanged, no queued
   order appended; a fresh reopen reproduces the same consumption projection
   (head is log-derived). Companion RED for the rider: a permanently-failing
   "done" AppendStatus (closed-handle stub) + an invocation-counting
   resolveCycle → drainQueuedOrders RETURNS within a bounded attempt count, the
   order is NOT in QueuedWorkOrders afterward (append-only terminal failed
   line), and resolveCycle fires at most max-attempts times (today it spins
   forever — the hard RED).

3. RANKED ROADMAP: [work-source + rider THIS WAVE]; [UX provenance split #2]
   RenderStock connect-minted vs reinvested (Producer "wo:" prefix) — now
   meaningful because the backlog gives a second distinct mint, makes
   compounding LEGIBLE; [triage/attention-queue #3] unblocked by the per-card
   funded-Target backlog (the node-with-target the dependency-graph signal
   lacked); [#16f cross-process producer + the security trio #4] gate together,
   heaviest, taken only AFTER the in-process termination invariant is proven;
   [#13 multiset], [#11.5 rename-cliff]; trust-economy bricks — 8/15 risks,
   need a live multi-cycle economy to calibrate against.

4. BLOCKERS: (a) the configured-target singleton makes "compounds" true only
   for n=1 — at n≥2 the economy is a deflationary honest-loss treadmill; the
   trust-economy bricks have no live multi-cycle economy to calibrate against;
   (b) the runner can spin a CPU-bound suite loop forever on one permanent
   status-write error under a held runMu — not deployable, addressed by the
   bundled rider; (c) #16f stays blocked until the in-process termination
   invariant is proven (wrong to carry a still-spinnable runner across an
   OS-process boundary where runMu can't reach it).

CONVERGED (18th consecutive round, 5/6 on target): #16e closed the loop in
DATA but every lens converged on the same truth — the loop is a CLOSED TOY
fuelled by a single hand-configured cfg.DispatchTarget, so the honest
anti-farm dedup mints 0 on the second dispatch (silent no-op), "compounds"
true once and deflationary thereafter. The work-order's self-describing
Target{base/fix/tip,path,line,lineHash}+Producer is already the seam a backlog
AND the triage graph both want, so the slice is schema-free reuse (NATS
unearned — one file, one l.mu). The smallest testable slice replaces the
singleton with an ordered ≥2-distinct-target backlog that AppendDispatch
consumes head-first as a pure log-derived projection (survives reopen), pays
back TWICE, then HONESTLY EXHAUSTS with an explicit refusal — turning the
silent dedup loss into a legible scarcity signal. The contested build-ORDER
dissent is honored by SCOPE not sequence: the runner-termination guard lands
INSIDE this slice as a non-negotiable rider with its own hard RED, before the
loop widens and long before any cross-process #16f drainer. The bounded
backlog strictly improves the #15 cost posture; the funded-Target node set
unblocks the rank-2 UX provenance split and the rank-3 triage queue. #16f and
its security trio stay deferred with unanimous assent. The next event is a
BUILD — backlog []Target + head-consumption projection → exhaustion refusal →
the runner-termination rider (break-on-status-error + max-attempts).
