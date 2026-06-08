# Round 4 — latency + the question-thread artifact

Trigger: closing the two threads round 3 left open.

## New evidence

- Latency benchmark (`BenchmarkRunManySites`, 30-site fixture): cold
  3.24s (~108 ms/mutant), warm 0.91s (~30 ms/mutant). Settle-loop
  viable; parallelizable if needed. → Clash F resolved for serial
  viability (see §3).
- Question-thread artifact (`internal/review`): a surviving mutant now
  converts to an open `question:` thread authored by `agntpr`, anchored
  to the line, rendering as a Conventional Comment ("question: …"). The
  chain mutation→finding→thread→render is proven at the unit level.
  Still a data layer — no UI, no harness wiring yet.

## Clashes touched

- F (resolved on feasibility).

Verdicts updated: F, and the mutation swing in §4.

New clashes opened: none. Next likely: rendering threads in the actual
review surface (Via), and wiring the oracle to run at settle against a
real diff (needs the §17 pipe).

## Decisions

No VISION/DESIGN redesign; evidence only.
