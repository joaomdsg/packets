# Round 27 — the spine is real but PARALLEL: the council weighs the NATS-first pivot against its own deferral ruling — 2026-06-08

Trigger: a new KIND of build evidence and a top-down DIRECTION change.
Four green Phase-0 slices shipped and pushed to main —
`internal/fabric`: an embedded in-process nats-server with JetStream
as a durable append-only log (publish/replay survives restart); the
`packets.session.<sid>.events.<inst>.<status>.<kind>` subject taxonomy
with a JetStream-native scratch/minted demux; a live
catch-up-then-tail subscription; and `orchestrator.PublishRevision`,
putting a typed minted-revision `TurnOutcome` onto the stream. The MVP
goal was reframed OUTSIDE the loop: build the NATS/JetStream spine
FIRST (Phase 0), then the pipe (Phase A) and Board (Phase B) on top.
First round on INFRASTRUCTURE evidence and the first since the thesis
was recorded PROVEN (Round 26).

Panelists: all six. Did NOT converge to a build — converged on the
diagnosis of the binding pitfall, split on the architectural fork it
exposes. Recorded honestly as a deliberation round whose next event is
ANOTHER round, not a build.

Shared diagnosis: build quality is not in question — every slice is
real embedded JetStream (no mocks for the log), green under -race, the
demux retiring event-log-concurrency-phantom-edits structurally and
the publish path enforcing anti-forgery (only a real mint reaches the
minted subject). The binding limit every lens names is NOT a code
defect: the spine is PARALLEL, not underneath. The proven in-process
economy (`internal/ledger` JSONL append-log + board/balance
projections that read THAT file) is still the live source of truth;
the JetStream stream is a second authoritative log nothing downstream
reads. `PublishRevision` writes to the stream but ledger.Append does
not, and no projection rebuilds from the stream. So the pivot's
headline — "JetStream is authoritative, projections are rebuilt, never
written ahead of the stream" — is currently FALSE by omission: two
unreconciled append-only logs. This is
dual-source-of-truth-projection resurrected one level up — the exact
risk the pivot claimed to retire.

Direction question, adjudicated: building NATS embedded/in-process
appears to CONTRADICT the panel's own ruling (Rounds 19/22) that "the
bus earns its keep only once an order crosses a process boundary." The
chair resolves the contradiction but not the consequence: the deferral
was scoped to "while the in-process loop is the frontier" — Round 26
concluded the prototype thesis is DONE and the only remaining marginal
value lies in cross-process / real-user deployment. The frontier moved
(top-down), so NATS as the substrate for that goal is not premature
relative to the NEW goal; it IS that goal's first slice. Direction
ENDORSED — conditionally on the unresolved fork below.

## Per panelist

- Systems / Economy (→ CUTOVER before breadth): the spine is clean but
  parallel; two authoritative logs is the binding pitfall. Next slice
  must be the cutover — the existing economy projects FROM the stream
  — not new spine capability. Adding 0.5/0.6 entrenches divergence.
- Pragmatic TDD (→ the missing RED defines convergence): there is no
  test that the live ledger/board state EQUALS a state rebuilt purely
  from the stream. That equivalence test converts the spine from
  decorative to authoritative — but can't be written until producers
  publish the same mint/miss events the ledger records.
- CI/CD & Delivery (→ embedded was right; integrate first): in-process
  embedded is the correct prototype call and defuses egress/auth
  scrutiny for now. But shipping the spine UNINTEGRATED is dead weight
  that will rot; cut over before widening.
- UX (→ felt value is premature): the SSE bridge (0.6) is what makes
  the spine felt, but rendering a stream-derived projection while the
  economy runs off JSONL would show a board that can DISAGREE with the
  real ledger. Felt value must wait for the single-log cutover.
- Game design (→ the hit-rate must not fork): the bet/mint/miss loop
  still runs on the in-process ledger; the stream carries a parallel
  revision event. If both persist, the hit-rate (Reinvested/Done) can
  be computed two ways and fork. ONE log must own mint/miss events.
- Refactoring (→ name the duplication; sequence behind a lock): two
  implementations of "the event log" (ledger JSONL + JetStream) is the
  debt. Honest end-state is JetStream as the ONE log with the ledger a
  projection — a large risky migration that MUST sit behind the
  characterization lock TDD names (the state-equivalence test).

## Clashes / open-questions touched

- dual-source-of-truth-projection — PROMOTED from latent to the
  round's central live finding (two real logs now in the tree).
