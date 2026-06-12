# packets — Visual Language Proposal

A refinement of the calm "control-room" visual language: a coherent
`--pk-*` token system, a reusable component vocabulary, and paste-able
CSS for `internal/app/style.go`. Nothing here betrays the sacred ethos —
calm, dark, restrained; honest state over decoration; one row never
speaking for another; readable with the stylesheet stripped; accessible
by default. This is a polish pass, not a redesign.

This document is a proposal only. No source files are edited.

## 1. Audit

What follows is an honest read of the current `style.go` and the markup
it dresses.

### What is strong

- A genuine token discipline already exists. `--pk-*` custom properties
  for surface, ink, the honest-state hues, and a spacing rhythm mean the
  palette is centralized and inheritable. This is the right foundation.
- The honest-state palette is well-chosen and principled. The muted
  teal-green (confirmed), cool blue-gray (balance), amber-tan (in-flight),
  desaturated mauve (lost), and warm bronze (accent) read as calm
  reinforcement, never alarm. The per-state `data-state` hooks are applied
  consistently across verdict, land, readiness, and dispatch outcomes.
- Structure carries meaning. Bets are sealed in their own labelled
  cluster; dispatches in another; counts live in `data-*` markers. Strip
  the CSS and the page still reads — the ethos is real, not aspirational.
- Color is used as reinforcement of a state the text already names, with
  comments documenting the intent at every site. That intent should be
  preserved verbatim.
- Accessibility scaffolding is present: `role=main`, `aria-live=polite`,
  `aria-label` landmarks, `font-variant-numeric: tabular-nums` on the
  balance, dotted/wavy underlines instead of red squiggles.

### What is inconsistent or unpolished

1. No type scale. There is one base `font-size: 14px` and a scattering of
   relative sizes — `0.82em`, `0.85em`, `0.88em`, `0.9em`, `0.92em`,
   `0.95em` — chosen ad hoc per component. Six near-identical "slightly
   smaller" sizes that no human can distinguish and no token governs. There
   is no heading scale at all; "headlines" are just `font-weight: 600`.
2. Spacing has only four steps (`4 / 8 / 14 / 22`) and the jump from `14`
   to `22` is large with nothing between. `padding: var(--pk-sm)
   var(--pk-md)` is the de-facto card padding but it is retyped at ~10
   sites rather than tokenized.
3. Radius is a hardcoded `6px` everywhere except the answer submit (`4px`)
   and the survivor glyph (`3px`). Three values, no token, one of them
   (the `4px`) is almost certainly an oversight.
4. Borders are all `1px solid var(--pk-line)`, but the "accent edge" motif
   (the left-border that marks a thread, a bet cluster, onboarding, the
   bench) is `2px solid` and retyped at 6 sites with no token. This is a
   real, recurring component idea with no name.
5. Buttons are duplicated. `.board-create__btn`, `.settings__save`,
   `.compose__place`, `.spend-action` are byte-for-byte the same
   "secondary button in the balance hue" — four copies of one component.
   `.compose__analyze` and `.bench__item` are a second (ghost) variant,
   also duplicated. There is no button component; there are eight
   bespoke buttons.
6. Inputs are likewise triplicated (`.board-create__key`,
   `.settings__token-input`) — same surface, border, radius, focus rule.
7. Focus is inconsistent and a little unsafe. Every input does `outline:
   none; border-color: var(--pk-accent)` — a 1px hue shift is a weak focus
   indicator and removing the outline without a strong replacement is an
   a11y risk. Buttons have no focus-visible style at all; they rely on the
   UA default, which is invisible on this dark surface for keyboard users.
8. No elevation language. Two surface shades (`--pk-surface`,
   `--pk-surface-2`) do double duty as both "card" and "hover/raised,"
   so hover and elevation are the same signal. No shadow token exists, so
   nothing can lift off the page when it should (the editor frame, a future
   menu).
9. Motion is almost entirely absent and what exists is one-off: a single
   `transition: opacity 0.2s` on the analyzing indicator, `0.2s` hardcoded.
   Hover state changes are instant (a small jank on the board rows). SSE
   re-renders pop in. There is no motion token and no easing.
10. The label motif (uppercase, `0.82em`, `letter-spacing: 0.04em`,
    ink-dim) is a real reusable "eyebrow/label" primitive, retyped in two
    places (`.board-row__bets-label`, `.bench__label`) with no shared
    class.
