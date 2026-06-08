# Round 24 — the fleet-read board: one cross-card surface, ordered by activity, never by faked leverage — 2026-06-05

Trigger: the Round-23 wave BUILT and SHIPPED green (the provenance split —
compounding is now legible on ONE card). Sixteenth consecutive build-evidence
wave.

Panelists: all six. CONVERGED 5/6 on the cross-card fleet-read Board; Systems
dissents on ORDER (generator-first), chair-resolved toward the Board; no new
target-level clash (all six recorded newTargetClash: NONE).

Shared diagnosis: R23 made compounding legible on ONE card and thereby
EXPOSED the binding blocker (verified in code): liveReg is a sync.Map read by
NO fleet-wide projection — every View is keyed on a single session, so
"running multiple shops" is N isolated browser tabs the Lead diffs in their
head. What R23 made readable does NOT aggregate. The whole council engaged the
leverage-needs-a-dependency-graph finding HEAD-ON: a real "highest-leverage
review next" needs a blocked-downstream dependency graph that does NOT exist
in code (and even §29.7's file-OVERLAP conflict relation is UNBUILT), so any
leverage ranking shipped today would be FAKED — a mis-ranking trap laundering
a shallow scalar as authority.

## Per panelist

- UX (→ board as ACTIVITY strip): a new card whose View ranges liveReg, one
  row per session (key | confirmed/reinvested | balance | queued→running→done),
  ordered by queued-dispatch descending — an HONEST already-logged "where
  motion is" signal, framed as "activity," never "priority." Refuses leverage
  (the graph is the real blocker).
- Game design (→ overlap-as-contention): ship triage only as a
  NON-authoritative signal; rank by NOTHING, surface the one COMPUTABLE
  relation — file-OVERLAP between cards' funded Target Paths as CONTENTION.
  [Chair: a real future signal but the overlap relation is UNBUILT — its RED
  secretly requires building it first; deferred to #3.]
- Systems / Economy (→ generator FIRST, the dissent): the work-source
  generator (a Backlog interface, Next() (Target,bool)) is the true binding
  constraint — ranking attention across backlogs that both drain to a puddle is
  "rearranging deck chairs." [Chair: ORDER dissent, resolved toward the Board;
  supply seated firmly at #2.]
- Pragmatic TDD (→ board, labeled aggregation NOT leverage): a Board read; the
  honest signal is a labeled aggregation, and a characterization test must FAIL
  on a leverage/priority label.
- CI/CD & Delivery (→ runtime intake, supply-side): the smallest supply slice
  is a runtime INTAKE seam (HTTP POST / watched-dir → AddSession on a live
  server) so work can arrive without a restart. [Chair: co-equal supply
  down-payment at #2b.]
- Refactoring (→ board as starvation/liveness queue): a small honest liveness
  queue, explicitly NOT a leverage ranking; reuses the per-key registry + the
  already-computed projections.

## Clashes / findings touched

leverage-needs-a-dependency-graph (CENTRAL — engaged by 6/6; the Board ships a
non-leverage queued-activity signal and refuses the uncomputable rank).
multiuser-rewrites-the-scoring-spine (read-only projection over isolated
single-Lead economies — no co-review). conventions-compounding-unbounded
(resist a dashboard of meters; keep a calm row-per-card tally in the
RenderStock idiom). harness-context-unbounded (stateless re-projection, no
durable state). treasury-raw-vs-priced (rows show counts/balance, not cost).
The configured-not-generated backlog limitation (backlog-remaining is honest
only because supply is finite today).

## Verdicts updated

None flip; the Board is the seam every later cross-card signal (generator
provenance, intake, overlap-contention, eventual leverage) plugs into.

## New clashes opened

NONE at target level. Three conditional dissents (UX, TDD, Game, Refactoring)
— "I WOULD dissent if the board sorted/labeled by synthetic leverage or
balance" — are PRE-RESOLVED by scope (Queued-only order, leverage-label-banned
characterization test). Game-design's overlap-contention is a real future
signal, not adopted now (its relation is unbuilt). Systems' generator-first is
an ORDER dissent, not a target clash.

## Decisions

1. NEXT BUILD (fleet-read Board, CONVERGED 5/6): add ONE pure fleet-read seam
   — `app.BoardRows() []CardRow` — that ranges liveReg and, per entry, reads
   ONLY projections already computed from each session's OWN log (Key,
   Stock{Count,Reinvested}, Balance, DispatchStatusCounts{Queued,Running,Done},
   backlog-remaining). Order rows by Queued DESC, tie-broken by REGISTRATION
   order (the only stable ordinal — CatchRecord carries NO timestamp/seq, so
   "age" is rejected as fabricated). Render as a calm row-per-card tally in the
   RenderStock idiom (no gauges/meters). Stateless re-projection per render; NO
   new schema, write path, durable state, or cross-session credit. HARD FENCES:
   (1) the surface says "queued"/"activity"/"needs attention" — a
   characterization test FAILS if it ever emits
   "leverage"/"priority"/"rank"/"highest-impact"; (2) ordering is by Queued
   only, NOT balance (ranks the hoard) and NOT a synthetic leverage score
   (uncomputable, mis-ranking trap). Does NOT touch: the generator, runtime
   intake, the security trio, or any trust-economy brick.

2. ACCEPTANCE FIXTURES (hard RED): register keyA/keyB into liveReg; drive
   AppendDispatch directly (no oracle) so keyB has 2 queued+undrained orders and
   keyA 0; BoardRows() → both rows in order [keyB, keyA] with keyB.Queued==2,
   keyA.Queued==0, each row's Balance/Count/Reinvested/backlog-remaining
   matching that session's OWN log (isolation). Drain keyB to Done + fund keyA
   → re-sort [keyA, keyB]. Honesty boundary: two sessions with EQUAL queued
   counts fall back to registration order DETERMINISTICALLY across repeated
   renders (no flaky sync.Map iteration order); a characterization test asserts
   the rendered surface contains NO token in {priority, leverage, rank,
   highest-impact}.

3. RANKED ROADMAP: [#1 THIS WAVE] fleet-read Board (the queued-activity
   cross-card surface, the seam every later signal plugs into); [#2 next]
   work-source generator with from-cycle provenance (the faucet — ALSO grows
   the first real downstream provenance edge, the precursor to a genuine
   dependency graph that would make leverage HONESTLY computable); [#2b]
   runtime intake seam (HTTP POST / watched-dir → AddSession on a live server);
   [#3] overlap-as-contention (build the §29.7 file-overlap edge, THEN a true
   contention decision is shippable on the Board); [#4] #16f cross-process
   producer + the security trio (gate together, heaviest, deferred while the
   in-process loop is solvent/legible/termination-proven); [#5 #13 multiset,
   #11.5 rename-cliff]; [#6] the trust-economy bricks
   (calibration/the-bet/Focus/tiers/Ship-Quality — 8/15 risks; now have a
   readable multi-cycle compounding economy to calibrate against).

4. BLOCKERS: the binding one this round — NO surface ranges liveReg (verified:
   read by nothing fleet-wide), which the Board removes. Real leverage stays
   uncomputable (no timestamp/seq → age fabricated; no dependency/blocked
   relation → leverage only fakeable), sidestepped by ordering on
   Queued-awaiting-drain (purely log-derived, causal). Downstream: supply is a
   hand-seeded finite backlog (the board's backlog-remaining honestly mirrors
   that scarcity); work can't arrive at a running process (no intake); the #15
   cost-multiplier persists (a fleet view multiplies the ≥1000 stream-poll
   signature alias across cards — flag, don't fix).

CONVERGED (20th consecutive round, 5/6 on target): R23's legibility win
exposed the binding blocker (liveReg read by no fleet-wide projection —
verified), so "multiple shops" is N tabs the Lead diffs by hand. Five of six
lenses converge on the smallest honest slice — a pure liveReg.Range projection
of already-computed per-log scalars ordered by queued-awaiting-drain,
tie-broken by registration order (CatchRecord has no timestamp/seq, verified)
— and four lodge the SAME conditional dissent the chair pre-resolves by scope:
the Board ships as ACTIVITY/liveness, NEVER priority or leverage, enforced by a
characterization test failing on {leverage,priority,rank,highest-impact}. This
engages leverage-needs-a-dependency-graph honestly: no blocked/downstream
relation exists in code and even §29.7's file-overlap is unbuilt, so
Game-design's overlap-contention is a real future signal that cannot be this
round's slice (its RED secretly requires building the relation first; sequenced
to #3 after the generator grows a from-cycle provenance edge that finally makes
leverage computable rather than faked). Systems' dissent is ORDER not target —
supply before attention — resolved toward the Board because it is the strictly
smaller, schema-free, lower-risk read that removes the verified hard blocker
and is the seam the generator (#2), runtime intake (#2b), and overlap signal
(#3) all later plug into; the dissent is honored by placing supply firmly next.
Hard RED: register keyA/keyB, fund keyB with 2 queued orders and keyA with 0,
call the non-existent BoardRows() and assert order [keyB,keyA] with per-row
scalars matching each session's own log, re-sort on drain+refund, deterministic
tie-break on equal queued counts, and zero leverage tokens in the rendered
surface. The next event is a BUILD — BoardRows() liveReg.Range projection
(registration-ordered) → the calm row-per-card surface + the leverage-token-ban
characterization test.
