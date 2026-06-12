# Round 102 — DESIGN-LANGUAGE POLISH close-out — 2026-06-12

The thread that opened at R101 ships, is audited against canon, and closes
complete-for-V1.

Trigger: the two sequenced PRs scoped at R101 (the token + component system
layer, then the flows layer) are built in the working tree and green; the
scribe convenes the close-out to record what shipped, run the canon
reconciliation, and mark the thread done.

Panelists present: the Web Designer and the UX/UI Specialist (the two R101
lenses), plus the standing canon auditor pass (VISION / DESIGN / RISKS /
clashes).

New evidence on the table: the working-tree diff (`internal/app/style.go`,
`live.go`, `fund_work.go`, `nav.go`, and the surface renderers) plus the new
test files (`card_sections_internal_test.go`, `drill_return_internal_test.go`,
`fund_work_internal_test.go`) and the extended `style_internal_test.go` /
`board_inflight_internal_test.go`. Suite green:
`go test -race -count=1 ./internal/app/ ./internal/surface/` → ok.

## What shipped

PR1 — the system layer (Web-led, UX/UI co-signed):

- Scale + scaffolding tokens added to the inline `packetsStyle`
  (`internal/app/style.go`): `--pk-radius` / `--pk-radius-sm`, `--pk-border`,
  `--pk-font-sm` / `--pk-font-xs`, and a single shared
  `:focus-visible { outline: 2px solid var(--pk-accent); outline-offset }`.
- A tiny component layer reused via multi-class: `.pk-btn` (+ `.pk-btn--quiet`),
  `.pk-input`, `.pk-chip`, `.pk-section-label`, `.pk-card`. Surfaces keep their
  semantic class as a thin hue/state hook (e.g. `pk-card board-row`,
  `pk-btn spend-action`), so stripped-CSS markup and labels are unchanged.
- Honest-state color tokens (`--pk-confirmed` / `--pk-balance` /
  `--pk-inflight` / `--pk-lost` / `--pk-accent`) untouched, byte-for-byte; the
  focus ring reuses the documented `--pk-accent` bronze, adding no new hue.

The two reuse-gap closures found during PR1:

- Gap 1 (CSS DRY): `.review-answer__submit` now takes its border from
  `var(--pk-accent)` instead of a hand-rolled color, folding the last
  off-token button border into the system.
- Gap 2 (a11y): the drill-return crumbs are wrapped in a `nav` landmark on
  `/review` and `/settings` so the return affordance is a proper landmark, not
  a bare link.

PR2 — the flows layer (UX/UI-led, Web co-signed):

- Flow A — the live card is sectioned into an act-now `<section>` (rendered
  first) and a state/history `<section>`, each with a `.pk-section-label`
  heading via `aria-labelledby`. The wrapper keeps `role="main"` +
  `aria-live="polite"` + `aria-label="session economy"`; both sub-sections nest
  inside main, the nav stays a sibling landmark outside main.
