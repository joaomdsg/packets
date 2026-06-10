# Round 36 — #6c slice C4: the bets-vs-confirmed visual without a stylesheet — CONVERGED (no CSS; structural grouping) — 2026-06-10

Trigger: C3b (R35) put the producer claim lifecycle data (in-flight bets,
verified-lost) on `/board` and live on `/fleet`, closing the lie-green gap. C4 is
the purely-presentational step — make a BET read as distinct from a confirmed
CATCH — and it forced Clash J: the repo has NO stylesheet, so "gray" can't be a
color today. (a) introduce a minimal CSS asset, or (b) stay CSS-free with class
hooks + label semantics.

Panelists: Calm-UI Minimalist, Two-scores Guardian, Frontend/Asset-serving
Engineer, Shipping Pragmatist. (Autonomous council per the standing steering
directive — convened, debated, converged without maintainer input.)

## Per panelist

- 🎯 Calm-UI Minimalist (CSS-free / b): the bet-vs-catch separation is already
  semantically honest in distinct class hooks (`board-row__inflight`/`__rejected`
  vs `__stock`) and distinct labels ("in flight"/"verified-lost" vs "confirmed").
  Introducing a stylesheet to gray one span is premature decoration that breaks
  the repo's CSS-free contract for no functional gain. Leave the hooks; color
  later if a stylesheet ever lands for a bigger reason.
- ⚖️ Two-scores Guardian (no-CSS, but STRUCTURE needed): color would be safety,
  not polish — BUT achievable without CSS. The current FLAT span ordering
  ("5 confirmed, 3 reinvested · 2 in flight · 1 verified-lost") blends at
  glance-speed; labels are precise but parsed serially. Fix = a structural seal:
  cluster the unresolved-bet spans under an explicit "bets:" boundary, distinct
  from the confirmed "caught" stock — typographic/structural separation, no
  stylesheet. Flagged the flat ordering as a genuine confusability hazard.
- 🔧 Asset/serving Engineer (CSS-free / b): `via`'s `h` package DOES expose
  `h.StyleEl`/`h.Raw`, so an inline `<style>` is mechanically cheap — but it
  adopts a visual layer the project deliberately avoided, and only `/` and
  `/board` are HTML; `/stream` and `/fleet` are machine SSE/JSON APIs a
  stylesheet can't touch. The class hooks are a clean seam an external consumer
  (or a future stylesheet) can map. Recommend (b); defer CSS until a real driver.
- 🚀 Shipping Pragmatist (defer color): the lie-green gap is ALREADY closed by
  C3b — verified-lost is a labeled count on both surfaces. A stylesheet pays an
  asset-pipeline cost for cosmetic gain with a clean retrofit path (the hooks are
  there). Defer actual color to a real stylesheet driver.

## Chair adjudication

Unanimous (4/4): NO stylesheet this slice. The split was defer-entirely
(Pragmatist) vs a minimal no-CSS structural fix (Guardian, accepted by Minimalist
& Engineer as structure+labels, not color/gauge). RECONCILED: C4 ships the
Guardian's structural grouping — it is a real HONESTY improvement (the two-scores
separation becomes legible at a glance, not just on serial label-parsing), costs
no asset pipeline, stays within the calm idiom, and leaves the class hooks intact
for a future stylesheet to add color. Pure-defer is rejected because the flat
ordering is a genuine confusability hazard the grouping cheaply removes;
stylesheet-now is rejected because color is polish with a clean retrofit and the
machine surfaces (`/fleet`,`/stream`) can't use it anyway.

## Decision — converged C4 (to build)

CSS-FREE structural grouping on the in-process `/board` row: render the producer
bet lifecycle (in-flight + verified-lost) as one explicitly-labelled "bets"
cluster, structurally/typographically sealed off from the confirmed "caught"
stock — distinct from balance/activity too. Keep the existing class hooks
(`board-row__inflight`, `board-row__rejected`) so a future stylesheet can color
them with no server change. NO stylesheet, NO color, NO gauge. The `/fleet` JSON
frame already carries distinct `in_flight`/`rejected`/`confirmed` keys (a machine
API — no grouping change needed there). Color is explicitly DEFERRED to a future
stylesheet driver.

## Verdicts updated

- **Clash J — RESOLVED (R36):** no stylesheet now; bet-vs-confirmed separation is
  carried by structure + label semantics (a grouped "bets" cluster vs the
  "caught" stock), with class hooks left ready for future color. Color deferred
  until a real stylesheet driver, where it becomes a no-cost addition on the
  existing hooks.

## New clashes opened

NONE. A future "introduce the first stylesheet" decision (dark mode / design
system / cross-row color harmony) is noted as the natural driver that would
retrofit color onto the C4 hooks — not a clash, a deferred enhancement.
