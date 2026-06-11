# Round 80 — sweep: the rename-similarity cliff stops asserting a false "edited" cause — 2026-06-11

Trigger: maintainer picked the rename-cliff residual (#11.5) as the next item.
RISKS.md frames the minimum bar precisely: "The fix must at minimum never assert
a false cause."

## The residual (RISKS.md "reanchor-rename-similarity-cliff" / build-surfaced #11.5)

git's `--find-renames` is similarity-threshold based: a renamed-AND-heavily-edited
file falls below the threshold and shows as delete + add, indistinguishable from a
real deletion. `reanchor.Reanchor` mapped that deletion to `Outdated` — the SAME
state as a genuine in-place line edit — so the seam carried `ReasonAnchorEdited`
and the card rendered "Anchor edited / The anchored line was edited, so the oracle
can no longer speak to the original line."

That is a confidently-wrong cause for a file that VANISHED: the line was not
edited in place, and the true cause might be a sub-threshold rename, not a
deletion at all. R11 established the binding rule that every verdict dimension is
an orthogonal typed field stating a TRUE cause; the deleted-file path violated it
by reusing the "edited" reason.

## The fix (TDD, RED first) — admit the uncertainty, never assert a false cause

A new orthogonal state threaded through all four layers, NO economy change (the
outcome stays `NoOracleSignal`, fail-closed; nothing is minted):

- `reanchor.Deleted` — a new State for "the anchored file is gone (deleted, OR
  renamed below git's similarity threshold)", returned for `statusDeleted` instead
  of overloading `Outdated`. `Outdated` now means ONLY an in-place line edit.
- `pipe.ReasonAnchorDeleted` — CatchAcross maps `reanchor.Deleted` to it; the
  in-place-edit `Outdated` path keeps `ReasonAnchorEdited`. pipe_cycle's lost-anchor
  else branch already handled any non-Same/Moved state generically — unchanged.
- `surface.AnchorDeleted` + a distinct card state `anchor-deleted`: headline
  "Anchor lost: file gone", detail "The anchored file was deleted — or renamed
  beyond recognition — so the oracle cannot follow this line." It admits BOTH
  possibilities and asserts neither as certain.

RED tests (one per layer): `TestReanchor_reportsDeletedWhenFileRemoved`,
`TestCatchAcross_reportsAnchorDeletedWhenFileRemoved`,
`TestReviewCard_rendersAnchorDeletedWithoutClaimingEditedOrRenamed` (asserts the
card neither claims "edited" nor asserts a rename, and DOES surface both the
"delet" and "renamed" possibilities), plus the distinct-token + only-renders-known
extensions in present_test. The old `TestReanchor_outdatesAnchorWhenFileDeleted`
became `…reportsDeletedWhenFileRemoved` — it encoded the coarse behavior this
residual flagged, not a deliberate decision, so updating it is the fix landing.

## Scope held (the detection-tightening half stays deferred — honestly)

RISKS.md also suggests "attempt content-hash relocation before giving up" — i.e.
actively FOLLOW a sub-threshold rename by hashing the anchored content against
added files. That is a detection HEURISTIC with real false-positive risk (two
files can share the anchored lines), a design choice with a taste/precision
tradeoff, not a clearly-correct fix. This round does the minimum RISKS.md
demands — never assert a false cause, admit the uncertainty in the copy — and
leaves active relocation to a deliberate future slice. The card is now HONEST;
making it cleverer is a separate, optional bet.

## Verdict

Full suite green with `-race`. The rename cliff no longer produces a
confidently-wrong "edited" verdict; a vanished anchor reads as the gone-file
state it truly is, with the deletion-vs-rename ambiguity admitted rather than
papered over. Loop returns to maintenance mode — the autonomous-safe honesty fix
is done; active rename relocation remains a maintainer-gated heuristic.

## New clashes opened / resolved

Resolved: the deleted-file path is now an orthogonal typed state
(`reanchor.Deleted` → `ReasonAnchorDeleted` → `AnchorDeleted`), so `Outdated`
means in-place-edit exclusively and the surface states a true cause for a
vanished file (R11 honesty rule extended to the deletion/sub-threshold-rename
case). Open (maintainer-gated): active content-hash rename relocation — a
detection heuristic, not a correctness fix.
</content>
