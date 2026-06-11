# Round 79 — sweep continued: the diff.go half of the -z path fix (Moved now composes) — 2026-06-11

Trigger: the maintainer kept Clash F as-is (R78) and asked for the next item. The
natural next is the residual R76 explicitly left open: `internal/diff/diff.go`
`Compute` was the second half of the same path-quoting exposure, so a `Moved`
verdict on a TAB/control-char-named file still mis-read even after R76 fixed
`reanchor.fileStatus`.

## The residual (R76 scope note, RISKS.md non-ASCII residual)

R40 pinned `core.quotepath=false` on `Compute`, fixing the non-ASCII common case.
But git C-quotes any path containing a tab/newline/double-quote/control char in
the patch headers REGARDLESS of quotepath (`+++ "b/tab\tname.txt"`), and `Compute`
read the path from those headers (`pathFromDiffGit` / the `+++ b/` line). A quoted
header neither matched `+++ b/` nor split cleanly, so `FileDiff.Path` came back
empty/mangled — a consumer matching on the anchor path never found the file's
hunks, and `reanchor` silently degraded a `Moved` to `Outdated` (delta computed
as 0 → hash mismatch at the unshifted position).

## The fix (TDD, RED first)

Re-sourced paths AND add/delete counts from `git diff --numstat -z` instead of the
patch headers. `--numstat -z` emits RAW paths NUL-terminated (immune to ALL
quoting, since NUL cannot occur in a pathname), with counts tab-separated as
`added\tdeleted\tpath` — the path is taken as everything after the second tab
(`SplitN(rec, "\t", 3)`), so a tab IN the filename is preserved; a binary file's
`-` count maps to 0. Hunks still come from the unified patch via a new
`parseHunkGroups`, associated to files by git's stable emission order (one group
per `diff --git`, identical order for both invocations given identical args). The
header-path parsing (`pathFromDiffGit`, the `+++ b/` branch, the manual +/- line
counting) is gone — counts are now authoritative from numstat.

RED tests:
- `TestCompute_reportsTheRealPathAndHunksForAFileWhoseNameContainsATab` (diff) —
  the tab-named file surfaces under its real path WITH its hunks. Failed
  (Path empty) before, green after.
- `TestReanchor_movesAnchorOnAFileWhoseNameContainsATab` (reanchor, the capstone)
  — lines inserted above the anchor on a tab-named file resolve `Moved` with the
  shifted range, not `Outdated`. This proves the two halves (fileStatus -z from
  R76 + Compute -z here) compose; it was Outdated before this fix.

All existing diff tests stay green (binary `-`/`-`→0, rename-as-delete-add,
multi-hunk, non-ASCII, identical-revs-empty), and both consumers (reanchor,
orchestrator's count sum) pass with `-race`.

## Verdict

Full suite green with `-race`. The non-ASCII / control-char path exposure across
the re-anchor read path is now CLOSED end-to-end (both fileStatus and Compute);
no known quoting site remains in diff/reanchor. This was the last crisp,
autonomous-safe defect in the sweep — the rest stay design-gated (R78). Loop
returns to maintenance mode.

## New clashes opened / resolved

Resolved: the `diff.Compute` path source moved off the quotable patch headers
onto `--numstat -z` raw paths, completing the R76 fix. The one assumption it adds
— that `--numstat` and the unified patch emit files in the same order — is git's
deterministic same-traversal guarantee for identical args, and the index zip
degrades safely (trailing files get empty hunks) rather than crashing if it ever
broke.
</content>
