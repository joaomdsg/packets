# Round 42 — direction (maintainer re-delegated to the council): make the work-order round-trip legible — CONVERGED — 2026-06-10

Trigger: the maintainer re-authorized autonomous progress and handed DIRECTION
back to the council ("convene council for direction and keep working"). Round 41
had paused the loop at the edge of autonomous-safe work; this round picks a
NON-GATED, IDIOM-SAFE thread to build.

Standing constraints (unchanged): NOT the live #6 boundary ("keep working" is not
the explicit security sign-off that gate needs); no fabricated leverage/priority
rank (refused R24/R36/R41); no invented cross-session treasury pool; calm
CSS-free server-rendered idiom.

Panelists: Product/Vision, Calm-UI + Pragmatic TDD.

## Per panelist

- 🎨 Product/Vision: VISION promises the Lead "watches a funded order round-trip,"
  but today the board shows only AGGREGATE queued/running/done counts — the Lead
  cannot connect "the order I funded" to "this catch landed." The honest,
  no-fabrication gap is per-ORDER legibility: surface recent dispatches (which
  work-order ran what target, and whether it CAUGHT or MISSED). All real logged
  data (WorkOrderRecord + status + the "wo:<id>" catch provenance). Recommends a
  calm "recent dispatches" surface.
- 🧪 Calm-UI + Pragmatic TDD: ranked the candidates — (a) making /board itself
  live off the stream is BLOCKED (it's a request-scoped via.Mount that recounts
  all sessions per render; going live needs the bridge-subscription plumbing, not
  a thin seam); (c) the control-char `-z` residual is exotic/low-value; (d) the
  bets-cluster legibility is already done (R40). The work-order round-trip (b) is
  the one clean, idiom-safe, TDD-able-through-the-vt-client slice with no CSS, no
  fabricated rank, no new untrusted surface.

## Chair adjudication — CONVERGED

Build the per-session RECENT-DISPATCHES legibility surface (Product's form, which
satisfies Calm-UI/TDD's buildability bar). It renders, per board row, that
session's recent work-orders with their target, status, and caught/missed
outcome — derived entirely from existing ledger data (WorkOrders + the folded
status + catches keyed Producer="wo:<id>"). Honest (no fabrication), idiom-safe
(calm spans, per-session "one row never speaks for another", no rank/treasury, no
CSS), non-gated, TDD-clean. Making /board fully live off the stream is a separate,
bigger slice (bridge subscription) — deferred.

## Decision — next build (2 units)

- UNIT 1 (ledger): `type DispatchView{ ID int; Target Target; Status string;
  Caught bool }` + `(*Log).RecentDispatches(n int) ([]DispatchView, error)` —
  fold the log: per work-order, its current status (latest status event, default
  "queued"), its Target, and Caught = a catch exists with Producer=="wo:"+id.
  Return the most-recent n (by stream/append order). Pure projection over the
  log, TDD-clean.
- UNIT 2 (board): CardRow gains the recent dispatches; BoardRows reads
  e.log.RecentDispatches(n) guarded (degrade to nil on error, like the other
  fields); BoardCard.View renders a calm per-row "dispatches" cluster
  (WO#id target status caught|missed) — CSS-free, distinct span/class hooks.
- Load-bearing tests: ledger — a funded+run order that minted shows Caught=true;
  a done order with no wo-catch shows Caught=false (missed); a queued order shows
  status "queued"; ordering/limit. board — the /board HTML shows a session's
  recent dispatch with its caught/missed outcome (vt.NewClient render).

## New clashes opened / resolved

None. Reaffirms the fabricated-leverage refusal as the reason the board surfaces
HONEST per-order outcomes (caught/missed) rather than a synthesized rank.
