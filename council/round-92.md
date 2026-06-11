# Round 92 — order-loop slice D-ii: surface the bandwidth meter — 2026-06-11

Trigger: the attention-bandwidth earn (R91) was a sound, tested projection but
invisible. The Lead must SEE the responsiveness they have earned, beside the
catch balance, before it can mean anything — the two meters the Lead chose to
keep, both shown.

## The change

`surface.RenderBandwidth` renders the earned bandwidth as its own row in the
balance idiom: a held quantity (`data-bandwidth` carries the count as a stable
marker), calm at zero, never a percentage gauge, distinct from the
balance/stock/verdict rows. The card's `View` reads `log.Bandwidth()` and renders
it right under the balance, so the two meters sit together. A ledger read failure
degrades to a calm zero, never breaks the card (like the balance read).

## The tests

`internal/surface`: the bandwidth row carries the earned count as a stable marker
and does not collide with the balance row; zero reads calm (no `%`).
`internal/app`: a fast-cleared block (earn 3) surfaces on the card as
`data-bandwidth="3"`.

## Verdict

Full repo green with `-race`. Both economy meters — the catch balance and the
attention bandwidth — are now shown on the card. The earn is visible; D-iii wires
the real block/unblock emission points, and D-iv spends bandwidth to dispatch.

## New clashes opened / resolved

Resolved: the attention-bandwidth meter is visible beside the balance. Open:
D-iii (emit block on a surfaced question / order-awaiting-review, unblock on the
Lead's answer/review) and D-iv (gate live-order dispatch on bandwidth).
