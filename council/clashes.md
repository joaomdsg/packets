# The open clashes (the heart — resume these after builds)

Each clash: the disagreement, the positions, the current (provisional)
resolution, **the experiment that settles it**, and the verdict trail
(condensed to the latest status).

## Clash A — Is _any_ per-card cost signal acceptable, or pure guilt?

- **UX:** cut all per-card economic meters; cost is guilt. Rank by
  leverage; reveal Focus only retrospectively at close-out.
- **Systems:** a per-card _burn-rate / thrash signature_ is diagnostic, not
  guilt — it's how you spot a runaway agent.
- **Provisional resolution:** no live drain meter; per-card cost appears
  only as a _thrashing diagnostic_ and only at decision moments.
- **Experiment:** build the Board with the thrash-diagnostic on vs off. Do
  users catch runaways without it? Does its presence read as nagging?
- **Verdict (R10):** STILL OPEN on framing, but no longer a thought
  experiment — the served card makes silent-vs-badge testable against
  pixels (zero-survivor renders an affirmative "Tested — ship it", distinct
  from blind "no-oracle-signal"). Meters are deliberately OFF the first
  screen. Settles the moment a human opens the wired card.

## Clash B — Can agent self-confidence route the reviewer's attention?

- **UX + TDD:** No — a confidently-wrong agent flags nothing; don't build
  the attention spine on the least trustworthy signal.
- **Game + Systems:** self-flags are the highest-leverage attention hint.
- **Provisional resolution:** self-flags are a _hint + a calibration input
  measured against outcomes_; the **independent** spine is mutation
  (§13.6). Flag-density-vs-review-time mismatch = the stamp detector.
- **Experiment:** on a real changeset corpus, measure correlation between
  self-flag density and mutation-discovered weak spots. If low, demote
  self-flags to decoration.
- **Verdict (slice 2025):** PARTIAL. The independent oracle exists and works
  (`internal/mutation`): weak test → surfaces the surviving mutant; strong
  test → silent. Self-confidence no longer _has_ to be the spine. The
  self-flag↔mutation correlation experiment still needs a corpus. STILL
  OPEN, baseline established.

## Clash C — Is it fair to score a human on downstream CI truth?

- **Systems:** regression hits feed Ship Quality and Trust calibration.
- **CI/CD:** downstream red is often flaky/environmental/mis-attributed;
  scoring a human on it is unfair and will kill trust in the Ledger.
- **Provisional resolution:** flaky quarantine + settle-gating +
  eventually-consistent back-application + depth-scaled penalty;
  **change-fail-rate replaces raw regression-hit** as the headline.
- **Experiment:** run against a repo with real flake. Does the score feel
  fair? How often does quarantine misfire?
