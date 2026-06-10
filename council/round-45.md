# Round 45 — per-state visual polish (verdict + land), in the calm palette — CONVERGED + BUILT — 2026-06-10

Trigger: R43 (base stylesheet) + R44 (nav/drill flow) shipped; the app is
navigable and calm. This round picks the next UX/UI slice.

Panelist: a combined UX-Designer + Calm-UI/Pragmatic-TDD voice (a single focused
council pass — a small, well-understood sequencing fork).

## The choice (A vs B)

- (A) KEYBOARD NAV (j/k rows, →/Enter drill, ←/Esc back) — client-JS via
  datastar; the markup (tabindex/data-on-*) is vt-testable but the key BEHAVIOR
  is browser-side (inspection-only). A power-user layer on the working click-nav.
- (B) PER-STATE VISUAL POLISH — color the card's verdict + land STATES in the
  honest palette. The surface renderers ALREADY emit per-state data-state hooks
  (verdict: catch/no-catch/partial-catch/no-oracle-signal/lost-via-rename/
  anchor-edited/tested/in-flight; land: land-clean/conflict/checks-red/pending),
  all already asserted in the surface tests → pure CSS on existing+tested hooks,
  no markup, no JS.

## Decision — CONVERGED on B (built this round)

B is the cleaner next slice: testable (pure CSS / selector-coverage, vs A's
inspection-only behavior), in the established R43 pure-CSS-on-hooks groove, and
higher immediate VALUE — it makes the verdict the Lead READS legible at a glance,
where today catch/miss/lost are undifferentiated text. A (keyboard nav) is a
sound LATER slice (R46), deferred as a client-JS power-user layer on the working
click-nav.

BUILT (commit 6b87095): per-state color rules in style.go mapping each real
data-state to the --pk-* palette — catch/tested→confirmed (calm), partial-catch/
in-flight→amber (working), no-catch/no-oracle-signal→dim (neutral, not a loss),
lost-via-rename/anchor-edited→muted mauve (anchor lost), land-clean→confirmed,
land-conflict→amber (muted warn, NOT alarm), land-checks-red→muted mauve,
land-pending→dim. Color REINFORCES the headline text (strip-the-CSS test holds);
no alarm red/green, no gauge, no animation (guardrails).

Load-bearing test: SELECTOR COVERAGE — every one of the 12 real verdict/land
data-states has its own per-state rule (a new state can't ship unstyled), plus a
no-#ff0000/#00ff00 guardrail. Colors are taste → never pinned. Pure additive CSS;
every selector targets a rendered+tested state, no dead rules. Full gate green.

## New clashes opened / resolved

None. Reaffirms the calm guardrails: color is honest-state reinforcement, never
an alarm or a fabricated signal. R46 = keyboard nav.