11. The Monaco editors (`.compose__editor`, `.review-editor`,
    `.order-diff-editor`, `.review-answer__editor`) all share the exact
    same frame (`1px solid line`, `6px` radius) but differ only in height —
    a single `.pk-editor` frame with a height modifier is begging to exist.
12. Density is good on the card but the fleet board rows can get visually
    busy: 8+ inline spans wrapping, with the only separation being color.
    A little more rhythm (consistent inline gap, a hairline between wrapped
    clusters) would help without adding chrome.

None of these are wrong, exactly — they are the natural sediment of a
prototype grown surface-by-surface. The fix is consolidation: name the
primitives that already exist by repetition, and give the type/space/
radius/motion axes real scales.

## 2. Refined, expanded token system

Dark and calm are preserved. The honest-state hues keep their meaning;
they are only nudged for a tighter, more deliberate relationship and a
couple of state variants (a dim "track" tint, hover/active shades) are
added. Everything below is a drop-in replacement for the current `:root`
block plus a `body` refinement.

### Color — surface and ink (refined)

The surface ramp gains one step so we have a true elevation ladder
(page → sunken → card → raised → overlay) rather than two shades doing
four jobs.

```
--pk-bg:        #121518;   /* page — slightly deeper than before for contrast */
--pk-sunken:    #0e1114;   /* wells: transcript, code, inset areas */
--pk-surface:   #1a1e23;   /* the card / row resting surface */
--pk-surface-2: #21262d;   /* raised: hover, the secondary-button face */
--pk-surface-3: #2a3038;   /* overlay: menus, popovers (future) */
--pk-line:      #2b323b;   /* hairline border (unchanged) */
--pk-line-soft: #232930;   /* a quieter divider for within-card splits */
```

Ink keeps three weights and gains a faint one for the lowest-emphasis
labels:

```
--pk-ink:       #e6e8eb;   /* primary text (unchanged) */
--pk-ink-dim:   #9aa3ad;   /* secondary (unchanged) */
--pk-ink-faint: #6b7480;   /* eyebrow labels, disabled, watermark */
```

### Color — honest-state hues (refined, with variants)

The five hues are kept and very lightly re-tuned for a consistent
~55–60% saturation feel and even perceived lightness, so no single state
shouts. Each gets a `-dim` track tint (a ~10–14% mix toward the surface)
for subtle backgrounds — used sparingly, e.g. a survivor line — and the
accent gets hover/active shades for interactive controls.

```
/* confirmed — a minted catch; a thing that happened (calm teal-green) */
--pk-confirmed:      #6fb59a;
--pk-confirmed-dim:  color-mix(in srgb, var(--pk-confirmed) 14%, transparent);

/* balance — a spendable, ready resource (cool blue-gray) */
--pk-balance:        #82a8c6;
--pk-balance-dim:    color-mix(in srgb, var(--pk-balance) 14%, transparent);

/* in-flight — a pending bet under verification (muted amber-tan) */
--pk-inflight:       #c6ab7c;
--pk-inflight-dim:   color-mix(in srgb, var(--pk-inflight) 14%, transparent);

/* lost — a verified loss/miss; acknowledged, not shamed (desaturated mauve) */
--pk-lost:           #b58e8e;
--pk-lost-dim:       color-mix(in srgb, var(--pk-lost) 14%, transparent);

/* accent — focus / keyboard cue / the "attention" edge (warm bronze) */
--pk-accent:         #d4a574;
--pk-accent-hover:   #e0b487;
--pk-accent-dim:     color-mix(in srgb, var(--pk-accent) 14%, transparent);
```

Rationale for the nudges: balance moves from `#7fa6c4` to `#82a8c6` and
in-flight from `#c2a878` to `#c6ab7c` — both a hair brighter so they hold
up against the slightly deeper `--pk-bg` without crossing into
saturation. Mauve `lost` is nudged up one step likewise. These are
sub-perceptual on their own; together they keep the five states evenly
weighted. The `-dim` mixes replace the one hardcoded
`color-mix(... 12% ...)` already in the survivor line, generalizing it.

### Type scale

One base, a real modular scale (~1.15 ratio, snapped to sane px), three
weights, and matched line-heights. `--pk-text-base` stays 14px so nothing
reflows.

