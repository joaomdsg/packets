# The packets Design Language

> The reusable design-language + flows spec the implementers follow. It
> EVOLVES the calm dark control-room system already in
> `internal/app/style.go` (R43): it does not replace it. The honest-state
> color tokens are exemplary and stay byte-for-byte; the gap is everything
> ELSE the file hand-rolls per surface — radius, border, focus, type scale,
> and the button/input/chip/label component idioms duplicated across the
> board, settings, authoring, bench, spend, and review surfaces.

## Why this spec exists

`style.go` is strong on soul and half-built as a system. Tokens exist for
color and a 4-step spacing rhythm, then stop. Audited against the real file:

- The button idiom repeats 5x near-verbatim (`.board-create__btn` line 84,
  `.settings__save` 113, `.compose__place` 164, `.spend-action` 391, plus
  the dim variant `.compose__analyze` 126 / `.bench__item` 417 /
  `.board-row__retire` 213).
- The inline-input idiom repeats 2x verbatim (`.board-create__key` 75,
  `.settings__token-input` 103).
- `border-radius: 6px` appears ~14x as a magic number; the lone drifter is
  `4px` on `.review-answer__submit` (line 344).
- `border: 1px solid var(--pk-line)` appears ~12x.
- `0.92em` appears ~10-12x; `0.82em` is the uppercase-label size (236, 414);
  one-offs `0.85em` (154, 221), `0.88em` (269), `0.9em` (158), `0.95em`
  (300, 386) drift around those two.
- The uppercase letter-spacing label idiom repeats 3x
  (`.board-row__bets-label` 234, `.bench__label` 414, the dispatches label).
- Focus is a WCAG 2.4.7 gap: every interactive element is
  `outline: none` + a border-color swap, which is near-invisible on the calm
  palette AND shifts layout; `.compose__analyze` and `.bench__item` have NO
  `:focus` rule at all, so keyboard focus is fully invisible there.

The fix is promotion, not redesign: name the implicit scale tokens, extract
one small component layer every surface hooks, fix focus once on those
components, then build the new flow sections on the shared vocabulary.

## 1. Token system

All tokens are `--pk-*` custom properties on `:root` so every surface
inherits one palette (constraint 6). The color ramp below is the EXISTING
honest-state palette — UNCHANGED (constraints 3/6). New tokens fill the
scale gaps the file currently hand-rolls.

### Color: surface + ink (unchanged)

- `--pk-bg: #14171a` — the page base.
- `--pk-surface: #1b1f24` — a resting card/input layer.
- `--pk-surface-2: #222831` — a raised control/button layer.
- `--pk-ink: #e6e8eb` — primary text.
- `--pk-ink-dim: #9aa3ad` — secondary/supporting text.
- `--pk-line: #2b323b` — hairline borders.

### Color: honest-state hues (unchanged, the project's soul)

- `--pk-confirmed: #6fb59a` — calm teal-green: a minted catch, a thing that
  happened.
- `--pk-balance: #7fa6c4` — cool blue-gray: a spendable resource, ready.
- `--pk-inflight: #c2a878` — muted amber/tan: a pending bet under
  verification.
- `--pk-lost: #b08a8a` — desaturated mauve: a verified loss/miss,
  acknowledged not shamed.
- `--pk-accent: #d4a574` — warm bronze: the documented focus/keyboard cue.

These are AA-legible against `--pk-bg`/`--pk-surface` for the dim-secondary
use they carry; color is always reinforcement of a state the text names, so
contrast carries meaning, never the only signal.

### Type scale (NEW — collapses the ad-hoc em fractions)

- base: `font-size: 14px` on `body` (unchanged).
- `--pk-font-sm: 0.92em` — the dominant secondary size (breadcrumb,
  activity, dispatch, needs-key, land summary, review meta).
- `--pk-font-xs: 0.82em` — the uppercase section labels.

