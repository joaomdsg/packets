# Round 19 — the second session: isolation is a test, the shop-vs-tax feel is a human session — 2026-06-04

Trigger: the Round-18 #16b wave BUILT and SHIPPED green (liveState re-keyed to
the per-session sync.Map registry, behavior-preserving). Eleventh consecutive
build-evidence wave.

Panelists: all six. No new lens. CLEAN 6/6 convergence on
#16c-second-session-isolation, zero new target-level clashes.

New evidence (verified by reading code + the vendored ../via):

- #16b made a second card STRUCTURAL (liveReg holds *liveEntry per key, each its
  OWN *ledger.Log from cfg.LedgerPath) but changed nothing a Lead can SEE:
  setLiveState hardcodes defaultSessionKey, NewServer mounts ONE route
  (via.Mount[LiveCard](app,"/")), lookupLiveEntry falls every connect back to
  default, cmd boots one -base/-fix/-file/-line target.
- The economic substrate for ISOLATION is already structural: two entries = two
  ledger files = two independent balances; nothing in the credit/debit path
  crosses entries.
- ROUTING (unanimous diagnosis): ctx.ID() is a per-TAB id (server-minted,
  random), NOT a Lead-chosen session selector — so #16b's ctx.ID() keying can't
  aim a connect at a chosen session. A connect must reach a SPECIFIC session via
  a Lead/URL-controlled key. VERIFIED in ../via: a struct field tagged for URL
  decode is written into the per-connection component INSTANCE (render.go:17
  reflect.New(d.typ); :39/40 decodePathParams/decodeQueryParams), and the action
  handler retrieves the SAME persisted instance by tab id (action.go:109
  getCtx(tabID)) — so the key, decoded once at the initial render, PERSISTS and
  is readable in View AND in the action handlers (OnConnect/Spend), resolving the
  worry that the action ctx exposes only ctx.ID().

## Per panelist (ALL SIX advocate #16c-second-session-isolation, identical mechanism)

Re-key the session selector from ctx.ID() to a Lead/URL-controlled key on the
LiveCard instance; add a registerSession(key,cfg,log) that Stores a distinct
*liveEntry (own ledger + sem) so cmd can seed ≥2 targets each with its own
LedgerPath; route View/OnConnect/Spend through the card's key (fallback to
default when empty, preserving the single-card wire). The testable deliverable is
the inverting isolation RED (distinct keys → distinct ledgers → keyA.Balance==1
AND keyB.Balance==1, Spend on keyA leaves keyB) — every lens specifies it
identically. UX/Game/CI/CD/Refactoring add: deliver a watchable two-card surface
a human opens side by side; TDD + all are HONEST that the shop-vs-tax FEEL is a
human session, not a green test. Refactoring also flags the two stale "liveState"
prose comments (cost_test/cap_internal) to fix in-scope.

## Build-verified routing decision

(A correction to the chair's presumptive /s/{key} path route, grounded in
../via): checkPathParams (composition.go:194) PANICS if a path-tagged field has
no matching {seg} in the route — so mounting LiveCard (with a path key field) at
BOTH "/" and "/s/{key}" is impossible, and dropping "/" would force editing the
preservation suite (which connects to "/"), breaking the zero-edits proof. A
QUERY param (query:"key") is route-segment-FREE (querySlots decode via
decodeQueryParams the same way and persist per-tab identically) — so "/" stays
mounted (key="" → default, preservation suite untouched) and the Lead selects a
session via /?key=a vs /?key=b. The build uses query:"key", not the path route,
for this verified back-compat reason.

Clashes touched: D (1:N shop vs context-switch tax) — NOT resolved by this slice,
BY DESIGN: it is the first headline goal a green test CANNOT close. #16c makes D
WATCHABLE for the first time (two live cards) + instruments dwell/idle/rework as
RAW OBSERVATION ONLY (explicitly NOT a pass/fail oracle — the flaky-vs-intermittent
and catch-weight findings warn against laundering watched telemetry into a
score). The economic boundary (isolated ledger vs shared Treasury) is RATIFIED in
favor of per-session ISOLATED ledgers (R18 farm-denial default); the isolation
RED enforces it. The shared-Treasury farm exploit is pushed to #16d (where
dispatch makes it reachable; the isolated-ledger schema forecloses it).

Verdicts updated: none flip; #16c is the SETUP that makes Clash D adjudicable by
a human and locks the isolated-ledger economic boundary.

New clashes opened: NONE (6/6 report none). The OnConnect/Spend-only-have-ctx.ID()
concern is the shared diagnosis (resolved by the persisted-instance key,
build-verified), not a clash. Build-order (TDD: RED-first) is ordering inside the
agreed brick.

## Decisions

