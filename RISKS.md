# agntpr — Risk Register

> Status: **Draft audit.** 40 validated design risks + 12 internal contradictions,
> surfaced by a recurring validation loop against `VISION.md` and `DESIGN.md`.
> Most are validated empirically (git / `go test` / mutation oracle / simulation);
> the rest are analytical, grounded in cited sections. Each entry: where it bites,
> the finding, the fix. Section refs are to DESIGN unless prefixed `V` (VISION).

## How to read this

The codebase today is design docs + a small mutation-oracle/review slice. These
risks are about the *design as written* — the spec the P0 build (§17) will read.
Nothing here is a code bug in the current tree; it's "this assumption will break
when built as described."

## Root-cause families

Almost every finding reduces to one of four patterns:

1. **Mechanical-fact-as-guarantee** — a clean signal read as a semantic/causal/
   durability truth: disjoint files = safe merge, exit 0 = tests passed, mutant
   killed = real catch, 0 survivors = tested, coverage = causation, "state in git"
   = durable, turn boundary = changeset.
2. **Inferred/declared-linkage-as-unit** — a model judgment or agent-controlled
   string presented as an objective, logged unit: catch-weight `P(...)`, plan↔test
   name matching, self-flags.
3. **Statistical-ambiguity the signal can't resolve** — flaky test vs real
   intermittent bug, both indistinguishable from pass/fail variance.
4. **Easy-layer scoping** — claims reasoned at the easy layer (schema / git / the
   pipe) while the hard layer (the economy / durability / the full thesis) is the
   real work: "v2 additive," "hibernation lossless," "P0→P2 proves the thesis."

A cross-cutting consequence: **every trust/safety mechanism (confirmed-catch,
flaky-quarantine, Shadow Review, force-deep, self-flag routing) is itself built on
an agent's judgment or a fragile oracle — there is no independent ground-truth
anchor anywhere in the trust stack.**

---

## CRITICAL — day-one P0/P1 blockers

- ~~**Settle commits secrets & artifacts (§12.2).**~~ **[FIXED 2026-06-04, TDD — both halves,
  secrets-blocked + artifacts-surfaced]** Settle swept secrets + artifacts into every revision. **Fixed (secrets):**
  `internal/settle.Settle` now runs a **per-settle secret scan** over the *added* lines of the
  staged diff before committing; a hit **blocks the commit** and surfaces `Result.Secrets`
  ([]SecretHit{File,Line,Rule}), so a secret never enters history (this is the
  "secret-scrub-scans-wrong-scope" / "scan at every settle" fix). Diff output is pinned
  canonical (`--no-color --no-ext-diff --src-prefix/--dst-prefix`) so a hostile/customized git
  config can't sneak a secret past the parser (regression-tested). Rule set is now 7
  high-confidence patterns (PEM private key, AWS key id, secret-named long-value assignment,
  GitHub classic+fine-grained tokens, Google API key, Slack token, Stripe secret/restricted
  key), each positive + no-over-match tested; entropy detection / allowlist / further
  providers remain a deferred policy decision.
  **Fixed (artifacts) — via SURFACING, never dropping:** staging stays `git add -A` (no
  false-drops); staged **binary** files (the clearest unreviewable-pollution signal) are
  surfaced in `Result.Artifacts` so the reviewer sees them — detected via
  `git diff --cached --numstat --no-renames --diff-filter=d -z` (deletions excluded; `-z` →
  raw unquoted paths incl. spaces/non-ASCII; binary = `-\t-` columns), regression-tested.
  **Deferred policy (not bugs):** further artifact heuristics (large *text* files, size
  thresholds, extension globs) and **auto-exclusion** (deliberately NOT done — prefer
  surfacing over silently dropping an intended file); secret-scan **policy** (entropy /
  allowlist / more providers / FP tuning).
- ~~**Settle fails on no-edit turns (§12.2 vs §4.4).**~~ **[FIXED 2026-06-04, TDD — first
  P0 brick]** A `question:` reply or net-zero turn made `git add -A && git commit` exit 1
  ("nothing to commit"), breaking the turn=revision invariant. **Fixed:** new `internal/settle`
  package; `Settle` mints a revision only when there's a real change — guards on
  `git status --porcelain` AND (after staging) `git diff --cached --quiet`, so both a clean
  tree and a stage-then-revert net-zero turn return `{Committed:false}` with no error/commit.
  Regression tests cover clean / changed / new-untracked / net-reverted. *Residual:* still
  uses `git add -A` (the **iter-15** bug) — explicitly the NEXT P0 brick (TODO in code).
  **(The other settle bug, iter-15, remains embedded in §17.2 step 4 as written.)**
