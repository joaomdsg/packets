# Round 70 — slice 4b design: the live-order catch anchor (anti-farming) — 2026-06-11

Trigger: slice 4a shipped a live work order that PRODUCES a revision but mints
nothing. Slice 4b wires the catch onto a live revision — but a free-form live task
has no obvious anchor for the catch. A short 3-lens council (Systems, TDD,
Refactoring) convened on the anchor model.

## The fork

A PRE-FUNDED order has `Target.{BaseRev,FixRev,Path,Line}` — the catch is checked
against a KNOWN line (`anchorFromTarget(order.Target)`). A live order produces an
agent-chosen diff; where does the catch's anchor come from?

## Convergence — the critical Systems catch

TDD and Refactoring both reached for "derive the anchor from the live diff's changed
lines" (a pure `liveOrderAnchor(base, liveHEAD)` function). **Systems flagged this as
the confirmed-catch FARMING exploit (V§13.5):** if the anchor is derived from the
AGENT's own diff, the agent NAMES THE DENOMINATOR it is scored against — it edits a
line, the system catches on that line, the agent writes a tautological test there and
harvests a confirmed-catch. That violates the project's core integrity thesis ("the
system defines the units; the producer only spends") and the non-gameable-oracle bet.

BINDING RESOLUTION: **a live order's catch anchor MUST be pre-specified independently
of the agent's diff.** The clean way that reuses everything: a live order carries
`Prompt` AND the existing `Target.{BaseRev,Path,Line}` — the prompt tells the agent
WHAT to do; `Path/Line` is the farming-resistant anchor (a known line with a surviving
mutant at base) the catch is checked against, regardless of what the agent edits. The
catch confirms the agent's fix turned that line's survivor-set non-empty→empty — an
oracle the agent did not author. So a live order is "fix the known weak spot at
handler.go:42", not a blank-cheque free-form task (free-form, anchor-from-diff catches
are explicitly OUT — they are the exploit).

## Per lens

- SYSTEMS: anchor independent of the agent diff (above). DEDUP: a live HEAD is
  nondeterministic, so identity cannot include HEAD; but a work order runs ONCE
  (queued→running→done; re-run only on failure under the attempts cap), so the
  single-run lifecycle largely covers it — revisit only if live orders become
  re-dispatchable. COST-GATE: a `context.WithTimeout` around RunProcess (wiring),
  deferrable but cheap; in-scope as a defensive wrap when 4b adds the real run.
- TDD: the `resolveCycle` seam is ALREADY swappable — 4b makes runLiveOrder call
  `resolveCycle(base, liveHEAD, liveHEAD, anchorFromTarget(order.Target), testCmd)`
  and mint/surface exactly like runOneOrder; tests stub resolveCycle to return a
  canned Resolution. First claim: "a live order whose produced revision yields a catch
  mints a CatchRecord with Producer wo:<id>, like a pre-funded order does; a no-catch
  yields no record, balance unchanged." No real oracle/agent in CI. Test-theater to
  avoid: asserting the oracle ran vs asserting the minted outcome.
- REFACTORING: runOneOrder's tail (resolveCycle → Append record → AppendWorkOrderVerdict
  → setOrderFindings → AppendStatus done) is the SAME post-revision logic a live order
  needs. EXTRACT it into a shared `settleCatch(e, order, base, fix, tip, anchor)` both
  runners call — eliminates duplication; reuse `anchorFromTarget` (no new dumping
  ground). runOneOrder has green coverage (status/verdict/findings tests) to refactor
  safely; add one characterization test on the extracted helper.

## Clash resolved

Anchor-from-diff (TDD/Refactoring instinct) vs anchor-pre-specified (Systems
integrity) → Systems wins on the integrity thesis. The live order reuses
`Target.Path/Line` as the pre-specified anchor; the agent's diff never defines the
catch denominator. This *simplifies* 4b: no new anchor-derivation function, just reuse
`anchorFromTarget` + extract the shared settle tail.

## Slice plan

- SLICE 4b (NEXT — BUILD via tdd-rygba): extract `settleCatch` (the shared tail) from
  runOneOrder (refactor, keep green); runLiveOrder, after RunProcess produces the live
  HEAD, calls resolveCycle on (Target.BaseRev, liveHEAD, liveHEAD,
  anchorFromTarget(order.Target), testCmd) and settles via the shared tail — minting a
  catch with Producer wo:<id> when the agent's fix kills the anchored survivor. Wrap
  RunProcess in a context.WithTimeout (cost-gate). Tested via the resolveCycle seam +
  a scripted runHarness; firewall: anchor is Target.Path/Line, never agent-derived.
- SLICE 4c: the live-activity surface line + the PublishActivity wiring (the
  fabric-handle question deferred from 4a).
- SLICE 5+: containerize the agent run.

## Build record — slice 4b SHIPPED

Extracted `settleCatch(e, orderID, res, err)` from runOneOrder's tail (one mint
path: `res.Record` → Append as `wo:<id>`; verdict + findings off-ledger; a cycle
error settles nothing) — behavior-preserving, runOneOrder's existing tests stayed
green. Added pure `lastMintedSHA(turns) (string,bool)` (table-tested). `runLiveOrder`
now, after the harness run (bounded by `liveHarnessTimeout` — the cost-gate), runs
`resolveCycle(BaseRev, liveHEAD, liveHEAD, anchorFromTarget(Target), testCmd)` on the
PRODUCED revision and mints via `settleCatch`; a no-revision run skips the cycle.
tdd-rygba: Red → Yellow (added the no-revision-skip integration test; kept the pure
helper test) → Green (updated a now-stale 4a assertion — the live path legitimately
calls resolveCycle now) → Blue (behavior-preservation of the extraction confirmed;
all paths covered) → Audit (context cancel reached on both paths — no leak; race-clean;
firewall confirmed: anchor=Target.Path/Line, settleCatch the sole mint). FIREWALL test
asserts the resolveCycle args: fix==agent-produced HEAD, anchor.Path/Start==Target's
pre-specified Path/Line. `-race` green; vet clean. (NB a local cage equivalence
failure during the run was purely an exhausted `/tmp` tmpfs — confirmed green with
`TMPDIR=/home`; see the local-tmpfs memory.)

## New clashes opened / resolved

Resolved: the live-order anchor model — pre-specified (Target.Path/Line), never
agent-diff-derived (anti-farming). No doc contradiction; reinforces V§13.5 / the
non-gameable-oracle thesis.
</content>
