# Prep Bench — User-Flow Spec

> The implementer-facing flow spec for the Prep Bench surface. Companion to
> `design-language.md` (it reuses that component vocabulary) and grounded in
> VISION §12.1 (dead-air → the Bench), §13.4 (never-empty + the pre-staged
> self-flag), §10.4 (anti-busywork), §12.6 (queue-zero is the win), §4.3 /
> §12.10 (a self-flag is a routing hint + calibration input, NEVER the spine
> of where you look). Every behavioral claim here is observable in
> server-rendered HTML (`vt.HTML()`-assertable). Build per the house TDD rule.

## Why this spec exists

The Bench today (`internal/app/prep_bench.go`) is a correct but partial
slice: a curation list of fundable `path:line` targets where each chip funds
the *chosen* target instead of the FIFO head (VISION §5 "Dispatch"). The
Bench's defining purpose — sharpening the next work orders during the ~90s
dead-air while an agent computes (split vague orders, write acceptance
criteria, pre-stage the computing agent's self-flagged spots) — is unbuilt.
This spec defines that build.

## Locked decisions

These were decided with the maintainer and are constraints, not options:

1. **Seeding is gated on ACTIVE dead-air only.** The Bench seeds prep cards
   only while an order is filling (`fillSnapshot` reports `id > 0`,
   `live.go:793-794`). Idle / queue-zero → the Bench is honestly empty
   (`renderBench` returns `nil`, the caller omits it). Never manufacture
   busywork (§10.4); an empty bench at queue-zero is the win (§12.6).
2. **The chip grows into a card.** Today's `bench__item` button is the
   *collapsed* form of one unified `pk-card`. The same card expands to carry
   a sharpen body. NOT two stacked lists.
3. **Sharpen persists as a new event-sourced ledger fact.** A refined
   work-order (a split result, attached acceptance criteria, an accepted
   convention) is appended to the log as one new `worefine` event; the
   `fundableBacklog` projection folds it on read. No Postgres write, no
   in-place mutation — mirrors the append-only `AppendDispatch` /
   `AppendStatus` idiom.
4. **Split is proposed-then-accept.** The system harvests candidate
   sub-targets from the live diff; the Lead edits/accepts them. This keeps
   the "system seeds, you sharpen" framing rather than pure manual authoring.
5. **Sequencing: ship flows a/b/d/e first; preflight-self-flag (flow c) is a
   fast-follow.** Flow c is blocked on a new mid-fill self-flag event that
   does not exist yet (see Dependency below); a/b/d/e carry no such
   dependency.

## The unified bench card

The Bench is a `<div class="bench">` inside the existing **act-now**
`<section aria-labelledby="act-now-label">` (`live.go:777-782`), rendered
after the `fund-work` group. Heading stays `.pk-section-label .bench__label`
"on the bench:".

Each item is promoted from a bare `<button>` to a `pk-card .bench__item`:

- a header row — the `path:line` target (mono) + a `.bench__kind`
  `.pk-section-label` tag (`split | criteria | convention | preflight`) +
  the FIFO-next marker on the head.
- a fund affordance — `.bench__fund .pk-btn--quiet[data-target="path:line"]`
  firing the existing `FundChosen` + `SetSignal(FundTarget)` wiring. Present
  on any *fundable* card; ABSENT on a `preflight-pending` card (nothing to
  fund until the diff lands).
- a sharpen body (only when expanded) — the kind-specific
  editable/actionable region.

### Class hooks (reuse `design-language.md` §2 vocabulary)

| hook | role |
|---|---|
| `.bench` | region wrapper (exists) |
| `.bench__label` `.pk-section-label` | "on the bench:" (exists) |
| `.bench__item` `.pk-card` | one unified card (was `.pk-chip .pk-btn--quiet`) |
| `.bench__kind` `.pk-section-label` | kind tag |
| `.bench__target` `.pk-chip` | the `path:line` mono key |
| `.bench__fund` `.pk-btn--quiet` | fund button (keeps balance-hue hover) |
| `.bench__sharpen` `.pk-btn--quiet` | expand/collapse toggle |
| `.bench__body` | the sharpen body (present only when expanded) |
| `.pk-input` | the editable region (criteria / convention) |
| `.bench__refine` `.pk-btn` | submit-refinement control (emits the fact) |
| `.bench__review` `.pk-btn--quiet` | (flow c) the `/review?wo=` handoff link |
| `data-state` | `collapsed \| split \| criteria \| convention \| preflight-pending \| preflight-ready` |
| `data-kind` | `fundable \| split \| criteria \| convention \| preflight` |
| `data-target` | `path:line` (exists, drives `FundChosen`) |

### Keyboard model

Each card is a flat list of real `<button>`/`<input>`/`<a>` elements, so
native Tab order holds — no roving tabindex. The `:focus-visible` outline
(`design-language.md` §1) now covers `.bench__fund` / `.bench__sharpen` /
`.bench__refine`, fixing the previously-invisible quiet-control focus.

- expand/collapse: `.bench__sharpen`, Enter/Space (server round-trips a
  `data-state` flip via a signal; no client JS beyond Datastar).
- edit: Tab into `.pk-input`, type, Tab/Enter to `.bench__refine`.
- submit: `.bench__refine`, Enter/Space → emits the ledger fact → SSE
  re-render.
- fund: `.bench__fund`, Enter/Space → existing `FundChosen` → dispatch.
- handoff (flow c, fast-follow): `.bench__review` `<a>`, Enter → `/review?wo=`.

### Mockups (stripped-CSS legible — the text alone carries meaning)

(i) collapsed fund-target (the existing minimal form):

```
on the bench:
┌──────────────────────────────────────────────┐
│ FUNDABLE   pay/charge.go:88  (next)            │
│ [ fund pay/charge.go:88 ]   [ sharpen ▸ ]      │
└──────────────────────────────────────────────┘
```

(ii) expanded "split" card (proposed-then-accept):

```
┌──────────────────────────────────────────────┐
│ SPLIT   pay/charge.go:88                       │
│ this order looks broad. split into:            │
│   • pay/charge.go:88  (refund path)            │
│   • pay/charge.go:120 (partial-capture path)   │
│ [ refine: split into 2 ]   [ fund as-is ] [▾]  │
└──────────────────────────────────────────────┘
```

(iii) expanded "criteria" card:

```
┌──────────────────────────────────────────────┐
│ CRITERIA   pay/charge.go:88                    │
│ acceptance criteria (one per line):            │
│ ┌────────────────────────────────────────┐    │
│ │ rejects a negative amount              ▏│    │
│ │ caps at the daily ceiling               │    │
│ └────────────────────────────────────────┘    │
│ [ refine: attach criteria ]  [ fund ]    [▾]   │
└──────────────────────────────────────────────┘
```

(iv) preflight-self-flag card during dead-air (FAST-FOLLOW — pending, no
fund button):

```
on the bench:
┌──────────────────────────────────────────────┐
│ PREFLIGHT   filling WO#7 → auth/token.go:47    │
│ wo#7's agent self-flagged this line.           │
│ pre-staged — points you here the moment it     │
│ lands. (a routing hint, not a verdict.)        │
│ waiting on wo#7…                               │
└──────────────────────────────────────────────┘
```

on land → preflight-ready:

```
┌──────────────────────────────────────────────┐
│ PREFLIGHT   wo#7 → auth/token.go:47            │
│ landed. self-flagged spot ready to review.     │
│ [ review wo#7 → ]  (→ /review?wo=7)            │
└──────────────────────────────────────────────┘
```

The "(a routing hint, not a verdict.)" line is load-bearing copy: it
discharges §4.3 / §12.10 in the HTML itself, so the guardrail is assertable.

## Primary flows

Format: actor → system → surface change (keyboard).

### (a) Dispatch → dead-air begins → bench SEEDS prep cards — IN SCOPE

1. Lead funds an order (Spend / FundChosen / PlaceOrder); each ends
   `go drainQueuedOrders` and writes a trigger cell → SSE re-render.
2. The background runner sets a live fill (`fillSnapshot` now `id>0`).
3. On the next render while fill is active, `renderBench` receives **seed
   cards** (split/criteria/convention harvested from the live diff) in
   addition to the plain fundable targets.
4. Surface: act-now shows the bench with seeded cards; state/history still
   shows `.order-filling` + transcript (unchanged).
5. Keyboard: Tab from `fund-work` into the first bench card; the `(next)`
   head is the natural first stop.

Critical gate: the seed set is a pure function of `(fundableBacklog,
fillSnapshot, diff-harvest)`. When `fillSnapshot.id==0` (idle), the seed set
is empty → only honest fundable targets render. No timer, no manufactured
cards (§10.4).

### (b) Work a card → emit refined-WO ledger fact → SSE re-projection — IN SCOPE

1. Lead expands a card, edits (the `.pk-input` for criteria/convention; or
   accepts/edits proposed splits), submits `.bench__refine`.
2. System appends ONE `worefine` ledger line (see Event sketch) — append-only,
   no mutation.
3. `fundableBacklog`'s projection folds the fact (split → 2 targets replacing
   1; criteria/convention → annotate the target).
