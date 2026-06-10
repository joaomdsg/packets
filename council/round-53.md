# Round 53 — create a session from the UI (session-management thread, slice 1) — CONVERGED + BUILT — 2026-06-10

Trigger: the order-diagnostics thread (R51 persist verdict, R52 render it) read
end-to-end. The creative council picked the next thread.

Panelists: Systems (reachability) + Product/flow — a 2-voice creative council.

## The choice + a rejected candidate

- REJECTED (skeptic gate): an order DRILL-IN detail page. R52's inline "why" already
  shows target/status/verdict/caught per order; a detail page adds nothing without
  per-order transcript persistence (more plumbing). Marginal now.
- CHOSEN: SESSION MANAGEMENT — the maintainer's original "interfaces and MENUS, full
  user flows" gap. Sessions were created only at boot (the cmd/packets AddSession
  loop); a Lead could not start a new economy from the UI. Both council voices
  converged here; the product voice tied it to VISION's parallel-economies /
  "running a shop" mastery shape (run a refactor in one session, a POC in another,
  zero cross-talk — R18 farm-denial per session).

## Reachability — verified BEFORE building (the standing gate)

The systems voice + a direct read confirmed: AddSession(key,cfg) validates
fabric.ValidToken, binds a per-key ledger, registerSession→liveReg.Store (sync.Map,
concurrency-safe vs the board's Range). The in-process CARD flow (View/OnConnect/
Spend) needs NO claim consumer — consumers serve only the #6c untrusted-producer
POST /claim path. So a runtime-created session works IMMEDIATELY for the card flow,
NO lazy-consumer refactor needed. The via action mechanics were also de-risked:
on.Click renders a BARE @post('/_action/<method>') (no cmpID), and the component is
resolved via the via_tab — so a BoardCard action fires exactly like LiveCard.Spend
despite the non-root /board mount.

BUILT (commit 78eb700): BoardCard gains a NewKey signal (two-way bound input) + a
CreateSession action: read NewKey → if empty/invalid-token/duplicate, no-op (never
forge a bad token nor clobber a live economy) → else clone the default session's
cfg and AddSession. View renders a calm .board-create input + button. The created
session appears on the board and is reachable at /?key=<new>.

HONEST LIMITATION (documented): a runtime-created session has no claim consumer →
producer POST /claim is unsupported for it in V1; the card flow works fully. The
lazy-consumer refactor is a later slice if producer-claims for runtime sessions are
wanted.

Load-bearing tests: create → reachable board row + card; the control renders
(/_action/CreateSession + data-bind="newkey"); invalid ("bad key") and duplicate
("default") are no-ops. Blue confirmed both guards load-bearing (the duplicate guard
prevents AddSession clobbering the default's ledger), no structural regression, cfg
copy sound (slice fields read-only or ledger-mediated). Audit confirmed isolation is
keyed by the fabric session token (LedgerPath is dead post-boot), no cross-talk.
Full-repo gate green.

## New clashes opened / resolved

None. SESSION-MANAGEMENT thread continues: R54+ options — switch/active-session
affordance; rename/retire a session; the lazy-consumer refactor (to give runtime
sessions producer-claim support). Reachability + calm + data-honesty + two-scores
guardrails stand; #6 live boundary gated.
