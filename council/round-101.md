# Round 101 — DESIGN-LANGUAGE POLISH thread opens — 2026-06-12

A sleek, cohesive, reusable design system + flow fixes.

Trigger: maintainer steer — "world-class web designer + UX/UI specialist;
make the design language sleek, industry-leading, cohesive, and reusable
across every surface; improve the flows; converge autonomously." The app
accreted surface-by-surface (R95-R98 bolted on authoring + settings), each
re-declaring the visual idioms instead of reusing a vocabulary; time for a
pass over the whole language and the end-to-end journey.

Panelists present: the Web Designer (visual system / token discipline) and
the UX/UI Specialist (flows / accessibility / information architecture). Two
orthogonal lenses, convened to convergence.

New evidence on the table: an audit of the REAL tree at round ~100, not the
openings. `internal/app/style.go` (~428 lines), the new
`authoring.go`/`settings_card.go` surfaces, and `style_internal_test.go`
(the tested stylesheet contract).

## Audit findings (both panelists re-verified against the files)

- The COLOR system is exemplary and is the project's soul: every honest-state
  hue is a `--pk-*` token, per-state selector coverage is pinned by the test
  contract, and color only ever reinforces a state the text names.
- The system STOPS at color + a 4-step spacing rhythm. No radius, border,
  focus, type-scale, or component tokens exist — they are hand-rolled per
  surface as magic numbers and duplicated rule blocks (~1/3 of the file).
- Duplication, counted: the button idiom 5x (`board-create__btn` 84,
  `settings__save` 113, `compose__place` 164, `spend-action` 391, plus the
  dim variant `compose__analyze` 126 / `bench__item` 417 /
  `board-row__retire` 213); the inline-input idiom 2x (`board-create__key`
  75, `settings__token-input` 103); `border-radius: 6px` ~14x; `0.92em`
  ~10-12x; the uppercase-label idiom 3x (234, 414, dispatches).
- A real WCAG 2.4.7 gap: every interactive element is `outline: none` + a
  border-color swap that is near-invisible on the calm palette AND reflows;
  `.compose__analyze` and `.bench__item` have NO `:focus` rule at all, so
  keyboard focus is fully invisible there.
- The flows accreted: the live card is `role="main"` + one flat scroll
  mixing retrospective state, two different spend affordances, the authoring
  editor, the bench, and live-fill, with no grouping; the two funding paths
  (Spend balance vs PlaceOrder bandwidth) read as unrelated with no shared
  explanation; `/review` and `/settings` drill-ins dead-end with no return,
  and per-order vs session review are asymmetric.

## The debate

The Web Designer opened on the system: this is half a design system; promote
the implicit patterns into named tokens + a tiny component layer, do not
redesign. The UX/UI Specialist opened on the flows: section the card with
real landmarks, unify the funding story, close the drill-return loops.

The two lenses turned out orthogonal and to FUSE on one component: the
`.pk-section-label` the Web Designer extracts is exactly the sub-landmark
heading the UX/UI Specialist needs to section the card. So one shared spec,
two sequenced PRs.

Three points were contested and settled:

- `--pk-surface-3`: the Web Designer listed an elevation token upfront; the
  UX/UI Specialist objected it has no consumer today (surface/surface-2 cover
  every layer) and is spec bloat. Resolved: DROP it from the upfront set,
  GATE it on flows-PR step A — add only if the card sectioning needs a third
  layer, with that section as its consumer.
- The focus-ring shape: the Web Designer proposed a box-shadow color-mix
  ring; the UX/UI Specialist (who owns a11y) held for a real
  `:focus-visible { outline: 2px solid var(--pk-accent); outline-offset: 2px }`
  — no reflow, bronze accent is the documented keyboard cue, the canonical
  2.4.7 fix. Resolved: the Web Designer conceded; outline it is, with the
  border-color swap kept as calm reinforcement.
- Whether the focus fix is a separate accessibility round: resolved IN this
  thread — it is free, since it lives on the shared components already being
  extracted, and the invisible-focus controls are a genuine gap, not taste.

## What they converged on

CONVERGED, no conflict remaining. One shared spec, written to
[`design-language.md`](design-language.md), two sequenced PRs:

- PR1 (system layer, Web-led, UX/UI co-signs): promote the scale tokens
  (`--pk-radius` 6px, `--pk-radius-sm` 4px, `--pk-border`, `--pk-focus`, a
  3-step type scale base + `--pk-font-sm` 0.92em + `--pk-font-xs` 0.82em);
  extract `.pk-btn` (+ `.pk-btn--quiet`), `.pk-input`, `.pk-chip`,
  `.pk-section-label` (+ `.pk-card`/`.pk-section` where a pure collapse);
  surfaces keep their semantic class as a thin hue/state hook via multi-class.
  Fix focus once on the components. Honest-state color tokens UNTOUCHED.
- PR2 (flows layer, UX/UI-led, Web co-signs the visual layer): (A) section
  the card into act-now vs state/history regions with
  `<section aria-labelledby>` + `.pk-section-label` headings, keeping
  `role="main"` + `aria-live`; (B) unify funding — co-locate Spend + Place
  under one labelled group + a dim two-currency explainer, a labelled
  affordance PAIR not a meter; (C) close drill-return loops on
  `/review` + `/settings` and make per-order vs session review symmetric.

Both PRs are entirely `vt.NewClient().HTML()`-assertable (token names +
class hooks + a `:focus-visible` selector in PR1; landmarks + aria wiring +
funding label + return hrefs in PR2). Constraints honored: CSS-only
behavior-preserving system layer keeps `style_internal_test.go` green
(constraint 2 + the existing guardrails); honest-state rules and the
no-gauge/no-alarm guardrails untouched (constraint 3); the live region is
not regressed (constraint 5); one shared `--pk-*` palette (constraint 6).

## Decisions / build sequence

The full spec — token values, component contracts, flow improvements, and the
phased test-first plan — lives in [`design-language.md`](design-language.md).

- PR1 first: characterization tests lock the current `vt.HTML()` output, then
  collapse the duplication to tokens + components, then the shared focus fix.
  Green throughout per the house TDD rule.
- PR2 second: the sectioning/funding/drill-return flows build on the new
  vocabulary, each through the TDD skill with a failing vt test first.

## New clashes opened / resolved

Resolved: the design language is no longer half a system — the implicit
radius/border/focus/type/component patterns are promoted to a reusable
vocabulary every surface (including the new authoring + settings surfaces)
hooks, and the WCAG 2.4.7 focus gap is closed on the shared components.
Open: whether `--pk-surface-3` is justified — deferred to flows-PR step A,
decided with the card section as its only possible consumer (no preemptive
token). Whether semantic classes become thin `.pk-btn` wrappers or adopt
`.pk-btn` + modifier directly is a PR1 impl detail; both are vt-assertable
and carry no design disagreement.
