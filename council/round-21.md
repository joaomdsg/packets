# Round 21 — execute the order: the first second producer, gated by an identity-idempotent mint so the loop compounds without farming — 2026-06-04

Trigger: the Round-20 #16d wave BUILT and SHIPPED green (AppendDispatch funds
one queued work-order atomically with the debit; monotonic id + pre-paid
producer + status="queued"; WorkOrders/PendingDispatches projections;
RenderDispatch row; Spend dual-broadcasts drain + risen tally in one render;
isolation carried through). Thirteenth consecutive build-evidence wave.

Panelists: all six. No new lens. CLEAN 6/6 convergence on the TARGET (#16e
execute-the-order); two WITHIN-target dissents (the anti-farm guard is
non-deferrable) + a sequencing insistence, all chair-resolved by binding the
guard into scope — NOT a frontier-level clash.

Shared diagnosis: #16d closed the first loop half (earn→hold→spend→SEE the
tally rise) but the work-order is a DEAD TALLY — it never runs, the
reinvestment loop stays open, the economy is a one-way SINK. Two code-verified
facts: CatchRecord (ledger.go:20) already carries the catch-identity tuple
(BeforeRev, AfterRev, Path, Line, ReasonTag) + before/after inventory, yet
Append (ledger.go:110) gates ONLY on ShouldRecord with ZERO identity dedup —
a dispatched cycle re-running the card's OWN base/fix/anchor re-mints a
byte-identical catch = infinite money; and WorkOrderRecord (ledger.go:64)
carries NO funded target, the asymmetry where the farm grows.

## Per panelist (all six advocate #16e, reuse RunCatchCycleStreaming + cycleSem)

- UX: make the order's STATUS watchable — queued→running→done as NEW appended
  ID-keyed ledger lines (replay pure), RenderDispatch projects THREE counts;
  beats stream on a per-order row keyed by Producer; a re-mint carries
  Producer="wo:<id>" so the stock row labels it "from reinvestment"
  (byte-distinguishable); a no-distinct-target run completes "dispatched,
  nothing to catch". CONSTRAINT: execution ships WITH a status-bearing surface.
- Game design: the anti-farm guard ships in the SAME slice, non-deferrable —
  (1) WorkOrderRecord gains a TARGET (base/fix/tip/anchor) that is NOT the
  card's own caught cycle, refused at AppendDispatch; (2) the mint is
  idempotent on catch IDENTITY not on the act of running. DISSENT
  (chair-resolved): won't accept #16e shipping the executor WITHOUT the
  distinct-target + identity-idempotent mint.
- Systems / Economy: lift farm-denial from outcome-level (ShouldRecord) to
  IDENTITY-level dedup inside Append, projected purely from the log; the
  work-order carries its target so the mint can be identity-deduped;
  conservation holds because dedup keys on identity-of-RESULT, never on replay.
- Pragmatic TDD: DOUBLE-MINT GUARD FIRST (RED-first, no execution) — Append
  gains a log-derived dedup on the identity tuple, then the consumer
  goroutine. SEQUENCING insistence (chair-upheld): the guard lands before any
  execution wiring, else the executor ships infinite money live for ≥1 round.
- CI/CD & Delivery: a Runner goroutine picks the lowest-id queued order,
  acquires cycleSem (BOUNDS the #15 suite-exec multiplier), runs it, routes a
  Catch through the idempotent Append; the pre-paid Producer field DEMUXES two
  real producers on one per-key log — connect="connect", dispatch="wo:<id>".
- Refactoring: a consumer drains one order through EXISTING
  RunCatchCycleStreaming under cycleSem (reuse, no new sem), moves
  queued→running→done via an appended status line (last-writer-wins per id).
  CAVEAT (chair-bound): the double-mint guard MUST ship inside this slice.

## Clashes / risks touched

- EVENT-LOG-CONCURRENCY P0 — PARTIALLY discharged: in-process two-producer
  demux via the Producer field proven (both serialize through the SAME single
  l.mu over one file, positional+monotonic, as
  TestLog_concurrentAppendAndSpend proves; status transitions appended
  ID-keyed; beats off-ledger). CROSS-PROCESS single-monotonic-seq
  reconciliation EXPLICITLY DEFERRED to #16f (untestable scope this slice).
- DOUBLE-MINT/FARM — RESOLVED in-process by identity-idempotent Append + the
  distinct-target requirement (dedup is load-bearing — a duplicate mint is
  structurally impossible even if the target guard is miswired).
- CONFIRMED-CATCH-MUTANT-IDENTITY — earns its keep. TIME-TRAVEL-RE-EXECUTION-
  NOT-PROJECTION — respected (replay log for the KEY, re-run cycle for the
  RESULT). #15 COST MULTIPLIER — bounded by cycleSem. SECURITY TRIO
  (shim/egress/secret-scrub) — DORMANT in-process, activates at #16f.

## Verdicts updated

None flip; #16e closes the reinvestment loop and turns the economy from a
one-way sink into a compounding engine, gated so compound pays only on
DISTINCT work.

## New clashes opened

NONE at the frontier. Two within-target dissents (Game design, Refactoring) +
a TDD sequencing insistence, all upheld and bound into #16e scope (the
identity-idempotent Append is mandatory, in-scope, RED-first, non-deferrable)
— a build-constraint, not a competing target. The 6/6 supermajority is real.