- Flow B — funding is unified in `internal/app/fund_work.go`: `renderFundWork`
  renders one labelled "fund work" group (`.pk-section-label`) + a dim
  two-currency explainer ("balance spends a catch; bandwidth places a live
  order.") + the two existing buttons. A labelled affordance PAIR, no
  bar / ratio / fill / meter.
- Flow C — drill-return crumbs (`cardReturnCrumb` / `reviewSessionCrumb` in
  `internal/app/nav.go`) add a return link plus symmetric per-order ↔ session
  links on `/review` and `/settings`; keys are `url.QueryEscape`'d so a
  NATS-token key with query metacharacters round-trips.

## Test evidence

Test-first throughout (red, then green). New behavior is pinned by:

- `card_sections_internal_test.go` (4 tests) — exactly one `role="main"` and
  one `aria-live="polite"` survive (not duplicated); the two
  `<section aria-labelledby>` regions nest inside main; a fresh session omits
  the act-now section (no empty-shouting) while onboarding still renders.
- `fund_work_internal_test.go` (2 tests) — pins the "fund work" label + the
  two-currency explainer text and asserts NotContains
  `progress-bar` / `<progress` / `<meter` / `role=progressbar` / `gauge`.
- `drill_return_internal_test.go` (4 tests) — the return + symmetric crumbs on
  `/review` + `/settings`, with query-metacharacter keys round-tripping.
- Extended `style_internal_test.go` + `board_inflight_internal_test.go` (3
  added test funcs) — the guardrail still asserts NotContains `#ff0000` /
  `#00ff00` / `progress-bar`; the new tokens + components are present; the
  bet-vs-confirmed structural separation is intact.

Suite green under `-race -count=1` for `./internal/app/` and
`./internal/surface/`.

## Canon reconciliation

The auditor ran VISION / DESIGN / RISKS / clashes against the diff. Result: no
violation, no regression, no clash reopened. Four notes recorded (all
defensible and council-sanctioned, none a blocker):

- VISION §13.1 ("a sort, never a score"; "never let both leverage/trust pull
  the eye") + §6 economy — Flow A's act-now-first / state-history-second card
  ordering is the first time the card asserts a top-down "do this now"
  hierarchy. It is NOT a leverage/score ranking (no impact metric, no per-item
  accent, no number drives the order — it is a fixed semantic grouping of
  act-vs-retrospect), so it fabricates no rank. Note, not a blocker. Future
  guard: if a slice ever adds ordering WITHIN act-now, gate it so it never
  becomes an implicit leverage score; keep the section a static semantic
  grouping.
- DESIGN §8 / VISION §8 (calm control-room; honest-state) + design-language.md
  Flow B — verified clean. The highest-risk change (a two-currency "fund"
  group, the classic place a designer reaches for a meter) stayed a labelled
  pair + text explainer; a full-diff scan for `progress|meter|gauge|width:%|
  valuenow` found only the pre-existing calm `.order-filling` text row and the
  Monaco editor `width:100%`. No fix; `fund_work_internal_test.go` pins it.
- council/clashes.md, Clash J — NOT reopened. PR1 introduces NO new served CSS
  asset; all additions live in the existing inline `packetsStyle` string and
  add no new color/hue. The bet-vs-confirmed separation still rests on
  structure + the untouched honest-state hue tokens. R101 is precisely the
  "real stylesheet driver" J anticipated, and it correctly left the
  honest-state hues byte-for-byte and added only scale/component tokens, so J
  stays RESOLVED.
- RISKS.md — "confidently-wrong quiet verdict on the served card" (Clash C /
  R11 elevated binding): NOT violated. The pass is presentation-only; it does
  not touch `surface/present.go`, `pipe.Reason`, or any verdict-to-copy
  mapping. `RenderVerdict` / `RenderLand` only gain an additive `pk-card`
  multi-class; no verdict token semantics change.

Risks audited, all CHECKED CLEAN / IMPROVED — none regressed:

- Live-region (RISKS / constraint 5): `role="main"` + `aria-live="polite"` +
  `aria-label="session economy"` stay on the wrapper that now holds both
  sub-sections; the SSE live announce is unchanged.
- Empty / first-run guard: the act-now section is appended only when
  `len(actNow) > 0`, so a fresh session renders no empty act-now heading;
  onboarding still renders (now `.pk-card`). Honest-silence preserved.
- Honest-state color (constraint 3/6): only scale tokens + a `:focus-visible`
  rule using the existing `--pk-accent` were added; no honest-state hue added,
  changed, or repurposed.
- Keyboard-native north star (VISION §4 / §10.2): IMPROVED — the shared
  `:focus-visible` outline closes the WCAG 2.4.7 gap on the previously-invisible
  quiet/transparent controls (`.compose__analyze`, `.bench__item`,
  `.board-row__retire`).
- Drill-return crumbs (VISION §5 IA philosophy): IMPROVED — return + symmetric
  per-order ↔ session links on `/review` + `/settings`, with QueryEscape'd keys.
- Stripped-CSS legibility (constraint 2): preserved — every component class is
  added via multi-class; markup structure + text labels unchanged.
- No-alarm / no-gauge-meter-progress: the funding group is a labelled
  affordance pair with a text explainer; the residual `meter` strings in
  `live.go` are pre-existing variable names (`BandwidthMeter`) and comments,
  not UI.

Fixes applied during the thread: Gap 1 CSS DRY
(`.review-answer__submit` border → `var(--pk-accent)`); Gap 2 a11y
(drill-return crumbs wrapped in `nav` on `/review` + `/settings`); each
red-first then green. No fixes declined.

## Clashes touched / verdicts

- Clash J — TOUCHED by the audit, verdict UNCHANGED (stays RESOLVED). The R101
  design-system pass is the "real stylesheet driver" J's R36 verdict
  anticipated; it added scale/component tokens and a focus rule, left the
  honest-state hues byte-for-byte, and added no new color. A confirming note
  is appended to J. No other clash moved; none reopened.

## Deferred

- `--pk-surface-3` — never needed; the card sectioning used
  surface/surface-2, so the preemptive elevation token stays unbuilt (R101's
  gate decided it with the section as its only possible consumer).
- Within-act-now ordering — explicitly NOT built; if ever added it must be
  gated against becoming an implicit leverage score (V§13.1).
- Color on the bet-vs-confirmed hooks beyond the honest-state tokens —
  remains a no-cost future addition on the existing class hooks (Clash J).

## Decisions

The DESIGN-LANGUAGE POLISH thread is COMPLETE-FOR-V1. The design language is a
reusable `--pk-*` token + component vocabulary every surface hooks; the WCAG
2.4.7 focus gap is closed on the shared components; the live card is sectioned
with honest landmarks; funding is one honest labelled pair; the drill-return
loops are closed. Canon reconciliation found no violation and no regression.
Back to maintenance / await the next maintainer steer.
