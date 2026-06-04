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
- **Sandbox egress allowlist breaks all builds (§19.1).** Allowlist
  (npm/proxy.golang.org/PyPI) misses `sum.golang.org` (Go checksum DB, *verified via
  `go env`*), `files.pythonhosted.org` (pip wheels), and VCS hosts — so default-deny
  fails `go build`/`pip`/`npm` on any dep fetch → the agent can't test → the live
  RED→GREEN flow breaks day one. *Fix:* front all package traffic with one internal
  mirror, not enumerated upstream hosts.
- **The shim can't enforce the sandbox (§15.3/§19.1).** Enforcement (egress, fs
  confinement, permission gating) is credited to an in-container shim that is a peer
  process to the hijackable harness + repo RCE surface (§26.1). Hostile in-container
  code bypasses it. *Fix:* enforce below/outside — kernel seccomp/LSM, netns + host
  egress proxy, out-of-container permission broker. §26.2's "primary defense" is, as
  specified, bypassable by its own threat model.

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

---

## Build-surfaced code risks

Concrete defects/limitations found while BUILDING the slices (distinct from the
design risks above, which are about the spec). Each: where, the finding, the fix.

### Non-ASCII paths break re-anchor + diff path-matching (slice: re-anchoring)

- **Where:** `internal/reanchor/reanchor.go` `fileStatus` (name-status parse) and
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