Audit during impl: collapse `0.85em -> --pk-font-xs`, `0.9em` and `0.95em`
`-> --pk-font-sm`. Keep `0.88em` on `.order-transcript` (269) as a literal
only if the audit shows visible drift; otherwise round to `--pk-font-sm`.
Three steps total (base + sm + xs); no `--pk-font-lg` until a consumer needs
one (no spec bloat).

### Spacing scale (unchanged)

- `--pk-xs: 4px`, `--pk-sm: 8px`, `--pk-md: 14px`, `--pk-lg: 22px`.

### Radius (NEW)

- `--pk-radius: 6px` — the canonical corner (replaces the ~14 literals).
- `--pk-radius-sm: 4px` — the tighter corner (the review-submit drifter).

### Border (NEW)

- `--pk-border: 1px solid var(--pk-line)` — the hairline (replaces the ~12
  literals).

### Surface/elevation layers

- Two layers ship today: `--pk-surface` (resting) and `--pk-surface-2`
  (raised control). A third elevation token is NOT in the upfront set — it
  would be a token without a consumer. It is GATED on flows-PR step A: if the
  card sectioning genuinely needs a third layer to separate act-now from
  state/history, add `--pk-surface-3` THEN with that section as its
  consumer; otherwise skip.

### Focus ring (NEW — the WCAG fix)

- `--pk-focus` documents the canonical focus treatment, applied via
  `:focus-visible` on the shared components:

```css
:focus-visible {
  outline: 2px solid var(--pk-accent);
  outline-offset: 2px;
}
```

A real outline, not a box-shadow ring: `outline-offset` does not reflow
(the current border-color swap shifts layout), and bronze accent is already
the documented keyboard cue (token comment, line 32). The existing
border-color swap stays as calm reinforcement; the outline is the visible
2.4.7 signal.

### Motion

- Motion reports a real state change only, never decoration. The single
  existing transition (`.compose__analyzing` opacity 0.2s) is the idiom;
  keep transitions at `0.2s` ease on opacity/border-color. No spinners, no
  pulsing call-to-action, no progress animation (constraint 3). No motion
  token is promoted until a second consumer exists.

## 2. Component vocabulary

One component layer, built from the tokens above. Each surface KEEPS its
semantic class as a thin color/state hook and adds the component class via
multi-class (`class="pk-btn spend-action"`). The box CSS lives once on the
component; the semantic class carries only what differs (hue, state, layout
nudge). Structure and labels are unchanged, so stripped-CSS legibility holds
(constraint 2). All hooks are server-rendered, so all are
`vt.NewClient(...).HTML()`-assertable (constraint 4).

### `.pk-btn` — the canonical control

Contract: `padding: var(--pk-xs) var(--pk-md)`; `background:
var(--pk-surface-2)`; `color: var(--pk-balance)`; `border: var(--pk-border)`;
`border-radius: var(--pk-radius)`; `font: inherit`; `cursor: pointer`.

States:

- `:hover` — `border-color: var(--pk-balance)` (the existing calm cue).
- `:focus-visible` — the `--pk-focus` outline (NEW, shared).
- `:disabled` — `color: var(--pk-ink-dim)`; `cursor: default`; no hover
  border change (define now so future disabled states are consistent).
- active/pressed — no separate rule; the calm system does not punch.

Maps onto: `.board-create__btn`, `.settings__save`, `.compose__place`,
`.spend-action` (all become `class="pk-btn <semantic>"`; the semantic class
keeps only layout nudges like `.compose__place { align-self: flex-start }`
or margins).

### `.pk-btn--quiet` — the dim variant

Contract: as `.pk-btn` but `color: var(--pk-ink-dim)` and `background:
transparent` (or `--pk-surface-2` where the surface already sat).

States: `:hover` lifts toward `--pk-accent`/`--pk-balance` + `color:
var(--pk-ink)` (the existing analyze/retire idiom); `:focus-visible` gets
the shared outline — this is the fix for the currently-invisible controls.

Maps onto: `.compose__analyze`, `.bench__item`, `.board-row__retire` (the
retire keeps its `:hover` lost-hue override as a semantic reinforcement).