1. NEXT BUILD (#16c, CLEAN 6/6): register a SECOND keyed session + route to it by
   a Lead-controlled QUERY key, so two DISTINCT, ISOLATED cards become reachable
   side by side (/?key=a, /?key=b), each its own ledger/balance/sem. THE FIRST
   SLICE WHOSE HEADLINE GOAL (Clash D) IS NOT CLOSEABLE BY A GREEN TEST.
2. PREREQUISITE sub-bricks IN ORDER (the testable core via tdd-rygba): [a] add
   `Key string \x60query:"key"\x60` to LiveCard; route View/OnConnect/Spend
   through c.Key (fallback to defaultSessionKey when empty); the "/" mount is
   unchanged (no key → default), preservation suite green; [b]
   registerSession(key, cfg, log) → liveReg.Store(key, &liveEntry{...}) with its
   OWN ledger.Open(LedgerPath) + sem; keep setLiveState seeding defaultSessionKey
   for the single-target wire; [c] the isolation RED (internal test, swap
   resolveCycle to a no-real-oracle fake that mints one Catch Record): register
   keyA{ledgerA} + keyB{ledgerB}; connect /?key=keyA (+SSE → mints to ledgerA)
   and /?key=keyB (→ ledgerB); assert ledgerA.Balance==1 AND ledgerB.Balance==1
   (NOT 2 shared); fire Spend on the keyA client → ledgerA→0, ledgerB stays 1;
   [d] cmd/agntpr grows ≥2 review targets (a repeatable -session flag) each
   registered with a distinct LedgerPath — WIRING, verified by build/vet; [e]
   fix the two stale "liveState" prose comments in cost_test/cap_internal.
3. ACCEPTANCE FIXTURES: [isolation RED, the testable half] as in [c];
   [same-key-still-shares, preservation] cost_test.go:87 (2 sequential
   same-default-key connects → one ledger, Len==2) stays GREEN UNEDITED +
   registry_internal_test fallback contract green; [back-compat wire] the "/"
   route + single-target cmd still serves the one default card; [routing unit] a
   connect with ?key=keyB resolves Spend/OnConnect/View to keyB's entry
   (c.Key=="keyB"), proving the URL — not ctx.ID() — is the selector.
4. HUMAN EXPERIMENT (NOT a green test): a Lead opens /?key=a and /?key=b in two
   tabs, lets both catch cycles run live, triages both (streamed beat rows,
   in-flight→resolved verdicts, draining balances), and JUDGES whether two live
   cards feel like a SHOP (1:N leverage) or a context-switch TAX (thrash). The
   build delivers: (1) testable isolation [green RED], (2) a watchable two-card
   surface [/?key=a + /?key=b], (3) dwell/idle/rework as logged RAW observation.
   The VERDICT on D is the watched session, never a go-test assertion.
5. RANKED ROADMAP: [#16c THIS WAVE] second keyed session + query routing + cmd ≥2
   targets + the isolation RED; [#16d dispatch-consequence] give Spend's dispatch
   a real consequence (the first cross-process producer — only here does a bus
   earn its keep, and the shared-Treasury farm exploit becomes reachable,
   foreclosed by the isolated-ledger schema); [NATS] deferred until #16d;
   [triage/attention-queue UI] deferred (leverage-needs-a-dependency-graph makes
   it half-uncomputable today); [multi-Lead/co-review] far-deferred
   (multiuser-rewrites-the-scoring-spine); [#13 multiset], [#11.5 rename-cliff].
6. BLOCKERS: (a) the Lead sees one card until #16c routes a second; (b) routing
   must be Lead/URL-controlled (query key), not ctx.ID(); (c) Clash D's verdict
   is a human session, not a test — the build only SETS IT UP; (d) the dispatch
   verb stays dangling until #16d; (e) NATS waits for a cross-process producer.
   NO VISION/DESIGN text changed (12-contradiction reconciliation queued per
   RISKS step 5).

CONVERGED (15th consecutive round, CLEAN 6/6): all six ratify
#16c-second-session-isolation — register a second keyed session, route to it by a
Lead-controlled key, isolate the ledgers — with the identical inverting isolation
RED. The headline goal (Clash D) is, for the FIRST time, NOT closeable by a green
test: the build delivers the testable isolation + a watchable two-card surface +
raw instrumentation, and the shop-vs-tax verdict is an explicitly-defined human
session. Build-verified correction: routing uses a query:"key" param, not the
chair's presumptive /s/{key} path route, because checkPathParams would panic on a
path key at "/" and break the preservation suite — query keys preserve "/" with
zero test edits. Economic boundary ratified as per-session ISOLATED ledgers
(farm-denial); the shared-Treasury exploit deferred to #16d. NATS deferred. Next
event is a BUILD — [a] key the LiveCard via query → [b] registerSession → [c] the
isolation RED → [d] cmd ≥2 targets — not another round.
