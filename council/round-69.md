# Round 69 — the work-order LIVE-EXECUTION model fork resolves — 2026-06-11

Trigger: R68 deferred a genuine architectural fork rather than guess it — wiring a
live harness into the work-order FILL path needs the work-order model to gain a task
prompt + a live-vs-prefunded mode. Full six convened as parallel Explore agents,
grounded in `internal/app/live.go` (runOneOrder/drainQueuedOrders/the fill-beat
buffer), `internal/ledger` (WorkOrderRecord/Target), `internal/harness`,
`internal/orchestrator`.

## Ground truth

- `runOneOrder` fills a work order by running `resolveCycle` on a PRE-FUNDED
  `Target.{BaseRev,FixRev,TipRev}` (a baked base→fix diff) + the `Target.{Path,Line}`
  anchor; it appends the catch Record, the work-order verdict, and the findings.
- A `ledger.Target`/`WorkOrderRecord` has NO task prompt. A live fill must run
  `harness.RunProcess(repoDir, prompt)` to PRODUCE the fix rev, then assess the catch.

## Convergence (6/6, with one productive clash)

- MODEL (Systems + TDD): add an OPTIONAL `Prompt` field to `Target` — empty = the
  legacy pre-funded path, non-empty = the live path. One shape (not a new
  WorkOrderRecord kind), so a test fixture reads end-to-end and the two modes stay
  structurally unified.
- DISPATCH SEAM (Refactoring): branch on prompt presence in `drainQueuedOrders` — keep
  `runOneOrder` (the pre-funded producer) intact, add a sibling live producer; don't
  fork the settle/append/firewall logic.
- TESTABILITY SEAM (TDD, binding): the live producer depends on a
  `runHarness func(ctx, repoDir, prompt) ([]harness.Turn, error)` boundary. Tests
  inject a scripted-supervisor stub (built from `harness.Supervisor.Run` over a
  `strings.NewReader` of stream-json lines — the same fixture pattern the harness
  tests use) so the live-fill path is exercised in CI WITHOUT a `claude` binary or
  API key. Production binds `harness.RunProcess`. RunProcess/ClaudeArgs stay wiring
  (build/vet/manual); the producer-branch logic + the reducer get real tests.
  Test-theater to avoid: mocking RunProcess to assert it was called.
- FIREWALL (Systems + all): only a settle-minted REVISION enters the catch economy;
  the live agent's activity rides the scratch bus (R68 `PublishActivity`), never the
  ledger. Single-minter holds. The runaway-token COST-GATE (R67) is DEFERRABLE past
  the first slice — enforce it as a ctx timeout/budget on `RunProcess` (wiring-level),
  not in the work-order contract.
- SURFACING (UX + Game + Refactoring): harness `UIEvent`s (thinking/editing/tool) do
  NOT mix into the oracle fill-beat row (that row is the oracle's cadence the Lead
  calibrates task-sizing against). They publish to the activity bus and render as a
  DISTINCT single-line "latest activity" indicator — latest beat only, advisory,
  never an action point (no auto-land/fund off a live beat; only settled turns are
  actionable — the R67 observer-vs-controller deferral holds). Dead-air = honest
  silence, no fabricated spinner (binding).
- DELIVERY (CI/CD): host-subprocess-first is correct for the first slice on a TRUSTED
  local repo — NOT gated by enforcement-below-container / push-before-teardown (those
  gate the verification CAGE, orthogonal; the agent box needs egress + a writable
  repo, the opposite profile, and is its own later gated round). Keep the live
  agent's own `go test` advisory; the cage oracle re-derives the authoritative
  verdict — never gate a verdict on the agent's self-checks.

### The clash (Systems vs Refactoring) — resolved into the slice split

- Systems proposed `resolveCycle(base, liveHEAD, liveHEAD)` as a clean reuse.
- Refactoring objected: that semantically conflates "we caught this" (oracle on
  base→fix) with "the agent happened to end here", a live order can't be deduped
  (live HEAD is nondeterministic, unlike a pre-funded identity), and a live Target
  has no base→fix pair until AFTER RunProcess — coupling the catch step to an
  anchor model that isn't defined for a free-form live task.
- RESOLUTION → split the slice: settle the live turns into a revision FIRST (the
  honest, testable vertical), and treat the catch-cycle-on-a-live-revision (its
  anchor model + dedup + cost-gate) as a SEPARATE downstream sub-slice once the
  live-order anchor question is designed.

## Slice plan (live-execution thread; tdd-rygba; gate green; docs fresh)

- SLICE 4a (NEXT): `Target.Prompt` (optional) + the `runHarness` injection seam +
  a live producer the drain loop dispatches to when `Prompt != ""`: it runs the
  (injected) harness to produce a live revision and publishes the turns' activity via
  `PublishActivity`. NO oracle/catch step yet (firewall-safe: settles a git revision,
  mints NO CatchRecord). Tested in CI with a scripted supervisor against a real temp
  git repo (a live order whose scripted harness edits a file settles a real revision;
  its activity reaches the scratch bus). `internal/app` live state is delicate (prior
  race history, R54) — build with a careful full TDD cycle.
- SLICE 4b: the catch-cycle on a live revision — design the live-order ANCHOR model
  (what line/behavior the catch is checked against for a free-form task) + dedup +
  the cost-gate. Likely its own short council step.
- SLICE 4c: surface the "latest activity" line on the card (UX single-line, advisory).
- SLICE 5+: containerize the agent run (its own gated round).

## Build record — slice 4a SHIPPED

`ledger.Target.Prompt` (optional) + `var runHarness = harness.RunProcess` seam +
a dispatch branch in `drainQueuedOrders` + `runLiveOrder` (sibling of runOneOrder,
reuses status/sem/fill machinery; runs the agent via the seam to produce the fix
revision; terminal status done/failed; mints NOTHING). tdd-rygba:
Red (two routing tests) → Yellow (added the firewall balance-unchanged assertion +
a promptless "done" assertion; fixed the missing balance-seed for AppendDispatch;
confirmed no hang — the attempts cap bounds a stuck order) → Green (found the status
read must use the projection's DispatchView, not the raw WorkOrders record) → Blue
(flagged the untested harness-error branch; my "leave for the attempts cap" comment
was WRONG — a "running" order leaves the queued set, so I changed it to a terminal
"failed" and added a covering test) → Audit (clean; race-checked; firewall + the
startFill/endFill ordering confirmed). Full suite + vet green.

## New clashes opened / resolved

Resolved: the work-order live-execution model fork (R68) — `Prompt` on `Target`, a
dispatch branch reusing the fill machinery, a `runHarness` injection seam for honest
CI testing, the scratch-bus activity firewall. Split out: the catch-on-live-revision
anchor model (slice 4b). No doc contradiction introduced.
</content>
