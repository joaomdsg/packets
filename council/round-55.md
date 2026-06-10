# Round 55 — retire a session from the fleet (session-management thread, slice 3) — CONVERGED + BUILT — 2026-06-10

Trigger: R53 (create a session) + R54 (per-session consumers) made the board a
command surface, but created experiment sessions accumulate with no way to clear
them. The creative council picked R55.

Panelist: a creative council pass (Product + Systems + Calm-UI synthesis).

## The choice + an honest deferral

The council weighed RETIRE-a-session vs the bigger PREP-BENCH / management-sim
depth and made a sharp call:
- RETIRE = the honest completion of the create affordance; reachable on existing
  plumbing (liveReg.Delete), fully testable, calm. CHOSEN.
- PREP-BENCH = REJECTED for now, NOT as marginal but as UNDER-SPECIFIED: VISION
  frames "sharpening the bench" but doesn't operationalize it (read-only backlog
  list? editable acceptance criteria? decompose hints? — different plumbing each).
  Shipping a council guess risks missing intent → this thread warrants a MAINTAINER
  PRODUCT DECISION, surfaced rather than guessed.

## Decision — CONVERGED on retire (built this round)

BUILT (commit b4ec407): BoardCard gains a RetireKey signal + a RetireSession action;
each NON-default row renders a quiet retire control that uses on.SetSignal to bind
THAT row's key into RetireKey just before the post (the per-row-arg pattern, via
on.SetSignal — researched + proven this round), so RetireSession removes the right
session. The seeded default is never retirable (it is the "/" route's single-card
fallback); an empty key is a no-op. Retire unmounts the registry entry only — the
ledger events persist on the fabric.

Adversarial audit (Blue + Audit) independently CONFIRMED the edges:
- An in-use tab on a retired key degrades gracefully — lookupLiveEntry falls back to
  the default, so View / Spend / drainQueuedOrders never nil-deref.
- POST /claim (and /stream, /bundle) 404 a retired key (the liveReg.Load gate), so a
  retired session receives no new claims.
- The retired session's durable claim-consumer goroutine parks on an empty fetch
  until shutdown — a BENIGN leak, documented on RetireSession, intentionally not torn
  down (teardown would risk racing an in-flight verify for no idle cost).
- The retire button is appended at row END, so it doesn't shift the bet-lifecycle
  structural index test; R53 create tests unaffected.

Load-bearing tests: retire removes a created session from the board; retire of the
default is a no-op (forces the default-guard); the retire control renders only on
non-default rows (default-only board has none). Full-repo -race gate green.

## New clashes opened / resolved

None. The session-management thread is now COMPLETE for V1: create (R53) +
producer-claim support (R54) + retire (R55). The next high-value thread (PREP-BENCH
/ management-sim depth) needs a maintainer product decision — surfaced this round.
Guardrails + reachability gate stand; #6 live boundary gated.