4. System writes a trigger cell (reuse `c.Dispatch`) → single SSE re-render.
5. Surface: the bench re-renders with refined cards (split → two cards;
   criteria → card shows attached criteria, collapses to fundable). No
   reload, no spinner.

### (c) Preflight self-flag: fill → land → pending → ready → handoff — FAST-FOLLOW

Blocked on the self-flag-during-fill event (see Dependency). When unblocked:

1. During fill, a `preflight-pending` card renders per self-flagged spot of
   the filling WO.
2. The diff lands (runner appends the WO's catch/verdict; `fillSnapshot`
   clears or advances).
3. On re-render, the card transitions `preflight-pending → preflight-ready`;
   the waiting line is replaced by `.bench__review` `<a href="/review?wo=<id>">`.
4. Keyboard: Enter on the review link → `/review?wo=<id>` (symmetric
   per-order/session nav, `design-language.md` §3.C; the return breadcrumb
   back to `/?key=<key>` already exists there).

### (d) Fund a collapsed target → dispatch — ALREADY BUILT, keep verbatim

`.bench__fund` → `SetSignal(FundTarget, "path:line")` + `FundChosen` →
`chosenFundable` validates membership → `AppendDispatch` → announce drained
Balance + risen Dispatch over SSE → `drainQueuedOrders`. Off-bench /
over-budget = silent no-op. Unchanged.