- timetravel-re-execution-not-projection — REAFFIRMED as a caveat the
  pivot must not over-claim: JetStream gives durable EVENT replay, not
  durable agent CONTEXT (harness-context-unbounded-nondurable stands;
  replay ≠ re-run).
- shim-cannot-be-enforcement-boundary / sandbox-egress-allowlist /
  secret-scrub — kept DORMANT and correctly so (embedded, in-process,
  no boundary crossed), with the warning that the moment the harness
  crosses an OS boundary the security trio activates unbuilt — do not
  cross it until enforcement is kernel/netns/broker, not the shim.
- NATS-deferral ruling — RE-SCOPED, not overturned:
  deferred-while-in-process-is-the-frontier, now unblocked because the
  frontier moved by fiat.

Verdicts updated: none flip. The thesis stays PROVEN (in-process); the
spine does not change the game yet. New line: the spine is endorsed in
DIRECTION but is a parallel island, and "authoritative" is unbacked
until the cutover lands with an equivalence test.

New clashes opened: ONE genuine target-level fork (why this round does
not converge) — MIGRATE vs PARALLEL-BY-DESIGN. (A) MIGRATE: JetStream
becomes the ONE authoritative log; ledger/board rebuilt as projections
over the stream; the proven in-process economy carried across behind a
state-equivalence characterization lock. (B) PARALLEL-BY-DESIGN: the
proven in-process economy stays as-is (its JSONL its own truth); NATS
hosts ONLY new cross-process capability (multi-instance fan-out, live
harness, browser bridge) and the two never reconcile (disjoint state).
Lenses lean toward (A) (5/6 name the single-log end-state as honest),
but (B) is not refuted — it may be the cheaper, lower-risk path to the
cross-process value Round 26 named, WITHOUT a risky migration. The
fork determines whether 0.5 is "rebuild EXISTING projections from the
stream" (A) or "a NEW projection for NEW events" (B) — different
builds. Resolve before any 0.5 is specced.

## Decisions (§3-style)

1. NEXT EVENT IS A QUESTION, NOT A BUILD: resolve MIGRATE (A) vs
   PARALLEL-BY-DESIGN (B). The convergence test: pick the fork, and
   define whether the cutover/equivalence RED is against the EXISTING
   ledger state (A) or whether a clean process-boundary line is drawn
   so the two logs are provably disjoint (B). No 0.5 slice may be
   specced until decided.
2. PITFALLS TO AVOID (converged 6/6, regardless of the fork): (a) do
   NOT build 0.5/0.6 breadth on the parallel spine before the fork is
   decided; (b) do NOT let two logs both own mint/miss events (the
   hit-rate forks); (c) do NOT claim "authoritative/replayable" until
   an equivalence or disjointness test backs it; (d) do NOT cross an
   OS-process boundary until enforcement is real (kernel/netns/broker,
   not the shim) and the secret-scrub covers full-history pushes; (e)
   do NOT over-claim time-travel — durable events ≠ reproducible runs.
3. RANKED ROADMAP (conditional on the fork): [#1 THIS QUESTION]
   MIGRATE-vs-PARALLEL; [#2 if A] cutover: orchestrator/ledger publish
   to the stream + ledger/board rebuilt as a stream projection, locked
   by a state-equivalence test; [#2 if B] draw the process-boundary
   line and a disjointness test, then the first NEW cross-process event
   class; [#3] the NATS→SSE bridge (felt value, after one log owns the
   rendered state); [#4] cross-process producer + the security trio
   (gate together, deferrable while embedded); then the previously
   ranked pipe-correctness (#13 multiset, #11.5 rename-cliff) and the
   blocked-by-unsoundness trust-economy bricks.
4. BLOCKERS: the equivalence/disjointness test (the artifact that makes
   "authoritative" honest) cannot be written until the fork is chosen
   AND the producers emit the same events the chosen single owner
   records — so the blocker on convergence is a DECISION, not code.

NOT YET CONVERGED (1st round on infra evidence): the six lenses
converge on the diagnosis — the spine is real, green, well-built but
PARALLEL, and the dual authoritative log is the binding pitfall — and
on the five pitfalls, but split on the one fork that determines the
next build (MIGRATE the proven economy onto the stream behind an
equivalence lock, vs keep it PARALLEL-BY-DESIGN and let NATS own only
disjoint new cross-process state). The direction (NATS-first toward the
cross-process value Round 26 named as the only marginal value left) is
endorsed; the path is not yet agreed. Next event is ANOTHER
deliberation round that picks the fork — explicitly not a build, and
not a manufactured convergence.
