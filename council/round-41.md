# Round 41 — scoping the Board; the loop reaches the edge of autonomous-safe work — 2026-06-10

Trigger: #6c feature-complete, the non-gated correctness backlog cleared
(non-ASCII fix, GC). Council R40 named the Board / management-sim experience the
next major thread. This round tried to scope its THINNEST autonomous slice.

Panelists: Product/Vision, Calm-UI + Pragmatic TDD.

## Finding — the council converged on a BOUNDARY, not a slice

The high-value "human Lead running a fleet" increments are each blocked for the
autonomous loop:

- **Fleet treasury total — RULED OUT (Calm-UI/TDD):** there is no cross-session
  currency pool. Each session's Balance is its own spendable economy;
  Confirmed/Reinvested are counts. Summing them INVENTS a fleet budget the ledger
  does not have — a fabricated concept, not a render of existing data.
- **Leverage / standing ordering or accent — RULED OUT:** this is the exact
  fabricated-leverage trap council R24/R36 explicitly REFUSED. BoardRows orders by
  queued ACTIVITY deliberately, never a priority/leverage rank (blocked-downstream
  is uncomputable; a fabricated rank lies). The Product voice's per-card "leverage
  accent" is this trap, and it conceded the *feel* "needs a lead's eye."
- **Legibility regroup — ALREADY DONE (R40 C4 bets cluster).**
- **What remains idiom-safe + autonomous:** only a marginal span reorder — too
  thin to be worth a slice.

## Conclusion

The genuinely valuable remaining work is gated on a HUMAN:
1. **The live #6 cross-process boundary** — HARD-GATED on maintainer sign-off
   (06-plan, rounds 28/32). The loop must not cross it.
2. **The Board management-sim feel** — needs the maintainer's PRODUCT/UX TASTE;
   the strongest candidates either violate the established calm/no-fabricated-
   leverage idiom or invent concepts (treasury pool) the data doesn't support.
   Building a large user-facing surface on taste guesses would erode trust
   (the loop's charter: inventing significant new work without authorization is
   a trust cost).
3. Everything else is marginal/exotic (the control-char-filename `-z` residual,
   the C3b2b perf hatch) or already-deferred (GC disk reclaim, per-target GC).

Every slice this session also passed a Blue + Audit subagent pass, so a fresh
bug-hunt has diminishing returns. This is a genuine DIRECTION inflection, not a
tactical fork the council can resolve alone.

## Decision

Defer the Board pending a maintainer steer. Surface to the maintainer (one ping):
#6c is complete + the correctness backlog clear; the remaining high-value work is
either maintainer-gated (#6 live boundary) or needs product/UX direction (the
Board). Ask which way to go. Hold the autonomous loop on a long heartbeat until
the maintainer chooses a direction (or interjects). Do NOT fabricate leverage,
invent a treasury pool, build a large UX surface on taste guesses, or cross the
#6 sign-off gate to keep busy.

## New clashes opened / resolved

None. The fabricated-leverage refusal (Clash A / R24 / R36) is reaffirmed as the
reason the obvious Board ranking is off-limits. The Board, when it proceeds, will
want its own round once the maintainer sets a product/UX direction.