### (e) Order completes / queue-zero → honest-empty — IN SCOPE

1. Runner finishes; no order is filling (`fillSnapshot.id==0`).
2. The seeding gate closes; preflight cards drop; if `fundableBacklog` is
   also empty (queue-zero), `renderBench` returns `nil` and the caller omits
   the region.
3. Surface: act-now shows only what remains. The §12.6 win state is an
   honest empty bench — never a "0 items" shout.

## States & transitions

Bench region:

```
absent ── no fundable work AND no active fill ──────────────▶ (region omitted)
present-fundable ── fundableBacklog non-empty, no fill ─────▶ collapsed cards only
seeded-dead-air ── active fill (fillSnapshot id>0) ─────────▶ fundable + seed (+ preflight*)

absent ──dispatch(fill starts)──▶ seeded-dead-air ──fill ends, backlog left──▶ present-fundable
present-fundable ──dispatch──▶ seeded-dead-air ──fill ends, queue-zero──▶ absent
```
(* preflight cards are the fast-follow.)

Card:

```
collapsed-fundable ──sharpen▸──▶ expanded-{split|criteria|convention}
expanded-* ──refine (emit fact)──▶ collapsed-fundable (re-projected) | consumed
expanded-split ──refine──▶ collapsed-fundable ×2 (replaces 1)
collapsed-fundable ──fund (FundChosen)──▶ consumed (dispatched; leaves backlog)

preflight-pending ──diff lands──▶ preflight-ready ──review link──▶ (drill to /review?wo)
preflight-pending ──fill ends w/o land / WO failed──▶ (card dropped)
```
`consumed` = the target left `fundableBacklog` (a funded WorkOrder marks it);
the card stops rendering on the next projection.

## Event / data sketch

### The refined-work-order ledger fact (fields only — conceptual)

Append-only line, a new kind sibling to `kindWorkOrder` / `kindWOStatus` /
`kindWOVerdict`:

```
RefinedOrderRecord {
  Kind     "worefine"     // new kind tag
  RefineID int            // monotonic, like WorkOrderRecord.ID
  Target   Target         // the bench target being sharpened (path:line + revs)
  Refine   string         // "split" | "criteria" | "convention"
  Splits   []Target       // when Refine=="split": resulting targets (>= 2)
  Criteria []string       // when Refine=="criteria": one acceptance line each
  Note     string         // when Refine=="convention": the accepted convention
}
```

`fundableBacklog` folds it on read (same discipline as
`candidatesFromCatches`): split replaces the parent Target with `Splits`;
criteria/convention keep the Target and attach the annotation for the card
body. A refined target is still subject to the existing consumed / own-cycle
/ dedup filters.

### SSE re-render trigger

Reuse the trigger-cell idiom: the refine action writes `c.Dispatch` (backlog
changed) whose `Write` fans out one re-render to the live stream — same as
`c.Balance.Write` / `c.Dispatch.Write` today. No new transport. (A dedicated
`c.Bench` cell is unnecessary unless a refine that does not change the
dispatch count later needs its own trigger.)

### Dependency: self-flag-during-fill event (gates flow c only)

