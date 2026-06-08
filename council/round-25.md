# Round 25 — the faucet refills: from-catch candidate supply turns the loop into a going concern — 2026-06-05

Trigger: the Round-24 wave BUILT and SHIPPED green (the cross-card
fleet board — activity, never leverage). Seventeenth consecutive
build-evidence wave.

Panelists: all six. CONVERGED CLEAN 6/6 on
supply-as-from-catch-candidate-generator; one real disagreement
(interface-vs-concrete SHAPE) chair-resolved toward a concrete
function; no new target-level clash.

Shared diagnosis: Round 24's board turned scarcity into a live column
(BacklogRemaining counting down to a SILENT PUDDLE — live.go:
fundableBacklog drains to nil, next Spend silently no-ops). The
backlog is a HAND-SEEDED FINITE list, so the economy can only DRAIN.
The trust-economy bricks have no live self-sustaining multi-cycle
economy WITH MISSES to calibrate against until supply regenerates.
Two facts keep the slice small + honest: the candidate→no-mint HONEST
LOSS path already EXISTS and is tested (spent 1, got 0), and the
supply point is one function pair (nextUnconsumedTarget/
fundableBacklog) reading cfg.DispatchBacklog with everything
downstream (dedup, own-skip, cycleSem, the res.Record!=nil mint guard)
already generator-agnostic over bare ledger.Target — so supply needs
NO new economic machinery, only a new SOURCE feeding the same Target.

## Per panelist

- UX (→ from-catch): worst felt failure is a button that silently
  stops; from-catch provenance is the only shape that FEELS
  self-sustaining. Binding constraint: the honest LOSS must be VISIBLE
  board activity, never a silent discard.
- Game design (→ supply with MISSES): spend-1/mint-0 downside already
  shipped+tested — a spend is a BET, today rigged. Generator MUST emit
  MISSES (RED-enforced) or it silently kills the game.
- Systems / Economy (→ from-catch derivation): a candidate is a triple
  the oracle JUDGES, never a fabricated catch; per-card from-catch can
  legitimately DRY UP (accepted as HONEST scarcity); the repo-DAG
  walker is the guaranteed-unbounded fallback behind a future seam.
- Pragmatic TDD (→ concrete func, NOT interface, NOT repo-DAG):
  testable WITHOUT a real oracle via the resolveCycle package-var
  seam; a one-impl interface is premature.
- CI/CD & Delivery (→ Backlog interface NOW + fold intake): [DISSENT
  on SHAPE — overruled: one+one impls don't earn a
  characterization-tested seam; intake drags in the security trio for
  zero new supply.]
- Refactoring (→ concrete from-catch func, interface only on a third
  source): UPHELD as the chair's ruling.

## Clashes / open-questions touched

- HONEST-SHAPE (Open Q1) RESOLVED unanimously — generator emits
  CANDIDATES the oracle judges, never fabricated catches; loss path
  already exists+tested, the confirmed runtime contract.
- INTERFACE-EARNS-KEEP (Open Q2) resolved toward NO interface this
  slice (see new clash).
- RUNTIME-MUTABLE-SURFACE (Open Q3) deferred with intake.
- COST #15 (Open Q4) respected — human-only spend + cycleSem;
  generator only PROPOSES, never auto-spends.
- LEVERAGE-NEEDS-A-DEPENDENCY-GRAPH: the
  (catch→candidate-derived-from-it) edge is the FIRST real provenance
  link — down-payment on the graph that makes blocked-downstream/
  leverage honestly computable where faked rank is banned.

Verdicts updated: none flip; from-catch supply converts the loop from
faucet-to-puddle into a going concern and seeds the provenance edge
the dependency graph (and the dead trust-economy bricks) need.

New clashes opened: NONE at target level. One SHAPE clash
(interface-vs-concrete) chair-RESOLVED toward a concrete
`candidatesFromCatches` func composed into fundableBacklog — no
premature abstraction without a second independent impl; extract when
the repo-DAG walker lands.

## Decisions (§3-style)