### `.pk-input` — the inline input

Contract: `padding: var(--pk-xs) var(--pk-sm)`; `background:
var(--pk-surface)`; `color: var(--pk-ink)`; `border: var(--pk-border)`;
`border-radius: var(--pk-radius)`; `font: inherit`.

States: `:focus-visible` — the `--pk-focus` outline plus the existing
`border-color: var(--pk-accent)` as reinforcement. Drop the old
`outline: none`.

Maps onto: `.board-create__key`, `.settings__token-input` (the latter keeps
only `min-width: 22ch`).

### `.pk-chip` — the mono chip

Contract: `padding: 1px var(--pk-sm)`; `border: var(--pk-border)`;
`border-radius: var(--pk-radius)`; `font-family: var(--pk-mono)`;
`font-size: var(--pk-font-sm)`; `color: var(--pk-ink-dim)`.

Maps onto: the bench/dispatch mono chips. A fundable chip composes
`.pk-chip.pk-btn--quiet` (interactive); a read-only dispatch line is plain
`.pk-chip`. Keeps per-outcome hues
(`[data-outcome="caught"|"missed"]`) as semantic reinforcement.

### `.pk-section-label` — the uppercase label / sub-landmark heading

Contract: `color: var(--pk-ink-dim)`; `font-size: var(--pk-font-xs)`;
`text-transform: uppercase`; `letter-spacing: 0.04em`.

Maps onto: `.board-row__bets-label`, `.board-row__dispatches-label`,
`.bench__label`. This is ALSO the heading used by the flows-PR card
sectioning (the two threads fuse on this one component).

### `.pk-card` / `.pk-section` (consolidation of the box idiom)

The card/box idiom is the single most-repeated block: `.board-row`,
`.stock-row`/`.balance-row`/etc. (the line-275 group), `.analysis`,
`.order-transcript`, `.review-thread`, `.compose__editor`,
`.review-editor`, `.order-diff-editor` all share `background:
var(--pk-surface)` + `border: var(--pk-border)` + `border-radius:
var(--pk-radius)`. Promote `.pk-card` (padding `var(--pk-sm) var(--pk-md)` +
the box) for the padded rows, and a borderless `.pk-section` wrapper for the
flows-PR landmark regions. Surfaces keep their semantic class for hue/state
and layout (`display: flex` directions, the left-accent rules). Do this in
PR1 only where it is a pure collapse; leave the editor mounts as-is (their
sizing is load-bearing).

## 3. Flow improvements

Sequenced AFTER the system layer so new sections reuse the vocabulary.
All are server-rendered and `vt.HTML()`-assertable; none add a gauge, meter,
or progress bar; none touch the honest-state rules.

### A. Section the session card

Today the live card is `role="main"` + a flat scroll mixing retrospective
state (stock, dispatches, beats, land verdict), two different spend
affordances, the authoring editor, the bench, and live-fill — with no
grouping. Add semantic sub-landmarks:

- an ACT-NOW region: spend, bench, authoring, place-order.
- a STATE/HISTORY region: stock, dispatches, beats, land verdict.