- **Hibernation destroys session work (§15.2).** The session branch is created
  local-only off `base_ref`; §16 pushes only at land; hibernation removes the
  container. A fresh-clone reopen can't find the branch — *verified: `git checkout`
  fails, all unpushed revisions lost.* "Lossless" / "keep 20 sessions open ~free"
  (§24.1) is false. *Fix:* push the branch to durable storage before any teardown.
- ~~**Sandbox egress allowlist breaks all builds (§19.1).**~~ **[SUPERSEDED 2026-06-09 — council R34, built]**
  The egress allowlist was DROPPED entirely. The verification cage (#6c) runs
  `--network=none`: no socket exists, so there is nothing to allowlist and no dep
  fetch to break. The host provides everything offline — a read-only module cache
  populated by a TRUSTED-SIDE prefetch (`go mod download` + `go.sum`/policy verify,
  `GOPROXY=off`, `GOTOOLCHAIN=local`). The allowlist "boundary" moved to that
  trusted-side prefetcher (the only network in the system), so the original
  break-all-builds failure mode cannot occur. See `internal/cage`, `council/round-34.md`.
- ~~**The shim can't enforce the sandbox (§15.3/§19.1).**~~ **[SUPERSEDED 2026-06-09 — council R33/R34, built]**
  There is no in-container shim. Enforcement is the kernel + the container runtime
  (`internal/sandbox.DockerRunner`): `--network=none`, `--cap-drop=ALL`, a seccomp
  profile (proven by a real denied syscall), read-only rootfs, non-root uid 65534,
  pids/memory/cpu caps (pids cap proven by a real fork-bomb denial), one-shot
  container killed on cancel. Hostile in-container code asserts NOTHING about the
  verdict: the cage emits only a transcript and the trusted HOST re-derives the
  catch (lie-green trap) and is the single minter. So "the shim is a bypassable
  peer process" no longer applies — nothing inside the cage is trusted. See
  `internal/sandbox`, `internal/cage`, `council/round-33.md`/`round-34.md`.

## HIGH — correctness / security / trust integrity

- ~~**Mutation oracle hangs & misclassifies (§29.4).**~~ **[FIXED 2026-06-04, TDD]**
  A `+`→`-` mutant can be non-terminating; `runTests` treated a ctx-timeout exit
  identically to a real failure → hung mutant silently counted "killed." **Fixed:**
  `runTests` is now tri-state; a ctx-killed run classifies as `Undetermined` (checked
  before the ExitError path), and `Run` surfaces it as a distinct `Finding{Outcome:
  Undetermined}` instead of dropping it as killed. Survivors are tagged
  `Outcome: Survived`. Regression test: `TestNonTerminatingMutantIsReportedUndetermined…`
  + `loop_hang` fixture. *Residual (by design):* `Run` still has no DEFAULT deadline —
  the caller (e.g. the settle step) must pass a bounded `ctx`; the fix guarantees that
  when it fires, the result is honestly `Undetermined`, never a false "killed."
- **Checks read pass/fail from agent Bash exit code (§12.2).** *Verified:* on a
  failing suite, `go test | tee`, `; echo`, `|| true` all exit 0 → green-when-red →
  the approve guard (§16) lands broken code. *Fix:* run checks via controlled exec
  with structured output (`go test -json`), like the mutation runner already does.
- **Re-anchor: incremental signature vs immutable schema (§28 vs §14).** §28's
  `prevRev→curRev` signature has no place to store the per-revision position; §14
  stores only the immutable base anchor. *Verified from-base re-anchor gives the
  right line; incremental needs state the schema lacks.* *Fix:* commit to from-base
  re-anchoring on read.
- **Re-anchor rename cliff (§28).** "Follow `git -M` rename" is similarity-threshold
  based — *verified: a renamed+heavily-edited file becomes delete+add, silently
  dropping the thread*, indistinguishable from a real deletion. *Fix:* pin the
  threshold, surface "lost via rename" as a distinct state, content-hash relocation.
- **Fan-out "never a correctness risk" is false (§18).** *Verified:* two disjoint-
  file edits (rename a symbol in A; call it in B) merge clean with no conflict →
  broken build. The conflict-guard can't fire for cross-file semantic breakage.
  *Fix:* gate fan-out on a build+test of the *integrated* branch; derive disjointness
  from a symbol graph, not file overlap.
- **Confirmed-catch only well-defined for test-only fixes (§29.3).** *Verified:*
  when the fix edits the anchored line, the surviving mutant (`>`→`>=`) and the
  post-fix killed mutant (`>=`→`>`) are different mutants — "same mutant now killed"
  is incoherent; the survivor's output can *be* the fix. *Fix:* define a catch as the
  line's survivor-set going non-empty→empty, not "same mutant killed."
- ~~**"0 survivors" is ambiguous (§29.4).**~~ **[FIXED 2026-06-04, TDD — ambiguity half]**
  An untested `return x * 0.9` (no mutable operator) reported "0 findings" identically to
  a fully-tested line. **Fixed:** `Run` now returns `Result{Findings, MutantsConsidered}`;
  `MutantsConsidered` = mutable sites generated, so `0 findings + MutantsConsidered>0` =
  genuinely killed/tested, while `0 findings + MutantsConsidered==0` = **no oracle signal**
  (must not read as verified). Regression test asserts the strong(1,killed)/weak(1,survived)/
  no-sites(0) cases. *Operator-set expansion (2026-06-04, TDD):* multiplicative `* / %` (AOR:
  `*`↔`/`, `%`→`*`, closing the financial/scaling blind spot) AND **unary `!`** (removed:
  `!x`→`x`, via a new ast.UnaryExpr path, closing the negated-guard blind spot) are now mutated.
  *Bit-op residual, brick 1 (2026-06-04, TDD):* shifts **`<<`↔`>>`** (token.SHL/SHR) now mutated —
  closing the bit-packing/scaling blind spot. (Same pass also fixed a latent coverage-test bug: the
  `no_mutable_ops` fixture's "no sites" scope had pointed at the func *signature* line, passing
  vacuously; it now scopes the actual operator line — switched to `&^`, still unsupported — so it
  genuinely exercises "unsupported-operator → 0 sites".)
  *Bit-op residual, brick 2 (2026-06-04, TDD):* bitwise **`&`↔`|`** (token.AND/OR) now mutated.
  Guarded the lexer gotcha that `&^` is a single token.AND_NOT (not `&`+`^`), so a supported `&`
  must NOT split `x &^ y` into a `&`→`|` mutant; `&^` stayed unsupported until brick 3.
  *Bit-op residual, brick 3 — CLOSED (2026-06-04, TDD):* bitwise **`^`↔`&^`** (token.XOR/AND_NOT)
  now mutated, completing the bit-op set. The whole residual **`<< >> & | ^ &^` is now CLOSED** —
  the binary-operator whitelist is complete (comparisons, `+ - * / %`, shifts, all bitwise, `&& ||`;
  19 operators). Guards added for the overload hazards this exposed: `&^` is treated as ONE token
  (`TestAndNotIsTreatedAsOneTokenNotSplitIntoAndAndXor`), unary `^x` complement is NOT mutated
  (`TestUnaryXorComplementIsNotMutated`), and compound-assignment ops `+= &= <<= &^=` (AssignStmt
  tokens, not BinaryExpr) are NOT mutated (`TestCompoundAssignmentOperatorsAreNotMutated`). The
  `no_mutable_ops` fixture moved to a compound-assignment body (`x &^= 2`) — categorically not a
  BinaryExpr, so it stays genuinely unmutable regardless of future operator support.
  *Residual (still deferred, NOT started — requires explicit direction):* statement-level and
  literal mutators only. These are a different, broader class (per-settle-cost tradeoff vs iter-1)
  and were never auto-continued; lines without a mutable operator still **honestly** report
  "no signal" rather than masquerading as tested.
- **Flaky vs real-intermittent indistinguishable (§13.8/§29.5).** *Simulated:* the
  rerun gate scores a real 30%-race regression <1% of landings (k=3) → it escapes;
  and variance-quarantine marks the bug-catching test "flaky → inadmissible," hiding
  it. *Fix:* failure-signature + change-correlation, not pass/fail variance.
- **Per-test coverage map isn't free (§29.3/§29.5).** *Verified:* `go test -cover` is
  per-run aggregate; per-test attribution needs N isolated runs or is inaccurate
  (shared setup). And coverage∩ ≠ causation. *Fix:* budget per-test coverage as real
  infra; treat file-intersection as a heuristic needing a bisect tie-break.
- **Secret-scrub scans the wrong scope (§26.2).** *Verified:* an added-then-removed
  secret is invisible to the net `base..head` diff but present in an intermediate
  pushed commit (branch/keep-revisions land). *Fix:* scan the full pushed commit
  range; scan at every settle; squash-on-land.
- **`trusted` auto-land gates on unreliable signals (§13.2/§13.3).** It surfaces only
  on agent self-uncertainty (a confidently-wrong agent flags nothing), classifies
  "low-risk" by diff size (small ≠ safe), and earns trust via the fragile oracles
  (self-sustaining false trust). §13.9's "gun before the safety" as the endgame.
  *Fix:* gate on independent risk + a shadow sample; require oracle integrity first.

## MEDIUM — economy / model / scaling

- ~~**Mutation cost (§29.4):**~~ **[PARTLY FIXED 2026-06-04, TDD]** was N_sites × full-suite,
  strictly **serial**. **Fixed:** `Run` now executes mutants **concurrently** across up to
  `maxWorkers`(=8) isolated working-copy dirs (worker pool; order preserved by mutant index;
  race-clean). Wall-clock now ≈ ceil(N_sites/workers) × suite instead of N_sites × suite. As a
  bonus, the original tree is never mutated (copy-based) — so read-only sources work and the
  package's own tests can run in parallel safely. *Residual:* total *work* is unchanged (still
  one full suite run per mutant) and copying the dir per worker adds I/O — affected-test
  selection / smaller-scope suites (the bigger lever) remain future work.