- **Verdict (R12 #12 — RESOLVED-IN-CODE on the integration seam):**
  `pipe.integrateOnTip` rebases fixRev onto a real `tipRev` in a throwaway
  worktree and runs checks on the INTEGRATED tree; `CycleResult.Land` is a
  typed {LandClean | LandConflict | LandChecksRed}, orthogonal to
  catch.Outcome (mint token byte-identical). Load-bearing RED is green:
  `cleanRebaseButChecksRedYieldsChecksRed` proves a green pre-integration
  catch is a red post-integration regression; a disjoint-trunk
  `landsCleanOnNonConflictingTip` guard proves the verdict constrains; the
  card shows Land as its own SSE row. The catch is minted against the tree
  that actually integrates — "Landed ≠ Merged" is computed, not labelled.
  What remains for the SCORING question is the flaky-quarantine /
  change-fail-rate layer (§13.8) + catch pricing (#17), gated on the #15
  K-concurrent benchmark. (History: R8 named the settling fixtures; R11
  elevated the typed-orthogonal-field seam discipline as a binding rule
  after `CycleResult.Outcome` lossily collapsed three reanchor states into
  one `NoOracleSignal` token.)

## Clash D — Does the fleet (1:N) actually scale, or cap at N≈2–3?

- **Vision thesis:** review is 1:N; reviewing five Claudes is just a queue.
- **Game design:** review is a deep context-load; the context-switch tax
  may make N>3 _slower and more error-prone_ — "The Board" above that is
  theater.
- **Provisional resolution:** earned concurrency (§12.7) gates N to
  measured calibration; start serial.
- **Experiment:** measure review quality (mutation-caught defects, rework)
  vs number of concurrent in-flight reviews. Find the real ceiling. Is it
  per-user-trainable?
- **Verdict (post-build):** _TBD_

## Clash E — Does the Prep Bench kill dead-air, or create a second job?

- **Game design (with itself):** unseeded, the Bench relocates the void to
  onboarding; unbounded, it's a parallel obligation that kills rest.
- **Provisional resolution:** Bench is seeded (never empty), lightweight,
  interruptible, and pre-flights the _incoming_ diff; onboarding uses a
  scripted prep track.
- **Experiment:** instrument idle time and self-reported load during agent
  compute. Do users idle (bench failed) or feel doubly-taxed (overshot)?
- **Verdict (post-build):** _TBD_

## Clash F — "Confirmed catch": test-flipped vs mutant-killed?

- **Systems (R1):** catch = a test flipped, weighted by hiddenness.
- **TDD (R2):** that's farmable — the agent authors the flipping test;
  require a **mutant that survived-before and is killed-after** (an oracle
  the agent didn't write).
- **Provisional resolution:** adopted TDD's definition (§29.3/§29.4).
- **Open cost question:** is diff-scoped mutation fast/cheap enough to run
  every settle without wrecking latency or token budget?
- **Verdict (#3 pipe — RESOLVED-IN-CODE on BOTH unit and loop):**
  `internal/catch.Detect` enforces the survivor-set identity key (the
  anchored line's current operator inventory per revision, never "same
  mutant killed") via one pure function, refusal arms tested. The §17 pipe
  (`pipe.RunCatchCycle`) mints the catch from two real revisions end-to-end
  (settle→worktree→mutation×2→reanchor→CatchAcross→Detect): strengthening
  the test mints a real Catch; editing the anchored line yields
  NoOracleSignal. What remains is ECONOMY, not oracle: the Catch is
  computed but not yet persisted as a ledger record (capture-at-mint, #7),
  and is minted on pre-integration coordinates (integrate-on-tip, #5).
  Findings logged: the reanchor gate (edited line → Outdated →
  NoOracleSignal) fires BEFORE Detect's inventory-change NoCatch rule, so
  an edited anchored line reads as NoOracleSignal not NoCatch (both safe, no
  phantom, but a consumer can't distinguish "edited" from "operator-free");
  set-not-multiset keying under-credits killing one of two same-operator
  survivors on a line (v1 risk). (History: R3-4 resolved latency — warm
  ~30ms/mutant, parallelizable, no fallback oracle needed; R5 re-opened on
  identity; R7 converged on the tri-state primitive; R8 made it
  build-ready; R9 shipped the unit green.)

## Clash G — One unified review model, or a refactor fork?

- **Original design:** one diff-first/anchored/fan-out model for all work.
- **Refactoring:** that model is hostile to refactors; needs a separate
  task-type, the Invariant View, transformation anchors, inverted stamp
  penalty.
- **Provisional resolution:** refactor is a first-class task-type (§29.6) —
  accepts the added surface area.
- **Experiment:** run a real 30+ file rename and an extract-module through
  the Invariant View. Genuinely reviewable? Does the behavior-preservation
  proof hold and feel trustworthy?
- **Verdict (R11 #11 — surface honesty RESOLVED-IN-CODE):** the carnage
  baselines stand (`internal/refactor/trace_test.go`: a 40-file rename
  orphans all 40 threads; neutral rename asserts LostViaRename != Catch;
  extract-module is invisible to the `--no-renames` diff and re-mutated as
  net-new), and the CatchAcross JOIN closed the cross-package gap at the
  oracle (fail-closed, no phantom catch). The closing gate is green: a real
  renamed anchor drives end-to-end to `surface.LostViaRename`, NOT the false
  operator-free token, and the rendered card names the rename and
  `NotContains "no mutable operator"`. A renamed file no longer lies on the
  surface. RESIDUAL: the rename-cliff is similarity-threshold based — a
  heavily-edited rename degrades to Outdated→AnchorEdited (still honest, not
  a phantom, but coarser); council #11.5 fast-follow. Refactor is an
  honest-at-surface task-type for the detected-rename case.

## Clash H — Trust Ledger: power-fantasy or self-assessment dread?

- **Game design:** only works if framed _outward_ (scouting report on
  agents) and cashed out as _promotion_ — never an inward calibration
  mirror.
- **Tension:** it's a _framing_ bet; the same data can read either way.
- **Experiment:** A/B the inward vs outward framing with real users. Does
  outward actually feel like a power-fantasy, or do users see through it to
  "this is grading me"?
- **Verdict (post-build):** _TBD_

## Clash I — Time-travel/forking power vs Board calm

- **UX:** powerful, falls free from the event spine — but a past-comment
  forking the timeline risks "what's current?" confusion.
- **Provisional resolution:** read-only sepia history; past-comment = named
  alt branch as a second Board card (A/B); deliberate pick-winner.
- **Tension:** does the A/B card add enough value to justify complicating
  the calm Board?
- **Experiment:** ship it behind the Ledger plugin; measure whether anyone
  uses retroactive forking, and whether it confuses "current."
- **Verdict (post-build):** _TBD_
