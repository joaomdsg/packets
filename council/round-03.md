# Round 3 — first build evidence (mutation oracle slice)

Trigger: built the keystone "mutation-as-question-thread" slice
(`internal/mutation`, validating slice #2). No panelists re-convened —
this logs the evidence the next round will argue over.

## New evidence

- Diff-scoped mutation oracle exists, Go-stdlib-only, TDD-built (RYGBA
  per unit; audit caught a real restore-error-swallowing bug).
- Validating experiment passes: a weak/tautological test (`IsAdult`
  checked only at 25) lets the `>=`→`>` mutant SURVIVE → exactly one
  finding on the right line; a strong test (pins 17/18) KILLS it → zero
  findings. Test-theater made to visibly fail, in miniature.

## Per panelist

- TDD: core thesis vindicated at the unit level — mutation is the
  independent oracle, discriminates weak from strong tests. Wants next
  slice to redefine "confirmed catch" against survived→killed and render
  survivors as real `question:` threads.
- Systems: feasibility confirmed; flags the cost model (one test-run per
  mutant) for the settle-loop budget — see Clash F.

## Clashes touched

- B (PARTIAL — independent signal now exists).
- F (MOSTLY RESOLVED on feasibility; latency-at-scale still open).

Verdicts updated: B, F (see §3).

New clashes opened: none yet. Likely next: mutation latency budget in
the settle loop, and the survived-mutant → `question:`-thread UX.

## Decisions

No VISION/DESIGN text changed; this is evidence, not redesign.