- **Harness context unbounded & non-durable (§6/§18.1):** "full prior context" grows
  ~200–300k tokens over a normal review (forced compaction → silent fidelity loss;
  *severity is model-dependent — a 1M-context model pushes exhaustion ~5× out*) and,
  more importantly, isn't in the DB (**lost on hibernate/restart** — window-independent).
- **Leverage needs a dependency graph (§6/§12.2):** "blocked-downstream" requires a
  work-order dependency graph §5 lacks; the only fleet graph (§29.7) is file-overlap
  = *conflict*, the opposite relation → the Board's core ranking mis-computes.
- **Earned-concurrency cold-start (§12.7):** the core 1:N game is gated behind
  calibration evidence that's slow (settles overnight), oracle-derived (over-grants),
  decaying, and per-lane → steep new-user/new-subsystem ramp.
- **Treasury raw-vs-priced (§6/§24.2):** prompt caching separates raw tokens from
  priced cost ~10×; "actual spend" needs priced units; the burn-rate thrashing
  detector is confounded by caching; cost-to-land at dispatch forecasts a heavy tail.
- **Catch-weight not redeemable (V§13.5):** `(1−P(self-flagged))(1−P(predictable))`
  uses model-inferred counterfactuals, violating V§12.3 "redeem against a logged
  event"; undoes §13.6's objective catch.