```
--pk-font: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
--pk-mono: ui-monospace, "SF Mono", Menlo, Monaco, "Cascadia Code", monospace;

/* sizes */
--pk-text-eyebrow: 11px;   /* uppercase labels (was 0.82em) */
--pk-text-xs:      12px;   /* meta, indicators (was 0.85/0.88em) */
--pk-text-sm:      13px;   /* secondary copy (was 0.9/0.92em) */
--pk-text-base:    14px;   /* body */
--pk-text-md:      16px;   /* row headlines, card titles */
--pk-text-lg:      19px;   /* surface lead / page heading */
--pk-text-xl:      24px;   /* the rare hero number, if ever */

/* weights */
--pk-weight-normal:   400;
--pk-weight-medium:   500;   /* headlines, the "lead" line */
--pk-weight-semibold: 600;   /* amounts, counts that anchor a row */
--pk-weight-bold:     700;   /* the home wordmark, the row key only */

/* line-heights */
--pk-leading-tight:  1.25;   /* headlines, single-line numbers */
--pk-leading-normal: 1.5;    /* body copy */

/* tracking */
--pk-tracking-label: 0.04em; /* the uppercase eyebrow */
```

This collapses the six ad-hoc `em` sizes into four named steps
(eyebrow / xs / sm / base) and introduces md/lg for the headlines that
are currently just bolded body text — giving real hierarchy without
loudness.

### Spacing scale

A 4px base, six steps, filling the gap between the old `md` (14) and `lg`
(22). Old names are kept as aliases so existing rules need no rewrite.

```
--pk-space-1: 4px;
--pk-space-2: 8px;
--pk-space-3: 12px;
--pk-space-4: 16px;
--pk-space-5: 24px;
--pk-space-6: 32px;

/* back-compat aliases (existing rules keep working unchanged) */
--pk-xs: var(--pk-space-1);
--pk-sm: var(--pk-space-2);
--pk-md: var(--pk-space-4);   /* 14 → 16: a touch more breathing room */
--pk-lg: var(--pk-space-5);   /* 22 → 24: snapped to the scale */

/* the canonical card inset, finally named */
--pk-pad-card: var(--pk-space-3) var(--pk-space-4);   /* 12px 16px */
```

### Radius scale

```
--pk-radius-sm:  4px;    /* chips, the survivor glyph context, small controls */
--pk-radius-md:  7px;    /* the card / row / button default (was 6) */
--pk-radius-lg:  10px;   /* the editor frame, larger panels */
--pk-radius-pill: 999px; /* status chips / pills */
```

Moving the default from 6 to 7 is a one-pixel softening that reads more
intentional at this surface darkness; everything that said `6px` becomes
`var(--pk-radius-md)`.

### Border

```
--pk-border:       1px solid var(--pk-line);
--pk-border-soft:  1px solid var(--pk-line-soft);
--pk-edge-width:   2px;                       /* the accent left-edge motif */
--pk-edge-accent:  var(--pk-edge-width) solid var(--pk-accent);
--pk-edge-line:    var(--pk-edge-width) solid var(--pk-line);
```

The `--pk-edge-*` tokens name the recurring left-border motif (threads,
bet clusters, onboarding, bench) so it stops being retyped.

### Elevation / shadow

Calm and dark means shadows must be subtle — diffuse and low-opacity,
never a drop-shadow card-on-white look. Two levels plus a focus ring.

```
--pk-shadow-1: 0 1px 2px rgba(0,0,0,0.30);                       /* resting card lift */
--pk-shadow-2: 0 4px 16px rgba(0,0,0,0.40);                      /* raised: editor, overlay */
--pk-ring:     0 0 0 2px var(--pk-bg), 0 0 0 4px var(--pk-accent); /* focus ring */
```

The focus ring uses a 2px gap of the page color then a 2px accent halo —
a strong, calm, theme-consistent keyboard indicator that does not rely on
the UA outline and reads on every surface shade.

### Motion

Subtle, fast, consistent easing. Calm means short and smooth, never
bouncy.

```
--pk-ease:        cubic-bezier(0.2, 0, 0, 1);   /* standard decelerate */
--pk-dur-fast:    120ms;   /* hover, focus, color shifts */
--pk-dur-normal:  200ms;   /* the analyzing indicator, reveals */
--pk-dur-slow:    320ms;   /* SSE re-render settle */
```

## 3. Reusable component language

Naming follows the existing BEM-ish `.block__elem--modifier` house style.
The recurring primitives get a `pk-` prefixed canonical class so they are
clearly the shared vocabulary, while existing surface-specific hooks
(`.board-row`, `.compose`, etc.) remain as-is and either compose with or
are restyled to match the primitive. The migration is additive: introduce
the `pk-*` primitives, then have the bespoke classes either adopt them or
be replaced at call sites in a later slice. No markup must change to land
the token block; the component CSS below can be added incrementally.

