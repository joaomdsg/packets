# Round 63 — EDITABLE-ANSWERING fork scoped (maintainer greenlit) — 2026-06-10

Trigger: after the read-only Monaco review editor landed + was browser-verified
(R62), the maintainer greenlit the gated editable-answering fork ("yes, proceed"):
let a reviewer ANSWER a surviving-mutant "question:" by writing the killing TEST in
the browser → submit → re-run the oracle → the mutant dies → the question VANISHES.
This is the reviewer's OWN trusted session (NOT the #6 untrusted-producer boundary,
which stays hard-gated). Full six convened to scope it.

REACHABILITY (grounded): pipe.runOracleAt builds a git WORKTREE at the fix rev and
runs mutation.Run(Dir:wt, File, Lines, TestCmd). So answering is buildable:
materialize a fix-rev worktree, INJECT the reviewer's test, re-run mutation.Run on
the same line, recompute findings, update the in-memory findings cache.

## Convergence (all six)

- Systems — THE FIREWALL: the re-run path updates ONLY the off-economy findings
  cache (liveEntry.setFindings); it MUST NOT call log.Append / mint / touch balance.
  The sole mint entry is the catch-record minter (producer flow); a re-run can't
  reach it. Two-scores held. DEGENERATE STRATEGY DISSOLVES: no reward beyond the
  question vanishing → no farm incentive; a vacuous test that kills the mutant is
  acceptable in diagnostic scope (the next full connect cycle re-runs and resurfaces
  truth). RESOURCE: re-run is the user's own in-process session, bounded like a
  connect cycle — no NEW admission control for slice 1 (debounce is later polish).
- CI/CD — FLAKY-TRUTH FENCE: a nondeterministic oracle (Undetermined via timeout)
  must not flip-flop an answer. RULE: a re-run clears a finding ONLY on a genuine
  KILL (the mutant is absent from the new findings); an Undetermined/Survived result
  leaves it OPEN (retryable). Naturally handled by recomputing findings and only
  dropping ones that truly disappeared.
- Refactoring + TDD — CLEAN SEAM: extract a shared buildWorktreeAt(ctx,repoDir,rev)
  from runOracleAt (no duplication), add an EXPORTED pipe.RerunWithTestOverlay(ctx,
  repoDir, rev, file, line, testCmd, overlay) ([]mutation.Finding, error) that reuses
  it, writes the reviewer's test into the worktree (overwrite the package's _test.go
  so `go test ./...` picks it up), runs mutation.Run on the same line, returns the
  fresh findings. App wires a new action/handler that calls the seam (injected as a
  func var, like reviewFileReader/resolveCycle) and updates the findings cache via
  setFindings, re-rendering /review. Reuse the existing cache — no new store.
- TDD — TESTABILITY: the server contract (submit test → re-run → cache shrinks iff
  killed) is testable with a STUB re-runner (load-bearing RED: stub returns fewer
  findings → cache shrinks → /review omits the thread; stub returns same → unchanged).
  The pipe seam gets its own unit test; the Monaco editing is the client island
  (browser-verified, NOT vt). Test-theater boundary: never assert Monaco
  keystrokes/decorations through vt.
- UX + Game — FLOW: a second EDITABLE Monaco pane (pre-loaded with the package's
  existing test file), submit via button + Ctrl+Enter, a calm "running…" status for
  the real oracle wait, the question + decoration VANISH live (SSE) on a kill, and a
  calm "still open — your test didn't constrain line N, try again" on not-killed (no
  scolding, no XP/confetti — the vanishing IS the reward).

## Decision — SLICE ORDER (build over next ticks, tdd-rygba, commit+push, CI)

1. REFACTOR: extract buildWorktreeAt from runOracleAt (behavior-preserving; existing
   pipe tests are the green characterization coverage — keep green, no new tests).
2. pipe.RerunWithTestOverlay (exported) + unit test (worktree + overlay + mutation.Run;
   real or stubbed). Flaky-fence: returns the fresh findings; caller drops only
   genuinely-killed ones.
3. APP: a /answer action (mirror Spend/claim) calling an injected re-run seam →
   setFindings (cache-only, FIREWALL: no ledger) → /review re-render. Server vt RED
   (stub re-runner): a killing test shrinks the cache + drops the thread; a
   non-killing one leaves it. This is the feature-usable server contract.
4. CLIENT: the editable Monaco test pane + submit (browser-verified on :3000).
   Pure-client island; not vt-tested.

Guardrails held: diagnostic-only (off the two-scores economy ledger), calm/honest
(no scolding, no fabricated reward), reachability-grounded, server-contract tested +
client browser-verified. #6 live boundary stays gated (this is the user's own session).

## New clashes opened / resolved

None — a clean convergence. The R62 deferral of editable answering is now RESOLVED
(maintainer greenlit; scoped diagnostic-only with the firewall + flaky-fence).
</content>
