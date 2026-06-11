# Round 76 — maintenance sweep reopened: the re-anchor TAB/control-char path residual — 2026-06-11

Trigger: the maintainer authorized a DEBT & DEFECT SWEEP of the autonomous-safe
residuals catalogued in RISKS.md (the loop was holding in R75's maintenance mode).
This round opens the sweep and closes its first crisp item: the re-anchor
non-ASCII residual that R40 explicitly deferred as "pathological."

## The residual (RISKS.md, "Non-ASCII paths break re-anchor + diff", residual note)

R40 fixed the COMMON non-ASCII case by pinning `-c core.quotepath=false` on the
name-status invocation, so accented/CJK paths (café.txt) emit unquoted and match
`Anchor.Path`. But it left a documented residual: a filename containing a literal
TAB, newline, double-quote, or control char is STILL C-quoted by git even with
quotepath=false (`"tab\tname.txt"`), because those bytes are unconditionally
escaped. `fileStatus` split each record on TAB, so such a record mis-split and the
anchored file phantom-resolved as `Same` — silently dropping a catch, the exact
confidently-wrong-silence the re-anchor honesty rule (R11) exists to prevent.

## The fix (TDD, RED first)

Switched `fileStatus` from `--name-status` (TAB-delimited, quoted paths) to
`--name-status -z` (NUL-delimited, RAW paths). `-z` dodges BOTH failure modes at
once: git suppresses all path quoting under `-z`, and the field separator (NUL)
cannot occur in a pathname — so quotepath is no longer even needed. The parse
changed from line/TAB-split to a flat token walk over the NUL stream: a status
code, then its path (two paths for an `R` rename record). Empirically confirmed
the `-z` format first (`R100\0old\0new\0`, `M\0path\0`, raw tab-bearing names).

RED test: `TestReanchor_outdatesAnchorOnAFileWhoseNameContainsATab` — a file named
`"tab\tname.txt"` whose anchored line is edited must resolve `Outdated`, not the
phantom `Same`. Failed as `same` before the fix, green after. The existing
`TestReanchor_followsARenameOfANonASCIIPath` (café→résumé) still passes, so the
R40 fix is subsumed, not regressed.

Dropped the speculative copy-record (`C`) branch from the walker: `--find-renames`
never emits `C`, and treating a copy's source as a rename would be a false
classification (the copied-from file is unchanged). Kept the parser honest to the
codes it can actually see.

## Scope held (the diff.go half stays open — honestly)

`internal/diff/diff.go` `Compute` has the SAME quoting exposure on its path
extraction. This slice did NOT touch it: the RED test resolves `Outdated` purely
via the line-hash-mismatch fallback even when diff path-matching misses, so it does
not prove a diff.go fix. A `Moved` verdict on a TAB-named file WOULD still mis-read
through diff.go — that is the remaining residual, deliberately left to its own
slice (the diff `-z` change is a broader parse rewrite with its own RED case). This
round fixes the documented reanchor `fileStatus` residual only, and says so rather
than claiming the whole non-ASCII surface is closed.

## Verdict

Full suite green with `-race`. One crisp residual closed, the adjacent one named
and left open with a true reason. Sweep continues next tick (tipRev
conflict-vs-error conflation, the catch multiset under-credit).

## New clashes opened / resolved

Resolved (narrow): the re-anchor `fileStatus` parse is now robust to every
pathname byte, not just the non-ASCII common case — `-z` over a token walk
supersedes the TAB-split. The diff.go path-extraction residual remains open and is
now the only known quoting exposure in the re-anchor read path.
</content>
</invoke>
