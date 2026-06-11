# Round 77 — maintenance sweep: integrate-on-tip fails closed on an unresolvable tip — 2026-06-11

Trigger: continuing the maintainer-authorized debt sweep (R76). Second crisp
residual: the integrate-on-tip "nonexistent tipRev conflated with conflict" item
RISKS.md deferred as low-severity.

## The residual (RISKS.md, "Integrate-on-tip residuals", finding 2)

`integrateOnTip` rebases the fix onto `tipRev` and maps ANY non-zero rebase exit
to `LandConflict`. R-empty-tip (the earlier fix) guarded `tipRev == ""`, but a tip
that is non-empty yet resolves to NO commit (a typo'd / stale / wrong-repo sha)
also exits the rebase non-zero — exactly like a real textual conflict — so it
rendered a confidently-wrong "Trunk moved — rebase needed" verdict for what is a
caller/config error. That is the GDT-failure shape RISKS.md warns about: a
backend verdict presenting a fabricated cause.

It was deferred because the only live caller (`app.Resolve`) passes an
already-validated rev. But the sweep's standard is that a verdict surface must
never assert a false cause regardless of caller discipline (the R11 honesty rule),
and the guard is one cheap command.

## The fix (TDD, RED first)

RED test: `TestRunCatchCycle_failsClosedOnANonexistentTip` — a 40-char all-`deadbeef`
sha as the tip must return an error, not a `LandConflict`. Failed (nil error /
LandConflict) before, green after.

Fix: `integrateOnTip` pre-validates the tip with
`git rev-parse --verify --quiet <tipRev>^{commit}` immediately after the empty-tip
guard. An unresolvable tip fails closed with `pipe: trunk tip <short> does not
resolve to a commit`; a tip that resolves proceeds to the rebase, so a genuine
textual conflict is now the ONLY path to `LandConflict`. The existing
`TestRunCatchCycle_landsConflictWhenFixDivergesFromTip` still yields `LandConflict`
(its tip resolves), confirming the real-conflict path is untouched.

## Scope held

The third integrate-on-tip residual (the integrated-cost 3×-suite multiplier) is
an economics/benchmark item gated on the #15 benchmark, not a correctness defect —
left to its own gate, unchanged.

## Verdict

Full pipe suite green with `-race`. A second confidently-wrong terminal removed
from the Land surface. Sweep continues: the catch multiset under-credit (Clash F
#13) is next, and is the first item that touches the oracle's identity model — it
will get a more careful, possibly council-divergent treatment rather than a
mechanical fix.

## New clashes opened / resolved

Resolved (narrow): the Land verdict now distinguishes three failure causes —
empty tip (config), unresolvable tip (caller/config), genuine conflict (real
LandConflict) — instead of folding the first two into the third. `LandConflict`
is once again a true statement whenever it is shown.
</content>
