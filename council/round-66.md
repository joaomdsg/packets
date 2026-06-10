# Round 66 — make the dispatch→fill→review loop CLI-reachable (`-backlog` flag) — 2026-06-10

Trigger: with the R65 thread complete (catch → spend → fund → watch it fill → see
the edits → review → answer in place), I flagged that the loop is NOT exercisable via
the CLI — `LiveConfig.DispatchBacklog` is populated only in tests; `cmd/packets` never
seeds it, so `Spend` is always a no-op outside the test suite. The maintainer's "watch
it fill" can't actually be watched live. Full six convened around the gap.

## Ground truth

- `internal/app/supply.go` `fundableBacklog` reads `cfg.DispatchBacklog` (+
  catch-derived candidates). It is the ONLY source of fundable work for `Spend`.
- `grep` confirms `DispatchBacklog` is set in ~15 test files and NOWHERE in
  `cmd/packets/main.go`. The whole funded-work-order economy is, via the shipped CLI,
  dead: nothing to fund → `Spend` no-ops → no order → no fill → no per-order review.
- `parseSessionSpec` already models the exact grammar a backlog target needs
  (base/fix/file/line[/tip], tip-defaults-to-fix, positive-int line, git LineHash).

## Convergence

- The enabling slice (unanimous): a repeatable `-backlog` flag seeding
  `LiveConfig.DispatchBacklog` on the primary session — the smallest change that makes
  the just-built R65 loop genuinely runnable, not just unit-tested. No new economy
  surface; it only *populates* an existing input the supply path already consumes.
- LOAD-BEARING DETAIL (Systems + Refactoring, confirmed in audit): the target's
  `LineHash` MUST be computed with the SAME `lineHashAt(repo, BASE, path, line)` the
  primary target uses. `fundableBacklog` de-dups by full-struct `ledger.Target`
  equality (incl. LineHash) against `ownTargetOf`; a missing/differently-hashed anchor
  would silently double-fund the primary target. The wiring computes it identically.
- TESTABILITY SPLIT (TDD): the pure parser `parseBacklogSpec(spec) → ledger.Target`
  (grammar, validation, tip-default) is unit-tested data→data; the git LineHash lookup
  + flag.Var + LiveConfig assignment are CLI wiring, verified by build/vet (mirrors how
  `parseSessionSpec` splits parse from `lineHashAt`).
- FIREWALL (Systems): off-economy by construction — the flag adds INPUT, no new mint,
  no new scoring. A seeded order funds + fills exactly as a test-seeded one does.
- COMPUTE (CI/CD): zero new test surface beyond the parser's table tests; no docker.

## Build record — SHIPPED

- `cmd/packets/backlog.go`: `parseBacklogSpec` (pure) + `backlogFlag` (repeatable
  flag.Var). `cmd/packets/main.go`: parse each `-backlog` spec → compute LineHash vs
  BASE → assign to the primary `LiveConfig.DispatchBacklog`.
- tdd-rygba: 6 parser tests (target fields, tip-default, missing-field echoes spec,
  non-positive/non-numeric line, whitespace-trim, malformed-pair). Red → Yellow (added
  the trim test) → Green → Blue (full coverage, no extra code) → Audit (no bugs; the
  LineHash-dedup chain traced + confirmed correct).
- Verified: `go build ./...`, `go vet`, `-h` shows the flag, the dataflow lands in
  `fundableBacklog`. The loop catch → spend → fund → watch it fill → review is now
  reachable from the shipped binary, not only the test harness.

## New clashes opened / resolved

None — a clean enabling slice. The only subtlety (LineHash must match for dedup) was
caught by the audit and is satisfied by computing it with the same function/base.
