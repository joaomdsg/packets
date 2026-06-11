# Round 73 — next thread: the Trust Ledger's autonomous-safe first slice — 2026-06-11

Trigger: the LIVE-HARNESS thread (R67–R72) is autonomous-complete + integration-
proven (a real Claude Code harness, dispatchable via `-live`, produces a fix and
mints a confirmed catch on a pre-specified anti-farming anchor, streaming activity to
the card; only the ISOLATED agent CONTAINER remains, GATED on maintainer sign-off). A
4-lens council (Game, Systems, TDD, UX) chose the next thread.

## The strategic context

RISKS' meta-finding: "P0→P2 proves the pipe thesis; all 8 scoring/trust-integrity
risks live BEYOND P2 — in the VISION trust-economy thesis VISION itself calls the
groundbreaking part." The pipe (P0→P2) is now built + proven, so the TRUST LEDGER /
calibrated-delegation economy is the strategically highest-value remaining thread.
History (R41/R59) flagged the DEEP economy (Delegation Tiers, force-deep, Focus) as
needing maintainer product/UX taste — so the question was whether there is an
autonomous-safe FIRST slice.

## Convergence (4/4)

YES — the autonomous-safe first slice is a READ-ONLY per-lane "SCOUTING REPORT": a
first-pass catch-rate computed PURELY from the already-logged `CatchRecord`s +
dispatch outcomes. No new mint, no model judgment, no taste call.

- TDD: the MOST testable candidate — a PURE projection `data→data` (mirrors the
  existing `ledger.Projection`/`Stock` methods), unit-testable with fixture records.
  The other candidates risk test-theater: (b) /fleet live-activity ticker is
  browser-coupled; (d) Ship-Quality/DORA needs the model-inferred catch-WEIGHT
  (V§13.5) + dwell/review-time telemetry that RISKS confirms does NOT exist; (c)
  RISKS-hardening is risk-mgmt, not a feature.
- Systems: it redeems against LOGGED facts only — `CatchRecord.Producer` ("wo:<id>"),
  `DispatchView.{Status, Caught}`, `SelfFlagged`, `WouldHaveShipped`. DEFER the
  un-grounded trust mechanisms RISKS flags: model catch-WEIGHT counterfactuals
  (violate "redeem against a logged event"), risk-tier partitioning, trust half-life,
  earned-concurrency cold-start, force-deep gates (all rest on fragile oracles / a
  skim-depth history the two-scores ledger doesn't carry). Counts-only, RETROSPECTIVE
  (what the logs said), never PREDICTIVE (what the agent will do) — prediction is the
  hard part the system earns later.
- UX: OUTWARD framing (V§13.2) — "this lane ships clean: 3/4 first-pass" (agent
  quality), NEVER "you skim accurately" (Lead self-grading → Sunday-night dread). Calm,
  on /board, server-render-testable. Autonomous-safe; only line PLACEMENT is a
  low-stakes IA call.
- Game: the honest FIRST BEAT of the "I know which agents I never need to read
  closely" fantasy (V§13.3) — evidence the Lead reads, not a system that grades. The
  tier/auto-land/cold-start mechanics are LATER beats, gated.

## Clashes resolved

- TDD vs Systems (pre-weighting): a tuned model to "surface agent quality early" is
  trust laundering (the R70 farming exploit via the back door — any model-inferred
  weight lets the denominator be gamed). RESOLUTION: counts-only, purely
  retrospective; the human's later review becomes ground truth, not the oracle. The
  catch-WEIGHT is a strictly-later layer on a solid foundation.
- Game vs UX (meta-optimization): a hot-lane stat could tempt the Lead to pick the
  hot lane over the right task. RESOLUTION: the scouting line sits BELOW the queue
  (a retrospective roster note), never ABOVE it as a dispatch hint — leverage orders
  the queue, trust is only depth (V§13.1).

## Slice plan (TRUST-LEDGER thread; tdd-rygba; gate green; docs fresh)

- SLICE 1 (NEXT — BUILD): a PURE projection over the logged catches + dispatch
  outcomes → a per-session FIRST-PASS catch-rate stat (caught orders ÷ completed
  orders, the session = the "lane"/"agent" — the cleanest mapping, the board is
  already per-session). Unit-tested data→data (fixture CatchRecords/dispatch
  verdicts). The session = lane partition is the minimal honest aggregate; per-path /
  per-task-type lanes + the self-flagged-accuracy dimension are LATER refinements.
- SLICE 2: RENDER it as a calm outward scouting line on /board per session
  (server-render-tested via the vt pattern; below the queue, never a dispatch hint).
- LATER (gated / un-grounded — DEFERRED): self-flagged-accuracy + would-have-shipped
  dimensions; per-path/task-type lanes; the catch-WEIGHT; risk-tier partitioning;
  trust half-life; earned concurrency; force-deep; Delegation Tiers (all need
  maintainer taste or a skim-depth history the ledger doesn't carry).

## Build record — slice 1 SHIPPED (the scouting projection)

`internal/ledger`: `ScoutReport{Caught, Completed int}` + `FirstPassRate()`
(Caught/Completed, 0 when Completed==0 — the caller gates on Completed>0 for
no-signal). `Projection.ScoutingReport()` (pure): Completed = orders with status
"done" (failed/queued/running are NOT a completed pass — a harness crash is infra,
not a missed catch); Caught = completed orders with a `wo:<id>` catch (the same
provenance logic as RecentDispatches — a "connect" catch never marks an order
caught). `Log.ScoutingReport()` wrapper. tdd-rygba: Red → Yellow (added a
catch-on-uncompleted-order test pinning Caught≤Completed) → Green → Blue (all
branches covered; error path acceptable parity) → Audit (clean; provenance-leak-free;
no panic on a stray wo:id; pure/deterministic). `-race` + full suite green. Slice 2
renders it on /board.

## New clashes opened / resolved

Resolved: the next thread (Trust Ledger) HAS an autonomous-safe first slice (a
counts-only retrospective scouting report); the predictive/weighted/tiered mechanics
stay deferred. Reinforces V§13.2 (outward framing) + the redeem-against-a-logged-fact
rule.
</content>