- **Plan↔test mapping is name-based (§13.6):** `go test -json` exposes only the test
  name; a tautological like-named test marks a contract item "fulfilled."
- **Human/agent worktree concurrency (§3/§13.1):** both write the same tree, no lock;
  `git add -A` folds human edits into the agent's revision → author misattribution
  corrupts the Trust Ledger.
- **Event-log concurrency / phantom edits (§13.3):** fan-out scratch-branch activity
  (may be discarded) is logged as source-of-truth → phantom edits on replay; no
  instance/branch dimension; P0 log schema must carry producer + commit-status.
- **Dual source of truth (§13.3 vs §13.1/§14):** "client is a pure projection" vs
  command handlers writing tables directly → divergence on partial failure; re-anchor
  state has no defined home (may be lost on replay).
- **Conventions compound unbounded (V§7):** one global `CONVENTIONS.md` fed to all
  agents, no scoping/conflict-resolution/decay → context tax + instruction dilution +
  contradictory/rework-increasing rules.
- **Landed-teardown vs bounce-retry (§16 vs §29.2):** §16 tears down at Landed; §29.2
  needs a "cheap rebase-retry" hours later — but a conflicting rebase needs the gone
  harness context; "rehydratable branch" ≠ rehydratable agent.
- **Refactor contract: green ≠ preservation (§13.7/§29.6):** *verified:* a refactor
  changing an uncovered path keeps the suite green & unchanged → "behavior preserved"
  passes; §29.6 then inverts the stamp penalty toward the riskiest skim.
- **Invariant View exception-detection (§29.6):** "exceptions = the whole review"
  needs a pure-instance hunk classifier — only textual refactors qualify; an
  unreliable classifier hides the behavior-changing hunk.
- **Shadow Review (V§13.10):** the shadow is another agent (opinion laundered into
  "evidence-backed trust"); "idle tokens" compete with the fleet for scarce Treasury.
  *(A third claimed flaw — self-contradictory targeting — was retracted on self-audit
  as a misparse of "high-but-untested.")*
