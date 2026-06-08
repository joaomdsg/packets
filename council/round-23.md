# Round 23 — make compounding legible: split the stock by provenance (connect vs reinvested) — 2026-06-05

Trigger: the Round-22 wave BUILT and SHIPPED green (work-source backlog → the
loop compounds >once on distinct fuel then exhausts honestly; the
runner-termination guard closed the #16e non-termination liability).
Fifteenth consecutive build-evidence wave.

Panelists: all six. CLEAN 6/6 convergence on the #2 UX provenance split, zero
new target-level clash; the only chair adjudication is a verified CODE-FACT
correction (key on the "wo:" prefix, not inProcessProducer), not a
disagreement.

Shared diagnosis (every lens read the code, same finding): the WRITE side is
complete — live.go stamps Producer "wo:<id>" (dispatch) / "connect" onto each
CatchRecord, ledger persists it to JSONL, consumption-from-log survives reopen
— but the READ side THROWS THE FIELD AWAY: ConfirmedCatches never reads
Producer, RenderStock emits a flat `s.Count+" confirmed"`. Round 22's
compounding lands on screen as two UNRELATED number-bumps, pixel-identical:
the one thing the Lead cannot see is the headline thesis. The gap between
"works" and "demonstrably/auditably works."

## Per panelist (all six advocate the provenance split, identical mechanism)

- UX: the loop works and is green but the human can't READ it. #1: ledger.Stock
  gains Reinvested (Producer "wo:" prefix), RenderStock emits a distinct span.
  NOTE for the chair: key on "wo:", NOT inProcessProducer="in-process" (two
  producer vocabularies coexist; the const tags a different struct).
- Game design: the first PROGRESSION arc (earn→spend→reinvest→draw-down→
  scarcity) and it's invisible; the configured-vs-generated backlog is real
  debt but NOT a prototype blocker (a finite list still demos the full arc).
- Systems / Economy: frame it as a read-model INTEGRITY fix, not cosmetics —
  an additive partition of Count (originated+reinvested==Count), pure
  projection auditable against the JSONL. DISSENT (build-order): the configured
  backlog is integrity debt; the generator must follow legibility immediately,
  not be deferred behind the security trio.
- Pragmatic TDD: smallest honest core, data already exists; two tested layers
  (ledger partition + surface span). [Its RED mis-keyed on inProcessProducer —
  chair-corrected.]
- CI/CD & Delivery: DISSENT (build-order): #16f is NOT yet warranted — lighting
  cross-process agent execution + the security trio before compounding is
  observable would commission the heaviest slice with no way to acceptance-test
  the remote mint; legibility is the observability precondition. [Its RED also
  mis-keyed on inProcessProducer — chair-corrected.]
- Refactoring: structural reuse — the Producer field is already pre-paid; the
  split adds a dimension to a projection without changing earn/spend/dedup
  semantics.

## Clashes / items touched

CLOSES Round-22 open item #2 "COMPOUNDING IS NOT LEGIBLE." Lights up the
pre-paid Producer field; consistent with "Stock is a pure projection,
auditable against the JSONL" (a decomposition, never a new source of truth).
Makes backlog draw-down/exhaustion legible (Reinvested stalls when the backlog
empties = the scarcity signal becomes readable). Does NOT touch:
configured-backlog-vs-generator, triage (#3), #15 fan-out cost, #16f, the
security trio.

## Verdicts updated

None flip; the split turns Round 22's invisible win into a felt, auditable one
and is the observability precondition for everything heavier.

## New clashes opened

NONE. The split is a strict superset-honest move — no lens loses ground. The
only chair adjudication is a VERIFIED CODE-FACT: classify reinvested on
strings.HasPrefix(r.Producer, "wo:") — the prefix the LIVE dispatch path
writes to the CatchRecord ConfirmedCatches projects — NOT the const
inProcessProducer="in-process" which tags the separate WorkOrderRecord struct.
Two of six experiments mis-keyed there and would read Reinvested==0; the four
"wo:"-prefix experiments are correct.

## Decisions

1. NEXT BUILD (UX provenance split, CLEAN 6/6): Brick 1 (land first) —
   ledger.Stock gains `Reinvested int`; ConfirmedCatches increments it inside
   the existing ShouldRecord-gated loop when strings.HasPrefix(r.Producer,
   "wo:"); an additive partition of Count (connect-minted = Count − Reinvested),
   total-function over the log, no new I/O or counter. Brick 2 (same cycle) —
   RenderStock appends one span carrying Reinvested, disjoint from
   stock__count/stock__reason/stock__self-flagged/stock__would-ship;
   Reinvested==0 renders calmly (no phantom badge).

2. ACCEPTANCE FIXTURES (two hard REDs under -race): (1) ledger stock_test —
   feed ConfirmedCatches {Producer:"connect" catch, Producer:"wo:7" catch,
   Producer:"wo:8" catch, Producer:"wo:9" NON-catch} → Count==3 AND
   Reinvested==2 AND Count−Reinvested==1 (non-catch never inflates Reinvested
   via the ShouldRecord gate; connect never reads as reinvested; replay-twice
   identical). (2) surface stock_test — RenderStock(Stock{Count:3,Reinvested:2})
   contains a reinvested span byte-disjoint from stock__count;
   RenderStock(Stock{Count:1,Reinvested:0}) renders no phantom reinvest badge.
   Both fail today.

3. RANKED ROADMAP: [#1 THIS WAVE] provenance split / read-model legibility;
   [#2 next] triage/attention-queue ranking ≥2 cards (unblocked by the
   funded-Target backlog seeding the node-with-target signal); [#3] a real
   WORK-SOURCE/generator (the configured-backlog debt — build AFTER legibility
   so we can read what work-shape it must emit, per Systems' dissent — ahead of
   the security trio); [#4] #16f cross-process producer + the SECURITY TRIO
   (shim-not-enforcement, egress-allowlist, secret-scrub-scope), gated on
   legibility; then [#13 multiset], [#11.5 rename-cliff], and the trust-economy
   bricks (calibration/the-bet/Focus/tiers/Ship-Quality — 8/15 risks) which
   need the now-readable multi-cycle economy to calibrate against.

4. BLOCKERS: none blocking the build. One code-fact the implementer MUST
   honor: classify on strings.HasPrefix(r.Producer, "wo:"), NOT
   inProcessProducer="in-process" — keying on the const reads zero against real
   reinvested mints.

CONVERGED (19th consecutive round, CLEAN 6/6): all six lenses independently
read the code and converged on the same diagnosis and #1 — the write side
stamps Producer "wo:<id>"/"connect" onto every CatchRecord and persists it,
but the read side (ConfirmedCatches/RenderStock) throws the field away and
emits a flat count, so Round 22's compounding lands as two unrelated
number-bumps. The build is the smallest honest slice: add
ledger.Stock.Reinvested as an additive partition of Count
(originated+reinvested==Count), tally it inside the existing ShouldRecord gate
on the "wo:" prefix, render one disjoint span — pure projection, zero new
write path, zero new event type, zero process/agent-execution surface, ∴ zero
security-trio exposure — and it doubles as the observability precondition for
everything heavier. The hard RED is genuine and two-layered (ledger partition
+ surface span), locking the non-inflation edges (non-catch + empty/malformed
Producer never increment Reinvested; connect-only renders no phantom badge;
replay identical). The only chair adjudication is a verified code-fact (key on
the "wo:" prefix the LIVE path writes, not the coexisting inProcessProducer
const). Two recorded dissents are pure build-ORDER over the agreed frontier
(valid convergence): Systems seats the work-source generator at #3 (ahead of
the security trio), CI/CD keeps #16f at #4 gated on legibility. The next event
is a BUILD — ledger.Stock.Reinvested partition (RED-first) → RenderStock
reinvested span.
