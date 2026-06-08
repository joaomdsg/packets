# Round 20 — the dangling verb gets a consequence: spend funds a logged work-order (queued, not run) — 2026-06-04

Trigger: the Round-19 #16c wave BUILT and SHIPPED green (a second keyed
live session; LiveCard.Key query routing; per-key isolated ledgers + sem;
cmd -session flag + validateSessions; inverting-isolation RED green;
preservation suite UNEDITED; whole suite green under -race). Twelfth
consecutive build-evidence wave.

Panelists: all six. No new lens. CLEAN 6/6 convergence on the TARGET (#16d
dispatch-consequence); a single intra-slice sub-clash (queued vs executing)
chair-adjudicated by scoping — NOT a frontier-level target clash.

Shared diagnosis: #16c put a second card on screen and made the hollow
center LOUDER — every lens names the SAME blocker: LiveCard.Spend does
AppendSpend(1,"dispatch") with ZERO downstream. The loop earn→hold→spend→
drain has no second half; "drain" buys nothing. Two empty shops instead of
one.

## Per panelist (all six advocate #16d, in-process, identical boundary)

- UX: on a successful Spend append a "dispatched" record to the SAME ledger
  + render a distinct data-state="dispatch" row (RenderDispatch(n) reads the
  ledger like Stock; RenderDispatch(0) reads calm); Balance.Write fan-out
  re-renders. Spend visibly MOVES a catch into a dispatched tally.
