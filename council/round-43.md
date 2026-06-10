# Round 43 — UX/UI visual-design direction OPENS (maintainer-authorized); CSS now in scope — CONVERGED — 2026-06-10

Trigger: the maintainer authorized a new direction — "make the council start
moving toward the UX/UI visual design of interfaces and menus, full user flows."
This is the product/UX taste steer R40/R41 said the Board needed; it UNBLOCKS the
visual work and authorizes CSS (the repo had none; R36 deferred color "until a
stylesheet lands for a bigger reason" — this is that reason).

Panelists: UX/Product Designer (lead), Visual/Frontend Engineer, Calm-UI
Minimalist + Pragmatic TDD.

## Converged design direction

**Visual language (calm "control-room", honest-state):** a restrained dark
surface, high-contrast system-font type, generous spacing rhythm, and color that
encodes HONEST STATE (confirmed / pending-bet / verified-lost / missed / balance)
— muted, never alarmist red-green, never a gauge or progress bar. Design tokens
(CSS custom properties) for color/space/type so later slices inherit them.

**Information architecture / flows (UX, for LATER slices):** `/board` is the
fleet overview (home); `/` (or `/?key=`) is the single-session detail+action
card; `/stream` + `/fleet` are machine SSE (not in the Lead's nav). Core flows:
land on the fleet board → drill into a session → review its verdict → spend/
dispatch → watch the round-trip resolve → back to the fleet. Nav: a minimal
top breadcrumb header + keyboard nav (j/k between rows, →/Enter to drill, ←/Esc
back). These are sequenced AFTER the visual language.

**Guardrails (Calm-UI, even with CSS now allowed):** REFUSE live gauges/progress
bars (VISION cut them — they fabricate urgency), alarmist red/green, eye-stealing
animation, and color that re-encodes a fabricated leverage RANK or invented
metric. The test: "strip the CSS — can you still read the truth?" — meaning must
live in structure+labels, with color as calm reinforcement, not the sole carrier.
The data-honesty refusals (no fabricated leverage rank R24/R36/R41, no invented
treasury number R39) stand.

**Mechanism (Frontend, grounded):** `via.App.AppendToHead(h.StyleEl(h.Raw(css)))`
attaches one inlined stylesheet to every rendered page's <head> (boot-time, in
NewServer before Start) — covers both `/` and `/board`; `/stream`+`/fleet` are
machine SSE (CSS irrelevant). No served-asset route needed; the CSS targets the
class hooks already in the markup (board-row__*, stock-row/stock__*, balance-row,
dispatch-row, beat-row, land-row, review-card__*) so the FIRST slice changes NO
server markup.

**Testability:** existing render/structure tests stay green (markup unchanged);
add ONE test that the stylesheet is attached (`<style>` present in the page HTML
with a known token/selector). Pixel/color values are taste → inspection, not
unit-pinned (assert presence/structure, never pixel values).

## Decision — first build slice (Slice 1)

Introduce the base stylesheet — calm design tokens + rules on the EXISTING class
hooks — and attach it via `AppendToHead` in NewServer. Zero markup change,
reversible, reviewable, unblocks the nav/menu/flow slices that follow.
- New `internal/app/style.go`: the CSS (`:root` tokens + base body + the board
  and card hooks), and the head node.
- NewServer: `app.AppendToHead(h.StyleEl(h.Raw(packetsStyle)))`.
- Load-bearing test: `vt.NewClient(t, server, "/board").HTML()` contains a
  `<style` tag carrying the stylesheet (a token like `--pk-` or a selector like
  `.board-row`); the existing board/card render tests stay green.

DEFERRED to later UX rounds (R44+): the top nav/breadcrumb header + keyboard nav
(markup), the drill-and-return flow, per-state visual polish, menus, and the
larger management-sim interactions. Each its own thin slice on the now-calm base.

## New clashes opened / resolved

None. Reaffirms the data-honesty refusals as the boundary the visual language
respects (color reinforces honest state; it never invents a rank or metric).
