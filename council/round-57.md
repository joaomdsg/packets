# Round 57 — the PREP BENCH thread (curate the work you fund), full 6-member council — CONVERGED — 2026-06-10

Trigger: the review-thread surface thread (R56) shipped feature-complete. The
maintainer delegated SCOPING itself to the council ("council decides over next
ticks what to scope and build"), so the council both PICKS the next thread and
operationalizes the under-specified ones.

## The vote (full six, each from its §1 seed)

- Game Designer → PREP BENCH: surface the fundable backlog as a visible bench; the
  Lead curates/reorders the next targets while compute runs (kills the dead-air the
  VISION names the make-or-break risk).
- Systems/Economy → PREP BENCH: today nextUnconsumedTarget is auto-FIFO (no choice);
  make the Spend fund a CHOSEN target — a real management-sim decision. Degenerate
  strategy checked: the scarce resource (tokens/time) still throttles, and from-catch
  candidates decay, so farming one line doesn't pay.
- Pragmatic TDD → PREP BENCH ranked #1 for testability (a server-side action over
  supply.go's backlog — fully vt-testable); KEYBOARD NAV ranked last = "test-theater"
  (markup testable, behavior is browser-only).
- UX → keyboard nav first (board j/k) then prep-bench — but acknowledges keyboard nav
  is browser-side, only markup server-testable.
- Refactoring → keyboard nav on /review; the refactor-as-task-type thread is NOT
  reachable (needs a multi-kind-dispatch maintainer steer + new anchor/surface types).
- CI/CD → a LAND/MERGE fleet surface (the Land verdict data exists, untapped) — a
  reachable, testable thread, higher than keyboard nav.

## Decision — CONVERGED on the PREP BENCH thread

3 strong votes (Game Designer + Systems + TDD) converge on the SAME concrete
mechanic, and it is the most VISION-central, reachable, AND server-render-testable
option. Keyboard nav (UX, Refactoring) is deferred — its core value is browser-side
behavior our tests can't verify ("prove it for real"); the markup-only test would be
theater. The Land/merge fleet surface (CI/CD) is a sound, reachable thread — noted as
the likely NEXT thread after prep-bench.

THE MECHANIC: turn dispatch from a blind auto-FIFO pick into a curated CHOICE — the
Lead sees the fundable work ("the bench") and decides what the next Spend funds,
sharpening the queue while compute runs. Plumbing exists: fundableBacklog(cfg, log)
already computes the ordered fundable targets (config backlog + from-catch supply);
Spend currently calls nextUnconsumedTarget (FIFO head) and AppendDispatch already
records the chosen Target (the auditable logged fact).

## Scoped slice path (build over next ticks, each via tdd-rygba)

1. RENDER THE BENCH — show the session's fundable backlog as a calm list on the card
   (the next N targets, path:line, the FIFO-next marked), so the Lead sees what's on
   deck while compute runs. Read-only; server-render-testable from fundableBacklog.
2. CHOOSE-TO-FUND — each bench item gets a "fund this" action (on.Click + on.SetSignal
   carrying the target id, the R55 per-row pattern) so Spend funds the CHOSEN target
   instead of the FIFO head. Spend validates the chosen target is in the fundable set
   (never fund the own-cycle target or an arbitrary one). Test: fire choose-and-fund →
   THAT target is dispatched (AppendDispatch records it), not the FIFO head.
3. (if warranted) REORDER / richer curation; otherwise judge the thread complete.

Guards (binding): calm + actionable-not-noise (UX); the chosen target must stay
within fundableBacklog so the economy's distinct-work rule holds and the own-cycle
target is never re-funded (Systems two-scores / no-degenerate-farm); every choice
backed by the AppendDispatch logged fact; server-render-testable only (no browser-
only behavior); #6 live boundary gated.

## Slice 1 (built, commit d857cbf)

renderBench(fundableBacklog(cfg, log)) on the card: a calm ".bench" list of the next
fundable targets (path:line), the FIFO-next marked "(next)", capped at benchCap.
Read-only — the Lead sees what's on deck while compute runs. Pure projection of
existing supply.go plumbing; omitted when no fundable work; guarded on log. Tests:
seeded backlog → bench shows both targets + "(next)"; no fundable work → no bench;
a backlog past the cap → exactly benchCap items. Full-repo -race green. (A small
render helper mirroring renderDispatches — verified via the three tests + the
full-repo gate rather than a separate Blue/Audit subagent, proportionate to size.)
Next: slice 2 — choose-to-fund.

## New clashes opened / resolved

Clash: prep-bench vs keyboard nav vs land-surface. Resolved on prep-bench (3 votes +
testability + VISION-centrality). Keyboard nav deferred (testability). Land/merge
surface queued as the next thread. No open clashes.
