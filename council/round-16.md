# Round 16 — make the lane scarce before you price it: the bounded-queue cap — 2026-06-04

Trigger: the Round-15 #15 wave BUILT and SHIPPED green (integrated cost is now a
COUNTED INVARIANT — a cycle fires exactly M_base+M_fix+1 suites, K concurrent →
3K, pinned — plus a measured K_max from BenchmarkConcurrentCycle). Eighth
consecutive build-evidence wave. 14 green packages; the wire SHOWS the economy
and now has MEASURED cost.

Panelists: all six. No new lens.

New evidence (verified by reading code):

- #15 measured the multiplier on an UNCAPPED engine: K=1→348ms/3execs,
  K=2→348ms/6 (flat), K=4→405ms/12 (~16% degradation, 20-core box).
  `cost_test.go` pins K cycles→3K (cost_test.go:54, uncapped fan-out as a
  passing FACT) and the 2-connect shared-ledger snapshot (cost_test.go:87, the
  characterization the re-key must preserve).
- `LiveCard.OnConnect` (live.go:93-110) fires `go func(){ ResolveStreaming(...)
  }()` per SSE connect UNCONDITIONALLY — NO semaphore/queue/cap. The 3K test
  DOCUMENTS the fan-out; it does not CONSTRAIN it. No RED fails when K_max
  exceeded.
- `liveState` is a package var (live.go:37, set once in NewServer); every
  LiveCard reads the SAME cfg+log. A second agent forces a per-session re-key —
  but the cap needs NO re-key.