- **Disagreement Replay (V§13.10):** "reuses data already captured" — but no
  scroll/dwell telemetry exists in the protocol/schema; "the line that mattered" needs
  line-level attribution; scroll ≠ attention ≠ causation; re-points the ledger inward.
- **Time-travel re-execution (V§12.8/§13.10):** replay (deterministic) vs re-execution
  (nondeterministic agent, past harness state not persisted) conflated → "rewind &
  branch" can't reproduce a baseline.
- **Multi-user rewrites the scoring spine (§22):** "v2 additive" reasons about the DB,
  but calibration/the-bet/Focus/tiers/Ship-Quality are single-Lead by construction →
  co-review is core-economy rework, not additive UI.

## Internal contradictions (the docs assert both halves)

VISION's closing "now coherent and defensible" is not met. 12 live contradictions —
the panel-hardening sections (§§12–13/29) supersede or compete with earlier text but
left it unedited, and sometimes conflict with each other:

1. §29.2 Landed-non-terminal vs §16 teardown-at-Landed.
2. §12.10 "self-flags not the spine" vs §13.4 pre-point-at-self-flags.
3. §13.3 pure-projection vs §13.1 direct table writes.
4. §28 incremental re-anchor vs §14 immutable anchor.
5. §4.4 no-edit turns vs §12.2 settle-every-turn.
6. §12.3 redeem-against-logged-event vs §13.5 inferred catch-weight.
7. §15.2/§16 container-removed vs §24.1 lossless-hibernation.
8. §6 idle-cost ranking vs §12.2 idle-leak-cut.
9. §4 no-chat vs §12.9 the-brief-rail (self-aware reframe).
10. §13.6 green-is-decorative vs §13.7/§29.6 green-=-preserved.
11. §13.2 ledger-outward vs §13.10 Disagreement-Replay-inward.
12. §12.5 Focus-never-a-gate vs §12.4/§13.3 forced-deep.

*(A former 13th — §13.1 "trust is a Focus discount" as self-contradictory — was
retracted on self-audit: the discount reconciles as a post-hoc measured-cost
adjustment, no live bar needed. It survives only as a wording-clarity nit.)*

*Fix:* a reconciliation pass that edits/strikes superseded passages (not just appends),
plus an "amends/superseded-by" table atop each doc.

## Two meta-findings

- **The build order de-risks the wrong thesis.** P0→P2 proves the DESIGN "pipe"
  thesis; all 8 scoring/trust-integrity risks live *beyond* P2 — in the VISION
  trust-economy thesis VISION itself calls the groundbreaking part.
- **The design's own acceptance bar would catch none of this.** §27's happy-path
  trace ("if this trace runs, the thesis holds") exercises only the pipe and dodges
  every §10 risk — it would green-light a system failing all 40 findings. *Build the
  acceptance suite from adversarial traces, not the happy path.*

## Recommended sequencing

1. **Fix the settle bugs (CRITICAL) inside P0 §17.2 step 4** — cheap now, load-bearing.
2. **Move enforcement below the container** (seccomp/netns/external broker) + internal
   package mirror — before any repo runs untrusted.
3. **Push branch to durable storage before teardown/hibernation.**
4. **Prototype the confirmed-catch oracle** against the mutation findings (cost, hang,
   identity, 0-survivors) *before* building Ship Quality / the Trust Ledger on it.
5. **Do the doc reconciliation pass** (13 contradictions) so P0 reads one spec.
6. **Resolve live-vs-post-hoc Focus** (the economy keystone) before any economy UI.
7. **Harden the oracle before surfacing its verdict — opacity kills, not bad math.**
   Game Dev Tycoon's review score was near-deterministic and skill-based, but
   hidden and unattributable, so players who repeated identical steps saw 10/10
   then 3/10, concluded it was RNG, and offloaded all mastery to wikis
   ([postmortem][gdt]). agntpr's confirmed-catch is structurally the same object —
   a backend verdict on a line — and inherits the same risk, made *worse* by the
   oracle-fragility entries above (hang-misclassify, catch-identity, 0-survivors):
   rendering a confident causal chain ("mutant killed ✓") that is fabricated for
   the common fix shapes is GDT's failure with a fake explanation bolted on, which
   is strictly worse than no explanation. *Rule:* the oracle must reach
   timeout-as-equivocal, survivor-set catch definition, and explicit
   coverage-surfacing (all tracked above) **before** any verdict is rendered inline
   or allowed to gate Ship Quality / Trust / auto-land. Sequencing constraint, not
   a feature.

[gdt]: https://gamecritics.com/tayo-stalnaker/game-dev-tycoon-review/

---

