# Round 68 — live-harness slice 3: the activity bus brick + the work-order-prompt fork — 2026-06-11

Trigger: continuing the live-harness thread (R67). With the supervisor (slice 1)
and the real-process adapter (slice 2) built, slice 3 was scoped as "wire the live
harness into the watchable surface / work-order fill." A scout of the integration
surface surfaced a genuine architectural FORK that must not be guessed.

## Ground truth (scouted in code)

- `internal/app/live.go` `runOneOrder` fills a work order by running `resolveCycle`
  on the order's PRE-FUNDED `Target.{BaseRev,FixRev,TipRev}` — a baked base→fix git
  diff. A `ledger.WorkOrderRecord`/`Target` carries rev/path/line, NOT a
  natural-language TASK. The live "fill beats" are the oracle cycle's
  `pipe.TraceEvent`s (settle-base → oracle-base → … → catch), accrued into the
  per-session fill buffer (R65 slice 4).
- So swapping a LIVE harness into the fill path requires the work-order MODEL to
  gain (a) a task intent (prompt) and (b) a live-vs-prefunded execution mode where
  `RunProcess(repoDir, prompt)` PRODUCES the fix revision, then the catch cycle runs
  on base→(live HEAD). That is a real model fork (where the prompt lives, how the
  two modes coexist, where activity beats come from — `translate.UIEvent`s vs the
  oracle's `TraceEvent`s — and how the two-scores firewall stays intact).

## Decision: split the fork from the brick

Rather than guess the model fork, slice 3 builds the load-bearing BUS BRICK the
watchable surface needs regardless of how the fork resolves, and DEFERS the
work-order-prompt model to a dedicated council round (the next thread step).

- THE BRICK (built this round): `orchestrator.PublishActivity` / `DecodeActivity`.
  A live turn's `[]translate.UIEvent` is published on the SCRATCH/activity subject
  (`EventSubject(session, instance, StatusScratch, "activity")`). Scratch because
  activity is non-authoritative diagnostic that must NEVER be replayed into
  source-of-truth state — the economy firewall. An empty batch is refused (no bus
  noise, no needless scratch refold per viewer), mirroring `PublishRevision`
  refusing an unminted turn. Fabric round-trip tested in CI; no API key.
  - FIREWALL VERIFIED (audit): every economy/ledger projection filters to
    `minted`/`claim` (`ReplayProjection`, `FleetBoard`, claim lifecycle); the only
    scratch reader is `bridge/fleet.go`'s `FleetEventsSubject` WAKE trigger, which
    still folds minted+claim only. Activity is watchable-but-unscored as designed.

- THE FORK (deferred to a council round): the work-order live-execution mode (the
  prompt model + live-vs-prefunded fill). This needs the full lenses — Systems
  (firewall + the runaway-token cost-gate flagged in R67), Refactoring (the
  runOneOrder seam: don't fork the pipeline), Game/UX (the honest "real worker"
  beats, no fake typing), TDD (reachability + how to test a live fill without an
  API key), CI/CD (containerization sequencing). Convene before building.

## Build record — SHIPPED (slice 3 brick)

`internal/orchestrator/activity.go` (+ activity_test.go): tdd-rygba —
Red (round-trip on scratch/activity, empty-batch refusal publishes nothing to ANY
subject, malformed-decode error) → Yellow (declined the zero-value-content refusal
as an unwarranted semantic judgment; `len==0` is the clean firewall) → Green
(minimal mirror of publish.go) → Blue (only the unreachable json.Marshal error
uncovered — defensive, kept for parity) → Audit (clean; firewall leak-checked).
Full suite + vet green.

## New clashes opened / resolved

Opened: the work-order live-execution model fork (prompt + live-vs-prefunded fill)
— deferred to its own council round, NOT guessed. No clash resolved/contradicted.
</content>
