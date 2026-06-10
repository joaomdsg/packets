# Round 44 — the first nav / drill-and-return flow, on the calm base — CONVERGED — 2026-06-10

Trigger: R43 shipped the base stylesheet (calm visual language). The app is still
a pile of disconnected URLs. This round designs the FIRST flow slice: make it
NAVIGABLE — a nav header + the clickable drill-and-return loop (fleet board →
session card → back). Keyboard nav is a SEPARATE later slice.

Panelists: UX/Product Designer, Calm-UI + Pragmatic TDD. CONVERGED (minor
class-name reconciliation by the chair).

## Converged design

A shared, stateless `navHeader(key string) h.H` PREPENDED to both Views (each
returns one root container, so prepending is trivial — no via plumbing):
- a "packets" home link → `/board` (`board-nav__home`)
- a breadcrumb: a "fleet" link → `/board` (`board-nav__crumb`); on a card, a
  separator (`board-nav__sep`) + the RAW session key (`board-nav__key`) — honest,
  the real key, never a fabricated label.
Rendered as `<nav class="board-nav">`. On `/board`: `navHeader("")` (crumb shows
just "fleet"). On `/` (the card): `navHeader(c.Key)` (crumb "fleet › <key>", the
fleet crumb IS the back-to-fleet link).

The drill-in: each `/board` row's `board-row__key` span becomes
`<a href="/?key=<key>">` (the default row → `/?key=default` — explicit + honest;
the handler already falls back to "default"). The loop: land on fleet → click a
row → the session card → "fleet" crumb → back to the board. One URL param, two
links, NO client state, NO modal, NO history API.

CSS: add calm nav rules to packetsStyle using the `--pk-*` tokens (a bottom-
bordered header row, home link bold, hover → accent). Meaning is in the markup
(`<nav>` + links) — strip the CSS and the nav still reads.

Guardrails (Calm-UI): a flat `<nav>` only — NO mega-menu/dropdown, NO fabricated
section labels, NO JS interactions (href-based browser nav), NO keyboard focus/
aria yet (the keyboard-nav slice is next). The breadcrumb key is the real
registry key.

Testability: load-bearing tests via vt HTML — structural checks scoped to
bodyOf() (R43: class names now also live in the head CSS), href Contains checks
are head-safe (URLs don't appear in the stylesheet):
- TestBoardCard_rendersNavAndDrillsIntoASession: /board body has `board-nav` +
  `board-nav__home`, and a registered session's row key is `<a href="/?key=<key>"`.
- TestLiveCard_rendersNavWithBackToFleet: "/" body has `board-nav` + a back link
  `href="/board"`.
Pixel/color values stay taste (asserted by presence/structure, never values).

## Decision — next build (slice)

New `internal/app/nav.go`: `navHeader(key string) h.H`. BoardCard.View prepends
`navHeader("")` and turns `board-row__key` into the drill `<a>`. LiveCard.View
prepends `navHeader(c.Key)`. Add the `.board-nav*` CSS to style.go. The two
load-bearing tests above. No removal, only additive markup.

DEFERRED (R45+): keyboard nav (j/k rows, →/Enter drill, ←/Esc back — needs via
action/event binding + focus/aria), per-state visual polish, menus, the larger
management-sim interactions.

## New clashes opened / resolved

None. The breadcrumb-shows-the-real-key choice reaffirms the honesty rule (no
fabricated labels), consistent with the no-fabricated-leverage refusals.
