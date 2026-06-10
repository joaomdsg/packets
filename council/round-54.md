# Round 54 — per-session claim consumers, including runtime-created sessions (session-management thread, slice 2) — CONVERGED + BUILT — 2026-06-10

Trigger: R53 let a Lead create a session from the UI, but documented a V1
limitation — a runtime-created session got no claim consumer (StartClaimConsumers
snapshotted liveReg once at boot), so its producer claims would publish and never
verify. This slice closes that gap.

Panelist: a creative council pass (Systems + Product synthesis). It weighed the
remaining session-management options (switch is already covered by R44 drill;
rename is REJECTED as unreachable — the key IS the fabric subject token / ledger
namespace, so renaming orphans the ledger; retire/remove = board hygiene, lower
leverage) and chose the LAZY-CONSUMER REFACTOR: it closes a real architectural gap
now and is fully testable in-process with the stub verifier (the consumer LIFECYCLE
is what's tested; real external producers stay behind the gated #6 live boundary).

## Decision — CONVERGED on the lazy-consumer refactor (built this round)

BUILT (commit 7d797e8): a claimConsumerSpawner (package global) spawns EXACTLY ONE
durable claim consumer per session — for sessions present when StartClaimConsumers
runs AND for any session registered later (registerSession calls onRegister, which
spawns under the lock if consumers are active). A `started` set dedups so a session
is never double-consumed. StartClaimConsumers now activates the spawner + spawns for
all current sessions (was: a one-shot Range). cmd/packets is unchanged — its single
StartCageClaimConsumers call now also arms per-session births, so a prod
runtime-created session gets a cage consumer.

Two real defects caught + fixed in the cycle (not by the green bar — by adversarial
audit):
1. CLOSURE-CAPTURE RACE (Blue mis-analyzed it as safe; re-derived from Go
   semantics): `go func(){ … s.ctx … }()` reads the spawner fields INSIDE the
   goroutine, after mu is released — a latent race with a later StartClaimConsumers
   write. FIX: copy ctx/verifier/ackWait/adm into locals under the lock; the
   goroutine closes over the locals. (-race passed before the fix only because no
   concurrent write occurred in the test timeline — a green bar that hid a real
   race.)
2. CROSS-TEST GLOBAL CONTAMINATION (Audit, reproduced with -count): the never-reset
   package globals leaked a prior test's stale liveReg entry; a later test's
   StartClaimConsumers Ranged the stale key, marked it `started`, and starved the
   fresh same-key session of a consumer → flaky fail. FIX: resetConsumersForTest
   clears liveReg + the spawner in place (zeroing fields, never reassigning the
   struct under its own held mutex), called in each consumer test's setup. Prod is
   untouched (one server per process; globals never torn down there).

Load-bearing test: a session created via AddSession AFTER StartClaimConsumers gets
a consumer that verifies a posted claim and mints (the R53 case). Existing
StartClaimConsumers tests + cmd cage-wiring test stay green; full-repo -race gate
green (consumer tests also pass -count=5 -race). Stale "call EXACTLY ONCE / snapshots
the registry" docs on StartClaimConsumers + StartCageClaimConsumers updated to the
new per-session behavior.

## New clashes opened / resolved

None. The session-management thread now delivers: runtime session CREATE (R53) +
full producer-claim support for runtime sessions (R54). R55+ options: a
retire/remove-from-fleet affordance (board hygiene); an active-session highlight;
or a fresh creative thread. Reachability + calm + data-honesty + two-scores
guardrails stand; #6 live boundary gated.