## Decisions

1. NEXT BUILD (#16e, CLEAN 6/6): drain one queued work-order through the
   existing RunCatchCycleStreaming under cycleSem as the FIRST real second
   producer, gated by an identity-idempotent mint shipping in the same slice.
   IN: (1) RED-FIRST double-mint guard BEFORE any execution wiring — Append
   gains a log-derived dedup on the catch-identity tuple (BeforeRev, AfterRev,
   Path, Line, ReasonTag), refusing a second Append of an identity already in
   Records() (pure, deterministic, no executor); (2) WorkOrderRecord gains a
   funded TARGET (rev/anchor triple) bound at AppendDispatch (one atomic Write
   under l.mu), refused if it equals the card's own already-caught
   (base,fix,anchor); (3) a consumer goroutine drains the lowest-id queued
   order, acquires cycleSem(key), runs resolveCycle/ResolveStreaming verbatim,
   routes any Catch through the idempotent Append; (4) status queued→running→
   done as NEW appended ID-keyed lines (never mutating), last-writer-wins per
   id; RenderDispatch projects three counts; (5) a Producer field on
   CatchRecord — connect stamps "connect", dispatch stamps "wo:<id>".
   DEFERRED: cross-process/OS boundary (#16f), NATS, the security trio; the
   cross-process monotonic-seq reconciliation is NOT closed here.

2. ACCEPTANCE FIXTURES (three hard REDs, RED before consumer/dedup exist,
   under `env -u GOROOT go test -race ./internal/app/... ./internal/ledger/...`):
   (A) FARM DENIAL — Append CatchRecord with identity T, then a second with
   the same T → second refused, Records() len + Balance() unchanged; (B)
   DISTINCT-WORK COMPOUND — dispatch an order yielding a NEW tuple, run under
   cycleSem → Balance nets -1 then +1, Records +1, status reaches done exactly
   once via appended lines; (C) TWO-PRODUCER -RACE REPLAY — fire connect-mint
   and dispatch-mint concurrently on one key's log under -race → no torn line,
   each CatchRecord carries a distinct Producer, Close+reopen replay from JSONL
   reproduces WorkOrders/Records/Balance identically with monotonic ids.

3. DOUBLE-MINT RULING: a dispatched run is HONEST iff its mint is idempotent
   on catch IDENTITY, NOT on the act of running. Lift farm-denial from
   outcome-level to identity-level dedup inside Append, projected purely from
   the log. Re-running OWN base/fix/anchor → byte-identical tuple already in
   Records() → no-op → spend 1, get 0 = honest LOSS. A DISTINCT target → new
   tuple → mints once → loop compounds. Two defenses ship together; dedup is
   load-bearing. A no-distinct-target run completes done reading "dispatched,
   nothing to catch"; a real re-mint carries Producer="wo:<id>".

4. P0 SEQ RULING: DISCHARGED NOW (in-process) — the pre-paid Producer field
   demuxes which producer minted each line on replay; Producer added to
   CatchRecord too; both serialize through the SAME single l.mu over one file
   (positional/monotonic regardless of goroutine count); status transitions
   appended ID-keyed (no phantom-edit). EXPLICITLY DEFERRED: the cross-process
   single-monotonic-seq reconciliation, to #16f, where NATS first earns its
   keep.

5. RANKED ROADMAP: [#16e THIS WAVE] execute + anti-farm identity gate +
   status-bearing surface + CatchRecord.Producer, in-process; [#16f
   cross-process producer] OS boundary, NATS earns its keep, cross-process
   monotonic-seq reconciliation, security trio activates together; [#13
   multiset]; [#11.5 rename-cliff]; [triage/attention-queue ranking ≥2 cards]
   (needs the dependency-graph signal the work-order target seeds);
   trust-economy bricks — 8/15 risks, post-pipe.

6. BLOCKERS: none to convergence. Hard prerequisite folded into scope: the
   double-mint identity guard must land RED-first, BEFORE any execution wiring.
   Open downstream risk (not blocking): each execution is a real suite fan-out
   (#15 multiplier), bounded but not eliminated by cycleSem; the cross-process
   seq stays an untested claim until #16f.

CONVERGED (17th consecutive round, CLEAN 6/6): #16d gave the spend a
conserved, atomic, visible consequence but the order is a DEAD TALLY. Every
lens names the same two code-verified facts: CatchRecord carries the identity
tuple yet Append has ZERO identity dedup, and WorkOrderRecord carries no
funded target. The slice: run one queued order through the existing
RunCatchCycleStreaming/cycleSem (reuse, no new primitive) as the first second
in-process producer, but RED-FIRST lift farm-denial to IDENTITY-level dedup,
give the order a funded target that is never the card's own, add Producer to
CatchRecord. Double-mint is honest iff idempotent on catch identity — own work
= -1 loss, distinct work = +1 compound. P0 two-producer seq DISCHARGED
in-process (single l.mu, Producer demux, append-only status, off-ledger beats)
and DEFERRED at cross-process to #16f. The only dissents are within-target —
the anti-farm guard ships in this slice, RED-first — so the supermajority is
real. The next event is a BUILD — identity-dedup Append (RED-first) →
WorkOrderRecord funded target → CatchRecord.Producer → consumer goroutine +
status transitions → RenderDispatch three counts.