Each region is a `<section aria-labelledby="...">` whose heading is a
`.pk-section-label` (PR1's component). Keep `role="main"` + `aria-live`
on the live card (R61, constraint 5) — the sections nest INSIDE main; do not
move the live region. Decide `--pk-surface-3` here per the gate above.

Testable: assert the `<section>` landmarks, the `aria-labelledby` wiring to
the label ids, and that the act-now controls render inside the act-now
section.

### B. Unify the funding story

Spend (balance hue) and PlaceOrder (accent/bandwidth hue) both fund work but
read as unrelated and gate on different currencies with no shared
explanation — the most confusing moment in the loop. Co-locate them in the
act-now section under ONE labelled group (`.pk-section-label` "fund work")
plus a dim one-line two-currency explainer (e.g. balance spends a catch;
bandwidth places a live order). A labelled affordance PAIR, never a
meter/gauge (constraint 3) — no bar, no ratio, no fill.

Testable: assert the funding group's label, the explainer text, and that
both affordances render under it.

### C. Close drill-return loops

`/review` and `/settings` are drill-ins with no return to the originating
card; per-order `/review?wo=` (diff + threads) and session `/review`
(threads + island) are asymmetric. Add:

- a back-affordance on `/review` and `/settings` returning to the
  originating session card (an `href` back to `/?key=<key>`), reusing the
  breadcrumb idiom.
- symmetric nav between per-order and session review so the Lead can move
  between the two without dead-ending.

Testable: assert the return `href` targets on `/review` and `/settings`,
and that per-order/session review expose symmetric breadcrumb links.

### Empty / first-run, live activity (unchanged, guarded)

The onboarding affordance (`data-state="empty"`) and the live activity row
stay as-is — they already follow the calm idiom. The sectioning in A must
not regress them: a fresh session still renders the onboarding guide, and
the act-now section is omitted (not empty-shouting) when there is nothing to
act on.

## 4. Phased, test-first implementation plan

Two sequenced PRs, one shared spec. Green throughout per the house TDD rule.

### PR1 — system layer (pure refactor, behavior-preserving)

This changes CSS only; server markup is unchanged, so the current
`vt.HTML()` output is preserved. Sequence per the TDD skill:

1. Characterization first: extend `style_internal_test.go` to lock the
   CURRENT contract before refactoring — the per-state selector coverage
   (already pinned) plus the class hooks that must survive. Keep it green.
2. T1 — add the scale tokens to `:root` (`--pk-radius`, `--pk-radius-sm`,
   `--pk-border`, `--pk-font-sm`, `--pk-font-xs`, and the `:focus-visible`
   rule documenting `--pk-focus`). Replace the literals with `var(...)`.
   Audit the one-off em fractions and collapse per section 1. Honest-state
   color tokens UNTOUCHED.
3. T2 — extract `.pk-btn` (+ `.pk-btn--quiet`), `.pk-input`, `.pk-chip`,
   `.pk-section-label`, and `.pk-card`/`.pk-section` where it is a pure
   collapse. Add the component class to each surface via multi-class in the
   `h.*` builders (board.go, settings_card.go, authoring.go, supply.go,
   review_surface.go); the semantic class keeps only hue/state/layout.
4. T3 — the `:focus-visible` outline now applies to every shared component,
   fixing the invisible quiet/transparent controls. Keep the border-color
   swap as reinforcement.

Stays vt-testable: pin the NEW token names (`--pk-radius`, `--pk-focus`,
etc.), the shared class hooks (`.pk-btn`, `.pk-input`, `.pk-chip`,
`.pk-section-label`), and a `:focus-visible` selector assertion in
`style_internal_test.go`. Constraint-compliant: CSS-only, structure/labels
unchanged (constraint 2), honest-state rules and guardrails (NotContains
progress-bar / alarm red/green) intact.

### PR2 — flows layer (new behavior, new tests)

Builds on PR1's vocabulary; each item is a behavior change, so each goes
through the TDD skill with a failing vt test first.

1. Flow A — section the card. New vt tests assert the `<section>`
   landmarks + `aria-labelledby` wiring; a NotRegress test confirms
   `role="main"` + `aria-live` survive on the live card. Decide
   `--pk-surface-3` here (add with the section as consumer, or skip).
2. Flow B — unify funding. New vt tests assert the funding-group label, the
   explainer line, and both affordances under it; a guardrail test confirms
   no meter/gauge markup was introduced.
3. Flow C — close drill-return. New vt tests assert the return `href`
   targets on `/review` and `/settings` and the symmetric per-order/session
   review nav.

Stays vt-testable: every assertion is over server-rendered HTML
(landmarks, aria wiring, label text, hrefs) — not browser-only behavior, so
not test-theater (constraint 4). None touch honest-state rules; the funding
group is a labelled pair, never a gauge (constraint 3).
