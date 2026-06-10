# Round 35 — #6c slice C3b: where the producer claim lifecycle surfaces — CONVERGED, shipped — 2026-06-10

Trigger: the cage verification flow (R34) is built and wired live — producers
POST claims, the host verifies in the cage and mints, and C3a made a rejection
DURABLE (`ledger.ClaimVerdict`). But a verified-but-rejected bet was visible
NOWHERE: it left the in-flight count and appeared on no surface — the "lie-green"
gap, a wrong bet silently vanishing. The fork: where does the claim lifecycle
(pending → verifying → confirmed | rejected) surface, and how much to build now?

Panelists: Calm-UI Minimalist, Producer-experience Advocate, Two-scores Guardian,
Streaming/SSE Engineer, Shipping Pragmatist. (Run autonomously as grounded
Explore subagents; persona seeds below for re-summoning.)

## Per panelist

- 🎯 Calm-UI Minimalist (REFUSE-on-live-card): claims do NOT belong on the live
  review card — its idiom is "one row never speaks for another," and the card is
  the HOST's single subject (its one verdict). A producer's bet is an alien
  actor's economy; even a calm "verifying N" span hijacks the verdict row's sole
  franchise. The board already renders peers (sessions), so cross-actor claim
  state is contextually honest THERE, foreign on the card.
- 📨 Producer-experience Advocate (per-claim, loud rejections): a producer must
  see received → verifying (~11s) → confirmed | rejected; a silent rejection is a
  broken contract (the system ran their work and discarded it wordlessly). Wanted
  per-claim status lines + a verifying pulse. Conceded the card has NO
  per-producer auth filter today, so "your bet" isn't yet distinguishable.
- ⚖️ Two-scores Guardian (labels, not color): pending bets and confirmed catches
  must occupy DISTINCT labeled regions; semantic labels ("bet"/"caught"/
  "verified-lost") carry the separation, not hue. A rejection is a RESOLVED
  loss, a normal outcome — render it neutral, never red/error. Confirmation is a
  PROMOTION (leaves the bet tally, enters stock), never a double-count.
- 🔌 Streaming/SSE Engineer (event-driven, no recount): the claim consumer is a
  background goroutine with no `*via.Ctx`, so it can't write the card's reactive
  cells. The raw SSE bridge already emits a frame PER COMMITTED EVENT; route
  claim state through it. "Verifying" needs no durable marker — derive it (a
  claim with no terminal verdict). DO NOT put `ClaimsInFlight` (2 stream replays)
  on a per-tick path — that's the C1a perf hazard.
- 🚀 Shipping Pragmatist (thinnest first): the highest-value move is just making
  the rejection VISIBLE (fix lie-green) as a labeled count; cut the verifying
  pulse, per-claim rows, and gray color to a later slice. Each count ships green
  independently.

## Clashes touched

- SURFACE LOCATION — Minimalist (board only) vs Advocate (live card, per-claim).
  RECONCILED: the board owns cross-actor tallies AND is request-scoped (no
  SSE-tick hazard) AND sidesteps the goroutine/Via-cell problem. The Advocate's
  per-claim case is premature without per-producer auth. → BOARD.
- SCOPE — Pragmatist (counts only) vs Advocate (pulse + per-claim). RECONCILED by
  the no-CSS finding (below): "verifying" == in-flight by definition (a claim
  with no terminal verdict), already shown by C1b, so "counts + verifying" needs
  no animation. → counts; the live-update is the pulse.
- MECHANISM — confirmed event-driven (the bridge re-folds per committed event),
  NOT a per-tick recount. The two-replay `ClaimsInFlight` stays on the
  request-scoped `/board` GET (already-cleared perf).

## Key finding (settles the scope debate)

The repo has NO stylesheet — the UI is server-rendered `h.H` spans with class
hooks only. So a literal animated "pulse" has no home; "verifying" is the
in-flight count, and the live update IS the pulse. This collapses the
pulse-vs-no-pulse axis: deliver lifecycle as live counts.

## Decisions — converged, all shipped

1. **C3b1** (commit 6961fac): `ledger.ClaimsRejected()` + a `board-row__rejected`
   "N verified-lost" span on the in-process `/board`, neutral language. A target
   now sits in exactly one bucket: in-flight, verified-lost, or confirmed.
2. **C3b2a** (commit c43a035): extract the pure projection seam
   (`claimsInFlightFrom`/`claimsRejectedFrom`) so the stream and the board share
   one classification — behavior-preserving refactor.
3. **C3b2b** (commit 4511732): the live cross-session fleet stream (`/fleet`) now
   wakes on the whole event taxonomy (`FleetEventsSubject`) and carries
   `in_flight`/`rejected` per row via `ledger.FleetBoard`/`FleetView` — a claim
   or verdict (no mint) drives a live frame. The verifying pulse, live.

## Verdicts updated

- Clash A (per-card cost signal / what shows on a card): reaffirmed the
  "one row never speaks for another" discipline — cross-actor claim state was
  kept OFF the live card and put on the board/fleet, where peers already render.
- Two-scores invariant held in code: in-flight and verified-lost are independent
  axes, never folded into balance/confirmed, proven on both `/board` and `/fleet`
  (full `-race -p 1` gate green at each commit; CI green for all three).

## New clashes opened

- **Clash J — the gray "bets vs confirmed" VISUAL (C4):** the data is now on
  both surfaces (in_flight/rejected); C4 is purely presentational. But the repo
  has no CSS, so "gray" can't be a color today. Positions: (a) introduce a
  minimal stylesheet (a real new asset/serving decision) to express bet-vs-
  confirmed as muted-vs-solid; (b) stay CSS-free and carry the distinction in
  class hooks + label semantics only, deferring color to whenever a stylesheet
  lands. Experiment: build C4 both ways on the rendered board and judge which
  reads as honestly-distinct without a stylesheet. → Round 36 settles it.

## Persona seeds (for re-summoning)

Calm-UI Minimalist: guards the card idiom "one row never speaks for another,"
calm stock spans, no gauges/priority/forecast; refuses anything that hijacks the
verdict row. Producer-experience Advocate: speaks for the cross-process producer
needing honest bet feedback; loud about silent rejections. Two-scores Guardian:
a pending bet is never a confirmed catch — distinct labeled regions, neutral
language, promotion-not-double-count. Streaming/SSE Engineer: owns cells vs the
raw per-commit bridge, the bg-goroutine constraint, and the per-tick-recount perf
hazard. Shipping Pragmatist: smallest green slice that fixes the most important
gap; ruthless cut-list.