### Surface / card — `.pk-surface`

The resting container: `--pk-surface` fill, hairline border,
`--pk-radius-md`, `--pk-pad-card`, `--pk-shadow-1`. This is the canonical
form behind `.board-row`, `.stock-row`, `.balance-row`, `.review-thread`,
`.analysis`, `.onboarding`, etc. Modifier `.pk-surface--sunken` for wells
(the transcript), `.pk-surface--raised` for the editor frame.

### Row — `.pk-row`

A `.pk-surface` laid out as a horizontal, baseline-aligned, wrapping flex
line with `--pk-space-2 --pk-space-4` gaps. This is exactly today's
`.board-row`. Hover raises to `--pk-surface-2` over `--pk-dur-fast`.

### Eyebrow label — `.pk-label`

The uppercase micro-label: `--pk-text-eyebrow`, `--pk-ink-faint`,
`text-transform: uppercase`, `letter-spacing: --pk-tracking-label`.
Replaces the duplicated `*-label` rules.

### Button — `.pk-btn` with three variants

One base + three variants, replacing all eight bespoke buttons.

- `.pk-btn` (base): `--pk-pad-control` inset, `--pk-radius-md`, `font:
  inherit`, `cursor: pointer`, a `--pk-dur-fast` transition on border/
  background/color, and a proper `:focus-visible { box-shadow: --pk-ring }`.
- `.pk-btn--primary`: the deliberate economic move (Spend, Place order,
  Save key, Create session). `--pk-surface-2` face, `--pk-balance` text,
  hairline border that warms to `--pk-balance` on hover. This is today's
  "balance-hue secondary button" — promoted to the primary action because
  in packets the calm balance-hue button *is* the strongest call to
  action the ethos allows (no pulsing CTAs).
- `.pk-btn--secondary`: `--pk-surface` face, `--pk-ink` text, hairline that
  warms to `--pk-accent` on hover. For non-economic confirms.
- `.pk-btn--ghost`: transparent face, `--pk-ink-dim` text, hairline; warms
  to `--pk-ink` / `--pk-accent` on hover. This is `.compose__analyze`,
  `.bench__item`, `.board-row__retire`. A `--danger` flavor of ghost
  (`.pk-btn--ghost.pk-btn--danger`) warms to `--pk-lost` — the retire
  control's honest "this removes something" cue, never an alarm-red button.

### Input / textarea — `.pk-input`

`--pk-surface` face, hairline, `--pk-radius-md`, `font: inherit`,
`--pk-pad-control`. Replaces `.board-create__key`, `.settings__token-input`.
Focus uses the ring token plus a border warm to `--pk-accent` (the ring is
the primary indicator; the border shift is reinforcement, fixing the
current outline:none risk).

### Badge / chip — `.pk-chip`

The mono inline marker used for bench items and dispatch fragments:
`--pk-text-sm`, `--pk-mono`, `1px var(--pk-space-2)` inset,
`--pk-radius-pill` (or `--sm` for square), hairline. A `.pk-chip--link`
flavor carries the dotted-accent underline of `.board-row__questions`.

### Meters — balance / bandwidth — `.pk-meter`

Critically NOT a gauge. A meter here is a labelled held quantity: a
`.pk-surface` row with a `.pk-meter__label` (`--pk-ink-dim`) and a
`.pk-meter__value` in its state hue, `--pk-text-md`,
`--pk-weight-semibold`, `font-variant-numeric: tabular-nums`. Balance uses
`--pk-balance`; bandwidth uses `--pk-accent`; confirmed stock uses
`--pk-confirmed`. No bar, no fill, no percentage — the value is the meter.

### Status chip — `.pk-status`

The per-`data-state` colored token used by verdict, land, readiness,
dispatch outcome. It does not introduce new color; it formalizes the
`[data-state="..."] { color: ... }` pattern as a single attribute-driven
component so verdict / land / readiness / outcome share one rule table.
A `.pk-status` reads its hue from a `--pk-state` custom property set per
state, e.g. `&[data-state="catch"] { --pk-state: var(--pk-confirmed) }`.

### Panel — `.pk-panel`

A `.pk-surface` carrying the accent left-edge (`--pk-edge-accent`) for the
"needs your attention / read this" clusters: review threads, the gated
review-questions badge, onboarding, the in-flight bet cluster. The neutral
variant `.pk-panel--line` uses `--pk-edge-line` (the bench, the bets/
dispatches clusters that are informational, not attention-seeking).