- Stock is SHOWN (read-only) but never SPENT — no sink. Pricing (#17) against an
  UNCAPPED lane has no opportunity cost: degenerate strategy (Systems) is "spam
  connects → 3N free suites → farm cheap catches against an un-contended
  budget." Price needs contention; contention needs the cap.

## Per panelist

- UX (→ #16-board-queue): #15's ceiling says a calm Board runs ~2-3 cards
  before tempo degrades. Stock is inert = a gauge you can't act on = noise. #1:
  a bounded semaphore enforcing K_max in OnConnect on the CURRENT
  single-instance wire, BEFORE per-session re-key. Defer NATS (in-process chan
  is the smaller honest step; NATS drags §15/§19 for zero pixels).
- Game design (→ #17-pricing, lone outlier): stock only CLIMBS — faucet, no
  drain. #1: pricing — smallest honest SINK (a logged DEBIT in the same JSONL
  so balance = credits − debits stays a pure projection; over-budget rejected).
  Concedes the framing requires a contended budget.
- Systems (→ #16-board-queue): #15 gave the denominator; economy needs its
  CONVERSION — but pricing against ONE uncapped lane is decoration. #1: the
  bounded queue/semaphore — the FIRST scarce/contended resource pricing needs
  to bite. Pricing is #2. Defer NATS.
- Pragmatic TDD (→ #16-board-queue): the 3K test DOCUMENTS but does not
  CONSTRAIN. #1: a bounded acquire/release wrapping the OnConnect goroutine.
  Deterministic RED (no sleeps): K_max+1 OnConnects whose cycle blocks on a
  barrier; assert peak in-flight == K_max while the +1 blocks on acquire. Keep
  cost_test.go:87 green. Defer NATS.
- CI/CD & Delivery (→ #16-board-queue): any cost is load-dependent
  (348ms@K=1 vs degrading@K=4), so pricing would denominate against a drifting
  figure. #1: bounded in-process semaphore (buffered chan of size K_max) gating
  OnConnect BEFORE the re-key. Connects beyond K_max QUEUE, never drop. Defer
  NATS.
- Refactoring (→ #16-board-queue): the cap and the re-key are SEPARABLE
  refactors with separate characterizations; conflating them is the big-bang
  risk. #1: the in-process semaphore on the CURRENT wire — needs NO re-key
  (liveState stays a package var); the per-session re-key is the NEXT slice,
  alone, with cost_test.go:87 as its baseline. Two commits, each
  behavior-preserving. Defer NATS.

Clashes touched: D (still UN-adjudicable — one card; the cap is its
prerequisite, not its settler); A (pre-empted; full adjudication needs a Board);
the FARM-THE-FAUCET exploit (Systems) — capping closes it and establishes the
contention pricing (#17) denominates against. No §3 clash flips.

Verdicts updated: none flip; the cap converts the unenforced 8N prose warning
(#15 measured) into a regression-guarded constraint and makes the cost
denominator STABLE for pricing.

New clashes opened: NONE. 5/6 on #16-board-queue, 1/6 (Game) on #17-pricing — a
within-arc build-ORDER preference, self-defeating without the cap (Systems
demonstrates Game's own exploit). Zero new target-level clashes. NATS-defer
unanimous across all five #16 advocates.

## Decisions

1. NEXT BUILD (#16-board-queue, 5/6; Game's pricing-first dissent chair-resolved
   as build-order, ratified roadmap #3): an in-process bounded semaphore
   (buffered chan of size K_max) gating `OnConnect`'s ResolveStreaming goroutine
   on the CURRENT single-instance wire — BEFORE any per-session re-key and
   BEFORE pricing. NATS DEFERRED to a dedicated round (the cap is a chan, not a
   broker; an external broker drags the §13.3 rewrite + §15/§19 egress/auth
   scrutiny into a ~10-line refactor, earns its place only at
   per-session demux/second-agent).
2. PREREQUISITE sub-bricks IN ORDER (each via tdd-rygba): [1a] RED at the
   OnConnect LAYER (NOT the Resolve layer — cost_test.go:54 drives app.Resolve
   one layer below the cap, stays an uncapped FACT): K_max+1 concurrent
   OnConnect cycles whose cycle blocks on a barrier (started/release channels,
   NO time.Sleep), assert peak in-flight == K_max while +1 blocks on acquire —
   fails today (peak == K_max+1); [1b] introduce K_max into LiveConfig (explicit
   default; 0 = unbounded back-compat) threaded through setLiveState; keep the
   2-connect sequential snapshot (cost_test.go:87) green UNCHANGED; [1c] add the
   buffered-channel semaphore + wrap the OnConnect goroutine: acquire before the
   cycle, release (defer) after delivery — connects beyond K_max BLOCK then
   proceed when a slot frees (queued, never dropped); [1d] AUDIT -race + no
   goroutine leak (every acquire matched by a release on BOTH success and
   cycle-error branch at live.go:103; Stream nil-channel drain untouched).
3. ACCEPTANCE FIXTURES: [cap holds] K_max+2 concurrent OnConnects with a
   barrier-blocked cycle → atomic in-flight gauge peaks EXACTLY at K_max,
   surplus block, release → all complete (no time.Sleep, green under -race);
   [no dropped work] real cycle, K_max=2, 8 concurrent OnConnects against one
   ledger → EXACTLY 8 catch records (queue serializes, never sheds);
   [felt-loop not frozen] an admitted card still receives beat frames while a
   queued connect waits (semaphore gates ADMISSION, not the Stream tick);
   [behavior-preserved] cost_test.go:87 stays green UNCHANGED;
   [farm-the-faucet closed] K beyond K_max do NOT fire 3K concurrent suites
   (≤3·K_max in-flight) — the exploit backpressures.
4. RANKED ROADMAP: [#16-board-queue THIS WAVE] in-process bounded semaphore
   (K_max) on OnConnect, single-instance, NATS deferred — cap-only; [#16b
   per-session re-key, NEXT wave alone] liveState package var → per-session
   keyed state, baseline cost_test.go:87; [#17 pricing, after #16b] the
   SINK/debit (Spend record + Stock.Balance = credits − debits as a pure JSONL
   projection, over-budget rejected) — NOTE: ledger.Log.Append gates on
   ShouldRecord (catch-only) and will REJECT a non-catch Spend; pricing must
   extend the append contract or use a sibling kind; [#16c second agent /
   per-session demux, LATER] where NATS is reconsidered; [#13 multiset], [#11.5
   rename-cliff].
5. BLOCKERS: (a) lane uncapped → pricing has no opportunity cost and
   farm-the-faucet is open until the cap; (b) cost denominator drifts under load
   until the cap; (c) re-key and second agent are downstream of the cap (cap
   needs no re-key); (d) #17's Spend record needs the ledger append contract
   extended. NO VISION/DESIGN text changed (12-contradiction reconciliation
   queued per RISKS step 5).

CONVERGED (12th consecutive round): 5/6 converge on #16-board-queue — an
in-process bounded semaphore enforcing the measured K_max on the OnConnect cycle,
the smallest honest Board slice (cap-only, no re-key, NATS deferred), turning the
uncapped 3N fan-out into the first contended resource. The lone dissent (Game:
pricing-first) is build-order, chair-resolved against by the panel's own
reasoning (price needs contention, contention needs the cap) — pricing ratified
roadmap #3, after the re-key. No new target-level clash; NATS-defer unanimous.
Chair sharpened the brick: the cap lives at the OnConnect layer
(live.go:98-110), so the RED is a NEW OnConnect-layer barrier test, not a
Resolve-layer mutation. Next event is a BUILD — 1a RED → 1c semaphore → 1d audit
— not another round.
