# Round 47 — the Spend control: close the core loop from the UI — CONVERGED + BUILT — 2026-06-10

Trigger: R46 (first-run onboarding) merged to main. The card now reads the
economy honestly and guides a fresh Lead — but the central economic MOVE (spend a
confirmed catch to fund a work-order) was still unreachable: `LiveCard.Spend`
existed and was well-tested, yet nothing in `View` rendered a control bound to it.
The loop could not be closed from the UI.

Panelists: Calm-UI/Pragmatic-TDD + Producer-experience/Full-user-flow (the
standing UX pair), grounded in the real code.

## The choice

R46 deferred keyboard nav to R47+. Re-examined, the highest-value AND fully
provable next slice was not keyboard nav (still browser-side, markup-testable
only) but the SPEND CONTROL:

- `via`'s `on.Click(c.Spend)` renders a server-side action binding
  (`@post('/_action/Spend')`) that the `vt` client can both READ in the rendered
  HTML and FIRE end-to-end (existing spend tests already fire it and assert the
  balance drains). So the Spend FLOW is fully verifiable in CI — exactly the
  "prove it for real" bar keyboard nav can't meet.
- It closes the product's core loop from the UI: catch → balance → **spend** →
  funded work-order → (caught) reinvest. Without a rendered trigger the Lead reads
  the balance but can never act on it.

## Decision — CONVERGED on the Spend control (built this round)

BUILT (commit 90dbeba): `LiveCard.View` renders
`h.Button(on.Click(c.Spend), class "spend-action", "Spend a catch → fund a
work-order")` directly under the balance row, ONLY when `balance > 0` — offering a
Spend control with nothing to spend is dishonest (the click would be a silent
no-op). style.go gives it a calm balance-hue button (no alarm, no pulsing CTA).
The Spend ACTION logic itself was already built + tested in earlier rounds; this
slice is purely the rendered, honest, balance-gated trigger.

Load-bearing tests (three):
1. balance > 0 → the card renders a control bound to the real `/_action/Spend`
   action (same path the existing spend tests fire), with its class hook and an
   honest label naming the outcome ("fund a work-order").
2. balance == 0 → NO spend control (asserted on the action binding, not the word
   "Spend", which the onboarding copy legitimately contains).
3. SSE retract: spending the LAST catch drains balance to 0 and the drain-to-zero
   re-render frame DROPS the control — View re-evaluates the `balance > 0` guard
   every render, so no dead button lingers. This locks the live disappearance, not
   just the initial-render zero case.

Full gate green (build + vet + `go test -race -p 1 ./...`).

## New clashes opened / resolved

None. Reaffirms "prove it for real" as a slice-SELECTION criterion (R46's
principle): the action that's fully testable end-to-end beats the one that isn't.
The honest-state guardrails hold — the control is shown iff the resource to spend
exists. R48+ candidates: keyboard nav (browser-side, markup-only — land it knowing
that limit or pair with an e2e check); a board empty-state; richer dispatch/claim
interactions. Prefer server-render-testable slices.