The preflight cards need *which line(s)* the filling WO self-flagged, surfaced
*before* the catch mints. Today `CatchRecord.SelfFlagged` is a **bool**
(`ledger.go:32`) and `stock.go` only counts it — there is no per-spot mid-fill
self-flag stream. That signal must come from the event-translation layer (the
live runner's activity stream — `activitySnapshot` / `activityTranscript`,
`live.go:215-243`). Until a structured "agent self-flagged anchor `path:line`
for WO#id" event exists, flow c cannot ship honestly — do NOT fake it from the
final bool (that violates §12.10 and §10.4). Flows a/b/d/e ship without it.

## Guardrail audit

| Flow | no gauge/meter/bar | calm / no spinner | keyboard-native | server-rendered / assertable | stripped-CSS legible | work-is-source-of-truth | self-flag = signal not spine |
|---|---|---|---|---|---|---|---|
| (a) seed on dead-air **[highest risk]** | cards are text | "waiting…" copy, no spinner; gated on real fill not a clock | native-tabbable | seed set = pure fn of log+fill; `data-*` assertable | each card names kind + target | seeds derive from diff+backlog, never invented | n/a until c |
| (b) refine→fact | yes | re-render on submit only | input + buttons | refined fact replays; re-projection assertable | criteria/convention as plain text | one append-only fact, folded | n/a |
| (c) preflight (FF) | yes | state flip on land, no animation | review link Enter-followable | `href="/review?wo="` + `data-state` assertable | "ready to review" text | driven by the WO landing | "routing hint, not a verdict" copy; no auto-route |
| (d) fund (built) | yes | existing | existing | `data-target` + `FundChosen` | "fund path:line" | `AppendDispatch` atomic | n/a |
| (e) queue-zero empty | region omitted | yes | n/a | absence assertable | empty = nothing | empty because backlog truly empty | n/a |

Dead-air seeding (a) is the one path that could manufacture busywork; the
design ties every seed to real backlog or the live diff and closes the gate
the instant fill ends.

## TDD-ready acceptance assertions (write as failing tests first)

Flow (a) — seed scoped to active fill:

- with an active fill, the act-now `<section>` contains
  `.bench .bench__item[data-kind="split"]` (and/or criteria/convention).
- with NO active fill, the bench contains only `[data-kind="fundable"]`
  items (or is absent) — assert NotContains split/criteria/convention/preflight.
- the bench renders inside `aria-labelledby="act-now-label"`, after
  `.fund-work`.
- each seed card carries `data-target="path:line"` and a `.bench__kind`
  `.pk-section-label`.

Flow (b) — refine emits a fact, re-projection re-renders:

- an expanded criteria card contains a `.pk-input` and a `.bench__refine`
  `.pk-btn` whose label contains "refine".
- after a refine-split, the bench shows 2 cards where there was 1 (both
  `data-target`s present, parent absent).
- the refine action results in exactly one new `Kind=="worefine"` ledger
  line and a single SSE re-render.

Flow (c) — preflight (fast-follow):

- during fill, `.bench__item[data-state="preflight-pending"]` exists, has NO
  `.bench__fund`, body text contains "routing hint, not a verdict" and
  "waiting on wo#<id>".
- after land, the same card is `data-state="preflight-ready"` and contains
  `<a class="bench__review" href="/review?wo=<id>">`.
- the review link round-trips to `/review?wo=<id>` exposing the symmetric
  breadcrumb back to `/?key=<key>` (extends `drill_return_internal_test.go`).

Flow (d) — fund (regression-lock the built path):

- a collapsed fundable card has `.bench__fund[data-target]` firing
  `FundChosen` + `SetSignal(FundTarget)`; FIFO head label contains "(next)".
- off-bench / over-budget fund is a no-op (no Dispatch write, no broadcast).

Flow (e) — honest empty:

- queue-zero with no fill → NotContains `.bench` in the rendered card.
- `role="main"` + `aria-live="polite"` survive on the live card across all
  bench states (NotRegress).

Component / focus (carries `design-language.md` PR1):

- `.bench__item` carries `.pk-card`; `.bench__fund` / `.bench__sharpen` /
  `.bench__refine` are `.pk-btn--quiet` / `.pk-btn`; a `:focus-visible`
  outline rule covers them.

## Residual open questions

1. **Self-flag-during-fill event shape** (gates flow c). The exact event the
   live runner emits for a mid-fill self-flagged anchor is undefined. Resolve
   when the fast-follow is scheduled.
2. **Seed cap interaction.** `benchCap=5` currently bounds fundable chips.
   With seeded + preflight + fundable cards competing during dead-air, the
   suggested priority is preflight-ready > preflight-pending > fundable-(next)
   > seeds > rest — confirm this curation policy at build time.
