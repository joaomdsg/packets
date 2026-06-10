# Round 48 — per-order dispatch round-trip on the live card — CONVERGED + BUILT — 2026-06-10

Trigger: R47 (the Spend control) merged to main — the Lead can now spend a catch
to fund a work-order from the card. But the card showed only aggregate dispatch
COUNTS (queued/running/done); to learn whether the order they just funded was
caught or missed, the Lead had to leave the card and cross to the fleet board
(which already renders the per-order round-trip via renderDispatches). The action
and its payoff lived on two different surfaces.

Panelists: Calm-UI/Pragmatic-TDD + Producer-experience/Full-user-flow (the
standing UX pair), grounded in the real code.

## The choice (A vs B)

- (A) PER-ORDER DISPATCH OUTCOMES ON THE CARD — render this session's
  RecentDispatches (WO#id path:line status caught/missed) on the live card,
  completing spend → dispatch → watch-it-resolve on one surface. Reuses the
  existing DispatchView projection + the board's renderDispatches helper; rides the
  card's existing SSE re-render; fully server-render testable.
- (B) "NOTHING TO FUND" affordance — when balance>0 but the backlog is exhausted,
  the Spend button is a silent no-op; render an honest note instead.

## Decision — CONVERGED on A (built this round)

Both personas converged independently on A. Deciding reason: the card is the
Lead's primary control surface — they spend HERE, so they must see the outcome
HERE. Showing only aggregate counts forces a context-switch to the board precisely
at the moment of highest interest (did my spend pay off?). A closes that loop on
one surface and reuses proven, already-tested machinery (DispatchView,
renderDispatches), so it's a thin, low-risk slice. B is a real honesty gap but
smaller and narrower; queued behind A.

BUILT (commit 8e3c16c): LiveCard.View reads `log.RecentDispatches(5)` and, below
the dispatch counts row, conditionally appends `renderDispatches(dispatches)` (the
same helper + classes the board uses — board-row__dispatch, calm mono treatment
already styled in style.go) when there is at least one order. The cluster is
omitted entirely when no orders are funded — an empty round-trip block would imply
activity where there is none. It re-reads the log on every render, so it rides the
card's existing SSE re-render: a funded order appears and resolves caught/missed in
place. n=5, newest-first — identical projection to the board (audited consistent).

Load-bearing tests:
1. a session with WO#1 caught + WO#2 missed renders both on the card (WO#1
   alpha.go:7 caught; WO#2 missed) — the round-trip is legible on the card.
2. omit path — a session with no funded orders renders NO dispatch cluster
   (asserted via bodyOf, since the stylesheet carries the class as a selector in
   the head).

Full gate green (build + vet + `go test -race -p 1 ./...`); audit confirmed SSE
liveness rides the existing render path (transitively covered) and the projection
is consistent with the board.

## New clashes opened / resolved

None. Class-naming reuse (board-row__dispatch on the card) is a deliberate
judgment call — same calm mono treatment on both surfaces, one styled helper, no
duplication. R49+ candidates: (B) the "nothing to fund" honesty affordance;
keyboard nav (browser-side, markup-only — land knowing the limit); more first-run
empty-states. Prefer server-render-testable slices.
