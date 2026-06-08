# Round 28 — the fork resolves: ONE authoritative log (JetStream), the proven economy migrated onto it behind an equivalence lock, gated to the first cross-process consumer — 2026-06-08

Trigger: reconvened to resolve Round 27's open MIGRATE-vs-
PARALLEL-BY-DESIGN fork. No new build evidence — a decision round.

Panelists: all six. CONVERGED 6/6 on target after the fork was
stress-tested and option B collapsed on inspection. One sub-dissent
(drop the JSONL entirely vs keep it as a cache) resolved as
not-target-level.

Shared diagnosis: the fork is not symmetric — PARALLEL-BY-DESIGN (B)
collapses under the only marginal value Round 26 left. The
cross-process payoffs (a cross-session board aggregator, the browser
bridge) REQUIRE the per-session economy's events to cross the process
boundary; therefore no genuinely disjoint state line exists that both
keeps the economy off the stream AND delivers the cross-process value.
Any path that delivers it either (A) migrates the economy onto the
stream, or double-writes a derived broadcast (a dual-write with a
sync/divergence debt — the worst of both). Since JetStream strictly
DOMINATES the ledger JSONL — durable and replayable (everything the
JSONL gives) plus cross-process subscribe, demux, and live tail — the
one honest log is JetStream. The migration risk is BOUNDED precisely
because `internal/ledger` is already an append-only log feeding pure
projections: the economy LOGIC (identity dedup, the res.Record!=nil
mint guard, Balance = credits−debits, per-account work-order
conservation, the hit-rate) is substrate-independent and moves
UNCHANGED — only the home of the bytes changes. A is a substrate swap
behind the existing projection logic, not an economy rewrite.

## Per panelist

- Systems / Economy (→ A, B collapses): the cross-process board needs
  economy events across the boundary, so disjointness is unattainable;
  one log, and it is the one subscribable across processes.
- Pragmatic TDD (→ the convergence ARTIFACT is the equivalence lock):
  a state-equivalence characterization test — every projection
  (balance, board, hit-rate, work-orders) yields IDENTICAL state
  whether folded from the JSONL or from fabric.Replay of the same
  event sequence. Build the lock FIRST; the safe-migration
  precondition and the only thing that makes "authoritative" honest.
- CI/CD & Delivery (→ in-time, the lock is cheap): retire the dual-log
  debt behind the lock before shipping more parallel spine; do not
  widen on an unintegrated substrate.
- UX (→ the felt payoff is downstream of the swap): the browser bridge
  becomes trivial once the economy rides the subscribable stream;
  rendering it before the swap would show a board that can disagree
  with the real ledger — build the bridge AFTER the swap, never on
  JSONL.
- Game design (→ ONE log owns mint/miss, hit-rate cannot fork): A
  guarantees it. Hard rider: 0.4's PublishRevision must NOT be wired as
  a double-write alongside ledger.Append — that forks the mint event;
  it is the FIRST STEP of the migration (it replaces the append), not
  a parallel writer.
- Refactoring (→ name the debt, retire behind the lock, full taxonomy
  first): the debt is two event-log implementations; the migration
  needs the FULL event taxonomy (catch, spend, work-order, dispatch —
  not just revision), or rebuild silently drops unmapped kinds.

## Clashes / open-questions touched

- MIGRATE-vs-PARALLEL — RESOLVED toward MIGRATE (A);
  PARALLEL-BY-DESIGN (B) recorded COLLAPSED (no disjoint line survives
  the cross-process board/browser requirement).
- dual-source-of-truth-projection — RETIRED BY CONSTRUCTION: one log;
  the JSONL ceases to be a second source (the stream's file storage is
  the durable file now). The sub-dissent (drop JSONL vs keep as a read
  cache) is not target-level — chair: drop it AS A SOURCE; keep no
  second authoritative log; a transient cache, if any, is an
  optimization behind the lock, never a truth.
- shim/egress/secret-scrub — kept DORMANT and correct: the swap is
  single-process embedded and crosses no OS boundary; the trio stays a
  hard gate before the cross-process producer.
- harness-context-unbounded-nondurable / timetravel-re-execution —
  REAFFIRMED unchanged: durable events ≠ reproducible agent runs; the
  swap makes no time-travel claim.

Verdicts updated: none flip; the thesis stays PROVEN. New line: the
NATS-first pivot's locked decisions (JetStream authoritative,
projections rebuilt from it, never written ahead) are AFFIRMED as the
target and Round 27's "parallel island" observation is the gap this
path closes.

New clashes opened: NONE at target level (all six pick A).

## Decisions (§3-style)

1. CONVERGED PATH — the sequenced build: (i) extend the typed event
   taxonomy from revision-only to EVERY ledger event kind (catch,
   spend, work-order, dispatch) with subject kinds + publish/decode,
   mirroring 0.4; (ii) build the STATE-EQUIVALENCE characterization
   lock (projections identical folded from the JSONL vs from
   fabric.Replay of the same events — RED before any swap); (iii) SWAP
   the substrate behind the green lock — ledger publishing becomes a
   stream publish (REPLACING the JSONL append, never double-writing),
   projections read from Replay/Subscribe; (iv) THEN the cross-process
   consumers — the NATS→SSE browser bridge and the cross-session board
   aggregator — now trivially enabled because the economy rides the
   subscribable stream.
2. PITFALLS (carried, sharpened, 6/6): NO double-write
   (PublishRevision replaces the append, never parallels it — else
   mint/miss forks); NO "authoritative/replayable" claim until the
   equivalence lock is green; FULL taxonomy before the swap (a partial
   taxonomy silently drops event kinds on rebuild); the swap is ALWAYS
   behind the characterization lock, never free-hand; the OS-process
   boundary stays gated on the unbuilt security trio (kernel/netns/
   broker, not the shim) and the full-history secret-scrub; NO
   time-travel over-claim (events durable, agent context not).
3. RANKED ROADMAP: [#1 NEXT BUILD] full event taxonomy
   (catch/spend/work-order/dispatch publish+decode); [#2]
   state-equivalence lock; [#3] substrate swap (retires dual-source);
   [#4] NATS→SSE browser bridge; [#5] cross-session board aggregator;
   [#6] cross-process producer + the security trio (gate together);
   then pipe-correctness (#13 multiset, #11.5 rename-cliff) and the
   blocked-by-unsoundness trust-economy bricks (calibration/the-bet/
   Focus/tiers — still need log-derivable inputs).
4. BLOCKERS: the swap touches proven code — the equivalence lock is the
   non-negotiable precondition that bounds the risk; the security trio
   is a hard gate before any OS-process boundary; harness-context
   durability is unaffected and must not be over-claimed.

CONVERGED (1st convergence on the infra arc, 6/6 on target): the
Round-27 fork resolves to ONE authoritative log (JetStream). B
collapses because the cross-process value requires the economy's events
to cross the boundary, so no disjoint line survives; JetStream strictly
dominates the JSONL log, so the one honest log is the stream; and the
proven economy migrates as a LOCKED substrate swap (logic unchanged,
bytes' home changed), gated to land with the first cross-process
consumer so it delivers value rather than churn. The state-equivalence
characterization test is the convergence artifact — the precondition
that makes "authoritative" honest and retires
dual-source-of-truth by construction. The path is sequenced taxonomy →
lock → swap → bridge/board, with the no-double-write rider and the
security/time-travel pitfalls carried as hard constraints. Next event
is a BUILD — the full event taxonomy — not another deliberation round.
