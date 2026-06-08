# Round 17 — the economy's missing half: the SINK (pricing) — 2026-06-04

Trigger: the Round-16 #16-board-queue wave BUILT and SHIPPED green (a bounded
admission semaphore caps concurrent cycles; the catch lane is the first
contended resource; farm-the-faucet closed). Ninth consecutive build-evidence
wave.

Panelists: all six. No new lens.

New evidence (verified by reading code):

- #16 capped the FAUCET RATE (liveState.sem, queued-never-dropped, slot released
  on every exit incl. cycle-error). But the TANK is undrained:
  `ConfirmedCatches` only does `s.Count++` (stock.go:24); Balance is credits with
  NO subtraction; `ledger.Append` HARD-REFUSES any non-catch via the catch-only
  `ShouldRecord` gate (ledger.go:67-69). A capped faucet over an undrained tank
  is a slower-filling tank, not an economy. Stock SHOWN, never SPENT.
- `Append(r CatchRecord)` and `Records() ([]CatchRecord, error)` are HARD-TYPED
  to CatchRecord (ledger.go:19,82) — a debit forces a discriminated record on the
  same JSONL or a parallel reader. The farm-denial invariant must NOT be diluted.
- liveState is STILL a package var; two DISTINCT cards do not exist — Clash D
  un-adjudicable until the per-session re-key + a real second session.

## Per panelist

