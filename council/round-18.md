# Round 18 — the structural unlock for ≥2 cards: re-key liveState to a per-session registry — 2026-06-04

Trigger: the Round-17 #17 wave BUILT and SHIPPED green (the economy's SINK +
visible drain — earn→hold→spend→drain complete on ONE card; the ledger is now
thread-safe). Tenth consecutive build-evidence wave.

Panelists: all six. No new lens.

New evidence (verified by reading code, cross-checked against ../via):

- #17 closed the full economy loop on ONE card. But all six lenses name the SAME
  residual: LiveCard.Spend's "dispatch a unit of agent work"
  (`AppendSpend(1,"dispatch")`) is a DANGLING VERB — a logged debit with ZERO
  downstream, because there is no second agent to dispatch to.
- Root cause structural and unanimous: `liveState` is a process-wide package var
  (live.go:47), read identically by View (CtxR, :102), Spend (Ctx, :130),
  OnConnect (Ctx, :149) via readLiveState()/cycleSem() — one cfg / one *Log /
  one sem for the whole process. Two DISTINCT cards cannot coexist → Clash D
  (1:N shop vs context-switch tax) literally un-exercisable.
- The re-key seam needs NO new Via capability: `CtxR.ID()` (ctx.go:128) and
  `Ctx.ID()` (ctx.go:207) both return the per-tab wire id, and `CtxR.Session()`
  (ctx.go:160) exists — so View is NOT key-starved (correcting Refactoring's
  stated blocker). A connect-derived tab-id key threads symmetrically through
  all three readers. (Session.id is unexported, sess.go:26 — a stable-across-tabs
  key would be minted via sess.Put/Get; the public tab id is sufficient and
  simplest.)

## Per panelist (ALL SIX advocate #16b-rekey-alone)

- UX: dispatch verb dispatches to nothing; the Board (two triageable cards + an
  attention queue) needs ≥2 cards. #1: re-key liveState to a sync.Map registry
  keyed by a connect-derived key, ONE seeded entry, behavior-preserving
  (cost_test.go:87 stays green); NATS deferred.
