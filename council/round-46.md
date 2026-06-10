# Round 46 — first-run onboarding affordance over keyboard nav — CONVERGED + BUILT — 2026-06-10

Trigger: R43 (base stylesheet) + R44 (nav/drill flow) + R45 (per-state color)
shipped; the app is navigable, calm, and legible. R45 had pre-named R46 =
KEYBOARD NAV. This round re-examined that and converged elsewhere.

Panelists: Calm-UI/Pragmatic-TDD + Producer-experience/Full-user-flow (two
parallel persona passes, grounded in the real code).

## The choice (A vs B)

- (A) KEYBOARD NAV (j/k rows, →/Enter drill, ←/Esc back) — the R45-named next
  slice. But it is BROWSER-SIDE behavior: the app is server-rendered (`via`
  h-trees) and tests run through `vt.NewClient(t,server,path).HTML()`, which
  executes NO client JS. So the keyboard BEHAVIOR can't be verified in CI — only
  the static markup (tabindex/data-on-keydown/role) is testable. That is lint, not
  proof, and cuts against this project's load-bearing ethos: "prove it for real,
  never fabricate green."
- (B) FIRST-RUN ONBOARDING AFFORDANCE — a brand-new session card today renders
  nothing but bare zeros (0 confirmed, balance 0, 0 dispatched), stranding a
  first-run Lead at the entry to the core loop with no signal for what to do or
  why nothing is moving. A calm affordance naming the real flow (oracle mints a
  catch → balance → spend funds a work-order → a caught order reinvests) is pure
  server-rendered markup + CSS, fully vt-testable, and directly serves the
  maintainer's "full user flows" direction.

## Decision — CONVERGED on B (built this round)

Both personas converged independently on B. The deciding reason: B is fully
PROVABLE in CI (server-rendered markup the vt client asserts), where A is
inspection-only behavior our test suite can't settle — landing A would mean
merging a slice CI can't verify, exactly what "prove it for real" forbids. B also
has higher immediate flow value: it's the entry point to the whole economy loop,
and a dead all-zeros first screen is a real onboarding gap. Keyboard nav remains a
sound LATER slice (R47+), deferred as a client-JS power-user layer on the working
click-nav.

BUILT (commit 248e9de): `onboardingHint(stock)` in internal/app/onboarding.go
renders a calm `<section data-state="empty" class="onboarding">` with three honest
lines (current state → how a catch mints to balance → spend funds work and
reinvests) ahead of the economy rows, ONLY when the session is truly fresh.
LiveCard.View conditionally appends it (nil otherwise). style.go gives it the card
surface + a quiet `--pk-accent` left border + dim supporting text — no alarm, no
gauge, no animation (guardrails); strip the CSS and the guidance still reads.

## The single-check emptiness guard (a correctness note, not a shortcut)

The affordance shows iff `stock.Count == 0`. That single check is the COMPLETE
emptiness test, proven from the economy invariants (audited this round): the stock
count is monotonic (a confirmed catch is never un-minted), and it is the
prerequisite for every other sign of activity — `ledger.Balance` rises only on a
confirmed catch, and a dispatch is created only by `Spend`→`AppendDispatch`, which
is refused unless `Balance() >= 1`. So balance>0 or any dispatch ⟹ a prior catch
⟹ stock.Count ≥ 1. Contrapositive: stock.Count == 0 holds exactly when there are
no catches, no balance, and no dispatches. Adding balance/dispatch clauses to the
guard would be dead, untested code (Blue would strip it), so the guard stays the
single honest check.

Load-bearing tests: fresh session renders `data-state="empty"` and each honest
step of the loop copy ("No confirmed catches yet" / "mints to your balance" /
"Spend"); an active session (one minted catch) does NOT render it. Full gate green
(build + vet + `go test -race -p 1 ./...`).

## New clashes opened / resolved

None. Reaffirms "prove it for real" as a slice-SELECTION criterion, not just a
build rule: a UX slice whose value can't be verified in CI yields to one that can.
R47+ = keyboard nav (client-JS power-user layer, markup-testable only).