## Build-surfaced code risks

Concrete defects/limitations found while BUILDING the slices (distinct from the
design risks above, which are about the spec). Each: where, the finding, the fix.

### Confidently-wrong quiet verdict on the served card (slice: the §17 wire) — FIXED 2026-06-04

- **Where:** `internal/pipe/pipe.go` `CatchAcross`, `internal/pipe/pipe_cycle.go`
  `CycleResult`, `internal/surface/present.go` `PresentVerdict`,
  `internal/surface/card.go` `present()`.
- **Finding (surfaced by Council Round 11, all six lenses, in shipped code):** the
  served review card (#10 wire) resolved a renamed OR edited anchor to a calm,
  terminal verdict reading "This line has no mutable operator — the oracle cannot
  speak to it." This was a CONFIDENT FALSEHOOD: `catch.NoOracleSignal` was triple-
  overloaded across two seams — `catch.Detect` returns it for a genuinely operator-
  free line (catch.go:52-54), and `CatchAcross` fail-closed BOTH `reanchor.Outdated`
  (edited anchor) and `reanchor.LostViaRename` (renamed file) to it. `reanchor.State`
  distinguished all four states but `CycleResult` carried only `Outcome`, dropping the
  State before the presenter could see it. The roadmap's earlier framing ("a renamed
  file spins 'Oracle running…' forever") was itself wrong — the card RESOLVED to a
  false terminal, which is worse than an honest spinner. This is precisely the
  confidently-wrong-terminal the confirmed-catch economy exists to prevent, living in
  the surface; it re-opened Clash G at the surface layer (resolved-at-oracle ≠
  honest-at-surface).
- **Fix:** a typed `pipe.Reason` {ReasonNone | ReasonNoMutableOperator |
  ReasonAnchorEdited | ReasonFileRenamed} carried as a dimension ORTHOGONAL to
  `catch.Outcome` (the economy/ledger token is unchanged) — `CatchAcross` returns it,
  `CycleResult` threads it. `PresentVerdict` splits a `NoOracleSignal` verdict by
  reason into three honest tokens (`surface.LostViaRename` / `surface.AnchorEdited` /
  the operator-free `no_oracle_signal`, kept only where that copy is true), each a
  distinct card data-state with a true detail. Gate tests:
  `TestResolve_rendersLostViaRenameVerdictForARenamedAnchor` (real rename end-to-end
  through the seam) + `TestReviewCard_rendersLostViaRenameWithoutClaimingNoOperator`.
  This also establishes the seam rule (Clash C, elevated to binding): every verdict
  dimension is an orthogonal typed field, never a new meaning on `NoOracleSignal` —
  integrate-on-tip's `Land` (#12) must follow the same pattern.
- **Residual (deferred, council #11.5):** rename detection is git `--find-renames`
  similarity-threshold based — a heavily-edited rename degrades to delete+add →
  `statusDeleted` → `Outdated` → `ReasonAnchorEdited`. Still honest (no phantom catch;
  the card says "edited" not "renamed", not actively false), but coarser than the
  true cause. The fix must at minimum never assert a false cause; tightening detection
  or admitting threshold-uncertainty in the copy is the fast-follow (tracks the
  re-anchor rename-similarity-cliff finding above).

### Integrate-on-tip residuals (slice: #12 integrate-on-tip)

- **Where:** `internal/pipe/pipe_cycle.go` `integrateOnTip`.
- **Findings (build-surfaced, two of three handled):**
  1. **Empty `tipRev` — FIXED.** A caller leaving `LiveConfig.TipRev`/the tipRev param
     empty made `git rebase ""` exit non-zero → silently mislabeled `LandConflict` (a
     confidently-wrong terminal). Now guarded: `integrateOnTip` returns an error on an
     empty testCmd OR empty tipRev, so the card stays honestly in-flight rather than
     showing a false integration verdict. `cmd/packets` defaults `-tip` to `-fix`.
  2. **Nonexistent `tipRev` conflated with conflict — DEFERRED (low severity).** A
     genuinely bad (nonexistent) tipRev also exits non-zero → reported as `LandConflict`
     rather than an error. Acceptable for now: the only caller (`app.Resolve` via the
     wire) passes an already-validated rev; a bad tip cannot arise from a realistic
     caller. Fix later: distinguish "rebase in progress" (real conflict) from an
     immediate rev-resolution failure before labeling `LandConflict`.
  3. **Integrated-cost multiplier — DEFERRED to the #15 benchmark gate.** A cycle now
     runs the oracle TWICE (base + fix) PLUS a third full `testCmd` run on the rebased
     integrated tree; a future merge queue re-runs integrate per tip move. The
     K-concurrent-settle benchmark must measure this INTEGRATED cost (3× suite + per-tip
     re-runs) before any catch PRICING (#17), per the Round-10/12 gate.

### Non-ASCII paths break re-anchor + diff path-matching (slice: re-anchoring) — FIXED 2026-06-10 (council R40)

- **Fix:** pinned `-c core.quotepath=false` on the git invocations in BOTH
  `internal/reanchor/reanchor.go` `fileStatus` and `internal/diff/diff.go`
  `Compute`, so git emits real non-ASCII pathnames instead of the octal-quoted
  `"caf\303\251.txt"` form — path-matching and path extraction now work on
  accented/CJK filenames. Locked by `TestReanchor_followsARenameOfANonASCIIPath`
  (a `café.txt`→`résumé.txt` rename resolves LostViaRename with the real new path)
  and `TestCompute_reportsTheRealPathForANonASCIIFile`. The Audit confirmed no
  third latent site (settle already uses `-z` raw paths; reanchor's `fileAt` and
  ingest's `for-each-ref` parse no file paths from output). **Residual (exotic,
  not fixed):** filenames containing a literal TAB / newline / `"` / control char
  are still C-quoted by git even with quotepath=false, so they could mis-split on
  the `\t` delimiter in `fileStatus`; the proper fix is `-z` (as settle does) — a
  larger change, deferred as pathological.
- ~~(original)~~ **Where:** `internal/reanchor/reanchor.go` `fileStatus` (name-status parse) and
  the `f.Path == a.Path` hunk loop; `internal/diff/diff.go` path extraction.
- **Finding:** git's default `core.quotepath=true` octal-quotes and double-quote-
  wraps non-ASCII paths in `--name-status`/`--numstat`/`diff` output (e.g.
  `café.txt` → `"caf\303\251.txt"`). So an `Anchor.Path` containing non-ASCII
  bytes never matches the quoted output → `fileStatus` falsely returns
  `statusUnchanged` (Same), and `diff.Compute`'s `Path` is the mangled quoted
  form. A real catch on such a file is silently mis-handled (phantom Same instead
  of Moved/Outdated). Surfaced by the re-anchoring audit; both packages share the
  defect, so a one-package fix is a misleading half-fix.
- **Fix:** pin `-c core.quotepath=false` on the git invocations in BOTH packages
  (reanchor's name-status and diff's diff), then re-verify path matching. Deferred
  to a dedicated brick so the fix lands coherently across both. Existing tests use
  ASCII paths, so the current suite stays green; this is a correctness gap for
  non-ASCII repos, not a present test failure.

### Claim AckWait omits the concurrency-semaphore queue-wait (slice: #6c governor) — KNOWN, ACCEPTED at prototype scale — surfaced by the R42 integration bug-hunt

- **Symptom:** the governor pins `claimAckWait` (240s) > `cageVerifyTimeout`
  (120s) so a slow-but-legal verify finishes before its durable redelivery
  (internal/app/claims.go). But the durable consumer acks only after the WHOLE
  `handle` returns, and handle latency = `Admission.Concurrency` semaphore
  queue-wait + verify. The queue-wait is unbounded under sustained saturation
  (many producers, few cage slots), so total handle time can exceed AckWait →
  NATS redelivers the still-in-progress claim. The claims.go comment ("ackWait
  MUST outlast cageVerifyTimeout") is necessary-but-NOT-sufficient — it omits the
  queue-wait term.
- **Why accepted:** `ConsumeDurable` Fetches one message at a time on a single
  serial loop (internal/fabric/consume_durable.go), so a redelivery is a
  SEQUENTIAL re-verify (wasted cage compute), never a concurrent double cage run;
  and `Append`'s locked identity gate dedupes any re-mint. So the economy stays
  correct — the cost is wasted compute under load, not a correctness bug. The
  R42 cross-cutting concurrency bug-hunt otherwise found the assembled #6c system
  sound (git clone/fetch/update-ref races empirically safe; double-mint
  impossible; GC self-healing holds; no goroutine leaks).
- **Fix when it matters (deployment/load):** make AckWait dynamic or generous
  vs the worst-case queue depth, or bound the queue-wait (reject/shed beyond a
  depth), or extend the ack (in-progress heartbeat). Couples with the deferred
  flood-defenses (gated on producer auth, below). Also noted: FleetBoard's
  two-pass replay can transiently over-count InFlight across a mint that commits
  between the passes — display-only, self-corrects next event (see WatchFleet's
  perf note).

### Unbounded producer-bundle ingest storage (slice: #6c A.2 POST /bundle) — KNOWN, DEFERRED (gated on producer auth) — council R39

- **Symptom:** POST /bundle (internal/app/live.go) accepts a producer git bundle
  and unbundles it into refs/producers/<key>/* of the session repo. It has a
  per-CALL 32 MiB cap but NO per-producer rate limit and NO aggregate storage
  quota, and ingested objects are never GC'd. A producer that POSTs bundles
  repeatedly grows the host store without bound → disk-fill DoS; objects from
  resolved or never-submitted claims are never reclaimed.
- **Not an economy threat (council R39, Systems):** objects are off-ledger — the
  two-scores ledger and single-minter invariants hold regardless of store size.
  This is an ops/security concern, not a correctness one.
- **Why deferred:** the per-producer flood-defenses (rate limit, aggregate quota)
  presuppose a producer-AUTH boundary the live HTTP surface does NOT have today —
  /claim and /bundle are SESSION-KEY-GATED, not authenticated (fabric's
  ProducerGrant is the NATS path, not wired into the live server). Governing "a
  producer" against a malicious flood is premature without that identity layer.
  *Fix sequence:* (1) producer auth on the live HTTP surface; (2) per-producer
  /bundle rate-limit (reuse the proven token bucket) + aggregate quota by
  bytes-ACCEPTED (deterministic — never git's on-disk size, which is brittle);
  (3) GC-by-resolved (BEING BUILT next — prune refs/producers/<key>/* for
  minted/rejected targets, keep in-flight) reclaims the working set and frees
  quota; (4) a TTL-reap for uploaded-but-never-claimed objects + a global disk
  ceiling as defense-in-depth.

### Fleet-stream refold amplification (slice: #6c C3b2b live claim lifecycle) — KNOWN, ACCEPTED at prototype scale

- **Symptom/cost:** to carry the claim lifecycle live, `WatchFleet` now wakes on
  the WHOLE event taxonomy (`FleetEventsSubject`: minted ∪ claim ∪ scratch), up
  from minted-only. So EVERY committed event — including high-frequency discarded
  *scratch* fan-out — drives a full refold, and each refold (`ledger.FleetBoard`)
  is TWO full `ReplaySubject` passes (minted + claim). Net: ~2 full-stream replays
  per committed event per connected `/fleet` viewer, O(stream)×events×viewers.
- **Why accepted now:** at prototype scale (few sessions, few viewers, modest
  stream) this is fine, and it bought the live verifying-pulse with one clean
  seam. Documented inline in `internal/bridge/fleet.go` (WatchFleet).
- **Escape hatch when it bites (scratch rate or viewer count grows):** debounce/
  coalesce wakes, or fold incrementally (apply each event to a cached board)
  instead of replaying the whole stream per event. This is the same hot-path
  caution first flagged for the live SSE path in the C1a perf note.

---

## Design direction: NATS/JetStream as the orchestration event-log/bus

Status: **noted, deferred** (Round-10 era). Not adopted for the single-user
`§17` wire (#10) — in-process server-push (via.Broadcast / OnConnect+Stream) is
sufficient to watch one card resolve, and an external broker there is infra
cost for no gain. Adopt at the event-sourcing / fan-out / Board slices.

A JetStream stream is an ordered, durable, replayable log — a near drop-in for
the substrate several findings need, and it RETIRES three of them:

- **dual-source-of-truth projection** — one append-only stream replaces the
  oscillation between event-sourced (§13.3) and direct-write CRUD; consumers
  (surface, ledger, Board) subscribe.
- **event-log concurrency phantom edits** — per-message seq + a
  producer/commit-status header on each published event demuxes fan-out
  (§18) scratch-branch activity from source-of-truth, instead of one ambiguous
  monotonic seq.
- **timetravel re-execution not projection** — JetStream replay-from-seq gives
  deterministic EVENT replay cleanly; the caveat stands (replaying events ≠
  re-running the agent), so "rewind & branch" still cannot reproduce a
  nondeterministic baseline — NATS provides the event substrate, not agent
  determinism.

Scope/fit: server-side only — the browser edge stays SSE (Via); the surface
subscribes to NATS and pushes to the browser. Also serves the replayable trace
(#14 timed trace), fan-out to N agents (§18, #16), and the async Bounce /
landing-outcomes bridge (§29.2/§29.3). As an external broker it inherits the
§15/§19 egress/auth/enforcement scrutiny. A council round should formally adopt
it (it rewrites §13.3) when the event-log/Board slice is next.