- Game design: the SHOP is imaginary (spend dispatches into a void); Clash D
  unanswerable with one card. #1: re-key to a sync.Map keyed by Ctx.ID() so two
  tabs = two cards, each its own *ledger.Log + sem. The inverse-of-shared-ledger
  RED is the structural proof D can finally be set up (lands #16c).
- Systems: the loop is complete but degenerate (one economy because one
  liveState singleton). #1: a sessionID-keyed registry with a SEPARATE
  per-session ledger path — the verdict is ISOLATED economies, not a shared
  Treasury: per-session ledgers keep each balance non-transferable so the faucet
  stays the sole credit source (matching #16 farm-denial). Boundary EXPRESSIBLE
  only after #16b, DECIDED in #16c.
- Pragmatic TDD: #17 made the ledger thread-safe, de-risking concurrent
  per-session ledgers. The re-key needs a test that CONSTRAINS its new behavior;
  the only snapshot (cost_test.go:87) stays green under a single key, proving
  nothing changed. Wanted the inverting isolation test shipped WITH #16b.
  (Chair-adjudicated to #16c — see dissent.)
- CI/CD & Delivery: the per-session registry lets N capped heads (a real
  merge-queue/Board) exist, each its own cap + ledger. #1: a
  sync.Map[sessionKey]→{cfg,*Log,sem} threaded through OnConnect/View/Spend; the
  second session's isolation (own LedgerPath) lands #16c. Defer NATS.
- Refactoring: the re-key is the big-bang flagged for rounds; the
  characterization snapshot now exists. #1: introduce the sync.Map behind
  readLiveState/cycleSem/setLiveState, seed exactly ONE entry under a single key
  so behavior is byte-identical — the second config is the NEXT slice. Separate
  commits. Defer NATS.

Clashes touched: D (UN-ADJUDICABLE today — liveState is a singleton; #16b is the
structural prerequisite that makes D adjudicable, #16c exercises it once two real
cards exist; no clash RESOLVED in #16b, D UNBLOCKED for #16c). A/H
(pre-empted/concrete in the small; full adjudication needs the Board). The
economic-isolation sub-question (per-session ledger vs shared Treasury) becomes
EXPRESSIBLE only after #16b, DECIDED in #16c.

Verdicts updated: none flip; #16b is a pure behavior-preserving refactor
unblocking the second agent (settles D) and making the economic-boundary
question expressible.

New clashes opened: NONE (all six report none). Divergences: (a) a build-SCOPE
split (isolation test in #16b vs #16c) — adjudicated as scope in favor of
re-key-alone; (b) Systems' per-session-vs-shared-ledger economic question — ruled
a scoping decision folded into #16c. Clean 6/6 convergence.

## Decisions

1. NEXT BUILD (#16b, CLEAN 6/6 — the council's first unanimous round, NOT
   chair-adjudicated like R17's 4/6): re-key `liveState` from a process-wide
   package var to an in-process `sync.Map` registry keyed by a connect-derived
   session key, threaded through the three readers (View, Spend, OnConnect) +
   setLiveState/cycleSem. BEHAVIOR-PRESERVING: seed ONE entry under a single key
   so single-card behavior stays byte-identical and the preservation suite
   (cost_test.go:87 + spend + cap) stays GREEN with zero edits. NATS DEFERRED
   (in-process sync.Map is the smaller honest step; the broker waits for a real
   second cross-process producer at #16d).
2. PREREQUISITE sub-bricks IN ORDER: [SUB-1] registry type — replace the
   liveState package-var struct with `var liveReg sync.Map` (sessionKey →
   *liveEntry{cfg, log, sem}); [SUB-2] seed one entry — setLiveState(cfg, log)
   stores ONE *liveEntry under a fixed default key (sem per-entry as today);
   [SUB-3] thread the key — readLiveState()→readLiveState(key) and
   cycleSem()→cycleSem(key) look up liveReg by key, FALLING BACK to the default
   key when the derived key isn't registered; update the three callsites;
   [SUB-4] derive the key from the PUBLIC tab id — ctx.ID() in Spend/OnConnect
   (*Ctx) and r.ID() in View (*CtxR), both verified symmetric; [SUB-5] green
   oracle — full suite passes UNCHANGED (zero edits == proof), plus a focused
   internal registry test pins the keyed lookup (seeded key resolves; unknown
   key falls back to default).
3. ACCEPTANCE FIXTURES: [preservation, zero edits — the refactor oracle]
   cost_test.go:87 (2 sequential connects → one shared ledger, recs Len==2), the
   spend drain/no-op cases, and cap_internal_test all stay GREEN unchanged;
   [registry unit, the new keyed-lookup constraint] after setLiveState seeds the
   default key, readLiveState(defaultKey) returns the entry (hit) and
   readLiveState("unregistered") falls back to the same entry (fallback) — both
   branches covered without a second card; [DEFERRED to #16c, written now as its
   entry gate, NOT landed in #16b] the inverting isolation RED — two connects
   under TWO DISTINCT keys, each its OWN LedgerPath, each mints a catch →
   keyA.Balance==1 AND keyB.Balance==1 (NOT 2 shared), a Spend on keyA drains
   ONLY keyA → Clash D made executable + the economic-isolation verdict made
   testable.
4. RANKED ROADMAP: [#16b THIS WAVE] re-key to the sync.Map registry, one seeded
   key, behavior-preserving; [#16c] real second session — distinct keys
   end-to-end, the inverting isolation RED, adjudicate Clash D + rule
   per-session-isolated-ledger vs shared-Treasury (presumptive default: isolated,
   per farm-denial); [#16-board] an attention queue ranking ≥2 cards (needs the
   leverage/dependency-graph signal); [#16d dispatch downstream] give Spend's
   "dispatch" a real second agent/producer (the FIRST real cross-process producer
   that justifies NATS); [#18 NATS] deferred to a dedicated round, after #16d;
   [#13 multiset], [#11.5 rename-cliff]; trust-economy thesis bricks
   (calibration/the-bet/Focus/tiers/Ship-Quality) remain post-pipe (8/15 risks
   live there).
5. BLOCKERS: (a) liveState is a singleton → two cards can't coexist → Clash D
   un-exercisable until #16b; (b) the dispatch verb dispatches into a void until
   #16d; (c) the economic-isolation boundary un-expressible until #16b, decided
   at #16c; (d) NATS waits for a real second cross-process producer. NO
   VISION/DESIGN text changed (12-contradiction reconciliation queued per RISKS
   step 5).

CONVERGED (14th consecutive round, CLEAN 6/6 — the council's strongest
convergence, a genuine unanimous result, NOT order-adjudicated like Round 17's
4/6): all six ratify #16b-rekey-alone — re-key the process-wide liveState package
var to an in-process sync.Map registry keyed by a connect-derived tab id, seeded
with ONE entry so the slice is byte-identical (preservation suite green unchanged
as the refactor oracle). The structural unlock for ≥2 cards, the prerequisite
that makes Clash D adjudicable at #16c. The scope split (TDD: isolation test
in-commit) is adjudicated to #16c on TDD's own green-discipline grounds — the
isolation RED needs two distinct keys threaded end-to-end, which is #16c's new
behavior; that RED is the mandatory acceptance gate OPENING #16c. The
economic-isolation question (isolated ledger vs shared Treasury) is folded into
#16c, not a new clash. NATS unanimously deferred. Next event is a BUILD — SUB-1
registry → SUB-4 key derivation → SUB-5 suite green unchanged + a
registry-lookup test — not another round.
