# Round 78 — maintenance sweep close-out: the remaining residuals are design-gated, not defects — 2026-06-11

Trigger: the debt sweep (R76–R77) cleared the two genuinely-crisp residuals (the
re-anchor TAB/control-char path; the unresolvable-tip LandConflict mislabel). This
round assesses the rest of the RISKS.md residual catalogue against the skeptic
gate and records why each is HELD for a maintainer decision rather than fixed
autonomously.

## Held #1 — the catch survivor multiset under-credit (Clash F #13): a LOCKED design decision, not a defect

RISKS.md frames "set-not-multiset keying under-credits (2-same-op survivors→1-kill
as NoCatch)" as a fast-follow. On inspection it is NOT a latent defect — it is a
deliberate decision cemented in THREE places:

- `catch` package doc: "keyed by the line's OPERATOR INVENTORY per revision —
  never by individual mutant identity — so that the incoherent 'the same mutant
  was killed' claim cannot be expressed across a fix that edits the line."
- `LineState` doc: "Both are treated as SETS; duplicate operator sites collapse to
  one entry (a deliberate v1 simplification that keys the denominator on the
  operator alphabet rather than unstable per-site identity)."
- `TestLineStateAt_deduplicatesRepeatedOperatorsIntoASet` — built on the EXACT
  Clash F source (`a >= b && c >= d`, two `>=` sites) and asserts the two
  survivors collapse to one. The test exists specifically to lock set-keying.

What the change would and would not cost:

- It is ECONOMY-NEUTRAL: the under-credit only suppresses a `PartialCatch`, which
  `ShouldRecord` never persists — no catch record, no ledger line, no effect on
  the equivalence lock. The change is display-only (a "partial" badge appears in
  one more edge case).
- A safe implementation exists: keep `Inventory` deduped (record/golden/lock
  bytes unchanged) and compare `Survivors` as a multiset, gated on the unchanged
  inventory set-equality. Coherent because equal alphabet ⟹ stable site
  correspondence ⟹ a survivor-count shrink is a real partial constraint.
- But it REQUIRES deleting/inverting a test that documents the opposite intent,
  and contradicts two doc-comments stating the simplification is deliberate.

Verdict: reversing a multiply-documented, test-locked design decision for a
display-only edge-case gain is a maintainer call, not autonomous-safe sweep work
(R59/R74/R75 skeptic gate). HELD pending an explicit steer. If the maintainer
wants it, the safe multiset-survivor implementation above is the path, and
`TestLineStateAt_deduplicatesRepeatedOperatorsIntoASet` is rewritten to assert the
new (multiset) contract.

## Held #2 — re-anchor rename-similarity cliff (#11.5): already honest, tightening is taste-gated

A heavily-edited rename degrades to git delete+add → `statusDeleted` → `Outdated`
→ `ReasonAnchorEdited`, so the card says "edited" not "renamed." RISKS.md already
classes this as "still honest (no phantom catch); not actively false." The only
open move is to TIGHTEN detection or soften the copy to admit
threshold-uncertainty — a UX/taste change (R41/R59 precedent: visual/copy steers
are maintainer-gated), not a correctness fix. HELD.

## Held #3 — bundle-ingest GC-by-resolved: a feature gated on an unbuilt prerequisite

The unbounded-producer-bundle item's full fix sequence starts with a producer-AUTH
boundary the live HTTP surface does not have (/claim and /bundle are session-key
gated, not authenticated). GC-by-resolved is one step in that sequence; building it
in isolation, ahead of the auth boundary it presupposes, is premature
(RISKS.md says so explicitly). HELD until the auth boundary is a sanctioned thread.

## Held #4 — checks read exit code (§12.2.2): no code surface yet

The remaining instance of this is the §16 approve guard, which is unbuilt; the
built check paths (`integrateOnTip`, the mutation runner) already use controlled
exec and read the real exit code. Nothing to fix until the guard is built. HELD.

## Verdict

The autonomous-safe crisp defects in the sweep are CLEARED (R76, R77). Everything
remaining is design-reversal (Held #1), taste-gated (Held #2), or gated on an
unbuilt prerequisite (Held #3, #4). The loop returns to maintenance mode and
surfaces the one decision that is genuinely the maintainer's: whether to reverse
the set-keying simplification (Held #1). No code changed this round.

## New clashes opened / resolved

Opened (for the maintainer): reverse the deliberate set-keying simplification
(Clash F) to credit a same-operator survivor-count shrink as PartialCatch? Cost is
one display-only edge case + rewriting a test that locks the current contract;
benefit is a marginally more generous partial badge. Default (skeptic gate): keep
the simplification.
</content>
