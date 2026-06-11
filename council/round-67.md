# Round 67 — the LIVE-HARNESS thread opens: the stateful turn-reducer — 2026-06-10

Trigger: the autonomous loop re-oriented on the GOAL (a user does REAL work — a
real Claude Code harness instance does the task in a container, you review its
changeset as a PR). Built-vs-spec audit found the integrity spine, review surface,
Monaco UI, fleet board, and work-order economy all built — BUT **no real Claude
Code harness is ever spawned**. Work orders "fill" via a pre-supplied base→fix git
diff (R65 made this honesty caveat binding: "NO live code-editing agent"). That
live-harness supervisor + stateful reducer IS the P0→P2 product gap (~0%). Full six
convened as parallel Explore agents, grounded in the actual code + docs.

## Ground truth (verified in code)

- `internal/translate.Translate` is a PURE, stateless per-event mapper: one
  stream-json event → `[]UIEvent`. `assistant` text → `activity.agent`/thinking;
  `Edit|Write|MultiEdit` → editing(file); `Bash` → tool(command); other tools →
  tool(name); `result` → a single `{Type:"turn.ended", Detail:subtype}` signal.
  The package doc explicitly DEFERS the stateful reducer + orchestrator wiring.
- `internal/orchestrator.SettleTurn(ctx, repoDir, baseRev, msg)` settles the tree
  into a revision when the turn changed something (guards no-edit / net-revert /
  secret-block via `internal/settle`) and computes the base..head diff →
  `TurnOutcome{Minted, SHA, Added, Deleted, Diff, Secrets}`. `PublishRevision`
  emits a genuinely-minted outcome on the bus (refuses an unminted turn).
- NOTHING spawns a `claude` process or drives these bricks from a live stdout
  stream. The work-order "fill" (internal/app runOneOrder) runs the catch cycle on
  a PRE-FUNDED base→fix diff — not a live agent.
- Env: `claude` + `docker` present; `ANTHROPIC_API_KEY` unset (so a live API call
  can't run in this env — the slice must be testable without one).

## Convergence (6/6, tight)

The smallest shippable vertical = a **stateful turn-reducer** that reads a harness
stream-json event stream and settles a revision at each turn boundary. Spawning the
real `claude` subprocess is a thin `io.Reader` adapter in the NEXT slice.

Per lens:

- **TDD:** new `internal/harness` package; public API takes an `io.Reader` (the
  true process boundary), NOT a premature `HarnessReader` interface. Test with a
  scripted fixture stream (a `strings.Reader` of stream-json lines) against a real
  temp git repo — real>stub done honestly: stub the stream *source*, not the
  concept, and never fake-test by mocking. No API key needed. First behavioral
  claim: "a stream with two turn boundaries settles exactly two revisions, the
  second diffed against the first's SHA." Test-theater to avoid: asserting on the
  subprocess spawn or on call counts.
- **CI/CD:** host-subprocess FIRST, defer containerization. This slice is NOT gated
  by push-before-teardown / enforcement-below-container (those gate the verification
  cage + trust verdicts; this slice's output flows through the already-hardened
  settle→diff catch pipe). The agent container's profile (needs egress + a writable
  repo) is the OPPOSITE of the `--network=none` verification cage — a separate,
  gated slice. The settle CRITICAL fixes it depends on are already built.
- **Systems:** ECONOMY FIREWALL (binding) — the harness MINTS NOTHING. Activity
  events are diagnostic / off-ledger; only the host's settle step produces a
  revision; the single-minter + two-scores invariants hold untouched. Defer the
  catch/ledger integration (which catch a live revision redeems against) to a later
  slice. Flagged a real future cost-gate need: a runaway live agent has no
  pre-funded token cap (couples with the deferred governor flood-defenses).
- **Refactoring:** the supervisor owns a distinct concept (process lifecycle +
  stream reduction + base-rev threading across turns) → its own package, reusing
  `translate` (the pure reducer) and `orchestrator.SettleTurn` as bricks the same
  way `runOneOrder` reuses `pipe`. Debt to avoid: do NOT reinvent the
  accumulate-then-settle turn loop per adopter; keep the turn-boundary settle in one
  place (orchestrator's brick). Permission mediation is UI plumbing, NOT a security
  lever (enforcement stays kernel+container) — out of scope for slice 1.
- **UX:** surface the live thinking/editing/tool beats honestly — real latency, no
  fabricated "typing" suspense; the diff crystallizes only at turn-end (real gap,
  not theater). First visible increment = the activity stream. Raised the
  observer-vs-controller clash (below).
- **Game:** the first FELT moment is "oh — a real worker is doing my task." Dead-air
  (§12.1) over a 30–90s real run is handled by streaming each real event the instant
  it arrives, never by a fake spinner; honest silence is a real signal the Lead
  learns to price into task-scoping.

## Clashes

- **Observer vs. controller (UX, deferred):** a live harness is a black box that may
  backtrack mid-turn; should the Board show the live (nondeterministic) beat stream
  or only the settled minted trace? Resolution for slice 1: the reducer settles at
  turn boundaries (deterministic outcome) and surfaces activity as advisory — no
  mid-turn UI decisions (auto-land/fund) key off live beats. Mid-turn
  interrupt/redirect is a later (P3) coupling.
- **Premature interface (TDD vs Refactoring, resolved):** take `io.Reader`, not a
  `HarnessReader` abstraction — both the real `exec.Cmd` stdout and the fixture
  satisfy `io.Reader`; if the stream contract changes, refactor the translator.

## Slice plan (this thread, tdd-rygba; commit+push; CI; docs)

- SLICE 1 (this tick): `internal/harness.Supervisor` — reads a harness stream-json
  stream from an `io.Reader`, accumulates `translate` UI events per turn, and at each
  `turn.ended` calls `orchestrator.SettleTurn`, threading the new SHA forward as the
  next turn's base. Returns one settled `Turn{Events, Outcome}` per turn boundary.
  Fully testable with a scripted fixture stream — no subprocess, no API key.
- SLICE 2 (next): the real-process adapter — spawn `claude -p --output-format
  stream-json` via `os/exec`, expose its stdout as the `io.Reader`. Verified by
  build/vet + a manual run (API-key-gated), never fake-tested.
- SLICE 3+: publish activity events live to the existing surface (off-ledger);
  wire the settled revision into the work-order fill path (replace the pre-funded
  diff with the live one); containerize the agent run (its own gated round).

## New clashes opened / resolved

Opened: observer-vs-controller (deferred to P3). Resolved in convergence: package
placement, the `io.Reader` boundary, the economy firewall (harness mints nothing).
</content>