- UX (→ #17): stock row is structurally inert (only climbs) = noise; AND #16's
  scarcity is invisible (View renders exactly 4 rows). #1: smallest Spend slice
  — a sibling DebitRecord with its OWN append path (Append's gate UNTOUCHED) +
  a balance row + ONE keyboard Spend affordance on today's card. FIRST ACTION on
  the stock, no re-key.
- Game design (→ #17): a meter that only goes up is a score. #1: the SINK — a
  sibling debit kind via Log.AppendSpend, balance = credits − debits as a pure
  JSONL projection, over-budget rejected; ONE spend verb → one logged fact. No
  fake XP.
- Systems (→ #17): #16 gave scarcity; conversion is finally honest. #1: a
  sibling DebitRecord{Amount,Reason} (NOT through CatchRecord/ShouldRecord),
  balance = credits − debits, over-budget rejected-not-logged. Mint-then-refund
  self-deal is punished because debits log against a REAL credit balance derived
  only from oracle-confirmed catches: you cannot spend what you did not catch.
- Pragmatic TDD (→ #17): #16's scarcity is genuinely TESTED. #1: a sibling debit
  kind (RecordKind: Catch|Spend) keeping Append's gate INTACT for catches,
  balance = credits − debits, over-budget rejected, pure JSONL projection,
  replay-auditable. RED: 3 catches + Spend{2} → Balance==1, identical on
  re-read.
- CI/CD & Delivery (→ #16b, dissent): #16 serializes N connects onto ONE head;
  no second head exists (every OnConnect reads the SAME liveState). #1: the
  per-session re-key — package var → per-session keyed map (sessionID →
  {cfg,*Log,sem}), the ONLY structural unlock for a real Board (N capped heads).
  Pricing is DOWNSTREAM of having more than one thing to spend against. DEFER
  NATS.
- Refactoring (→ #16b, dissent): #16 gave the characterization net (shared-
  liveState snapshot). #1: re-key package-var liveState → per-session keyed map
  by a connect-derived session key, re-routing only readLiveState/cycleSem/
  setLiveState, NO behavior change yet. DEFER NATS (in-process sync.Map is the
  smaller honest step; broker waits for #16c). Pricing adds a new ledger record
  kind — a separable concern.

Clashes touched: D (touched, DEFERRED — un-adjudicable until two distinct cards
coexist, which #16b enables structurally and #16c exercises; all six agree D is
blocked on the re-key, not on pricing); A (pre-empted; balance row + drain start
to give the Lead an action, full adjudication needs the Board); H (the SPENT
economy makes the Trust-Ledger framing more concrete). NATS-decision juncture
ruled DEFERRED by unanimous position. No §3 clash closes; #17's sibling-kind
design opens no clash (farm-denial intact).

Verdicts updated: none flip; #17 installs the economy's missing SINK so the
stock can finally drain (first non-climbing transition), and the lane being
contended (#16) makes a Spend trade against something real.

New clashes opened: NONE (all six report none). The only divergence is SEQUENCE
(#17 vs #16b) over an undisputed two-brick frontier.

## Decisions

1. NEXT BUILD (#17-pricing — 4/6 advocate directly; 2/6 (CI/CD, Refactoring)
   advocate #16b-rekey as build-ORDER, NOT a target-level clash — both concede
   #17's necessity, scope, and the sibling-kind/gate-intact design;
   CHAIR-ADJUDICATED CONVERGED per the loop's recorded-dissent rule, sequenced
   #17-first because it adds NEW behavior with a hard RED (first stock drain) on
   the ONE card today, while #16b is a pure relocation whose payoff is two bricks
   away at #16c): the smallest honest Spend slice. Add a SIBLING debit record
   kind, project Balance = credits − debits as a pure JSONL replay, reject
   over-budget Spend WITHOUT appending. KEEP `ledger.Append`/`ShouldRecord`
   CATCH-ONLY (byte-identical); debits travel a separate guarded `AppendSpend`
   (Amount>0, Balance>=Amount). NATS NOT adopted (orthogonal).
2. PREREQUISITE sub-bricks IN ORDER (each via tdd-rygba): [1] a record-kind
   discriminator on the SAME JSONL — a `Kind` field DEFAULTING to catch so
   existing catch-only logs re-read byte-identical (replay-compat RED); [2]
   `Log.AppendSpend(amount, reason)` SEPARATE from Append — validates Amount>0
   AND Balance>=Amount against the projection, appends one Spend line on success,
   returns error and writes NOTHING on over-budget (RED: over-budget → error +
   log byte-length unchanged); Append+ShouldRecord stay catch-only and
   regression-pinned; [3] the Balance projection — `Balance(recs) =
   ConfirmedCatches(recs).Count − sum(Spend.Amount)`, a pure function in the
   stock.go idiom (RED: 3 catches + Spend{2} → Balance==1, identical on re-read);
   [4] surface — a 5th `RenderBalance` row on the LiveCard reading Balance
   read-only (degrade-to-empty on error, like RenderStock) + ONE keyboard Spend
   verb calling AppendSpend and re-rendering over SSE (the first DRAIN motion;
   over-budget rejected → row unchanged). NO re-key, NO second-agent plumbing —
   rides the package-var card.
3. ACCEPTANCE FIXTURES: [balance projection] 3 catches → Count==3; AppendSpend{2}
   → Balance==1; Balance from a FRESH re-read == 1 (pure replay);
   [over-budget rejection + byte-immutability] Balance==1, AppendSpend{5} →
   non-nil error, log byte-length unchanged, Balance stays 1; [farm-denial
   regression] Append(non-catch) still refuses; a Spend mis-routed through Append
   still refused (debits ONLY via AppendSpend); [backward-compat replay] an
   existing catch-only JSONL (no Kind) re-reads to the identical Stock
   (discriminator defaults to catch); [surface drain motion] 3 catches → row
   reads 3; Spend(2) → row re-renders to 1 over SSE; Spend(5) → rejected, row
   unchanged at 1, no new line.
4. RANKED ROADMAP: [#17-pricing THIS WAVE] the SINK (sibling debit kind,
   AppendSpend guard, Balance projection, balance row + spend verb); [#16b-rekey
   #2 immediate next] behavior-preserving package-var liveState → per-session
   keyed map (sessionID → {cfg,*Log,sem}), single derived session so byte-
   identical, snapshots green by construction; [#16c second session] two DISTINCT
   cards with isolated ledgers + independent caps — first point Clash D is
   exercised; [#18 NATS broker, deferred] adopt the external bus ONLY when #16c
   proves a real second producer needs cross-process fan-out (§13.3 + §15/§19);
   [#19 spend taxonomy, deferred] broaden beyond Catch|Spend after the single
   verb proves the projection + guard; [#13 multiset], [#11.5 rename-cliff].
5. BLOCKERS: (a) tank never drains until #17's sink; (b) ledger.Append is
   catch-only by construction — debits need a sibling kind + AppendSpend, never a
   relaxed gate; (c) Clash D stays blocked on the re-key (#16b) + a real second
   session (#16c); (d) NATS waits for a real second producer. NO VISION/DESIGN
   text changed (12-contradiction reconciliation queued per RISKS step 5).

CONVERGED (13th consecutive round, 4/6 + chair-adjudicated): 4/6 (UX, Game,
Systems, TDD) advocate #17-pricing — the missing SINK — as #1; 2/6 (CI/CD,
Refactoring) advocate #16b per-session re-key, a build-ORDER preference over an
undisputed two-brick frontier (both concede #17's necessity, scope, design;
neither opens a target-level clash). Per the recorded-dissent rule the chair
ratifies #17-first — NEW behavior with a hard RED (first drain) on today's single
card, while #16b is a pure relocation whose payoff is two bricks away (#16c) —
and ratifies #16b as the immediate #2 with its spec preserved verbatim.
Load-bearing ruling: KEEP ledger.Append catch-only (farm-denial intact); admit
debits via a sibling kind + a guarded AppendSpend; Balance = credits − debits as
a pure replay, over-budget rejected-not-logged. No new target-level clash; NATS
unanimously deferred to #18. FIRST round under ≥5/6 — recorded honestly as a 4/6
order-only dissent the chair resolved, not a manufactured supermajority. Next
event is a BUILD — discriminator → AppendSpend guard → Balance projection →
balance row + spend verb — not another round.
