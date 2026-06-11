# Round 71 — slice 4c: surface the live agent's activity on the card — 2026-06-11

Trigger: slices 4a/4b shipped (a live work order produces a revision and mints a
catch). Slice 4c surfaces the live agent's activity (thinking/editing/tool) on the
card. A 3-lens council (UX, Systems, TDD) chose the surfacing mechanism, reconciling
R69's "activity → scratch bus" guidance against new implementation facts.

## New info since R69

- The card's existing "watch it fill" beats are surfaced SERVER-SIDE by polling a
  per-session in-memory buffer (`fillSnapshot`), NOT via the bus — `runOneOrder`
  accrues `pipe.TraceEvent` kinds into `fillBeats`; the card's `via.Stream` polls it
  every 100ms and writes a re-render cell.
- `orchestrator.PublishActivity` (R68) puts activity on the scratch BUS, but the
  single card doesn't otherwise need cross-process fabric coupling, and `ledger.Log`'s
  fabric is unexported.

## Convergence (3/3) — mechanism A: a per-session activity buffer

UX, Systems, TDD all chose the BUFFER-POLL mechanism over the bus for the card line:

- UX: a DISTINCT "latest activity" row (separate from the oracle fill-beat row),
  showing the LATEST beat only (e.g. "editing internal/auth/token.go"), replaced not
  appended; ABSENT during dead-air (no spinner — honest silence). The agent's activity
  row and the oracle's fill-beat row coexist as two concurrent observers of the run
  (agent works → oracle verifies), both vanishing when the verdict lands.
- Systems: both mechanisms are firewall-safe (activity never touches the ledger; only
  minted/claim project the economy). The bus (B) is YAGNI — no fleet-activity consumer
  exists yet; keep `PublishActivity` as a READY brick (not dead code) for a future
  cross-session /fleet activity ticker, marked as such.
- TDD: the buffer is HONESTLY server-testable (drain a live order with a scripted
  runHarness emitting thinking+editing, assert the session's activity snapshot surfaces
  those beats; the card cell re-render is vt-testable). The bus path needs the CARD to
  subscribe client-side → browser-only verification. Anti-theater: do NOT mock
  PublishActivity / fake the SSE client / assert internal buffer slice state directly.

CLASH RESOLVED: card = session-scoped activity buffer (the card's own live beats); bus
= future cross-session /fleet monitoring. Conflating them would break session
isolation (R18). The bus brick stays on the shelf for the fleet slice.

## The build wrinkle the council surfaced — LIVE streaming needs a supervisor seam

The agents assumed the agent's beats accrue live like the oracle fill-beats. They do
NOT yet: `harness.Supervisor.Run` / `RunProcess` return the full `[]harness.Turn` only
at EOF (run completion). So feeding the activity buffer LIVE during the 30–90s run
requires a STREAMING callback through the supervisor (emit each turn's activity events
as they are processed), not just reading the returned turns post-hoc. Surfacing the
beats only after the run completes would be honest but NOT "watch it work live".

## Slice plan (4c; tdd-rygba; gate green; docs fresh)

- SLICE 4c-i (NEXT — BUILD): add a live-activity callback seam to the supervisor —
  `Supervisor.Run` (and `RunProcess`) take an optional `onActivity func([]translate.UIEvent)`
  invoked per turn as events are processed (the reducer already loops per line). Pure-ish
  to test: a scripted stream invokes the callback with each turn's activity in order.
  The `runHarness` seam signature gains the callback.
- SLICE 4c-ii: `liveEntry` gains an activity buffer (mirror fillBeats:
  `addActivityBeat`/`activitySnapshot`, latest-beat-only); `runLiveOrder` passes a
  callback that accrues into it; the card's Stream poll renders a distinct "latest
  activity" line (absent on dead-air). Server-render-tested via the vt pattern; the
  live SSE update is browser-verified.
- SLICE 5+: containerize the agent run; (later) the /fleet cross-session activity
  ticker off the PublishActivity bus brick.

## Build record — slice 4c-i SHIPPED

`harness.Supervisor` gained a non-breaking functional option `WithActivity(fn
func([]translate.UIEvent))` (+ `Option` type; `New` is now variadic — existing
`New(dir,base)` calls unaffected). `Run` fires the callback per stream line with
that line's activity events the moment they are read — BEFORE the turn settles — so
beats stream live. Purely additive: the pending/turns accumulation, settle-at-turn-
end, and base-threading are unchanged. tdd-rygba: Red → Yellow (added a test proving
activity streams for an in-progress/unsettled turn — the essence of "live") → Green →
Blue (all paths covered; per-line scoping + ordering sound) → Audit (clean; no race;
14 New call sites still compile). `-race` green; vet clean. (A flaky
`internal/sandbox` container-reap test failed once under the degraded local env —
full tmpfs pressuring Docker — and passed on retry; 4c-i touches only
internal/harness. CI is the gate.)

## Build record — slice 4c-ii-a SHIPPED (the live data path)

`harness.RunProcess` gained an `onActivity func([]translate.UIEvent)` param
(threads `WithActivity` into the supervisor when non-nil). The `runHarness` seam +
all 6 existing stubs updated to the 4-arg signature. `liveEntry` gained a per-session
`activityBeat` (latest activity line, under `fillMu`, reset in startFill / cleared in
endFill — bracketed to the fill lifecycle); `addActivityBeat`/`activitySnapshot`. A
pure `formatActivity(UIEvent)` ("thinking" / "editing <file>" / "running <cmd>" /
detail-or-kind fallback). `runLiveOrder` passes a callback that pushes the LATEST
event of each streamed batch into the buffer. tdd-rygba: Red → Yellow (added a
multi-event-batch assertion proving the latest wins) → Green → Blue (all arms +
branches covered) → Audit (clean; -race green; endFill is defer'd so the harness-error
path can't leak the beat to a later order). The live test observes `activitySnapshot`
MID-RUN from inside the stub (synchronous drain) → ["thinking","editing auth.go",
"running go test ./..."]. Full suite 20/20, vet clean. Remaining: 4c-ii-b renders the
beat on the card via the Stream poll.

## New clashes opened / resolved

Resolved: the 4c surfacing mechanism — per-session buffer poll (not the bus) for the
card; bus reserved for future fleet monitoring. Surfaced: live streaming needs a
supervisor callback seam (the returned-turns API is batch-only).
</content>