- Game design: make Spend produce a state-changing GOOD; advocated the
  EXECUTING cycle (spend→run a real catch cycle that mints back).
  [Sub-clash dissent — re-sequenced to #16e.]
- Systems / Economy: one debit ⇒ exactly ONE funded work-order, same
  session, CONSERVED (debits == dispatched count, per-account). Guardrail:
  consequence stays in-process + same-account; cross-process would be a
  target clash — no lens proposed it.
- Pragmatic TDD: make debit + consequence ONE atomic ledger fact —
  `ledger.AppendDispatch(reason)` under the same `mu` (reuses the AppendSpend
  guard) writing the debit AND a paired kind:"workorder" line carrying a
  monotonic id + producer field; `PendingDispatches()` projects open orders
  (skipped by Records like spends). Forces the P0 event-log producer field
  NOW, earned by a real second line-kind.
- CI/CD & Delivery: enqueue ONE work-order (id, key, status=queued, producer
  tag) atomically with the debit under the balance lock; one new SSE row.
  Carry producer+status from line one for the later cross-process step.
  Defends as TARGET-level (not order): #16d must NOT spawn a cross-process
  executor or stand up NATS.
- Refactoring: REUSE RunCatchCycleStreaming under the per-key sem to RUN the
  dispatched cycle — the first in-process producer. [Sub-clash dissent —
  re-sequenced to #16e.]

## Clashes touched

- DANGLING VERB (headline R17 open item) — CLOSED: Spend now buys a funded,
  projectable, conserved work-order.
- Clash D (1:N shop vs context-switch tax) — made WATCHABLE-WITH-CONTENT but
  verdict stays a human side-by-side session, NOT a green test.
- Event-log-concurrency P0 — PRE-PAID (producer+status FIELDS forced onto the
  new workorder line-kind now) while a single in-process writer keeps the
  monotonic seq honest; full multi-producer reconciliation NOT discharged
  (deferred to #16e).
- NATS-deferral — CONFIRMED (the bus earns its keep only once an order
  crosses a process boundary).
- Shim-enforcement / sandbox-egress / secret-scrub — kept DORMANT (no agent
  RUNS this round). Farm-denial extends to the work side via per-account
  conservation. Seeds leverage-needs-a-dependency-graph: the work-order is
  the first node that graph will need.

## Verdicts updated

None flip; #16d installs the consequence the spend has lacked since R17 and
converts the economy from bookkeeping into a purchase the Lead can SEE.

## New clashes opened

NONE at the frontier. One intra-#16d SUB-CLASH (adjudicated by scoping): a
passive QUEUED work-order (UX, Systems, Pragmatic-TDD, CI/CD — 4 lenses) vs a
work-order that RUNS an in-process catch cycle (Game design, Refactoring — 2
lenses). RULING: the QUEUED non-executing order wins THIS round — (i)
strictly smaller honest slice with a clean green RED; (ii) running a cycle
introduces the genuine second in-process producer the P0 single-seq finding
warns about — funding-without-running pre-pays the schema fields while
honestly deferring seq reconciliation; (iii) the 4-lens framing already
carries the producer field, so #16e is a clean follow-on, not rework. The
2-lens runner is NOT rejected — re-sequenced to #16e with its spec preserved.
Convergence holds 6/6.

## Decisions

1. NEXT BUILD (#16d, CLEAN 6/6): a successful Spend atomically funds exactly
   ONE logged work-order in the SAME isolated ledger and renders a
   Dispatched/Queue row; it does NOT execute code. IN: (a)
   `ledger.AppendDispatch(reason)` — under the same `mu`, refuses on
   balance<1, else writes the debit AND a paired kind:"workorder" line with a
   monotonic id + producer field + status=queued; (b)
   `PendingDispatches()`/`Dispatched()` projection (workorder lines skipped by
   Records); (c) `surface.RenderDispatch(n)` — distinct data-state="dispatch"
   row, RenderDispatch(0) calm; (d) LiveCard.Spend calls AppendDispatch and
   broadcasts BOTH the drained Balance cell AND the new Dispatch row over the
   existing SSE fan-out; (e) per-key isolation through the consequence.
   DEFERRED: the order RUNNING a cycle (#16e); cross-process executor;
   NATS/JetStream; triage/attention-queue ranking; the multi-producer
   RECONCILIATION (only FIELDS pre-paid); all shim/egress/secret-scrub.

2. ACCEPTANCE FIXTURES (hard RED, internal/app + ledger + surface,
   preservation suite UNEDITED): (1) ledger core — mint 2 catches, two
   AppendDispatch → Balance==0, PendingDispatches==2, distinct monotonic ids +
   producer field + status=queued, debit-count==workorder-count; (2)
   atomicity under -race — balance=1, N goroutines → exactly ONE succeeds,
   Balance==0, PendingDispatches==1; (3) over-budget — Spend at balance 0
   funds NO order, emits NO dispatch frame; (4) surface/integration — POST
   Spend → streamed View has the dispatch row at 1 AND balance drained to 0 in
   the SAME render, RenderDispatch(0) calm; (5) isolation — Spend on A funds
   an order ONLY in A. NOT test-closeable: Clash D's worth-it verdict.

3. RANKED ROADMAP: [#16d THIS WAVE] queued in-process work-order + Dispatch
   row; [#16e execute-the-order] run the dispatched order through
   RunCatchCycleStreaming under the per-key sem (the 2-lens runner
   re-sequenced here) — the FIRST real in-process second producer, forcing
   the P0 single-seq reconciliation; [#16f cross-process producer] order
   crosses an OS-process boundary — where NATS/JetStream earns its keep AND
   shim/egress/secret-scrub activate together; [#13 multiset]; [#11.5
   rename-cliff]; [triage/attention-queue ranking ≥2 cards] still blocked on
   the dependency-graph signal; trust-economy bricks
   (calibration/the-bet/Focus/tiers/Ship-Quality) — 8/15 risks, post-pipe.

4. BLOCKERS: (a) the only hazard inside #16d is ATOMICITY — debit + workorder
   line MUST be written under the single `mu` (covered by RED #2 under -race);
   (b) pre-paying producer+status fields does NOT discharge the P0
   multi-producer seq reconciliation (deferred to #16e); (c) cross-process
   security trio stays dormant until #16e/#16f; (d) triage ranking stays
   half-uncomputable until the dependency graph matures.

CONVERGED (16th consecutive round, CLEAN 6/6): #16c made the hollow center
louder — every lens names the dangling verb (a debit that buys nothing). All
six name #16d as #1 and all scope it IN-PROCESS, pre-emptively flagging the
cross-process/NATS framing as the one thing that WOULD be a target clash;
since no lens proposed it, the shared guardrail confirms the boundary. The
chair adjudicates the queued-vs-executing sub-clash for the QUEUED order:
smaller honest slice, clean green RED, pre-pays the producer+status FIELDS the
P0 finding demands while a single writer keeps the monotonic seq honest; the
executing cycle re-sequenced to #16e. Fully test-closeable except Clash D
(human session). The next event is a BUILD — AppendDispatch → PendingDispatches
→ RenderDispatch → LiveCard.Spend dual broadcast.