### Nav — `.pk-nav`

Today's `.board-nav` formalized: a baseline flex header, hairline bottom
border, the wordmark in `--pk-weight-bold`, crumbs in `--pk-ink-dim`
warming to `--pk-accent` on hover. Gains `--pk-dur-fast` color transitions
and the focus ring on its links.

### Empty state — `.pk-empty`

The calm "nothing here yet" line (today's `.review__empty`): centered-left
`--pk-ink-dim` text in `--pk-pad-card`, no border, no icon — silence is
honest. Reused for empty review, empty board (future), drained backlog.

### Monaco editor frame — `.pk-editor`

The shared editor frame: `--pk-radius-lg`, `--pk-border`, `--pk-shadow-2`
(it is a raised work surface), `overflow: hidden` so Monaco's corners
clip to the radius. Height is a modifier, not baked in:
`.pk-editor--compose { height: 180px }`, `--review { height: 60vh }`,
`--diff { height: 45vh }`, `--answer { height: 14em }`. Replaces the four
duplicate editor rules. Monaco stays `theme: 'vs-dark'`, which sits
naturally inside this frame.

## 4. Concrete CSS for `style.go`

Two paste-able blocks. The first replaces the current `:root` + `body`.
The second adds the reusable component layer; it can be appended to the
existing component CSS and adopted incrementally (the existing bespoke
rules keep working until call sites migrate).

### Token block (replaces `:root` and refines `body`)

```css
:root {
  /* ---- surface + ink ---- */
  --pk-bg: #121518;
  --pk-sunken: #0e1114;
  --pk-surface: #1a1e23;
  --pk-surface-2: #21262d;
  --pk-surface-3: #2a3038;
  --pk-line: #2b323b;
  --pk-line-soft: #232930;
  --pk-ink: #e6e8eb;
  --pk-ink-dim: #9aa3ad;
  --pk-ink-faint: #6b7480;

  /* ---- honest-state hues (calm reinforcement, never alarm) ---- */
  --pk-confirmed: #6fb59a;
  --pk-confirmed-dim: color-mix(in srgb, var(--pk-confirmed) 14%, transparent);
  --pk-balance: #82a8c6;
  --pk-balance-dim: color-mix(in srgb, var(--pk-balance) 14%, transparent);
  --pk-inflight: #c6ab7c;
  --pk-inflight-dim: color-mix(in srgb, var(--pk-inflight) 14%, transparent);
  --pk-lost: #b58e8e;
  --pk-lost-dim: color-mix(in srgb, var(--pk-lost) 14%, transparent);
  --pk-accent: #d4a574;
  --pk-accent-hover: #e0b487;
  --pk-accent-dim: color-mix(in srgb, var(--pk-accent) 14%, transparent);

  /* ---- type ---- */
  --pk-font: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
  --pk-mono: ui-monospace, "SF Mono", Menlo, Monaco, "Cascadia Code", monospace;
  --pk-text-eyebrow: 11px;
  --pk-text-xs: 12px;
  --pk-text-sm: 13px;
  --pk-text-base: 14px;
  --pk-text-md: 16px;
  --pk-text-lg: 19px;
  --pk-text-xl: 24px;
  --pk-weight-normal: 400;
  --pk-weight-medium: 500;
  --pk-weight-semibold: 600;
  --pk-weight-bold: 700;
  --pk-leading-tight: 1.25;
  --pk-leading-normal: 1.5;
  --pk-tracking-label: 0.04em;

  /* ---- spacing ---- */
  --pk-space-1: 4px;
  --pk-space-2: 8px;
  --pk-space-3: 12px;
  --pk-space-4: 16px;
  --pk-space-5: 24px;
  --pk-space-6: 32px;
  --pk-xs: var(--pk-space-1);
  --pk-sm: var(--pk-space-2);
  --pk-md: var(--pk-space-4);
  --pk-lg: var(--pk-space-5);
  --pk-pad-card: var(--pk-space-3) var(--pk-space-4);
  --pk-pad-control: var(--pk-space-1) var(--pk-space-4);

  /* ---- radius ---- */
  --pk-radius-sm: 4px;
  --pk-radius-md: 7px;
  --pk-radius-lg: 10px;
  --pk-radius-pill: 999px;

  /* ---- border / edge ---- */
  --pk-border: 1px solid var(--pk-line);
  --pk-border-soft: 1px solid var(--pk-line-soft);
  --pk-edge-width: 2px;
  --pk-edge-accent: var(--pk-edge-width) solid var(--pk-accent);
  --pk-edge-line: var(--pk-edge-width) solid var(--pk-line);

  /* ---- elevation ---- */
  --pk-shadow-1: 0 1px 2px rgba(0,0,0,0.30);
  --pk-shadow-2: 0 4px 16px rgba(0,0,0,0.40);
  --pk-ring: 0 0 0 2px var(--pk-bg), 0 0 0 4px var(--pk-accent);

  /* ---- motion ---- */
  --pk-ease: cubic-bezier(0.2, 0, 0, 1);
  --pk-dur-fast: 120ms;
  --pk-dur-normal: 200ms;
  --pk-dur-slow: 320ms;
}

body {
  margin: 0;
  padding: var(--pk-space-5);
  background: var(--pk-bg);
  color: var(--pk-ink);
  font-family: var(--pk-font);
  font-size: var(--pk-text-base);
  line-height: var(--pk-leading-normal);
  -webkit-font-smoothing: antialiased;
}

/* numbers everywhere line up under their own labels */
.pk-num, [class$="__amount"], [class$="__count"], [class*="balance"],
[class*="hitrate"] { font-variant-numeric: tabular-nums; }

/* one global, theme-consistent keyboard indicator */
:where(a, button, input, textarea, [tabindex]):focus-visible {
  outline: none;
  box-shadow: var(--pk-ring);
  border-radius: var(--pk-radius-sm);
}

@media (prefers-reduced-motion: reduce) {
  * { transition-duration: 0.01ms !important; animation-duration: 0.01ms !important; }
}
```

### Reusable component layer (append; adopt incrementally)

```css
/* ===== surface / card ===== */
.pk-surface {
  padding: var(--pk-pad-card);
  background: var(--pk-surface);
  border: var(--pk-border);
  border-radius: var(--pk-radius-md);
  box-shadow: var(--pk-shadow-1);
}
.pk-surface--sunken { background: var(--pk-sunken); box-shadow: none; }
.pk-surface--raised { background: var(--pk-surface-2); box-shadow: var(--pk-shadow-2); }

/* ===== row ===== */
.pk-row {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--pk-space-2) var(--pk-space-4);
  padding: var(--pk-pad-card);
  background: var(--pk-surface);
  border: var(--pk-border);
  border-radius: var(--pk-radius-md);
  transition: background var(--pk-dur-fast) var(--pk-ease);
}
.pk-row:hover { background: var(--pk-surface-2); }

/* ===== eyebrow label ===== */
.pk-label {
  color: var(--pk-ink-faint);
  font-size: var(--pk-text-eyebrow);
  text-transform: uppercase;
  letter-spacing: var(--pk-tracking-label);
}

/* ===== button ===== */
.pk-btn {
  padding: var(--pk-pad-control);
  border: var(--pk-border);
  border-radius: var(--pk-radius-md);
  font: inherit;
  cursor: pointer;
  background: var(--pk-surface);
  color: var(--pk-ink);
  transition: background var(--pk-dur-fast) var(--pk-ease),
              border-color var(--pk-dur-fast) var(--pk-ease),
              color var(--pk-dur-fast) var(--pk-ease);
}
.pk-btn--primary   { background: var(--pk-surface-2); color: var(--pk-balance); }
.pk-btn--primary:hover   { border-color: var(--pk-balance); }
.pk-btn--secondary:hover { border-color: var(--pk-accent); }
.pk-btn--ghost {
  background: transparent;
  color: var(--pk-ink-dim);
}
.pk-btn--ghost:hover { color: var(--pk-ink); border-color: var(--pk-accent); }
.pk-btn--ghost.pk-btn--danger:hover { color: var(--pk-lost); border-color: var(--pk-lost); }
.pk-btn:disabled { opacity: 0.5; cursor: default; }

/* ===== input / textarea ===== */
.pk-input {
  padding: var(--pk-pad-control);
  background: var(--pk-surface);
  color: var(--pk-ink);
  border: var(--pk-border);
  border-radius: var(--pk-radius-md);
  font: inherit;
  transition: border-color var(--pk-dur-fast) var(--pk-ease);
}
.pk-input:focus { border-color: var(--pk-accent); } /* + ring via :focus-visible */

/* ===== chip ===== */
.pk-chip {
  padding: 1px var(--pk-space-2);
  border: var(--pk-border);
  border-radius: var(--pk-radius-pill);
  font-family: var(--pk-mono);
  font-size: var(--pk-text-sm);
  color: var(--pk-ink-dim);
}
.pk-chip--link {
  border: 0;
  border-bottom: 1px dotted var(--pk-accent);
  border-radius: 0;
  text-decoration: none;
  transition: color var(--pk-dur-fast) var(--pk-ease);
}
.pk-chip--link:hover { color: var(--pk-ink); }

/* ===== meter (a held quantity, NOT a gauge) ===== */
.pk-meter {
  display: flex;
  align-items: baseline;
  gap: var(--pk-space-2);
  padding: var(--pk-pad-card);
  background: var(--pk-surface);
  border: var(--pk-border);
  border-radius: var(--pk-radius-md);
}
.pk-meter__label { color: var(--pk-ink-dim); }
.pk-meter__value {
  font-size: var(--pk-text-md);
  font-weight: var(--pk-weight-semibold);
  font-variant-numeric: tabular-nums;
  color: var(--pk-state, var(--pk-ink));
}
.pk-meter--balance   { --pk-state: var(--pk-balance); }
.pk-meter--bandwidth { --pk-state: var(--pk-accent); }
.pk-meter--stock     { --pk-state: var(--pk-confirmed); }

/* ===== status chip (attribute-driven hue; introduces NO new color) ===== */
.pk-status { color: var(--pk-state, var(--pk-ink-dim)); }
.pk-status[data-state="catch"],
.pk-status[data-state="tested"],
.pk-status[data-state="land-clean"]      { --pk-state: var(--pk-confirmed); }
.pk-status[data-state="ready"]           { --pk-state: var(--pk-balance); }
.pk-status[data-state="partial-catch"],
.pk-status[data-state="in-flight"],
.pk-status[data-state="land-conflict"]   { --pk-state: var(--pk-inflight); }
.pk-status[data-state="lost-via-rename"],
.pk-status[data-state="anchor-edited"],
.pk-status[data-state="land-checks-red"] { --pk-state: var(--pk-lost); }
.pk-status[data-state="no-catch"],
.pk-status[data-state="blocked"],
.pk-status[data-state="caution"],
.pk-status[data-state="land-pending"]    { --pk-state: var(--pk-ink-dim); }

/* ===== panel (accent / line left-edge clusters) ===== */
.pk-panel {
  padding: var(--pk-space-1) var(--pk-space-4);
  border-left: var(--pk-edge-accent);
  color: var(--pk-ink-dim);
}
.pk-panel--line { border-left: var(--pk-edge-line); }

/* ===== nav ===== */
.pk-nav {
  display: flex;
  align-items: baseline;
  gap: var(--pk-space-4);
  padding-bottom: var(--pk-space-2);
  margin-bottom: var(--pk-space-5);
  border-bottom: var(--pk-border);
}
.pk-nav__home { color: var(--pk-ink); font-weight: var(--pk-weight-bold); text-decoration: none; }
.pk-nav a { transition: color var(--pk-dur-fast) var(--pk-ease); }
.pk-nav a:hover { color: var(--pk-accent); }

/* ===== empty state ===== */
.pk-empty { padding: var(--pk-pad-card); color: var(--pk-ink-dim); }

/* ===== Monaco editor frame ===== */
.pk-editor {
  border: var(--pk-border);
  border-radius: var(--pk-radius-lg);
  box-shadow: var(--pk-shadow-2);
  overflow: hidden;
}
.pk-editor--compose { height: 180px; }
.pk-editor--review  { width: 100%; height: 60vh; }
.pk-editor--diff    { width: 100%; height: 45vh; }
.pk-editor--answer  { width: 100%; height: 14em; }
.pk-editor:empty    { height: 0; border: 0; box-shadow: none; }
```

A migration note: the eight existing buttons can be retired by adding the
relevant `pk-btn pk-btn--*` classes alongside the current hook in markup
(e.g. `.spend-action` keeps its class for any spend-specific rule but
gains `.pk-btn .pk-btn--primary`), then deleting the duplicated rule
bodies. Same for inputs and the editor frames. The token block and the
component layer can land first with zero markup change because the
existing bespoke rules already reference `--pk-*` aliases that still
resolve.

## 5. Micro-interactions and motion

All calm, all short, all using the motion tokens. The principle: motion
clarifies state change, it never performs.

### Focus

The single global `:focus-visible` ring (token block above) is the
biggest a11y win — a strong, theme-consistent 2px bronze halo with a 2px
page-color gap, appearing only for keyboard users (`:focus-visible`), on
every interactive element. It replaces the current weak/absent focus
treatment. No animation on the ring (instant is correct for focus).

### Hover

Board rows, buttons, nav links, and chips transition their
background/border/color over `--pk-dur-fast` (120ms) with `--pk-ease`.
This removes the current instant snap on `.board-row:hover` without
drawing attention — it just feels less abrupt.

### The analyzing indicator

Keep the existing dim→visible fade, retimed to the token: opacity
0 → 1 over `--pk-dur-normal`. Optionally, a very subtle "breathing" of the
ellipsis text via opacity (0.6 ↔ 1.0, 1.4s ease-in-out, infinite) — calm,
not a spinner, and disabled under `prefers-reduced-motion`. The dim
"analyzing…" text already carries the meaning; the breath is a faint
reassurance the run is alive.

```css
.compose__analyzing {
  color: var(--pk-ink-dim);
  font-size: var(--pk-text-xs);
  opacity: 0;
  transition: opacity var(--pk-dur-normal) var(--pk-ease);
}
.compose__analyzing[data-state="pending"],
.compose__analyzing[data-state="analyzing"] { opacity: 1; }
.compose__analyzing[data-state="analyzing"] { animation: pk-breathe 1.4s ease-in-out infinite; }
@keyframes pk-breathe { 0%,100% { opacity: 0.6; } 50% { opacity: 1; } }
```

### SSE re-render settle

When a row re-renders over SSE (a verdict resolves, balance drains, an
order moves queued→running→done), the new content should settle, not pop.
A lightweight approach that needs no JS: animate the re-rendered surfaces
in with a 1-frame fade+lift on mount.

```css
@keyframes pk-settle {
  from { opacity: 0; transform: translateY(2px); }
  to   { opacity: 1; transform: none; }
}
.review-card, .land-row, .dispatch-row, .order-filling {
  animation: pk-settle var(--pk-dur-slow) var(--pk-ease);
}
```

Because morphing reuses DOM nodes where possible, this fires on genuinely
new/replaced rows — exactly the moments worth acknowledging — and stays
silent on untouched ones. Disabled under `prefers-reduced-motion`. The
2px lift is deliberately tiny: a settle, not an entrance.

### The live transcript

The scrolling transcript well becomes `.pk-surface--sunken` (the code
idiom). New lines benefit from `scroll-behavior: smooth` on the container
so an appended beat eases into view rather than jumping.

### What deliberately does NOT move

The honest-state hues never pulse, flash, or animate color. Numbers never
count up. There is no progress bar to fill, no gauge to sweep. A loss is
shown, never shaken. This restraint is the brand.

## Executive summary

1. This is a consolidation-and-polish pass, not a redesign; every sacred
   ethos rule (calm, dark, honest-state color, one-row-independence,
   stylesheet-strippable, accessible) is preserved.
2. The current language is strong in intent (real tokens, principled
   honest-state palette, structural meaning) but has accreted ad-hoc
   sizes, duplicated buttons/inputs/editors, and no type/radius/motion
   scales.
3. The token system gains a real type scale, a six-step spacing scale, a
   radius scale, an elevation/shadow ladder, edge/border tokens, and
   motion tokens — with back-compat aliases so nothing reflows.
4. The five honest-state hues are kept and only sub-perceptually re-tuned
   for even weight against a slightly deeper background, each gaining a
   reusable `-dim` track tint.
5. Eight bespoke buttons collapse into `.pk-btn` + primary/secondary/ghost
   (+danger); inputs into `.pk-input`; four editor frames into `.pk-editor`
   with height modifiers.
6. New reusable primitives — `.pk-surface`, `.pk-row`, `.pk-label`,
   `.pk-chip`, `.pk-meter`, `.pk-status`, `.pk-panel`, `.pk-nav`,
   `.pk-empty` — name the patterns that already exist by repetition.
7. `.pk-meter` and `.pk-status` are explicitly NOT gauges: a held value in
   its state hue, and an attribute-driven hue that introduces no new color.
8. A single global `:focus-visible` ring is the headline a11y win,
   replacing the current weak/missing focus treatment on every control.
9. Motion is subtle and tokenized: 120ms hover, a calm analyzing breath, a
   2px SSE-settle fade — all disabled under `prefers-reduced-motion`;
   honest-state color never animates.
10. The token block and component layer can land first with zero markup
    change (existing rules resolve through aliases); call sites migrate to
    the `pk-*` primitives incrementally afterward.
