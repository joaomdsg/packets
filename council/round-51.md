# Round 51 — persist the per-order oracle verdict (order-diagnostics thread, slice 1) — CONVERGED + BUILT — 2026-06-10

Trigger: the maintainer handed the council CREATIVE latitude ("council should
creatively steer next ticks") after the card/board visual+flow thread completed
(R43–R50). A 3-voice creative council convened to pick a bold new thread.

## The creative council (3 voices)

- Product-visionary: a "reinvest rhythm" loop (after a catch, suggest re-spending).
  Low plumbing but marginal + risks nagging (calm guardrail).
- Systems/data: ORDER DIAGNOSTICS — persist a per-order verdict so a missed order
  shows WHY. Reachability VERIFIED: the oracle's verdict is computed for every order
  run and currently discarded.
- Calm-UI/flow: a session-create MENU. Bold + on-mandate, but its first slice is
  REACHABILITY-BLOCKED: StartClaimConsumers snapshots the registry at boot, so a
  runtime-created session is a ghost (no claim consumer) — needs a lazy-consumer
  refactor first.

## Decision — CONVERGED on the ORDER-DIAGNOSTICS thread, slice 1

The systems/data thread won: highest value (the round-50 survey's named deepest
gap — "why did my order miss?"), its first slice is both reachable AND thin, and it
persists what the oracle ACTUALLY computed (honest, no fabrication). The session-menu
thread was set aside (its first slice needs the lazy-consumer refactor — a future
prerequisite slice). Reachability was independently verified before building:
runOneOrder computes `res` from resolveCycle, and `res.Verdict` (=
surface.PresentVerdict, one of the 8 verdict states) is available for EVERY order
run including misses — but only `res.Record` was used (to mint), so the verdict was
thrown away.

BUILT (commit 35f1688): a new ledger fact WorkOrderVerdictRecord (kind "woverdict")
on the same minted subtree as the work-order/status lines; AppendWorkOrderVerdict;
a per-id `verdicts` projection (last-writer-wins, mirroring status); and a
DispatchView.Verdict field surfaced by RecentDispatches. runOneOrder persists
res.Verdict for every order it runs (err==nil), before the "done" status. The
verdict is DIAGNOSTIC ONLY — it mints no balance and is not a confirmed catch
(two-scores), and an absent verdict reads empty (never a fabricated default).

Load-bearing tests:
- ledger: a persisted verdict surfaces on the order's DispatchView; an order with
  none reads ""; last-writer-wins across re-runs; the verdict never moves Balance
  nor counts as a ConfirmedCatch (two-scores).
- app: the runner persists res.Verdict end-to-end (Spend → run → RecentDispatches
  shows the verdict), via a hermetic resolveCycle stub.

Blue + Audit cleared the key risk: foldEvents is the SINGLE kind-demux on the
minted subtree and each kind has a distinct token, so no StatusMinted consumer
(SSE bridge, WatchFleet, counts) mis-decodes the new woverdict event; old logs
replay fine (empty verdict); no DispatchView struct-shape regression. Full-repo
gate green.

## New clashes opened / resolved

None. THREAD CONTINUES: R52+ renders the per-order verdict on the card + board (the
"why" made visible — reuse the data-outcome groove from R50), then a possible
drill-in order detail. The session-menu thread remains available (start with the
lazy-consumer refactor to make runtime sessions reachable). Guardrails stand;
reachability gate stays a slice-selection criterion.