1. NEXT BUILD (from-catch candidate supply, concrete func, CLEAN 6/6):
   (1) `candidatesFromCatches(log) []ledger.Target` — a PURE
   derivation over the card's own CatchRecords, emitting DISTINCT
   candidate triples advancing each catch's anchor/fix (same revs,
   adjacent anchor line — never a guaranteed mint); (2)
   `fundableBacklog` composes its config-list output WITH
   candidatesFromCatches after the SAME t!=own && !consumed filter so
   dedup/own-skip/consumed-projection apply uniformly (a funded
   candidate, mint OR miss, is consumed → never re-funded); (3) the
   honest LOSS made VISIBLE (a misses signal = Done − Reinvested,
   clamped ≥0). Testable via the resolveCycle seam. NO Backlog
   interface, NO Next() extraction, NO Enqueue/intake, NO network/fs.
2. ACCEPTANCE FIXTURES (hard RED, pure-JSONL replay under -race): (a)
   one confirmed catch + drawn config backlog → fundableBacklog
   returns ≥1 DERIVED candidate distinct from the catch's identity AND
   own (today nil → provable silent no-op); (b) swap resolveCycle to
   mint a distinct catch for the derived candidate, drive one
   Spend→drain → log gains the new catch AND BacklogRemaining stays >0
   (faucet refills); (c) HONEST-LOSS/anti-farm — a derived candidate
   reproducing a seen identity funds an order that RUNS but mints
   NOTHING (deduped, Confirmed/balance unchanged) and is NOT
   re-proposed; a "distinct-but-always-catches" generator fails this
   RED; (d) race — concurrent BoardRows mid-derivation stays clean;
   (e) board misses signal — done-but-no-mint shows as a miss.
3. RANKED ROADMAP: [#1 THIS WAVE] from-catch candidate supply
   (concrete func — going concern + first provenance edge); [#2]
   Backlog{Next()} INTERFACE extraction (deferred until a second
   impl); [#2-ext] repo-DAG-walk generator (unbounded fallback behind
   the extracted interface); [#2b] runtime intake (deferred); [#3]
   overlap-as-contention (§29.7, now shippable on the board); [#4]
   #16f cross-process producer + the security trio (gate together);
   then [#13 multiset, #11.5 rename-cliff], and the trust-economy
   bricks (calibration/the-bet/Focus/tiers/Ship-Quality — 8/15 risks;
   all need the live multi-cycle economy WITH MISSES this build
   creates).
4. BLOCKERS: from-catch supply is PER-CARD and can legitimately DRY UP
   (accepted as honest scarcity, repo-DAG walker held as fallback). UX
   binding: the loss must be VISIBLE board activity. Game-design
   invariant (RED-enforced): generator MUST emit misses. Cost #15:
   bounded by human-only spend + cycleSem; watch fan-out.

CONVERGED (21st consecutive round, CLEAN 6/6): every lens names the
same binding limit (supply is a hand-seeded finite list drawn to a
silent puddle) and the same #1 (from-catch candidate supply),
unblocked because Round 24's board made scarcity an assertable column
and the supply point is structurally tiny + generator-agnostic over
bare ledger.Target with the resolveCycle seam making it oracle-free
testable. Chair adjudicates interface-vs-concrete toward a CONCRETE
candidatesFromCatches func composed into fundableBacklog, and defers
runtime intake (relocates a finite list, drags in the security trio
for zero new capability). The honest shape is unanimous and already
the shipped runtime contract — generator emits CANDIDATES the oracle
judges, identity-dedup refuses any candidate reproducing a seen catch,
the loss must be VISIBLE (UX) and the generator MUST emit misses
(Game-design, RED-enforced), per-card sterility accepted as honest
scarcity (Systems), repo-DAG walker held as the unbounded fallback.
The prize beyond unbounded supply: the
(catch→candidate-derived-from-it) edge is the first real provenance
link the dependency graph needs, and the live multi-cycle economy WITH
MISSES the dead trust-economy bricks need. Next event is a BUILD —
candidatesFromCatches → fundableBacklog composition → board misses
signal — not another deliberation round.
