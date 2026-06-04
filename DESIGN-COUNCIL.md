# The agntpr Design Council

> A living record of the expert panel that shaped `VISION.md` and
> `DESIGN.md` — who they are, what they argued, and **where they still
> disagree**. Structured to be *reconvened*: most clashes below are not
> settled by argument, they're settled by **evidence from a built,
> tested slice.** Come back here after each validating slice and fill in
> the verdicts.

## How to use this document

1. **Before building** — read the open clashes (§3). Each names the
   experiment that would settle it. Let that shape what the slice
   measures.
2. **After building/testing a slice** — find the clashes it touched,
   fill in the `Verdict (post-build)` field with what actually happened,
   and mark the clash `RESOLVED` / `STILL OPEN` / `NEW QUESTION`.
3. **To re-summon a panelist** — the persona seeds (§1) are written so a
   future session can re-instantiate the same voice via the Agent tool.
   Agent IDs from the original rounds are session-scoped and will not
   survive; use the seeds, not the IDs.
4. **To run a new round** — use the round template (§5).

Session-scoped agent IDs from rounds 1–2 (likely dead now, kept for
provenance only): UX `a985fda4…`, Game design `af9d2f4c…`, Systems
`a494dd62…`, TDD `afcf847e…`, CI/CD `a5b74ebb…`, Refactoring `a172b669…`.

---

## 1. The panelists (persona cards)

Each card: their lens, north star, what they reliably push for, their
signature bold swing, and a **re-summon seed** (the essence of the prompt
that produces this voice).

### 🎨 The UX Designer

- **Lens:** product UI/UX; Linear / Vercel / Raycast sensibility.
- **North star:** *calm, power-user-first, keyboard-native.* A meter you
  can't act on right now is noise.
- **Reliably pushes for:** fewer surfaces, information gated to the
  moment it's actionable, motion that reports a real state change (never
  to fill time), density without clutter.
- **Bold swings:** Time-travel review (bidirectional Ledger scrubber);
  the Disagreement Replay (calibration you *feel*, not read).
- **Re-summon seed:** *"World-class product designer. Constraint: pure-Go
  Via framework over SSE, no SPA, Monaco only as a plugin island.
  Calm control-room aesthetic, keyboard-native. Allergic to gauges that
  induce guilt or can't be acted on. Render mechanics without building a
  cockpit of meters."*

### 🎮 The Game Designer

- **Lens:** management/tycoon games; Factorio / Frostpunk / Two Point
  Hospital / Slay the Spire.
- **North star:** *the work already is a game; make it feel like one
  without lying.* Honest hooks, not dark patterns. Queue-zero is a win
  you're allowed to walk away from.
- **Reliably pushes for:** killing dead-air, an ethical "just one more,"
  framing that flatters the player's competence, a real mastery curve,
  beats that punctuate.
- **Bold swings:** The Trust Ledger (calibrated delegation is the game);
  Delegation Tiers (Ascension — opt-in shrinking safety net).
- **Re-summon seed:** *"World-class tycoon/management game designer.
  agntpr is an HONEST game (mechanics must map to real dynamics, no fake
  XP/confetti, no engagement dark patterns). Obsessed with moment-to-
  moment feel, pacing across 30s/30m/session loops, and whether a
  mechanic flatters or grades the player."*

### ⚙️ The Systems / Economy Designer

- **Lens:** game systems & economies; Factorio logistics / RimWorld
  incident economy / Into the Breach perfect-information.
- **North star:** *one scarce resource, one conversion, one loop that
  punishes the obvious cheat.* Meters aren't an economy until they
  trade. The system defines the units; the user only spends.
- **Reliably pushes for:** collapsing meters into roles, red-teaming for
  degenerate strategies, tying every reward to a logged downstream fact.
- **Bold swings:** The Focus meter (attention as the central spent
  resource); the Shadow Review (spend tokens to audit untested trust).
- **Re-summon seed:** *"World-class game systems/economy designer.
  Hostile to 'meter soup' — insists on stocks vs rates vs scores. Always
  asks: what's the degenerate strategy, and what logged fact redeems each
  point? Every exploit is an attempt to control the denominator."*

### 🧪 The Pragmatic TDD Expert

- **Lens:** test-driven development, Kent-Beck-pragmatic (not dogma).
- **North star:** *tests must constrain behavior, not just exist and
  pass.* Ceremony decoupled from constraint is theater.
- **Reliably pushes for:** an independent oracle (mutation testing),
  test-list-as-contract at the plan gate, reviewing tests before code,
  distinguishing "RED for the right reason" from a birth-cry.
- **Bold swing:** Mutation-driven adversarial review (surviving mutants
  become `question:` threads — "green is a lie here").
- **Re-summon seed:** *"World-class pragmatic TDD practitioner.核心
  question: does this mechanic produce well-tested code or confidently-
  green test-theater? Knows RED→GREEN proves sequence not constraint, and
  that mutation score is the only non-gameable signal. Knows when a test
  earns its keep and when it's ceremony."*

### �l The CI/CD & Delivery Expert

- **Lens:** continuous delivery, DORA, trunk-based dev, merge queues,
  progressive delivery, flaky-test management.
- **North star:** *"Landed" ≠ done; only "Merged" through real CI is.*
  Don't reinvent integration — feed a merge queue.
- **Reliably pushes for:** an explicit integration point, two-tier checks
  (container-advisory vs real-pipeline), flaky quarantine before scoring,
  honest DORA metrics, controlling superlinear integration cost.
- **Bold swing:** Speculative integration preview (background-rebase onto
  tip + CI before you approve).
- **Re-summon seed:** *"World-class CI/CD & delivery expert. Sees the
  seam where the design hands N changesets to downstream CI and walks
  away. Worries about stale-base collisions across the fleet, scoring
  humans on noisy/flaky downstream truth, and O(N²) merge-queue cost."*

### 🔧 The Refactoring Expert

- **Lens:** large-scale refactoring; Fowler / Feathers "Working
  Effectively with Legacy Code."
- **North star:** *behavior-preserving change keeps codebases alive — and
  it's the safest thing to skim when tests are trustworthy.*
- **Reliably pushes for:** refactor as a first-class task-type with a
  proof (tests unchanged & green), reviewing the invariant not the hunks,
  transformation-anchored threads, cross-session collision safety.
- **Bold swing:** Characterization Gate + mechanical-equivalence replay
  (review the proof of invariance, time-travel to where it broke).
- **Re-summon seed:** *"World-class refactoring expert (Feathers/Fowler).
  Tests whether the diff-first/anchored/fan-out model supports healthy
  refactoring or punishes it as churn. Knows a 40-file rename breaks
  line-anchoring and that a clean refactor with green unchanged tests is
  the safest possible skim."*

---

## 2. Contribution ledger

Where each panelist's ideas landed in the docs.

| Panelist     | Round 1 → adopted in        | Round 2 → adopted in              |
|--------------|------------------------------|------------------------------------|
| UX           | leverage-not-cost, brief rail (V §12.2/§12.9) | time-separated economy, outward Trust render, time-travel disambiguation, Disagreement Replay (V §13.1/§13.2/§13.10) |
| Game design  | Prep Bench, session arc, earned concurrency (V §12.1/§12.6/§12.7) | outward Trust Ledger, Delegation Tiers, seeded+pre-flight Bench (V §13.2/§13.3/§13.4) |
| Systems      | leverage formula, net-quality scoring, Focus, exploit framing (V §12.3/§12.5) | unified economy, exploit patches, CI-truth loop, Shadow Review (V §13.1/§13.5/§13.10; D §29.3) |
| TDD          | — (joined R2)                | mutation oracle, test-list contract, confirmed-catch redefinition (V §13.6; D §29.4) |
| CI/CD        | — (joined R2)                | merge queue, two-tier checks, "Merged not Landed", flaky quarantine, DORA (V §13.8; D §29.1/§29.2/§29.5/§29.9) |
| Refactoring  | — (joined R2)                | refactor task-type, Invariant View, fleet collision guard (V §13.7/§13.9; D §29.6/§29.7) |

---

## 3. The open clashes (the heart — resume these after builds)

Each clash: the disagreement, the positions, the current (provisional)
resolution, **the experiment that settles it**, and a blank verdict.

### Clash A — Is *any* per-card cost signal acceptable, or pure guilt?

- **UX:** cut all per-card economic meters; cost is guilt. Rank by
  leverage; reveal Focus only retrospectively at close-out.
- **Systems:** a per-card *burn-rate / thrash signature* is diagnostic,
  not guilt — it's how you spot a runaway agent.
- **Provisional resolution:** no live drain meter; per-card cost appears
  only as a *thrashing diagnostic* and only at decision moments.
- **Experiment:** build the Board with (a) thrash-diagnostic on, (b) off.
  Do users catch runaways without it? Does its presence read as nagging?
- **Verdict (post-build, round 10 — first runnable experiment):** still open on
  the framing question, but no longer a thought experiment — the served card
  (R10 #10 wire) makes silent-vs-badge concretely testable against pixels:
  zero-survivor renders an affirmative "Tested — ship it", distinct from blind
  "no-oracle-signal", and a live HTTP/SSE server lets a human watch whether the
  most-common calm-win screen reassures or nags. The two-quiet-meanings
  collision is closed at the pipe→card presenter. Meters are deliberately OFF
  the first screen. Settles empirically the moment a human opens the wired card.

### Clash B — Can agent self-confidence route the reviewer's attention?

- **UX + TDD:** No — a confidently-wrong agent flags nothing; don't build
  the attention spine on the least trustworthy signal.
- **Game + Systems:** self-flags are the highest-leverage attention hint.
- **Provisional resolution:** self-flags are a *hint + a calibration
  input measured against outcomes*; the **independent** spine is mutation
  (§13.6). Flag-density-vs-review-time mismatch = the stamp detector.
- **Experiment:** on a real changeset corpus, measure correlation between
  agent self-flag density and mutation-discovered weak spots. If low,
  demote self-flags to decoration.
- **Verdict (post-build, slice 2025 — mutation oracle):** PARTIAL. The
  *independent* oracle the resolution depends on now exists and works
  (`internal/mutation`): on a weak test it surfaces the surviving mutant
  as a finding; on a strong test it stays silent. So self-confidence no
  longer *has* to be the spine — there's a real signal to lean on. The
  self-flag↔mutation *correlation* experiment still needs a corpus.
  STILL OPEN, but the baseline (a trustworthy independent signal) is
  established.

### Clash C — Is it fair to score a human on downstream CI truth?

- **Systems:** regression hits feed Ship Quality and Trust calibration.
- **CI/CD:** downstream red is often flaky/environmental/mis-attributed;
  scoring a human on it is unfair and will kill trust in the Ledger.
- **Provisional resolution:** flaky quarantine + settle-gating +
  eventually-consistent back-application + depth-scaled penalty;
  **change-fail-rate replaces raw regression-hit** as the headline.
- **Experiment:** run against a repo with real flake. Does the score feel
  fair to the human? How often does quarantine misfire?
- **Verdict (post-build, round 8):** gains both settling fixtures and a
  refined ordering — integrate-on-tip (roadmap #5) precedes catch PRICING,
  not catch MINTING. Fixture (A) trunk-moved-underneath decides whether the
  survivor-set transition survives a rebase (if not, #5 jumps ahead of
  pricing); fixture (B) clean-rebase-but-checks-red proves a green
  pre-integration catch can be a red post-integration regression. STILL OPEN.
- **Verdict (post-build, round 11 — the seam itself elevated to a design rule):**
  the downstream-truth question is unchanged (integrate-on-tip still the experiment,
  #12), but the SEAM that will carry the {clean|conflict|checks-red} Land verdict was
  shown this round to be actively dangerous: `CycleResult.Outcome` lossily collapsed
  THREE distinguishable reanchor states (genuine-no-operator / edited / renamed) into
  one `NoOracleSignal` token, and on a rename that produced a confident falsehood at
  the surface (card.go:66-68). ELEVATED to a binding design rule: every verdict
  dimension — this round's `Reason`, #12's `Land` (already stubbed) — is an orthogonal
  TYPED field on CycleResult, NEVER a new meaning bolted onto an existing Outcome
  token. #12 MUST reuse #11's widened-seam pattern. STILL OPEN on the scoring
  question; the seam discipline is now decided.
- **Verdict (post-build, round 12 — resolution path RATIFIED):** #12 integrate-on-tip
  is ratified (5/6) as both the experiment and the fix. A typed orthogonal `Land`
  verdict {LandClean | LandConflict | LandChecksRed}, computed by rebasing fixRev onto
  trunk tip and re-running checks on the REBASED tree (reusing #11's seam pattern; the
  catch.Outcome/ledger token stays byte-identical), replaces the dead `Unintegrated`
  const so the catch is minted against the tree that actually integrates. The
  load-bearing closing RED is `clean-rebase-but-checks-red` (a green pre-integration
  catch going red post-integration), with a disjoint-trunk `LandClean` degenerate guard
  proving the verdict CONSTRAINS. Flips to RESOLVED-IN-CODE on the green #12 build;
  catch PRICING (#17) remains gated behind it. STILL OPEN until that code ships.
- **Verdict (post-build, round 12 #12 — RESOLVED-IN-CODE):** `internal/pipe.integrateOnTip`
  rebases fixRev onto a real `tipRev` in a throwaway worktree and runs the checks on
  the INTEGRATED tree via controlled exec; `CycleResult.Land` is now a typed
  {LandClean | LandConflict | LandChecksRed}, orthogonal to catch.Outcome (mint token
  byte-identical). The load-bearing RED is green:
  `TestRunCatchCycle_cleanRebaseButChecksRedYieldsChecksRed` proves a green
  pre-integration catch is a red post-integration regression; the disjoint-trunk
  `landsCleanOnNonConflictingTip` degenerate guard proves the verdict CONSTRAINS;
  `TestResolve_threadsTheTipSoADivergentTrunkReportsLandConflict` proves the seam
  threads the real tip (a tip-ignoring stub would land clean). The card shows the
  Land verdict as its own honest row over SSE (`surface.RenderLand`, asserted
  end-to-end in the live wire test). The catch is now minted against the tree that
  actually integrates — "Landed ≠ Merged" is a computed verdict, not a label.
  RESOLVED-IN-CODE on the integration seam. What remains for the SCORING question
  (is it fair to score a human on this?) is the flaky-quarantine / change-fail-rate
  layer (§13.8) + catch PRICING (#17), still gated on the #15 K-concurrent benchmark.
- **Verdict (post-build, round 11 #11 — the seam discipline DEMONSTRATED in code):**
  the rule is no longer prose: `pipe.Reason` ships as a typed field on
  `CycleResult` ORTHOGONAL to `catch.Outcome` (the economy/ledger token is byte-for-
  byte unchanged — `catch.Detect` and `ledger.ShouldRecord` untouched). `CatchAcross`
  now returns `(Outcome, Reason, error)`; the three quiet causes are three distinct
  values, asserted distinct end-to-end. This is the concrete template #12's `Land`
  ({clean|conflict|checks-red}, already stubbed) must follow — a new dimension is a
  new field, never a new meaning on `NoOracleSignal`. STILL OPEN on downstream-truth
  scoring (integrate-on-tip #12); the seam pattern is now load-bearing code.

### Clash D — Does the fleet (1:N) actually scale, or cap at N≈2–3?

- **Vision thesis:** review is 1:N; reviewing five Claudes is just a
  queue.
- **Game design:** review is a deep context-load; the context-switch tax
  may make N>3 *slower and more error-prone* — "The Board" above that is
  theater.
- **Provisional resolution:** earned concurrency (§12.7) gates N to
  measured calibration; start serial.
- **Experiment:** measure review quality (mutation-caught defects,
  rework) vs. number of concurrent in-flight reviews. Find the real
  ceiling. Is it per-user-trainable?
- **Verdict (post-build):** _TBD_

### Clash E — Does the Prep Bench kill dead-air, or create a second job?

- **Game design (with itself):** unseeded, the Bench relocates the void
  to onboarding; unbounded, it's a parallel obligation that kills rest.
- **Provisional resolution:** Bench is seeded (never empty), lightweight,
  interruptible, and pre-flights the *incoming* diff; onboarding uses a
  scripted prep track.
- **Experiment:** instrument idle time and self-reported load during
  agent compute. Do users idle (bench failed) or feel doubly-taxed (bench
  overshot)?
- **Verdict (post-build):** _TBD_

### Clash F — "Confirmed catch": test-flipped vs mutant-killed?

- **Systems (R1):** catch = a test flipped, weighted by hiddenness.
- **TDD (R2):** that's farmable — the agent authors the flipping test;
  require a **mutant that survived-before and is killed-after** (an oracle
  the agent didn't write).
- **Provisional resolution:** adopted TDD's definition (§29.3/§29.4).
- **Open cost question:** is diff-scoped mutation fast/cheap enough to run
  every settle without wrecking the loop's latency or token budget?
- **Experiment:** measure mutation latency & cost on real diffs in the
  settle step. If too slow, what's the fallback oracle?
- **Verdict (post-build, slice 2025 — mutation oracle):** MOSTLY
  RESOLVED on feasibility. Diff-scoped mutation is buildable with the
  Go stdlib alone (`go/ast`+`go/parser`+`os/exec`), no external deps,
  and the mutant-killed definition works end-to-end. Cost shape
  confirmed: **one test-run per mutant per changed-line operator site**
  — bounded by diff size, exactly as predicted. A 1-operator file ran in
  ~0.03–0.09s incl. compile.
- **Verdict (post-build, round 4 — latency benchmark):** RESOLVED for
  serial viability. A 30-site fixture (`testdata/bench_many`,
  `BenchmarkRunManySites`): **cold 3.24s (~108 ms/mutant), warm 0.91s
  (~30 ms/mutant)** — warm is the relevant figure since the settle loop
  keeps the build cache hot. A realistic 10–40-site diff ≈ 0.3–1.2s
  warm. Mutants are independent → trivially parallelizable (run K
  concurrently → ÷K) if needed. Conclusion: cheap enough to run every
  settle for normal diffs; add mutant-level parallelism only for
  pathologically large diffs. No fallback oracle needed.
- **Verdict (post-build, round 5 — re-opened on identity):** DOWNGRADED to
  PARTIALLY RESOLVED. R3-4 resolved only *latency/feasibility* (that
  stands). The **catch-identity** half is untouched and has zero code:
  the survivor-set non-empty→empty *definition* (vs the incoherent "same
  mutant killed" when the fix edits the anchored line) is the live gate
  for the entire trust economy (RISKS §29.3, HIGH), and it is coupled to
  the still-unbuilt from-base re-anchoring (§28) — "the same line's
  survivor set across two revisions" is undefined without it. This is the
  council's ranked **#1 next build** (two-lens TDD+Systems convergence).
  STILL OPEN on identity.
- **Verdict (post-build, round 7 — CONVERGED on the build, open on identity):**
  The council converged (rounds 5→7, 5/6 lenses, render-camp conceding
  economy-first) that #1 is this catch oracle, now specified as a
  **tri-state** primitive `{catch | no-catch | no-oracle-signal | partial-catch}`
  with a mandatory **survivor-set identity key** = the anchored line's
  *current operator inventory per revision* (+ an explicit rule for
  inventory change under the fix), so that a no-op churn revision and a
  fix-that-edits-the-line are distinguishable rather than the same failure.
  The CI/CD ordering challenge (catch is minted on pre-integration
  coordinates; anchor-survives-rebase is a 2nd hidden dependency beyond
  §28) is **deferred to empirical resolution**: build the
  fix-edits-anchored-line fixture, then a trunk-moved-underneath variant;
  if the transition doesn't survive a rebase, integrate-on-tip precedes
  catch pricing. Identity half STILL OPEN until that code exists.
- **Verdict (post-build, round 8 — build-ready & type-committed):** still
  PARTIALLY RESOLVED on paper, but the #1 unit is now BUILD-READY: a typed
  `Catch{Anchor, BeforeInventory, AfterInventory, Outcome}` with
  `Outcome ∈ {Catch | NoCatch | NoOracleSignal | PartialCatch}`. The
  survivor-set IDENTITY KEY (denominator = the line's current operator
  inventory per revision) becomes a REAL Go type owned by one pure
  function, no longer prose; the explicit inventory-change rule (fix edits
  L + changes inventory → ill-typed → NoCatch) is the load-bearing RED that
  proves "same mutant killed" incoherent. Flips to RESOLVED-IN-CODE on the
  green #1 build (fix-edits-anchored-line RED + degenerate suite green).
- **Verdict (post-build, round 9 — unit SHIPPED green; re-scoped open on the
  loop):** RESOLVED-IN-CODE on the UNIT — `internal/catch` ships `Detect` with
  the identity key enforced by one pure function and the refusal arms TESTED
  (constrains, not just fires). RE-SCOPED OPEN on the LOOP: `catch.Detect` has
  ZERO prod callers, asserted only against hand-built `LineState` literals; it
  has never adjudicated two real revisions through settle→diff→mutation. Flips
  fully on the green #3 pipe (a real Catch minted from two real settles +
  agent-edits-anchored-line → NoCatch end-to-end through the real reanchor path).
  Logged v1 risk: set-not-multiset keying under-credits killing one of two
  same-operator survivors on a line.
- **Verdict (post-build, #3 pipe — the loop now RESOLVED-IN-CODE):** the §17
  pipe (`internal/pipe.RunCatchCycle`) mints the catch from TWO REAL revisions
  end-to-end (settle→worktree→mutation×2→reanchor→CatchAcross→Detect): an
  agent strengthening the test only mints a real Catch; editing the anchored
  line yields NoOracleSignal end-to-end. `catch.Detect` now has a prod caller
  and is exercised against real settles, not literals. Clash F is
  RESOLVED-IN-CODE on BOTH the unit and the loop. What remains is ECONOMY, not
  oracle: the Catch is computed but not yet PERSISTED as a ledger record
  (capture-at-mint, roadmap #7), and is minted on pre-integration coordinates
  (integrate-on-tip, #5). New cross-layer finding logged: the reanchor gate
  (edited line → Outdated → NoOracleSignal) fires BEFORE Detect's
  inventory-change NoCatch rule, so end-to-end an edited anchored line reads as
  NoOracleSignal, not NoCatch — both safe (no phantom), but a consumer can't
  distinguish "line edited" from "operator-free" (see also the NoOracleSignal
  overload note for the next round).

### Clash G — One unified review model, or a refactor fork?

- **Original design:** one diff-first/anchored/fan-out model for all work.
- **Refactoring:** that model is hostile to refactors; needs a separate
  task-type, the Invariant View, transformation anchors, inverted stamp
  penalty.
- **Provisional resolution:** refactor is a first-class task-type
  (§29.6) — accepts the added surface area.
- **Experiment:** run a real 30+ file rename and an extract-module through
  the Invariant View. Is it genuinely reviewable? Does the
  behavior-preservation proof hold and feel trustworthy?
- **Verdict (post-build, round 8):** still TBD, but the settling experiment
  is re-ratified CONCURRENT-with-#1 and re-scoped as the ACCEPTANCE BAR for
  #1's re-anchoring sub-brick (same build wave, not after). RED baselines:
  orphanedThreadCount>0 on a 30+-file rename; survivor-set ill-typed across
  rename (lost-via-rename != Catch); extract-module re-mutated as net-new
  (invisibility). STILL OPEN.
- **Verdict (post-build, round 9 — carnage baselines RESOLVED in code):**
  `internal/refactor/trace_test.go` is executable evidence — a 40-file rename
  orphans all 40 threads; the neutral rename asserts LostViaRename != Catch
  (the oracle refuses a phantom mint); extract-module is invisible to the
  `--no-renames` diff and re-mutated as net-new. RESOLVED on the carnage
  question. Residual (not a new clash): the safety property "lost via rename →
  NoOracleSignal, never a phantom Catch" is asserted across two separate test
  packages with no prod function joining them; the reanchor→catch JOIN
  (CatchAcross, roadmap #3 prereq) closes this by construction.
- **Verdict (post-build, round 11 — RE-OPENED at the surface layer):** the
  carnage baselines stand and the JOIN (CatchAcross) closed the cross-package gap
  AT THE ORACLE — fail-closed, no phantom catch, ledger-honest. But the served
  card (#10) gave the refactor failure a surface to be observed on, and it is
  DISHONEST there: a renamed file resolves to `data-state="no-oracle-signal"` with
  detail "This line has no mutable operator" (card.go:66-68) — a confident
  falsehood about WHY the oracle is silent. Resolved-at-oracle ≠ honest-at-surface.
  Refactor cannot be an honest first-class task-type until the rename renders a true
  terminal. CLOSING GATE: Round-11 #11's httptest SSE renamed-file fixture asserting
  the card names the rename and does NOT claim "no mutable operator". RE-OPENED
  (sub-target-level; not a new clash).
- **Verdict (post-build, round 11 #11 — surface honesty RESOLVED-IN-CODE):** the
  closing gate is GREEN. `TestResolve_rendersLostViaRenameVerdictForARenamedAnchor`
  drives a REAL renamed anchor end-to-end through the seam (settle→reanchor→
  CatchAcross→PresentVerdict) and asserts the verdict reaches the card as
  `surface.LostViaRename`, NOT the false operator-free token;
  `TestReviewCard_rendersLostViaRenameWithoutClaimingNoOperator` asserts the
  rendered card names the rename and `NotContains "no mutable operator"`. A renamed
  file no longer lies on the surface. RESIDUAL (not a new clash): the rename-cliff
  itself is similarity-threshold based — a heavily-edited rename degrades to
  Outdated→AnchorEdited (still honest, not a phantom catch, but coarser); council
  #11.5 fast-follow. Refactor is now an honest-at-surface task-type for the
  detected-rename case.

### Clash H — Trust Ledger: power-fantasy or self-assessment dread?

- **Game design:** only works if framed *outward* (scouting report on
  agents) and cashed out as *promotion* — never an inward calibration
  mirror.
- **Tension:** it's a *framing* bet; the same data can read either way.
- **Experiment:** A/B the inward vs outward framing with real users. Does
  the outward framing actually feel like a power-fantasy, or do users see
  through it to "this is grading me"?
- **Verdict (post-build):** _TBD_

### Clash I — Time-travel/forking power vs Board calm

- **UX:** powerful, falls free from the event spine — but a past-comment
  forking the timeline risks "what's current?" confusion.
- **Provisional resolution:** read-only sepia history; past-comment =
  named alt branch as a second Board card (A/B); deliberate pick-winner.
- **Tension:** does the A/B card add enough value to justify complicating
  the calm Board?
- **Experiment:** ship it behind the Ledger plugin; measure whether
  anyone uses retroactive forking, and whether it confuses "current."
- **Verdict (post-build):** _TBD_

---

## 4. The bold swings scoreboard

The signature bets, and their status. Fill `Validated?` after builds.

| Swing                              | By        | Status        | Validated? |
|------------------------------------|-----------|---------------|------------|
| Mutation-driven adversarial review | TDD       | high conviction | **validated** + CONFIRMED-CATCH now minted END-TO-END through the §17 pipe (`internal/pipe.RunCatchCycle`, #3): two real settles → worktree mutation×2 → reanchor → CatchAcross → real Catch; edited-anchor → NoOracleSignal — AND rendered as a distinct live state (`internal/surface.ReviewCard`, #4). Next (R10 #10 wire): a runnable HTTP/SSE binary so a human WATCHES it resolve + persist the CatchRecord (capture-at-mint). Pending pricing: integrate-on-tip (#12) |
| Trust Ledger (calibrated delegation)| Game     | spine, framing-risk (Clash H) | _TBD_ |
| Merge-queue-as-integrator          | CI/CD     | low-risk, standard practice | **R12 #12: integrate-on-tip SHIPPED** — `pipe.integrateOnTip` rebases the fix onto a real tip + runs checks on the integrated tree → typed Land {clean\|conflict\|checks-red}, one serialized lane (no N-concurrent-rebase fan-out). Clash C resolved-in-code; the catch is minted against the tree that integrates. Remaining: the single-lane QUEUE over K branches (throughput-to-zero) + the #15 K-concurrent benchmark before PRICING |
| Focus as central resource          | Systems   | adopted, render-risk (Clash A) | _TBD_ |
| Speculative integration preview    | CI/CD     | high value, infra cost | _TBD_ |
| Characterization Gate + replay     | Refactor  | high value, scoped to refactors | _TBD_ → roadmap #2 (concurrent w/ #1): adversarial refactor trace as RED baselines (rename_40 / rename_neutral_move / extract_module); settles Clash G, de-risks #1's re-anchor sub-brick. **R11 #11: Clash G's surface-honesty half RESOLVED-IN-CODE** — a renamed/edited anchor now renders a distinct, TRUE terminal card (`surface.LostViaRename`/`AnchorEdited`) instead of the false "no mutable operator"; the refactor task-type is honest-at-surface for the detected-rename case (residual: rename-cliff coarsening → #11.5) |
| Time-travel review                 | UX        | distinctive, value-unproven (Clash I) | _TBD_ |
| Delegation Tiers (Ascension)       | Game      | late-game depth, premature | _TBD_ |
| Shadow Review (anti-survivorship)  | Systems   | elegant, token-cost unclear | _TBD_ |
| Disagreement Replay (coaching)     | UX        | cheap, depends on dwell-tracking | _TBD_ |

---

## 5. Reconvene template (for future rounds)

Copy this block per new round and fill it in.

```markdown
## Round N — <focus> — <date>

Trigger: <what prompted it — e.g. "after building the mutation-thread slice">
Panelists present: <which personas, + any new lens added>
New evidence on the table: <build/test results that inform the debate>

Per panelist:
- <persona>: <their take this round, 1-3 lines>

Clashes touched: <A–I + any new>
Verdicts updated: <which §3 clashes moved, and to what>
New clashes opened: <…>
Decisions: <what changed in VISION/DESIGN as a result>
```

### Round 3 — first build evidence (mutation oracle slice)

Trigger: built the keystone "mutation-as-question-thread" slice
(`internal/mutation`, validating slice #2).
Panelists present: none re-convened yet — this logs the *evidence* the
next round will argue over.
New evidence on the table:

- A diff-scoped mutation oracle exists, Go-stdlib-only, TDD-built
  (RYGBA per unit; audit caught a real restore-error-swallowing bug).
- The validating experiment passes: a weak/tautological test (`IsAdult`
  checked only at 25) lets the `>=`→`>` mutant SURVIVE → exactly one
  finding on the right line; a strong test (pins 17/18) KILLS it → zero
  findings. **Test-theater is made to visibly fail, in miniature.**

Per panelist (the relevant ones, to argue next round):

- TDD: core thesis vindicated at the unit level — mutation is the
  independent oracle, and it discriminates weak from strong tests. Wants
  the next slice to redefine "confirmed catch" against survived→killed
  and to render survivors as real `question:` threads.
- Systems: feasibility confirmed; flags the cost model (one test-run per
  mutant) for the settle-loop budget — see Clash F.

Clashes touched: B (PARTIAL — independent signal now exists),
F (MOSTLY RESOLVED on feasibility; latency-at-scale still open).
Verdicts updated: B, F (see §3).
New clashes opened: none yet. Likely next: mutation *latency budget* in
the settle loop, and the survived-mutant → `question:`-thread UX.
Decisions: no VISION/DESIGN text changed; this is evidence, not redesign.

### Round 4 — latency + the question-thread artifact

Trigger: closing the two threads round 3 left open.
New evidence:

- **Latency benchmark** (`BenchmarkRunManySites`, 30-site fixture):
  cold 3.24s (~108 ms/mutant), warm 0.91s (~30 ms/mutant). Settle-loop
  viable; parallelizable if needed. → **Clash F resolved** for serial
  viability (see §3).
- **Question-thread artifact** (`internal/review`): a surviving mutant
  now converts to an open `question:` thread authored by `agntpr`,
  anchored to the line, rendering as a Conventional Comment
  ("question: …"). The full chain mutation→finding→thread→render is
  proven at the unit level. Still a data layer — no UI, no harness
  wiring yet.

Clashes touched: F (resolved on feasibility). Verdicts updated: F, and
the mutation swing in §4.
New clashes opened: none. Next likely: rendering threads in the actual
review surface (Via), and wiring the oracle to run at settle against a
real diff (needs the §17 pipe).
Decisions: no VISION/DESIGN redesign; evidence only.

## Round 5 — de-risking the right thesis: first economy primitive vs first rendered surface — 2026-06-04

Trigger: first real re-convening since Round 2 (Rounds 3-4 were evidence-only). Prompted by the two RISKS.md meta-findings — (1) "build order de-risks the WRONG thesis" and (2) "the design's own acceptance bar would catch none of this" — against a build state of 6 green backend packages (mutation/review/settle/diff/translate/orchestrator), NO end-to-end pipe, NO UI, NO trust-economy code.

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD, Refactoring). No new lens.

New evidence on the table:
- Mutation oracle is the keystone, now parallel (maxWorkers=8, copy-per-worker), ~30ms/mutant warm, survivors render as question: threads. Operator set complete (19 ops). Settle has secret-scan + artifact-surfacing + no-edit guard.
- Verified-by-reading-code this round: review.Thread.Render() returns a string, no HTTP/Via/SSE/template anywhere; orchestrator.go:37 diffs against a caller-supplied IMMUTABLE baseRev and never reconciles with trunk tip; diff.go's TestRenameIsDeleteAddRegardlessOfConfig proves the rename cliff is live; QuestionThreadsFromMutations anchors to mutation lines a refactor moves wholesale.

Per panelist:
- UX: build is mono-axial — 5 rounds, 6 packages, zero rendered surface. Every clash she owns (A Focus-guilt, D scaling, E Bench, H Ledger framing, I time-travel) is gated on a screen that doesn't exist; you can't ask "does this meter induce guilt" of a Go struct. Build §17's pipe WITH a real Via/SSE review surface (one card, the question: thread on its line, one comment->revision round-trip, no meters). New concern: the review surface has NO defined empty/zero state — the mutation-silent case (0 findings, MutantsConsidered>0 = "tested, ship it") is the MOST COMMON screen and would read as "broken/nothing happened."
- Game design: rigorous but grading the wrong difficulty — 100% of evidence lives in the mechanical pipe nobody doubted. You can't FEEL anything: no loop, no second agent, no queue to drain. Build §6 slice #3 — minimal two-agent Board, queue-to-zero, instrumented day-one for idle time and per-review dwell, no meters yet. Clash D is the load-bearing feel question; find the real N ceiling by making a human switch between two live cards and measuring rework. New concern: the oracle's INTERRUPTION RATE per session (how often it actually produces an acted-on thread) is an untuned feel-knob — fires-often = nag (Clash A relocated to the oracle), fires-rarely = dead weight.
- Systems: sound engineering, miscalibrated economy — five rounds polished a SIGNAL GENERATOR; the economy (Focus/Trust/Ship-Quality as stocks that TRADE against logged facts) has zero code and zero adversarial test. Build the confirmed-catch as the FIRST economy primitive: a typed append-only Catch{line, survivorSetBefore:nonempty, survivorSetAfter:empty, revID, author} per Clash F's survivor-set definition, with a degenerate-strategy test suite. New concern: the parallel oracle's 30ms/mutant is a SINGLE-TENANT number — under an N-agent Board, K concurrent settles spawn up to 8N concurrent full-suite runs contending for the same Treasury+CPU; the economy would price catches against latency that degrades superlinearly in the exact regime earned-concurrency unlocks. Unmeasured.
- Pragmatic TDD: RYGBA discipline is real (caught restore-swallow + vacuous-pass bugs) but pointed at the easy thesis — Rounds 3-5 hardened a SINGLE-REVISION oracle while the load-bearing thing (does a mutation survive across a fix to become a CONFIRMED CATCH) has zero code. "A beautiful odometer and no trip counter." Build the confirmed-catch oracle as a two-revision differential (survivor-SET non-empty->empty on the same anchored line), TDD'd against the fix-EDITS-the-anchored-line fixture. New concern: this oracle inherits from-base re-anchoring (RISKS §28) as a HARD dependency — you cannot define "the same line's survivor set across two revisions" without solving re-anchoring; the two risks are coupled and neither slice has touched it.
- CI/CD & Delivery: everything built sits UPSTREAM of the integration seam — settle mints a revision, orchestrator diffs a FIXED baseRev, nothing rebases or re-reads trunk tip. The whole downstream-truth surface (Merged!=Landed, stale-base collisions, fan-out safety) has zero code; §27's happy-path trace passes precisely because it never integrates anything. Build the integrate-on-tip brick (rebase session branch onto trunk tip, run checks on the INTEGRATED tree, tri-state clean/conflict/checks-red), TDD'd from the disjoint-file cross-symbol break (RISKS line 117). New concern: orchestrator.go takes baseRev as an immutable caller string and never reconciles with tip — the stale-base assumption is now baked into CODE with no TODO.
- Refactoring: build is sound but scope-misselected — six P0 bricks all serve the one work-type the diff-first model already handles (small local operator-bearing diffs); zero code touches refactoring, the work-type the model is structurally hostile to (Clash G). Run a real 30+-file rename + extract-module through diff->mutation->thread as an ADVERSARIAL trace and assert on the carnage (orphaned threads from the rename cliff; mutation re-litigating untouched behavior). New concern: QuestionThreadsFromMutations treats behavior-preserving refactor churn as MAXIMUM-suspicion surface — the keystone oracle is actively miscalibrated for the refactor task-type (the inverse of the §29.6 stamp-penalty inversion, on the oracle side). Not previously captured.

Clashes touched: F (re-opened — its IDENTITY half is unresolved, only latency was resolved in R3-4), B (UX reframes the framing baseline as a RENDER question, not logic), D (Game + CI/CD both nominate it as load-bearing but propose opposite experiments — human-dwell vs integrated-build-RED), G (Refactor — now settleable against the current green tree), A/E/H/I (UX + Game: all gated on a rendered/instrumented loop that doesn't exist).

Verdicts updated:
- Clash F: downgraded from "MOSTLY RESOLVED" to PARTIALLY RESOLVED — latency/feasibility resolved (R3-4) stands, but the catch-IDENTITY definition (survivor-set transition vs "same mutant killed") has zero code and is the live gate for the entire trust economy (RISKS §29.3 HIGH). Coupled to the unbuilt from-base re-anchoring (§28).
- Clash D, G, A, E, H, I: remain TBD, but their settling experiments are now named as concrete adversarial traces / a rendered slice rather than arguments.

New clashes opened:
- Render-camp vs economy-camp on what the next gate IS: feel/scaling (UX+Game, needs a visible loop) vs economy-integrity (TDD+Systems, needs the logged primitive first). CI/CD + Refactor sit between: adversarial traces that need neither a human nor an economy.
- Three new CODE-level risks (none yet in RISKS.md as code observations): oracle latency under fleet contention (Systems); QuestionThreadsFromMutations miscalibration on behavior-preserving churn (Refactor); orchestrator immutable-baseRev stale-base gap (CI/CD).
- Two render-only concerns that only surface once a screen exists: the undefined empty/zero state (UX) and the untuned oracle interruption rate (Game).

Decisions:
1. NEXT BUILD (ranked #1, two-lens convergence TDD+Systems): the confirmed-catch oracle — a logged, append-only, two-revision survivor-set non-empty->empty Catch event on the anchored line, NEVER "same mutant killed", TDD'd against the three-case fixture (test-only fix / fix-edits-anchored-line / fix-adds-branch); the fix-edits-anchored-line case is the killer that proves "same mutant killed" incoherent. This is the first acceptance-suite entry per meta-finding 2.
2. PREREQUISITE sub-brick of #1: from-base re-anchoring (RISKS §28), with "lost via rename" surfaced as a distinct state.
3. Then the integrate-on-tip brick (CI/CD) and the adversarial refactor trace (Refactor) as the next two adversarial-suite entries — both costed against #1 next round.
4. The rendered §17 surface (UX) + two-agent instrumented loop (Game) are gated AFTER the catch primitive; the §17 slice MUST ship a designed empty/zero state.
5. Add the three new code-level risks to RISKS.md; benchmark oracle latency under K-concurrent-settle contention.
NO VISION/DESIGN text changed this round (the doc reconciliation pass for the 12 contradictions remains queued per RISKS sequencing step 5). Not converged — one more round to adjudicate render-camp vs economy-camp and rank the three adversarial bricks.

## Round 6 — adjudicating the next gate: economy-primitive-first vs surface/integration-first, and the catch's undesigned unhappy path — 2026-06-04

Trigger: reconvening after Round 5 named two camps (render: UX+Game vs economy: TDD+Systems) and three between-bricks (CI/CD integrate-on-tip, Refactor adversarial trace) but explicitly did NOT converge — "one more round to adjudicate render-camp vs economy-camp and rank the three adversarial bricks." Build state unchanged: 6 green backend packages, no pipe, no UI, no economy code (re-verified in code this round).

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD, Refactoring). No new lens.

New evidence on the table (verified by reading code this round, not argued):
- thread.go:27-28 `Render()` is `t.Tag + ": " + t.Body` — confirmed string concat; no http/template/SSE anywhere in internal/.
- orchestrator.go:37 `SettleTurn(..., baseRev, ...)` with line 49 `diff.Compute(ctx, repoDir, baseRev, res.SHA)` — baseRev is an immutable caller string, never reconciled with trunk tip; no TODO.
- runner.go:72 `const maxWorkers = 8`, copy-per-worker — so K concurrent settles → up to 8N concurrent full-suite processes.
- grep for Catch/Trust/Focus/Treasury/Ledger across internal/ returns ZERO typed events — meta-finding 1 confirmed in code.

Per panelist:
- UX: Dissents on Round-5's ordering (surface gated AFTER catch). The render surface needs ONLY the survivor-set state the CURRENT single-revision oracle already emits, so surface and two-revision catch are NOT a strict sequence — build the §17 Via/SSE surface (one card, one pinned question: thread, one comment->revision round-trip, no meters) against today's oracle. New concern: the council's #1 catch primitive has NO render for its most-likely outcome — the round-1 fix where the survivor-set is STILL non-empty (partial catch); the binary Catch{} schema has no event type for it, repeating §27 one layer down.
- Game design: Came in render-camp but will not be a yes-man to it — the catch IS the economy spine AND the game's first honest reward beat, so economy-first is correct. Narrow friction: mint the Catch WITH the self-flag bit recorded (cheap once the event is typed) so Clash B's self-flag-vs-mutation correlation can later run on REAL catches. New concern: nobody has asked what earning a catch FEELS like — a catch on the agent's 3rd self-corrected revision of a never-shipped trivial bug is participation-trophy XP; the schema logs the survivor-set transition but not the counterfactual "would this have shipped," so the reward is uncalibrated (flatter vs grade indistinguishable).
- Systems: Build the confirmed-catch as the FIRST economy primitive — typed append-only Catch{anchorLine, survivorSetBefore:nonempty, survivorSetAfter:empty, revID, author}, defined ONLY as the line's survivor-set transition, NEVER "same mutant killed." Ship it WITH a degenerate-strategy suite as the first acceptance entry (agent-authors-the-killing-test farming case; fix-edits-anchored-line incoherence proof; no-op churn must-not-mint). From-base re-anchoring (§28) is a hard sub-dependency. New concern: 30ms/mutant is SINGLE-TENANT — 8N concurrent full-suites under a busy Board degrade superlinearly in the exact earned-concurrency regime; if catches are ever priced, that becomes a "time your settles to a quiet Board" exploit. Needs a K-concurrent-settle benchmark before any pricing.
- Pragmatic TDD: Build the two-revision differential (oracle on BOTH pre- and post-fix revisions of the same anchored line), TDD'd against the three-case fixture with fix-edits-anchored-line as the RED proving "same mutant killed" incoherent. Carry from-base re-anchoring as the in-scope prerequisite. New concern: the catch silently inherits the "0 survivors ambiguous" failure at the catch layer — if the pre-fix line had MutantsConsidered==0 (operator-free), a real human fix mints NO catch, systematically under-crediting operator-free code; the catch must be a THIRD explicit outcome (catch / no-catch / no-oracle-signal), not binary.
- CI/CD & Delivery: integrate-on-tip should come BEFORE the catch. A mutant killed on a stale base says nothing about survival on a moved trunk; BOTH catch revisions are computed against the SAME immutable baseRev (orchestrator.go), so the catch inherits a SECOND hidden dependency — anchor-survives-rebase — that neither §28 (which re-anchors base->cur WITHIN the session branch) nor the #1 brick mentions. Build integrate-on-tip (rebase onto tip, checks on the INTEGRATED tree, tri-state clean/conflict/checks-red), TDD'd from the disjoint-file cross-symbol break (RISKS line 117); assert "integrated checks go RED," not "no conflict." New concern: logging the first economy primitive on pre-integration coordinates bakes the stale-base lie one layer deeper.
- Refactoring: Run a real 30+-file rename + extract-module through the EXISTING green pipe NOW and assert on the carnage as the first refactor acceptance entry — (a) count orphaned threads from the rename delete+add cliff, (b) assert mutation fires question: threads on behavior-PRESERVING relocated lines, (c) record both as expected-failure baselines. Unlike the catch (coupled to unbuilt re-anchoring), this needs NO prerequisite, runs on today's tree, settles Clash G with evidence, and quantifies the carnage the re-anchor work must absorb. New concern: QuestionThreadsFromMutations (thread.go:40) turns every non-killed mutant into a thread with no behavior-changing-vs-preserving distinction — the keystone oracle inverts the refactor stamp-penalty AT THE SOURCE, a code-level miscalibration not yet in RISKS.md.

Clashes touched: F (identity half still the live gate; now additionally attacked for an undesigned UNHAPPY path — three lenses converge that the proposed binary repeats §27); B (Game/Systems want the self-flag bit captured at mint time; UX wants the framing question answered against a real surface); C (CI/CD — integrate-on-tip is the experiment that begins to settle scoring-on-downstream-truth, and re-argues it should precede the catch); G (Refactor — now settleable on today's tree, promoted to concurrent); D/A/E/H/I (render-camp — still gated on a surface/loop, dissent on ordering registered).

Verdicts updated:
- Clash F: remains PARTIALLY RESOLVED, but the #1 DELIVERABLE IS REDEFINED — from Round-5's binary Catch{} to a TRI-STATE-plus-intermediate primitive: {catch | no-catch | no-oracle-signal (pre-fix MutantsConsidered==0) | partial-catch (survivor-set still non-empty post-revision)}. The survivor-set-transition definition and "never same mutant killed" stand and harden; the binary outcome schema does NOT.
- Clash G: still TBD but its settling experiment is promoted from "gated after #1" (Round 5) to "run concurrently on today's tree" — it has no unbuilt prerequisite.
- Clashes C, D, A, E, H, I: remain TBD; C gains a concrete experiment (integrate-on-tip) and a live ordering challenge to the #1 brick.

New clashes opened:
- The catch's UNDESIGNED UNHAPPY PATH (material new finding, three-lens convergent): partial-catch (UX), no-oracle-signal third state (TDD), missing "would-this-have-shipped" counterfactual (Game). The Round-5 binary primitive repeats the §27 happy-path-only mistake one layer down; #1 must ship tri-state + intermediate outcomes.
- Catch is minted on PRE-INTEGRATION coordinates (CI/CD): a second hidden dependency (anchor-survives-rebase) beyond §28 re-anchoring; live ordering dispute over whether integrate-on-tip must precede the catch.

Decisions:
1. NEXT BUILD (#1, TDD+Systems convergence, REDEFINED): the confirmed-catch oracle as a two-revision differential, survivor-set non-empty->empty, NEVER "same mutant killed" — but as a TRI-STATE + intermediate primitive {catch | no-catch | no-oracle-signal | partial-catch}, NOT the Round-5 binary. TDD'd against the three-case fixture (test-only / fix-edits-anchored-line=RED / fix-adds-branch) plus the degenerate-strategy cases (agent-authored killing test; no-op churn must-not-mint). First adversarial acceptance-suite entry.
2. PREREQUISITE sub-brick of #1: from-base re-anchoring (§28/§14) with "lost via rename" as a distinct state. Document the OPEN gap CI/CD raised — re-anchoring as scoped does NOT survive an integration rebase — on the #1 brick, since #1 is minted on pre-integration coordinates.
3. PROMOTED to CONCURRENT with #1 (shares no code, no prerequisite): the adversarial refactor trace on today's green tree — assert orphaned-thread count + mutation-on-relocated-lines as expected-failure baselines. Settles Clash G; quantifies the re-anchor carnage.
4. Capture the self-flag bit on every minted Catch while the schema is typed (Game+Systems), IF it does not delay #1 — the only path to later answering Clash B / flatter-vs-grade on real catches.
5. Then integrate-on-tip (CI/CD) — but its ordering vs #1 is now a LIVE disagreement (CI/CD argues precede); revisit next round.
6. Render §17 surface + two-agent loop remain gated after #1, with UX's two non-negotiables adopted: a designed empty/zero state for the mutation-silent case, and built against the CURRENT single-revision oracle's survivor-set state (so not strictly sequenced behind the two-revision catch).
7. Add three code-level risks to RISKS.md (oracle latency under fleet contention; thread.go:40 miscalibration on behavior-preserving churn; orchestrator.go:37 immutable-baseRev gap); run a K-concurrent-settle contention benchmark before any catch pricing.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5). NOT converged — definition of #1's unit agreed, but ordering (render-camp dissent; CI/CD integrate-first; Refactor-concurrent) and the newly-opened unhappy-path scope need one more round.

## Round 7 — CONVERGED: ordering & mint-scope closed around the agreed #1 — 2026-06-04

Trigger: closing round, charged narrowly to resolve the two ordering disputes Round 6 left live (CI/CD integrate-first; UX render-gating) and rank the bricks against the now-settled #1 unit — NOT to re-litigate settled points (the survivor-set tri-state definition, re-anchoring prereq, refactor-trace concurrency). Build state re-verified in code, unchanged: 6 green packages, no pipe, no UI, no economy code.

Panelists present: all six. No new lens.

New evidence on the table (verified by reading code this round):
- `grep Catch|Trust|Focus|Treasury|Ledger internal/` → ZERO typed events. Meta-finding 1 confirmed in code.
- `mutation/generate.go` — a Finding is keyed only by `(Line, Original, Mutated)` operator-transition STRINGS; there is NO stable mutant identity across revisions (validates Systems' new set-identity concern).
- `orchestrator.go:37/49` — baseRev immutable, never reconciled with tip; `diff.go:47` — `--no-renames` hardcoded; `review/thread.go` — a `question:` thread on EVERY non-killed mutant, no behavior-preserving distinction; `Render()` is string concat (no http/SSE).

Per panelist (final positions):
- UX: 7 rounds, 0 pixels — standing render-dissent holds and is verified: the §17 Via/SSE surface needs ONLY today's single-revision survivor-set state (Survived/Undetermined + MutantsConsidered), ZERO new backend, so it is NOT strictly sequenced behind the two-revision catch. Build it now, all FOUR oracle outcomes as distinct designed states. NEW concern: no designed in-flight/streaming state for "oracle still running" — SSE means a live, half-rendered card the reviewer WILL stare at. Clash A is now buildable (render all-killed silent vs "0 survivors" badge → reassurance or guilt-meter?).
- Game: economy-first is correct (concession stands) — the catch IS the first honest reward beat. RATIFY the catch as #1, but mint it with a "would-this-have-shipped" counterfactual proxy AT MINT (un-backfillable; past harness state not persisted), else the Ledger is born inflationary → bolts on the forbidden model-inferred catch-weight (V§13.5).
- Systems: feasibility ≠ economy; zero economy object exists to attack. RATIFY catch-first. NEW non-negotiable: the survivor-set has NO identity key — across a fix that changes the line's operator, before/after sets are over DIFFERENT operator alphabets, so "non-empty→empty" is ill-typed; the denominator must be a function of the line's CURRENT operator inventory per revision, else no-op churn and fix-edits-line are the SAME failure mode.
- TDD: "a beautiful odometer, no trip counter" — the two-revision catch has zero code; RATIFY it as #1, fix-edits-anchored-line the non-negotiable RED. CONCEDES CI/CD's ordering "has teeth, ranks just under #1": both revisions compute against the SAME baseRev → a 2nd hidden dependency (anchor-survives-rebase). NEW: 2× oracle cost/catch, contended; no-oracle-signal MUST be a first-class third outcome.
- CI/CD: BLOCKS the field's "integrate-after" ordering as a correctness dependency, not a preference — a mutant killed on a stale base says nothing about a moved trunk; §28 re-anchors WITHIN the branch, not across a rebase. Build integrate-on-tip (tri-state {clean|conflict|checks-red}) from the disjoint-file cross-symbol break, assert integrated checks RED. NEW: O(N²)/8N contention → design the merge-queue (batch+bisect) in from this brick.
- Refactoring: RATIFY the refactor trace as concurrent — it runs on today's tree, needs NO unbuilt prerequisite, settles Clash G, and QUANTIFIES the re-anchor carnage #1's sub-brick must absorb (so it de-risks #1, not competes). NEW: extract-module is invisible to BOTH halves — diff can't link A→B (no rename), mutation re-mutates relocated operators as net-new; no hunk classifier can SEE it as a refactor.

Clashes touched: F (identity half — the live gate; unit hardened a 3rd round + new identity-key requirement), G (settleable concurrently on today's tree), C (integrate-on-tip experiment + live-but-empirically-resolvable ordering), B (self-flag + would-have-shipped bits captured at mint for later correlation), A (now has a concrete render experiment), D/E/H/I (render-camp — gated on a surface/loop, strict-gating dissent registered).

Verdicts updated:
- Clash F: remains PARTIALLY RESOLVED; #1 unit CONFIRMED a 3rd round (tri-state survivor-set transition, never "same mutant killed") and HARDENED — Systems' missing survivor-set IDENTITY KEY (denominator = line's current operator inventory per revision + explicit inventory-change rule) added to the #1 spec; no-oracle-signal locked as a first-class outcome.
- Clash G: still TBD but experiment CONFIRMED concurrent-on-today's-tree and de-risking; thread.go source-level miscalibration logged.
- Clash C: gains the integrate-on-tip experiment AND a sharpened, partly-conceded ordering argument; to be settled empirically by the trunk-moved-underneath stress variant.
- Clashes A, D, E, H, I: remain TBD; A gains its first buildable experiment.

New clashes opened: none at the target level — convergence on #1 held and strengthened. The would-have-shipped mint-scope (Game) and strict-gating-of-the-surface (UX) are scheduling/scope sub-disputes inside agreed bricks; Systems' identity-key finding is an additive sharpening of #1.

Decisions (the marching orders):
1. NEXT BUILD (#1, 5-lens convergence, definition hardened a 3rd round): tri-state confirmed-catch oracle — two-revision survivor-SET non-empty→empty on the same anchored line, NEVER "same mutant killed", {catch | no-catch | no-oracle-signal | partial-catch}. TDD'd against the three-case fixture (test-only / fix-edits-anchored-line=RED / fix-adds-branch) + degenerate suite (agent-authored killing test; no-op churn must-not-mint). NEW non-negotiable: define the survivor-set IDENTITY KEY as a function of the line's current operator inventory per revision, with an explicit inventory-change rule. First adversarial acceptance-suite entry (meta-finding 2).
2. #1 PREREQUISITE sub-brick: from-base re-anchoring (§28/§14), "lost via rename" a distinct state. Document the OPEN gap that this re-anchoring does NOT survive an integration rebase (CI/CD's 2nd hidden dependency).
3. CONCURRENT with #1 (no shared code/prereq, today's green tree): the adversarial refactor trace — 30+-file rename + extract-module; assert orphaned-thread count + behavior-preserving-suspicion + extract-module-invisibility as expected-failure RED baselines. Settles Clash G.
4. CAPTURE AT MINT while the Catch schema is typed (cheap, un-backfillable): the self-flag bit AND the would-have-shipped counterfactual proxy — data-capture only, the guard against an inflationary Ledger / forbidden catch-weight; adopt IF it does not delay #1's definition work.
5. INTEGRATE-ON-TIP brick (CI/CD): rebase onto tip, checks on the INTEGRATED tree, tri-state, merge-queue (batch+bisect) designed in. Ordering vs #1 LIVE but RESOLVED EMPIRICALLY: build the fix-edits-anchored-line fixture first (cheap), then a trunk-moved-underneath variant; if the survivor-set transition does NOT survive the rebase, integrate-on-tip MUST precede catch pricing.
6. RENDER §17 surface + two-agent loop: gated after #1's definition but NOT strictly behind the two-revision oracle (verified: needs only today's survivor-set state). Adopt all FOUR designed outcome states + a NEW designed in-flight/streaming state; Clash A's silent-vs-badge experiment falls out of it.
7. Add FOUR code-level risks to RISKS.md (oracle latency under fleet contention; thread.go behavior-preserving-churn miscalibration; orchestrator.go immutable-baseRev gap; survivor-set has no identity key / ill-typed denominator under operator change). Run a K-concurrent-settle benchmark before any catch pricing.

CONVERGED on the #1 build (3rd consecutive round, 5/6 lenses incl. render-camp conceding economy-first). Residual disputes are ordering/mint-scope to be settled WHILE building #1 — notably CI/CD's integrate-first, which the fix-edits-anchored-line + trunk-moved stress variant resolves empirically rather than by another argument round. NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5). The council loop terminates here on genuine convergence; the next event is a BUILD (slice #1), not another deliberation round.

## Round 8 — from validated clashes to a committed BUILD SEQUENCE: ranking the slices to a working full prototype — 2026-06-04

Trigger: new charge — commit to building agntpr as a WORKING FULL PROTOTYPE, advancing one validated, green-tested slice at a time. Round 7 converged on the #1 brick and said "the next event is a BUILD, not a round," yet the tree is byte-identical (only test-infra commits since: testify migration + CONVENTIONS compliance). The council's job is no longer only to validate clashes but to CHART and CONFIRM the build sequence that reaches a usable prototype. Build state re-verified in code, unchanged: 6 green backend packages (mutation/review/settle/diff/translate/orchestrator); `grep Catch|Trust|Focus|Treasury|Ledger internal/` → ZERO typed events; `grep fetch|rebase|merge-base|integrate|onto` (non-test) → ZERO integration primitives; no http/SSE/template surface (the incidental http/via grep hits are comment/struct text in runner.go/translate.go, not a surface).

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD, Refactoring). No new lens.

New evidence on the table (verified by reading code this round):
- `mutation.Result` = `{Findings []Finding, MutantsConsidered int}`; each Finding keyed only by `(Line, Original, Mutated)` operator STRINGS (generate.go:24-26, runner.go:40-57) — no stable mutant identity across revisions. The tri-state catch maps cleanly onto this substrate as a pure differential over `Run()` — no oracle rewrite.
- `translate.go:80-95` emits `activity.agent {thinking|editing|tool}` + `turn.ended`; `orchestrator` emits `TurnOutcome{Minted,SHA,Added,Deleted,Diff,Secrets}` — a live agent heartbeat + a settle beat with NO surface to play them on.
- `diff.go:42` `--no-renames` hardcoded (rename → delete+add → anchors evaporate); `orchestrator.go:37,49` baseRev immutable, never reconciled with tip; `thread.go:40` emits a `question:` on EVERY survivor with zero behavior-preserving distinction; `review/thread.go:27` `Render()` is string concat.

Per panelist:
- UX: 7 rounds, 0 pixels — render-dissent re-verified and now SHARPENED by the prototype goal: "full WORKING prototype" is a different bar than "validated clash" — a prototype is unusable until a human can SEE one card, so the roadmap MUST end at a usable surface, not at another economy primitive. Ratify the catch (concession holds, it is un-backfillable), but build the §17 pipe + Via/SSE surface as a PAIR right behind it — all FOUR outcome states + a designed IN-FLIGHT/streaming state, and a designed empty/zero "N considered, 0 survived — tested" state (the MOST COMMON screen, still undesigned, will read as "broken"). Keep every meter OFF the first screen.
- Game design: seven rounds, zero pixels, and now we want a "usable prototype" — one you can't sit in front of isn't one. The catch is the right spine but minting it before there's a Board is building the scoring before the game. FLIP the order for the prototype goal: pipe-to-a-live-card FIRST (zero new backend, only wiring), catch as build #2 layered onto a loop with a human in the chair, so the first catch is a beat someone WITNESSES, not a row in a log. NEW: the in-flight state has no defined TEMPO — raw event passthrough reads as log-spew or a frozen card; the in-flight state needs a designed CADENCE (debounce/coalesce into legible beats), tunable only against a live replay.
- Systems: six packages green, economy still zero typed events, Finding keyed only on string triples → no set identity exists. RATIFY #1 a 4th time, but it converts from spec to a real Go TYPE this round or it is not built: one typed `Catch{Anchor, BeforeInventory, AfterInventory, Outcome}` where Outcome is a PURE function of the two operator-inventory sets on the anchored line. Build the identity key as a real type FIRST inside this brick so one function owns the denominator. The §17 surface is buildable now on today's single-revision set (zero new backend) so it is NOT behind the two-revision catch — build it in parallel, but do NOT mint an inferred catch-weight alongside it.
- Pragmatic TDD: the substrate is confirmed — the tri-state catch is a pure differential ON TOP of `Run()`, no rewrite. Stop deliberating; build #1 via tdd-rygba with fix-edits-anchored-line as the RED that proves "same mutant killed" incoherent, carry re-anchoring in-scope, ship no-oracle-signal as a first-class outcome. One caveat that will not drop: a green RED→GREEN proves the transition FIRES, not that it CONSTRAINS — the degenerate suite (agent-authored killing test, no-op churn must-not-mint) is what earns its keep. CI/CD's rebase dependency is documented as an open gap on the brick, settled by the trunk-moved variant, not by argument.
- CI/CD & Delivery: still ZERO integration primitives. "Landed" is not done; only "Merged" through real CI is — and this prototype has no Merged state at all. Concede #1 (economy spine), do NOT block it; but the catch is minted on PRE-INTEGRATION coordinates — re-anchoring (§28) maps WITHIN the branch, NOT across a moved trunk. Build integrate-on-tip {clean|conflict|checks-red} on the rebased tree as the unit that converts Landed→Merged and gives the catch a sound base; fixture (B) clean-rebase-but-checks-red is the killer proving a green pre-integration catch can be a red post-integration regression. Under the N-agent goal this is a SERIALIZATION POINT — build it as ONE merge-queue lane, not N independent rebases (the O(N^2)/8N contention regime). Price the catch only against an integrated base.
- Refactoring: ratify #1, but its re-anchor prerequisite is exactly where refactors go to die, so the concurrent refactor trace is NOT optional garnish — it is the acceptance bar that prerequisite is built against. `--no-renames` (diff.go:42) downgrades a 30-file rename to delete+add → every thread orphans and the survivor-set transition is computed against a line that no longer exists → silent no-oracle-signal at best, phantom catch at worst. A clean refactor with green unchanged tests is the SAFEST skim, yet today's tree treats it as MAXIMAL noise (every survivor → `question:`) and maximal anchor-carnage. Build the RED baselines NOW, in the SAME build wave as the re-anchor sub-brick, BEFORE the §17 surface — else the prototype's first real rename mints phantom catches that poison the Ledger on day one.

Clashes touched: F (identity half — now ratified a 4th round AND converted from spec to a required TYPE this round: Systems' `BeforeInventory`/`AfterInventory` pure-function denominator becomes the brick's contract, the explicit inventory-change rule the load-bearing RED); C (integrate-on-tip remains the experiment, ordering re-stated as "precede catch PRICING not minting," to be settled by trunk-moved fixture (A) + clean-rebase-checks-red fixture (B)); G (refactor trace re-ratified concurrent; rename_neutral_move directly stresses F's identity key); D (Game nominates human-dwell, CI/CD nominates integrated-checks-RED — opposite experiments on the same card, both deferred to #7/#5); A (silent-vs-badge becomes #4's acceptance bar); B (self-flag + would-have-shipped captured at mint, data-only); E/H/I (render-camp — unblock ONLY once the surface + two-agent loop exist; roadmap now explicitly REACHES them).

Verdicts updated:
- Clash F: remains PARTIALLY RESOLVED but the #1 unit is now BUILD-READY and TYPE-COMMITTED — the survivor-set identity key (denominator = line's current operator inventory per revision + explicit inventory-change rule) becomes a real Go type owned by one pure function THIS build, not prose; tri-state + partial-catch + no-oracle-signal-as-first-class all stand. Moves off "spec" toward code; verdict will flip on the green #1 build.
- Clash G: still TBD; experiment re-CONFIRMED concurrent-on-today's-tree and re-scoped as the acceptance bar for the re-anchor sub-brick (must land in the SAME wave, not after the surface).
- Clash C: gains both settling fixtures (trunk-moved (A); clean-rebase-checks-red (B)); ordering refined — integrate-on-tip precedes catch PRICING, not catch MINTING.
- Clashes A, D, E, H, I: remain TBD; the roadmap now terminates at the surface + two-agent loop that unblock them, with named experiments.

New clashes opened: NONE at the target level — the #1 brick held a 4th round and HARDENED into a typed contract. The build-ORDER split (UX+Game pipe-first vs the field catch-first) is a scheduling sub-dispute inside an agreed roadmap, NOT a new clash; per the Round-7 bar it does not block convergence. Game's in-flight-CADENCE concern is an additive design requirement folded into surface build #4.

Decisions (the marching orders):
1. NEXT BUILD (#1, 6/6 ratify the brick; build-order dissent recorded, chair-resolved): the tri-state confirmed-catch oracle as a typed two-revision differential over `Run()` — `Catch{Anchor, BeforeInventory, AfterInventory, Outcome}`, Outcome ∈ {Catch | NoCatch | NoOracleSignal | PartialCatch}, survivor-SET non-empty→empty on the same anchored line, NEVER "same mutant killed." The identity key (denominator = line's current operator inventory per revision) becomes a REAL TYPE owned by one pure function THIS build, with the explicit inventory-change rule (fix edits L + changes inventory → ill-typed → NoCatch). Built via tdd-rygba; fix-edits-anchored-line the load-bearing RED. First economy object + first adversarial acceptance entry.
2. #1 PREREQUISITE sub-brick, BUILD FIRST: from-base re-anchoring (§28/§14), "lost via rename" a distinct state (→ NoOracleSignal, never a phantom Catch). Document the OPEN gap: this re-anchoring does NOT survive an integration rebase; the catch is minted on pre-integration coordinates.
3. CONCURRENT with #1 (no shared prereq, today's green tree), in the SAME build wave: the adversarial refactor trace — internal/refactor/testdata/ {rename_40, rename_neutral_move, extract_module}, each with a GREEN unchanged suite; RED baselines: orphanedThreadCount>0; survivor-set ill-typed across rename (lost-via-rename != Catch — stresses #1's key); extract-module re-mutated as net-new (invisibility). Settles Clash G; de-risks #1's sub-brick.
4. CAPTURE AT MINT (cheap, un-backfillable, data-only, IF it does not delay the definition): self-flag bit + would-have-shipped counterfactual proxy on the Catch record — NO weight, NO pricing (guards against an inflationary Ledger / forbidden catch-weight V§13.5).
5. RANKED ROADMAP to a USABLE PROTOTYPE (each with its validating experiment): [#1-sub] re-anchoring [rename re-anchors / lost-via-rename distinct]; [#1] tri-state catch [three-case + degenerate suite]; [#2 concurrent] refactor trace [RED carnage baselines]; [#3] §17 pipe end-to-end on ONE real changeset [one comment → one revision, mutation at settle, thread anchors — the single-user happy loop dispatch→settle→review→land]; [#4] Via/SSE single-card surface, FOUR outcome states + designed in-flight/streaming state with a designed cadence, against today's single-revision oracle [silent-vs-badge reads as "tested, ship it" not guilt; streaming-no-flash], METERS OFF the first screen; [#5] integrate-on-tip {clean|conflict|checks-red} [trunk-moved (A) + clean-rebase-checks-red (B)] — if (A) shows the transition fails to survive a rebase, #5 precedes catch PRICING; [#6] single-lane merge queue wrapping #5 [throughput-to-zero]; [#7] two-agent Board loop instrumented for idle/dwell [Clash D real N-ceiling via rework-vs-concurrency].
6. BLOCKERS that must be cleared before the prototype is reachable: (a) re-anchoring (§28) is a HARD prereq of #1 AND a known-incomplete one (does not survive rebase) — the refactor trace #2 quantifies the carnage it must absorb, in the same wave; (b) the survivor-set has no identity key until #1 makes it a type — until then non-empty→empty is ill-typed and no-op churn is indistinguishable from a real fix; (c) the prototype's single-user happy loop (#3) MUST exist before any economy stock is rendered — meters (Focus/Trust/Treasury) stay OFF the first screen or we ship a cockpit of gauges against an unproven loop; (d) catch PRICING (not minting) is gated on #5 if the trunk-moved fixture shows the transition is rebase-dependent.
7. RISKS.md already carries the four code-level risks from R5-7 (oracle latency under fleet contention; thread.go behavior-preserving-churn miscalibration; orchestrator.go immutable-baseRev gap; survivor-set no-identity-key); run the K-concurrent-settle benchmark before any catch pricing.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (4th consecutive round): 6/6 lenses ratify the tri-state catch oracle as the #1 brick — now type-committed — and 6/6 affirm the ranked roadmap REACHES a usable §17 pipe + Via/SSE surface (all four outcome states + a designed in-flight state) rather than terminating at an economy primitive. The sole dissent is build-ORDER (UX+Game: pipe-before-catch), a scheduling sub-dispute inside the agreed roadmap that the Round-7 bar does not let block convergence; the chair resolves it catch-first because the catch is the only un-backfillable primitive and the pipe+surface follow immediately at #3-#4. The next event is a BUILD — the re-anchoring sub-brick + the tri-state catch (#1) and the refactor trace (#2) in one wave — not another deliberation round.

## Round 9 — the #1 wave SHIPPED: from a green economy unit to its first real mint, and the ranked march to a usable prototype — 2026-06-04

Trigger: the Round-8 marching orders BUILT — for the first time the council reconvenes on post-build evidence, not a byte-identical tree. The charge is the full-prototype goal: confirm/refine the immediate next build, produce a ranked roadmap from #1 through a USABLE prototype, and flag prototype blockers. Build state re-verified in code: 9 green packages (`env -u GOROOT go test ./internal/...` → 9 ok); the R8 wave (`internal/catch`, `internal/reanchor`, `internal/refactor`) landed across 3 real commits.

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD, Refactoring). No new lens.

New evidence on the table (verified by reading code this round):
- `internal/catch` is a real Go type — `Detect(before, after LineState) Outcome`, `Outcome ∈ {Catch | NoCatch | NoOracleSignal | PartialCatch}` (catch.go:50-70). The identity key is enforced exactly as specced: `len(beforeInv)==0 → NoOracleSignal` (line 52); `!setsEqual(beforeInv, afterInv) → NoCatch` (line 55, the inventory-change rule); before-survivors empty → NoCatch; after-survivors empty → Catch; strict-subset → PartialCatch. `LineState` is SET-keyed, dedup'd per line (catch.go:35-43, a deliberate v1 simplification, comment-documented).
- The denominator is owned by ONE pure function; the refusal arms are TESTED, not just the birth-cry: `TestDetect_refusesCatchWhenFixChangesOperatorInventory`, `_refusesCatchForNoOpChurn`, `_refusesCatchWhenNewSurvivorAppears`, `_refusesCatchWhenInventoryShrinks`, plus the real-oracle `TestDetect_mintsCatchAcrossRealOracleRevisionsWhenTestStrengthened` (catch_test.go:178). The catch CONSTRAINS.
- `internal/reanchor` ships LostViaRename as a DISTINCT state (reanchor.go:35,75), alongside Same/Moved/Outdated; the refactor trace (`internal/refactor/trace_test.go`) locks the carnage as expected-PASS: 40-file rename orphans all 40 threads, neutral rename → LostViaRename != Catch, extract-module re-mutated as net-new.
- BUT `catch.Detect` has ZERO prod callers — `grep catch. internal/ (non-test)` returns only a COMMENT in reanchor.go. The mint is struck and never opened: `Detect` is dead code, nothing in orchestrator/settle mints a Catch.
- Still ZERO integration primitives (`grep rebase|merge-base|integrate|onto` non-test → 0); orchestrator.go:37,49 diffs an immutable baseRev never reconciled with tip. Still ZERO surface (`grep ListenAndServe|event-stream|Flusher` non-test → 0). Still ZERO economy callers beyond catch itself.

Per panelist:
- UX: real progress — 3 commits, not byte-identical, the #1 brick is green and type-committed. But the headline is now acute: 8 rounds, still zero pixels — catch.go emits an Outcome no surface renders, translate emits into the void, orchestrator emits TurnOutcome with nothing to play it on. A prototype you can't watch is a test suite. Build #3 pipe + #4 Via/SSE card as ONE wave: four outcome states + a designed coalesced in-flight beat, meters OFF the first screen, and the MOST-COMMON 'N considered, 0 survived — tested' screen designed as a calm WIN not empty/error chrome. NEW: the enum collapses two opposite 'quiet' meanings — NoOracleSignal (blind) vs NoCatch-already-constrained (verified-strong) — that a naive surface renders identically; the pipe must carry WHY it is quiet, not just the Outcome string.
- Game design: the build finally happened and it's honest and green — credit due. But a tycoon game with no screen is a spreadsheet. Nobody has ever FELT a catch; it has only been a row in a Go test. FLIP confirmed: pipe-first (#3, pure wiring, zero backend — orchestrator already emits TurnOutcome, translate already emits activity.agent + turn.ended), capture the replay, THEN the surface. NEW: the pipe emits a raw firehose with NO designed TEMPO — un-coalesced it reads as log-spew or a frozen card and the catch beat DROWNS; #3 must PERSIST the trace so #4 has an honest replay to tune cadence against.
- Systems: the marching order shipped — `Catch` is a real type, identity-key enforced, denominator owned by one pure function. But the economy object is an ISLAND: zero callers, `Detect` is dead code, we minted the unit of account and never opened the mint. Every property I red-team — does no-op churn fail to mint, is the agent-authored-killing-test farm denied, does PartialCatch read right — is proven ONLY against hand-built LineState literals, NEVER against two real revisions through settle→diff→mutation→Detect. The next build is the smallest driver that mints ONE real Catch from settle A→settle B, validated by the farm fixture run FOR REAL (agent-authors-killing-test → Catch; agent-edits-the-line → NoCatch). Acceptance must be 'a real Catch minted from two real settles', not 'catch package green'. Also: LineStateAt reads inventory from current src but survivors from a passed-in res — these can desync across re-anchoring, and reanchor is not wired into LineStateAt yet.
- Pragmatic TDD: deliberation debt paid — the catch is a pure differential over Run() with the refusal arms TESTED, so it constrains. Now build the §17 pipe so the catch runs at settle on a real two-revision sequence: today every package is green in isolation but nothing has run the before→after flow end-to-end through the orchestrator. The killer RED: the SAME pipe run on a turn that edits the anchored line (>=→>) must yield NoCatch, not a phantom catch — proving the inventory-change rule survives the REAL reanchor path, not just the unit table. NEW logged constraint: the set-not-multiset keying (catch.go:39) under-credits a line where a fix kills one of two same-operator survivors (reports NoCatch despite real partial progress) — RISKS.md before any pricing.
- CI/CD & Delivery: a build, not a round — credit due, the catch is real, typed, green. But my seam is untouched and WORSE: the catch is now minted on PRE-INTEGRATION coordinates IN SHIPPED CODE — 'Landed != Merged' is no longer a warning, it is a property of catch.go + orchestrator.go (immutable baseRev, no rebase path). I do not block #3 — a prototype needs a visible loop — I block #3 shipping a FAKE 'land'. Fold integrate-on-tip into #3's land as a tri-state {clean|conflict|checks-red}, or make 'land' return an explicit Unintegrated state. Settle the rebase question with the trunk-moved (A) and clean-rebase-checks-red (B) fixtures BEFORE the catch is ever PRICED. NEW: the roadmap lists integration LATE — that is backwards under the 1:N goal; integrate-on-tip is the serialization point and must not be retrofitted after the Board trains users on a fake 'land'.
- Refactoring: wave-1 landed and my baselines are green — rename is LostViaRename, the inventory rule kills the phantom catch, the refactor trace stands. But we built TWO correct bricks and NO edge between them: `grep catch.Detect (non-test)` is empty, Detect takes a `line` int the CALLER must have re-anchored, Detect does NOT call reanchor. The safety property the council ratified — 'lost via rename → NoOracleSignal, NEVER a phantom Catch' — is asserted in two SEPARATE test packages and enforced by NO single function. The safest possible change (a clean rename, green unchanged tests) is exactly where this oracle goes to die, and nothing in PROD prevents a phantom catch. Build the one-function JOIN CatchAcross(repoDir, anchor, beforeRev, afterRev) — reanchor first, NoOracleSignal-by-construction on LostViaRename/Outdated, Detect only on Same/Moved — BEFORE the pipe. Compose the bricks before building on an unproven composition.

Clashes touched: F (the unit FLIPS toward RESOLVED-IN-CODE — tri-state + identity-key shipped green with tested refusal arms; residual: the FARM half is asserted only against literals, Detect has never seen two real revisions, so 'runs every settle / not farmable' is still untested END-TO-END); G (the refactor trace is now EXECUTABLE evidence — neutral rename refuses to mint, extract-module invisible, 40-file orphan asserted — moves to RESOLVED for the carnage-baseline question, but exposes the unjoined-packages gap); C (unchanged in verdict but SHARPENED — the catch is now minted on pre-integration coords in SHIPPED code; ordering stands as 'precede catch PRICING not minting', settled by fixtures A+B); A (silent-vs-badge + NoOracleSignal-vs-Catch discrimination becomes #4's acceptance bar); B (capture-at-mint decided in R8 is NOT in the Catch type — no record is persisted, the un-backfillable data is being lost on every mint, of which there are zero); D/E/H/I (render-camp — the roadmap finally REACHES them at #4/#8; unblock the moment the surface + loop land).

Verdicts updated:
- Clash F: → RESOLVED-IN-CODE on the UNIT (tri-state + identity-key + tested refusal arms, all green), but RE-SCOPED OPEN on the LOOP — 'the oracle runs every settle' and 'the farm is denied in the wild' are asserted only against LineState literals; Detect has never adjudicated two real revisions. Flips fully on the green #3 pipe (real Catch minted from two real settles + agent-edits-line → NoCatch end-to-end).
- Clash G: → RESOLVED on the carnage-baseline question (refactor trace is executable: neutral rename != Catch, extract-module invisible, 40-thread orphan asserted). Residual reclassified into the new composition gap, NOT a fresh clash: the asserted safety lives in two unjoined packages, closed by the reanchor→catch JOIN.
- Clash C: unchanged verdict (catch minted pre-integration), now a PROPERTY OF SHIPPED CODE rather than a warning; both settling fixtures (trunk-moved A, clean-rebase-checks-red B) carried forward to #5; integrate-on-tip folded into #3's land as a tri-state or an explicit Unintegrated state.
- Clashes A, D, E, H, I: remain TBD; the roadmap now terminates at #4 (surface) + #8 (two-agent Board) that unblock them, with named experiments.

New clashes opened: NONE at the target level — the #1 brick shipped green and the council converges 6/6 on the §17 pipe as the next build. The build-ORDER split (UX: pipe+surface one wave; the field: pipe then surface) and the SCOPING refinements (Systems: real-mint not generic pipe; Refactoring: the reanchor→catch JOIN as prereq; CI/CD: no fake land) are scheduling/scope sub-disputes INSIDE the agreed brick — per the Round-7 bar they do not block convergence. Systems' island-not-economy and Refactoring's composition-gap are additive sharpenings folded into #3's spec.

Decisions (the marching orders):
1. NEXT BUILD (#3, 6/6 converge on the §17 pipe): the pipe AS THE CATCH'S FIRST REAL MINT — a minimal single-user driver settle A → settle B → diff → mutation.Run×2 → re-anchor B→A → CatchAcross → emit ONE typed Catch + resolve/keep the question: thread, on ONE real temp git repo. Acceptance bar (chair-set, Systems): 'a real Catch minted from two real settles', NOT 'catch package green'. Built via tdd-rygba.
2. #3 PREREQUISITE sub-brick, BUILD FIRST (Refactoring + Systems, chair-adopted): the reanchor→catch JOIN CatchAcross(repoDir, anchor, beforeRev, afterRev) — reanchor first; LostViaRename/Outdated → NoOracleSignal BY CONSTRUCTION (never reaches Detect); Detect only on Same/Moved against the RE-ANCHORED after-LineState. Load-bearing RED: rename_neutral_move yields a phantom Catch via a naive caller, GREEN when CatchAcross short-circuits to NoOracleSignal. This fuses the two green-but-unjoined bricks into one TYPED guarantee and is the single entry point #3 and #4 require.
3. CI/CD BINDING into #3 (chair-adopted, non-negotiable): #3's land step returns a tri-state {clean|conflict|checks-red} on the rebased/integrated tree, OR an explicit Unintegrated placeholder — NEVER a fake 'merged'. 'Landed != Merged' is a property of shipped code; #3 must not bake it one layer deeper.
4. CAPTURE-AT-MINT, WITH #3 (Systems' Clash-B blocker, data-only, no weight/pricing): persist the Catch record with a self-flag bit + would-have-shipped counterfactual proxy. There is currently NO Catch record persisted at all; the un-backfillable data must be captured on the first real mint or the Ledger is born without it.
5. RANKED ROADMAP to a USABLE PROTOTYPE (each with its validating experiment): [#3-prereq] reanchor→catch JOIN [neutral-rename phantom RED → NoOracleSignal]; [#3] §17 pipe / first real mint [two-revision farm: agent-strengthens-test → Catch; agent-edits-line → NoCatch end-to-end; catch as one discrete beat in a persisted replayable trace; land returns tri-state/Unintegrated]; [#4] Via/SSE single-card surface, FOUR outcome states + designed in-flight/coalesced state, meters OFF first screen [zero-survivor reads 'tested' not empty; NoOracleSignal visibly DISTINCT from Catch; 50-event burst coalesces no-flash-no-freeze; one-comment→one-revision loop]; [#5] integrate-on-tip {clean|conflict|checks-red} [trunk-moved A; clean-rebase-checks-red B] — if (A) flips the outcome, #5 precedes catch PRICING; [#6] single-lane merge queue wrapping #5 [throughput-to-zero, batch+bisect]; [#7] capture-at-mint persisted [self-flag + would-have-shipped, no weight]; [#8] first economy stock rendered + two-agent Board instrumented for idle/dwell [Clash D N-ceiling via rework-vs-concurrency]; [#9] catch PRICING [gated on #5(A) + K-concurrent-settle benchmark, against an integrated base].
6. BLOCKERS that must be cleared before the prototype is reachable: (a) the reanchor→catch JOIN is a HARD prereq of #3 — until CatchAcross exists the ratified safety property (lost-via-rename → NoOracleSignal) is enforced by NO prod function and a pipe author can mint a phantom catch; (b) catch.Detect has ZERO callers — the economy is an island until #3 opens the mint; the degenerate/farm suite is unproven against real revisions; (c) the single-user happy loop (#3) MUST exist and persist its trace before any meter renders — Focus/Trust/Treasury stay OFF the first screen; (d) 'land' must never be a fake merged — fold integrate-on-tip or return Unintegrated; (e) catch PRICING is gated on #5 if the trunk-moved fixture shows the transition is rebase-dependent.
7. RISKS.md additions/standing: log the set-not-multiset under-crediting gap (catch.go:39 — a fix that kills one of two same-operator survivors reports NoCatch); log LineStateAt inventory/survivor desync risk across re-anchoring; the four R5-7 code-level risks stand; run the K-concurrent-settle benchmark before any catch pricing.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (5th consecutive round): the #1 wave SHIPPED green — 6/6 lenses ratify the result — and 6/6 advocate the §17 pipe as the next build, refined to the catch's FIRST REAL MINT on a reanchor→catch JOIN prerequisite, with land returning a tri-state/Unintegrated state (never fake merged) and capture-at-mint landing alongside. The roadmap REACHES a usable Via/SSE surface (four outcome states + a designed in-flight state) and a two-agent Board, not an economy primitive in a vacuum. The sole dissent is build-ORDER/scoping inside the agreed brick (UX: pipe+surface one wave; the chair keeps #3 then #4 so the surface tunes against a real persisted trace and a real mint). No new target-level clash. The next event is a BUILD — the reanchor→catch JOIN, then the §17 pipe minting one real Catch from two real settles — not another deliberation round.

## Round 10 — the #2 wave SHIPPED (first real mint + the rendered card): from green islands to a watchable prototype — wiring the seam — 2026-06-04

Trigger: the Round-9 marching orders BUILT — the council reconvenes on post-build evidence for the second consecutive wave. The Round-8 brief snapshot is now STALE by 4 commits (it claims zero economy/surface — wrong). The charge holds: under the full-prototype goal, confirm/refine the immediate next build, rank the slices to a USABLE prototype, and flag prototype blockers. Build state re-verified in code: 11 green packages (`env -u GOROOT go test ./internal/...` → 11 ok); the R9 wave landed the reanchor→catch JOIN, the §17 first-real-mint pipe, and the Via/SSE surface card.

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD, Refactoring). No new lens.

New evidence on the table (verified by reading code this round):
- `internal/pipe.RunCatchCycle` (pipe_cycle.go:45) mints ONE typed catch from TWO REAL settles end-to-end: settle→throwaway worktree→mutation×2→reanchor→CatchAcross→Detect. `CatchAcross` is the JOIN (pipe.go:23). Land returns `Unintegrated` (pipe_cycle.go, the LandState const) — honest, never fake-merged; CI/CD's non-negotiable HELD in shipped code.
- `internal/surface/card.go` renders all four `catch.Outcome` states + `Tested` (the separate const, card.go:20) + in-flight, each a distinct `data-state` marker (present(), card.go:53-71). Tested reads "Tested — ship it" (the calm-win, card.go:67-68); NoOracleSignal reads distinct from Catch (card.go:59-61). Tested over httptest SSE incl. `_streamsEachVerdictAsLivePatch` and `_hostileVerdictCannotBreakOutOfTheStateAttribute`; `_carriesNoEconomyMetersOnTheFirstScreen` locks meters OFF.
- BUT `grep ListenAndServe|Flusher|event-stream|via.New|via.Run` (non-test) → ZERO. No `cmd/`. The card is exercised only by the via test client; NOTHING serves it — a human still cannot open a browser and watch a catch.
- The pipe→card EDGE does not exist: zero prod caller joins `CycleResult.Outcome` → `ReviewCard.Sig`. The mint and the card are two islands tested apart.
- `CycleResult.Trace` is a flat `[]string` of `fmt.Sprintf` lines (pipe_cycle.go:36,52,88) — NO timestamps, NO event types. Capture-at-mint (R6/R7/R8/R9 #4) is ABSENT: `CycleResult` carries Outcome/Path/Line/Land/Trace but NO persisted Catch record, no self-flag bit, no would-have-shipped proxy. `grep selfflag|would_ship|persist|ledger` (non-test) → 0. Every real mint discards un-backfillable data; there are zero integration primitives (`grep rebase|merge-base|integrate` non-test → 0).

Per panelist:
- UX: ten rounds and the card EXISTS and is HONEST — every state I fought for is green, including the in-flight beat and the calm-win "Tested — ship it." But a tested component is not a prototype; a prototype is something a human OPENS. The prototype is not watchable — that is now the ONLY thing between us and "usable." Build the SSE-served single-card surface wired to a real RunCatchCycle next: the server + the pipe→card edge, not new render logic. NEW: the bare `Outcome` cannot express Tested vs in-flight vs blind-NoOracleSignal — the seam must map pipe state and carry WHY a card is quiet, or the most-common calm-win screen collapses into the blind-silence screen (the R9 two-quiet-meanings collision, now at the wire). Keep meters off the first screen.
- Game design: we minted the coin, type-checked the card, and STILL nobody has watched a catch happen — 9 rounds, zero pixels; that is an exhibit behind glass, not a prototype. Boot a real Via SSE server and let a human watch one recorded catch land as a single beat. NEW (un-met R9 demand): the trace is an UNTIMED, UNTYPED `[]string` — the council shipped the pipe but threw away the one thing my lens needs (tempo); a replayable trace without time is a log, not a replay. Fix Trace to `[]TraceEvent{T,Kind,Msg}` so cadence can be tuned — feel lives on the time axis.
- Systems: big since R9 — 11 green pkgs, the mint is no longer an island at the catch-pkg level. But TWO NEW islands replaced the old one: pipe↔surface have NO edge, and capture-at-mint is STILL un-built — every mint discards the denominator. Build the end-to-end SEAM + the CatchRecord in ONE wave: one runnable cmd + SSE server streaming `CycleResult.Outcome` into the card, AND persist a typed `CatchRecord{Outcome, Anchor, BeforeInv, AfterInv, SelfFlagged, WouldHaveShipped}` as the mint's only durable artifact — data-only, NO weight/pricing. One scarce object, one conversion, one durable record, recorded BEFORE any stock renders. The seam must carry the quiet-discriminator, not reconstruct it surface-side.
- Pragmatic TDD: the mint is open and it CONSTRAINS — eight rounds of "beautiful odometer, no trip counter" is finally a trip counter that counted a real trip (Clash F's loop half RESOLVED-IN-CODE end-to-end). But we are now minting a catch and dropping it on the floor: RunCatchCycle returns a string and persists NOTHING. PULL capture-at-mint FORWARD as the immediate build — it is the only un-backfillable thing the now-live mint is actively destroying. In the SAME build, close the farm-untested-end-to-end gap: drive no-op-churn and agent-authored-killing-test fixtures THROUGH RunCatchCycle, not literals; the test that earns its keep is "no-op churn writes NOTHING to the log." Persist WITH the WHY (MutantsConsidered + a reason tag), or we store outcomes we can never audit for theater — the exact failure the oracle was built to prevent, re-introduced at the persistence layer.
- CI/CD & Delivery: credit due — a real mint and card shipped and my non-negotiable HELD (land is Unintegrated, not fake-merged). But honesty as a placeholder is where prototypes lie to first users; "Landed" is not done, only integration through real CI is, and the merged state is still a string constant. Next build is integrate-on-tip as ONE serialized merge-queue lane replacing the Unintegrated const: rebase onto tip in a throwaway worktree reusing runOracleAt — abort→Conflict, integrated checks non-zero→ChecksRed, else Clean. Load-bearing RED: clean-rebase-but-checks-red (disjoint lines, cross-file break). I do NOT block the surface; I block it being PRICED against a fake integration verdict. NEW cost-sharpening: the cycle already runs the oracle twice, integrate adds a third run, the queue re-runs every card per tip — the K-concurrent benchmark must run on the INTEGRATED cost before pricing.
- Refactoring: we did the right builds — #1, the refactor baselines, the pipe, the card — and they're green; my Clash-G trace stands (40-file rename orphans 40/40, neutral rename → LostViaRename != Catch, extract-module invisible to both halves). But six green islands are not a prototype. The single highest-leverage move is the end-to-end WIRE (cmd + via app + SSE-fed card driven by a real settle) — it needs NO new economy and NO integrate-on-tip, only built+green parts. NEW: the card is anchored to a SINGLE line's verdict and has NO designed state for "anchor lost to a rename" — a renamed file's card will sit in in-flight FOREVER, lying about a terminal lost state. The refactor task-type breaks the SURFACE state machine, not just the oracle; it needs a fifth designed terminal state. Wire it, then give lost-via-rename its state.

Clashes touched: F (the LOOP half FLIPS to RESOLVED-IN-CODE — RunCatchCycle mints a real Catch from two real settles and the inventory-change rule survives the real reanchor path; residual is the farm-denial-end-to-end gap — no-op-churn and agent-authored-killing-test still asserted only against literals, NOT driven through RunCatchCycle — closed by this wave's log-assertion fixtures); A (the silent-vs-badge + NoOracleSignal-vs-Catch discrimination becomes CONCRETELY runnable the moment a human watches the served card — the calm-win "Tested — ship it" vs blind silence finally testable against pixels); C (untouched in verdict — land stays honestly Unintegrated; integrate-on-tip moves from spec toward the #12 build with fixtures A+B settling whether the transition survives a rebase; pricing correctly gated downstream); G (the carnage baselines stand green but the underlying punishment of behavior-preserving change is about to become USER-VISIBLE — the surface has no honest terminal state for a renamed anchor); B (capture-at-mint finally lands the self-flag + would-have-shipped + reason-tag columns the Catch type has lacked since R6 — data-only, NOT routed to attention); D (single-lane vs N-rebase named as the first concrete N-ceiling mechanic, deferred to #13); E/H/I (render-camp — the roadmap now sits ONE build from the watchable surface that unblocks them).

Verdicts updated:
- Clash F: → RESOLVED-IN-CODE on BOTH the unit and the loop (carried from the R9 #3-pipe verdict, re-confirmed on the 11-green tree). Residual is no longer oracle but ECONOMY-PLUMBING: the Catch is minted but NOT PERSISTED (capture-at-mint, this wave) and the farm-denial is asserted on literals not through RunCatchCycle (closed this wave by asserting on the event LOG). NEW persistence-layer concern logged (TDD): a tautological-tests line (real fix kills nothing) is indistinguishable from already-strong at the NoCatch token — the CatchRecord MUST carry MutantsConsidered + a reason tag or the Ledger stores un-auditable outcomes.
- Clash A: gains its first RUNNABLE acceptance experiment — a served card whose silent/Tested vs catch render is watched by a human (reassurance vs guilt), no longer a thought experiment; flips off pure-TBD toward "buildable-this-wave."
- Clash C: unchanged verdict (catch minted pre-integration, land honestly Unintegrated); integrate-on-tip sharpened into the #12 build with fixtures A (trunk-moved) + B (clean-rebase-checks-red, the load-bearing RED), and the cost-sharpening (third integrated run + per-tip re-runs) folded into the #15 benchmark gate before pricing.
- Clash G: carnage baselines remain RESOLVED-in-code; a NEW surface-level residual logged (no terminal lost-via-rename state → renamed card lies in-flight forever) — NOT a fresh clash, an additive surface-state requirement scheduled at #11.
- Clashes D, E, H, I: remain TBD; the roadmap now terminates ONE build from the surface (#10) and reaches the two-agent Board (#16) + pricing (#17) that unblock them.

New clashes opened: NONE at the target level — the #2 (R9) wave shipped green and the council converges on the end-to-end wire as the next build. The build-ORDER/scoping splits (TDD: capture-at-mint as literal #1; CI/CD: integrate-on-tip as #1; Game: timed-trace-first) are scheduling sub-disputes INSIDE the agreed wave — per the Round-7 bar they do not block convergence. The chair folds capture-at-mint into the #10 wave (Systems pairs seam + record), keeps integrate-on-tip at #12 because the pipe already returns honest Unintegrated (CI/CD's non-negotiable is met; CI/CD blocks pricing, not the wire), and sequences the timed trace to #14 (the wire ships a live first-watch on the current trace; tempo TUNING needs the time axis). UX's two-quiet-meanings, TDD's audit-the-WHY, and Refactoring's lost-via-rename are additive sharpenings folded into #10/#11.

Decisions (the marching orders):
1. NEXT BUILD (#10, the converged wire): a runnable cmd/agntpr (or internal/app) that boots a real Via app over a live HTTP/SSE server (the first prod ListenAndServe/Flusher), mounts surface.ReviewCard, and on a real settle drives orchestrator→RunCatchCycle, feeding the result into the card over SSE — a human opens a browser and WATCHES one verdict go in-flight → resolve to the real Outcome of two real settles. NO new render logic, NO new oracle: this is the SSE server + the pipe→card EDGE. Built via tdd-rygba. Acceptance: a real Outcome from a real two-settle pipe reaches a rendered SSE frame, no flash/no-freeze on in-flight→resolved.
2. #10 PREREQUISITE sub-brick A, BUILD FIRST: the pipe→card PRESENTER — a pure mapping from pipe state → card verdict token that carries WHY a card is quiet (Tested vs in-flight vs NoOracleSignal vs the catch.Outcome). CycleResult.Outcome ALONE cannot express the surface's non-Outcome states (surface.Tested is a separate const; in-flight is the empty verdict). Map state, do not forward the bare enum, or the most-common calm-win screen collapses into the blind-silence screen. Test the pure function before the server.
3. #10 PREREQUISITE sub-brick B, BUILD FIRST (capture-at-mint, deferred 4 rounds, chair-pulled into this wave, data-only): a typed CatchRecord{Outcome, Anchor, BeforeRev, AfterRev, BeforeInventory, AfterInventory, MutantsConsidered, ReasonTag, SelfFlagged bool, WouldHaveShipped bool} appended to an event log on every real mint — NO weight, NO pricing (guards forbidden catch-weight V§13.5). The now-live mint returns a string and persists NOTHING; the un-backfillable data must be captured on the first real served mint. The ReasonTag carries the WHY so the Ledger is auditable for theater. Build the no-op-churn-writes-NOTHING assertion (on the LOG) first — it is the farm-denial claim still untested in the wild.
4. ADVERSARIAL ACCEPTANCE for #10, all end-to-end through the wired binary against the live SSE stream: (a) strengthen-test → in-flight→catch "Caught" as ONE discrete transition; (b) edit-anchored-line → quiet, NEVER "Caught" over the wire; (c) no-op-churn → NoCatch + NO CatchRecord appended (assert on the log); (d) zero-survivor → "tested" DISTINCT from operator-free "no-oracle-signal"; (e) CatchRecord byte-identical on replay; (f) 40-file rename → renamed anchors render a TERMINAL lost state (expected-RED today: card has none — locked as a visible baseline); (g) meters-off lock holds end-to-end.
5. RANKED ROADMAP to a USABLE PROTOTYPE (each with its validating experiment): [#10 + prereqs 2,3] the end-to-end wire + presenter + CatchRecord [watch a real catch resolve over SSE; farm-denial on the log; calm-win vs blind discrimination]; [#11] fifth TERMINAL surface state orphaned/lost-via-rename [40-file rename → terminal lost, not in-flight-forever]; [#12] integrate-on-tip {clean|conflict|checks-red} replacing the Unintegrated const [trunk-moved (A) — catch survives the rebase or pricing is hard-gated; clean-rebase-checks-red (B), the load-bearing RED]; [#13] single-lane merge queue wrapping #12 [throughput-to-zero on K branches; the first N-ceiling mechanic, Clash D]; [#14] timed/typed Trace ([]TraceEvent{T,Kind,Msg}) persisted as a replay artifact [50-event burst coalesces to one legible beat at honest tempo — Game's tempo demand]; [#15] K-concurrent-settle benchmark on the INTEGRATED cost [third-run + per-tip re-runs; MUST precede pricing]; [#16] first economy STOCK rendered + two-agent Board instrumented for idle/dwell [rework-vs-concurrency, Clash D — meters come ON here, never before]; [#17] catch PRICING against an integrated base [gated on #12(A) + #15; redeemed only against the logged CatchRecord's objective columns, never a model-inferred catch-weight V§13.5; unblocks H/E/I].
6. BLOCKERS that must be cleared before the prototype is reachable: (a) NO prod SSE server exists — the prototype is not watchable until #10's seam stands one up; this is the single gate between 11 green packages and "usable"; (b) the pipe→card edge does not exist — CycleResult.Outcome cannot express Tested/in-flight, so without the presenter the calm-win screen mis-renders as blind silence; (c) capture-at-mint is un-backfillable and the now-live mint destroys it on every run — it must land in this wave or the Ledger is born without its only objective columns; (d) the surface has NO terminal state for a lost-via-rename anchor — a renamed card lies in-flight forever (cleared at #11); (e) land must NEVER be fake-merged — it stays Unintegrated until integrate-on-tip (#12), and catch PRICING is gated behind #12(A) + the #15 benchmark; (f) the trace carries no time axis — cadence cannot be tuned until #14, though the wire can ship a live first-watch without it.
7. RISKS.md additions/standing: log the persistence-layer audit gap (CatchRecord must carry MutantsConsidered + ReasonTag or NoCatch-tautological is indistinguishable from NoCatch-already-strong); log the integrated-cost multiplier (oracle×2 + integrated run + per-tip queue re-runs); the set-not-multiset under-crediting gap and the four R5-7 code-level risks stand; run the K-concurrent-settle benchmark on the integrated cost path before any catch pricing.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (6th consecutive round): the R9 wave SHIPPED green — 6/6 lenses ratify the result (a real Catch minted from two real settles + a card rendering all four outcomes + Tested + in-flight, land honestly Unintegrated) — and the council converges on the END-TO-END WIRE as the next build: an SSE-served, runnable surface feeding a real RunCatchCycle into the card, so the first real catch becomes a beat a human WITNESSES instead of a row in a Go test. Four lenses (UX, Game, Systems, Refactoring) advocate the wire as #1 outright; the chair folds capture-at-mint (TDD's #1, paired by Systems) into the SAME wave as un-backfillable and pulls the presenter sub-brick first, and keeps integrate-on-tip at #12 (CI/CD's #1) because the pipe already returns honest Unintegrated and CI/CD blocks pricing, not the wire. No new target-level clash. The roadmap REACHES a usable Via/SSE prototype, a merge queue, and a priced catch against an integrated base — not an economy primitive in a vacuum. The next event is a BUILD — the presenter + CatchRecord + the SSE-served wire (#10) — not another deliberation round.

## Round 11 — the round the thesis caught the project lying: the served card resolves to a confidently-wrong terminal on rename — 2026-06-04

Trigger: the Round-10 wire (#10) BUILT — the council reconvenes for the third consecutive wave on post-build evidence, charged (full-prototype goal) to confirm/refine the immediate next build against the REAL code and rank the slices to a usable prototype. The Round-10 roadmap named #11 = "the missing TERMINAL lost-via-rename card state (a renamed file currently spins 'Oracle running…' forever)". Build state re-verified in code: 13 green packages (`env -u GOROOT go test -race ./...` → ok), the R10 wave landed cmd/agntpr + internal/app.LiveServer (first prod `http.ListenAndServe`/SSE), surface.PresentVerdict (pipe→card presenter), ledger.CatchRecord (capture-at-mint, data-only).

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD, Refactoring). No new lens.

New evidence on the table — THE VERIFIED CORRECTION (all six independently opened present.go/card.go/pipe.go/pipe_cycle.go and found the same defect; the roadmap's #11 premise was WRONG):
- The roadmap claimed a renamed file "spins 'Oracle running…' forever." FALSE. A LostViaRename anchor flows RunCatchCycle → CatchAcross fail-closes to `catch.NoOracleSignal` (pipe.go:33-36) → PresentVerdict(running=false, NoOracleSignal) → the card RESOLVES (does NOT spin) to `data-state="no-oracle-signal"` / "No oracle signal" / **"This line has no mutable operator — the oracle cannot speak to it."** (card.go:66-68). On a renamed file that detail is a CONFIDENT FALSEHOOD — the truth is the file was git-renamed and the anchor lost. A spinner is honest about not-knowing; this asserts a false fact with the visual authority of a settled verdict — the exact confidently-wrong-terminal the whole confirmed-catch economy exists to prevent, now living in the shipped surface.
- NoOracleSignal is TRIPLE-OVERLOADED across two seams: `catch.Detect` returns it for the genuine empty-inventory case (catch.go:52-54); `CatchAcross` fail-closes BOTH `reanchor.Outdated` (anchored line edited) AND `reanchor.LostViaRename` (file renamed) to it (pipe.go:33-36). Three distinct truths, one token.
- `reanchor.State` — the type whose entire reason for existing is to distinguish Same/Moved/Outdated/LostViaRename (reanchor.go:24-36) — is computed in RunCatchCycle at pipe_cycle.go:67, branched on (74-89), even stringified into the flat Trace at line 88, and then DROPPED: `CycleResult` (pipe_cycle.go:31-44) carries `Outcome` but no `State`/`Reason`. The presenter is structurally blind to a distinction the layer below it had in hand and threw away (Clash C — the lossy seam — made concrete).

Per panelist:
- UX: wire #10 was the first time the surface told the truth about WHY a card is quiet (calm-win "Tested — ship it" vs blind "No oracle signal", RenderVerdict shared by static card + SSE wire) — what she fought 10 rounds for. BUT the card now LIES on rename: card.go:66-68 asserts a false REASON, worse than a spinner. #1: split the triple-overload at the seam — thread reanchor.State (or a typed SignalCause: NoMutableOperator | LineEdited | FileRenamed) through CycleResult → PresentVerdict so three quiet states render three TRUE details. The bug is "resolves to a confident falsehood," not "spins forever" → a copy/data-plumbing fix, not a new spinner state. Second-order: rename-cliff itself unreliable (heavy edit → delete+add → Outdated, not LostViaRename).
- Game design: wire #10 gave a real witnessed beat to grade — in-flight→resolve over httptest SSE. But the card lies on its BEST-feeling resolution; "honest hooks, no lying" is the north star and this is the only lie in the shipped surface. #1: carry reanchor.State onto CycleResult and split the verdict — roadmap #11 done RIGHT (not "make the spinner terminal," it already is, but "make the terminal claim honest"). Explicitly DEFERS his own pet want (typed Trace, #14): "tuning a loop that is currently telling a lie is premature — fix the lie first."
- Systems: wire #10 gave the economy its first end-to-end mint + the farm-denial invariant in CODE (ShouldRecord gates Append to Catch ONLY, ledger.go:39-41 — spam no-op settles mints nothing). #1: plumb reanchor.State through CycleResult as a typed LostReason{none|no-operator|edited|renamed} sourced from ra.State (already in hand at pipe_cycle.go:67) and split the presenter — strictly higher-leverage than #11-as-scoped because #11 was premised on a non-existent spinner; the real defect is a truth-in-labeling bug on a verdict the human ACTS on. Secondary (ranked behind): set-not-multiset keying (catch.go:39) un-credits a 2-survivor→1-kill as NoCatch — lossy mint denominator before pricing exists.
- Pragmatic TDD: wire #10 made the calm-win demonstrable (NoCatch+considered>0+survivors==0 → "Tested — ship it", fired ONLY on positive oracle evidence) and gave an auditable ledger artifact. The confidently-wrong terminal is a test that GREENS for the wrong reason — exactly the theater this lens exists to catch. #1: add a typed Reason to CycleResult; this is a PRECONDITION for #11, not separate — "#11 is unbuildable while the surface cannot SEE that the state IS lost-via-rename; there is no spinner to fix, there is a lie to un-tell." KEYSTONE RED: the edited-anchor (Outdated) fixture — it forbids the suite greening on the NoOracleSignal token alone, so Outdated/LostViaRename/NoMutableOperator MUST become string-distinguishable.
- CI/CD & Delivery: wire #10 shipped the first honest serialization fact — `Land: Unintegrated` (pipe_cycle.go:25), "Landed ≠ Merged" as the shipped default. His nextBuild is #12 integrate-on-tip (the serialization point the 1:N economy rests on; pricing #17 + benchmark #15 already gated on it). BUT concedes — in his own words — the rename correction PROVES the lossy seam: "the same lossy seam will collapse {clean|conflict|checks-red}, so #12 MUST widen the seam (carry a typed Land), not bolt a 4th overload onto NoOracleSignal." → #11's seam-widening is the prerequisite #12 must reuse.
- Refactoring: wire #10 gave the refactor-carnage failure its first SERVED surface to test on. FORMALLY RE-OPENS Clash G (refactor carnage) as live-at-the-SURFACE: it was resolved at the ORACLE seam (fail-closed, no phantom catch) but a rename is reported to the human as an operator-free line — resolved-at-oracle ≠ honest-at-surface. #1: #11 RESHAPED — thread reanchor.State through CycleResult and split the presenter so LostViaRename renders "Anchor lost: file renamed; oracle cannot follow", the prerequisite for refactor ever being an honest first-class task-type. Validating fixture: an httptest SSE renamed-file case mirroring the #10 end-to-end test.

Clashes touched: C (the lossy CycleResult.Outcome seam — MOVED from latent to actively-binding: confirmed in code that ra.State is known at pipe_cycle.go:67 and dropped from CycleResult, and ELEVATED to a design rule — every future verdict dimension is carried as an orthogonal typed field, never a new meaning bolted onto an existing Outcome token); G (refactor carnage — RE-OPENED at the surface layer: resolved-at-oracle confirmed, surface dishonesty means NOT resolved; closing gate is now #11's renamed-file surface fixture); A (the calm-win-vs-guilt experiment is unaffected — the happy-path render is honest; the defect is the no-signal path); D/E/H/I (render-camp — roadmap still reaches them at #16/#17, unchanged).

Verdicts updated:
- Clash C: → ACTIVELY-BINDING (was latent). The seam that should carry orthogonal verdict dimensions collapses them into one Outcome token, and once (rename) that produced a confident falsehood. Design rule adopted: Reason (this round) and Land (#12, already stubbed pipe_cycle.go:21-25) are orthogonal typed fields on CycleResult, never overloads.
- Clash G: → RE-OPENED at the surface (was RESOLVED-IN-CODE on the carnage baselines). Fail-closed-at-oracle is correct AND insufficient: honest at the economy/ledger layer, dishonest at the surface. Closing gate: #11's httptest SSE renamed-file fixture asserting the card does NOT claim "no mutable operator".

New clashes opened: NONE at the target level. Three "new clashes" were proposed and all three self-declared "none": TDD's is a reframe of #11's RED (accepted as the official framing); CI/CD's is a sharpening of existing Clash C (the binding design constraint carried onto #12); Refactoring's is a re-opening of existing Clash G at a finer layer (legitimate, sub-target-level). Re-opening an existing clash at a finer layer is not a NEW target-level clash; convergence stands.

Decisions (the marching orders):
1. NEXT BUILD (#11, RESHAPED, 6/6 converge on the seam-fix; 0/6 ratify #11-as-scoped because its forever-spinner premise does not exist): split the NoOracleSignal triple-overload at the pipe→surface seam. Add a typed `Reason` (orthogonal to Outcome) to CycleResult distinguishing at least NoMutableOperator (catch.Detect empty-inventory, catch.go:52-54) / AnchorEdited (reanchor.Outdated) / FileRenamed (reanchor.LostViaRename); replace the single hard-coded card.go:66-68 detail with a branch rendering each cause with a TRUE detail — the renamed case must name the rename and must NOT say "has no mutable operator". Outcome (and the ledger token) stays unchanged; Reason is the NEW dimension. Built via tdd-rygba.
2. #11 PREREQUISITE sub-brick A, BUILD FIRST: widen the seam type. Carry the cause out of `CatchAcross` itself (pipe.go — the layer that fail-closes Outdated/LostViaRename to NoOracleSignal), not just inferred by the caller, or the genuine-no-operator vs anchor-lost split stays guessed. Populate CycleResult.Reason in RunCatchCycle from ra.State (already in hand) AND from the genuine-no-operator case. RED-then-GREEN at internal/pipe.
3. #11 PREREQUISITE sub-brick B, BUILD SECOND: the presenter split. PresentVerdict (present.go) gains the Reason and maps each no-signal cause to a distinct verdict token; ReviewCard.present (card.go) renders the true details. Add SIBLING tokens — do NOT add a fourth meaning onto the NoOracleSignal string. CI/CD's binding constraint: #12's Land verdict MUST reuse this widened-seam pattern.
4. ADVERSARIAL ACCEPTANCE for #11 (real git fixtures): (a) `TestRunCatchCycle_renamedAnchor_reasonFileRenamed` — git mv the anchored file base→fix above the --find-renames threshold → Outcome==NoOracleSignal AND Reason==FileRenamed (RED today: field absent); (b) `TestRunCatchCycle_editedAnchor_reasonAnchorEdited` — edit the anchored line at fix (reanchor.Outdated) → Reason==AnchorEdited (the KEYSTONE RED: forbids greening on the NoOracleSignal token alone); (c) `TestRunCatchCycle_literalOnlyLine_reasonNoMutableOperator` — operator-free RHS → Reason==NoMutableOperator (locks the split didn't regress the one case where "no mutable operator" is TRUE copy); (d) surface/httptest SSE renamed-file case (mirrors the #10 served-card test) — card resolves to a distinct data-state with a rename-naming detail and asserts NOT-contains "no mutable operator" (Clash G's closing gate).
5. RANKED ROADMAP to a USABLE PROTOTYPE (each with its validating experiment): [#11 THIS ROUND] split the NoOracleSignal triple-overload (typed Reason + presenter split) [the four fixtures; edited-anchor the keystone RED — de-risks the project's CENTRAL thesis-failure]; [#11.5 fast-follow, non-blocking] tighten reanchor rename detection OR make FileRenamed copy honestly admit threshold-uncertainty [heavily-edited rename → statusDeleted→Outdated, assert the surface copy is not actively false]; [#12 integrate-on-tip, CI/CD standing block] replace the Unintegrated const with a real {clean|conflict|checks-red} Land verdict, REUSING #11's widened-seam pattern (typed Land, never an overload) [non-conflicting→Clean / same-hunk→Conflict / tip-makes-tests-fail→ChecksRed]; [#13 multiset survivor accounting] catch.go SET→multiset so a 2-survivor→1-kill mints PartialCatch not NoCatch [deferred behind #12 — it changes the unit-of-account]; [#14 typed Trace] []TraceEvent{T,Kind,Msg} for cadence tuning [self-deferred by Game — premature while the surface lied]; [#15 K-concurrent benchmark on integrated cost]; [#16 first economy stock + two-agent Board — meters come ON here]; [#17 catch PRICING against an integrated base, gated on #12 + #15].
6. BLOCKERS that must be cleared before the prototype is honest: (a) the served card asserts a confident falsehood on every rename/edited-anchor — the single highest-priority defect because it is the exact failure mode the project exists to prevent, now in the shipped surface; (b) CycleResult is structurally blind to reanchor.State — until Reason is a field, the presenter cannot tell rename-loss from operator-free; (c) the seam-widening discipline must land in #11 so #12's Land does not re-introduce the overload.
7. RISKS.md additions/standing: log the surface-layer confidently-wrong-terminal (NoOracleSignal triple-overload renders a false cause on rename/edited-anchor); the set-not-multiset under-crediting gap and the rename-similarity-cliff stand (now with surface consequences); the integrated-cost multiplier and the four R5-7 code-level risks stand.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (7th consecutive round): 6/6 lenses independently opened the shipped surface and found the SAME defect — a renamed file resolves to a calm, terminal, confidently-WRONG "no mutable operator" verdict — and converge on the SAME #1: thread reanchor.State through CycleResult as a typed orthogonal Reason and split the NoOracleSignal triple-overload at the presenter so three quiet states render three TRUE details. 0/6 ratify roadmap #11 AS-SCOPED (a forever-spinner terminal state) because the verified correction shows the card already resolves terminally — the brick named by #11 is the brick built, only its premise was wrong (reshape, not clash). The sole dissent is build-ORDER (CI/CD: #12 integrate-on-tip first), chair-resolved in #11's favor by CI/CD's OWN argument — #11's seam-widening is the prerequisite #12 must reuse, so #11 establishes the typed-orthogonal-field pattern and #12 follows immediately. Clash C elevated to an actively-binding design rule; Clash G re-opened at the surface layer with #11's renamed-file fixture as its closing gate; no new target-level clash. The next event is a BUILD — sub-brick A (widen the seam: typed Reason out of CatchAcross + onto CycleResult), sub-brick B (presenter split), with the edited-anchor fixture as the keystone RED — not another deliberation round.

## Round 12 — the seam pattern's first reuse: integrate-on-tip makes the catch's base SOUND — 2026-06-04

Trigger: the Round-11 #11 wave BUILT and SHIPPED green (typed orthogonal pipe.Reason; presenter split; Clash G surface-honesty RESOLVED-IN-CODE; Clash C seam discipline demonstrated as code). The council reconvenes for the fourth consecutive build-evidence wave under the full-prototype goal: confirm/refine the immediate next build, rank the slices, flag blockers. Build state re-verified in code: 13 green packages; ZERO integration primitives (`grep rebase|merge-base|integrate|onto` non-test → only the `Unintegrated` const + comment); LandState single-valued; RunCatchCycle hardcodes `Land: Unintegrated` (pipe_cycle.go:103); catch minted on an immutable pre-integration baseRev.

Panelists present: all six (UX, Game design, Systems, TDD, CI/CD, Refactoring). No new lens.

New evidence on the table (verified by reading code this round):
- #11's orthogonal-seam pattern is now settled fact in code: `CatchAcross` returns `(catch.Outcome, Reason, error)` (pipe.go:45); `pipe.Reason {None|NoMutableOperator|AnchorEdited|FileRenamed}` is a typed field on CycleResult carried BESIDE catch.Outcome; the ledger mint token is untouched (catch.go:20-34 still 4 economy outcomes, ledger.go:39 ShouldRecord==Catch only). `present.go` splits NoOracleSignal into three TRUE data-states; gate tests green.
- The catch is minted on an UNSOUND base: `resolve.go:46` records BeforeRev=baseRev (immutable pre-integration coordinate); the fix oracle runs on fixRev directly (pipe_cycle.go:81), never reconciled to trunk tip; `LandState` is a dead single-value const. A fix green in isolation can mint a real Catch on a line that conflicts with — or goes checks-red against — trunk tip. "Landed ≠ Merged" is honestly LABELED but the verdict itself does not exist.
- Land is INVISIBLE on screen: present.go never reads the Land field, so the one state a reviewer must ACT on (rebase/conflict) is the one the calm card cannot show.
- Residuals carried: the four-beat Trace is dropped (resolve.go:38 never reads res.Trace) and untyped ([]string Sprintf) — #14; set-not-multiset survivor keying under-credits (catch.go:58-59) — #13; rename-cliff coarsening (reanchor.go:141) — #11.5.

Per panelist:
- UX (→ #12): #11 made the card honest on every quiet state (per-state h.Data("state",…) markers, RenderVerdict shared by reviewer card + live wire). But Land is invisible — present.go never reads it, so a reviewer sees a verdict on pre-integration coords with zero signal trunk moved under them. Build #12 as a typed Land field AND surface it as its OWN data-state ROW (separate from the oracle verdict, never overloading it): clean→quiet/no-chrome (info gated away), conflict/checks-red→the single actionable surface with a true detail; motion only on the real clean→conflict transition.
- Game design (→ #14, the lone dissent): #11 made the resolved beat tell the truth. But the loop has NO felt tempo — pipe_cycle.go:64-100 emits an ordered four-beat Trace that resolve.go:38 DROPS; live.go streams ONE in-flight→resolved patch, so the human watches a spinner snap to a verdict feeling none of the four real beats. Build #14 (typed []TraceEvent{T,Kind,Msg} + STREAM each event to the card) — the cheapest conversion of already-produced, already-dropped plumbing into the first felt moment. SCOPING NOTE (self-labeled chair-resolvable, not target-level): #14 must be type+STREAM, never type-only.
- Systems (→ #12): #11 added a verdict-display dimension without diluting the unit of account — exactly right, and it ratified the binding rule (#12 must reuse it). But the mint is priced against an UNSOUND base (resolve.go:46 BeforeRev=baseRev, zero rebase primitives): a "catch" can be minted on code that conflicts/red on tip — value minted on code that may never integrate. #12 makes the base SOUND; no pricing (#17) can be trusted until the base it prices against is the tree that actually integrates. (Secondary: catch.go:58-59 toSet collapses same-op survivor sites → NoCatch under-credits = #13.)
- Pragmatic TDD (→ #12): #11 proved the orthogonal-seam template in code; a Land field must reuse it (new typed dimension, never an Outcome overload). The catch is minted on fixRev-vs-baseRev, BOTH pre-integration; the mint is UNCONSTRAINED by integration truth. The load-bearing RED is NOT no-conflict and NOT clean — it is clean-MERGE-but-checks-red (trunk renamed a symbol the fix calls, or tightened a shared invariant → merges clean, rebased suite FAILS). Mandatory degenerate guard: a sibling test where trunk advanced DISJOINTLY → LandClean, proving the verdict CONSTRAINS, not that rebase merely ran.
- CI/CD & Delivery (→ #12): "Landed ≠ Merged" is honestly LABELED but the verdict doesn't exist; pricing sits on a fictional base. Build integrate-on-tip {Clean|Conflict|ChecksRed} by rebasing fixRev onto trunk tip and re-running checks on the rebased tree, as an orthogonal CycleResult.Land field, built as ONE serialized merge-queue lane (not N concurrent rebases — the O(N²)/8N contention regime). Clash C stays open as #12's experiment; #12 is its resolution path.
- Refactoring (→ #12): #11 gave a future Land verdict its proven template (orthogonal field, not an overload). Land is a dead const; catch minted on an immutable baseRev never reconciled with tip. My #11.5 rename-cliff is honest-but-coarse, NOT wrong, and it RECURS on the rebased tree #12 introduces — so #12 is where it actually bites and subsumes its urgency; #11.5 becomes the fast-follow, NOT a preemption of #12. Assert each of the three Land cases keeps catch.Outcome IDENTICAL to the pre-integration run (orthogonality).

Clashes touched: C (RESOLUTION RATIFIED via #12 — the typed Land verdict on the rebased tree closes "the mint is unconstrained by integration truth", witnessed by the clean-rebase-but-checks-red RED; STILL OPEN until that code ships green); G (surface half resolved by #11; #12 extends the SAME seam to integration — present.go gains a Land row — completing the honest-card half); D/E/H/I (render/Board-gated, unchanged; roadmap reaches them at #16/#17).

Verdicts updated:
- Clash C: RESOLUTION PATH RATIFIED. #12 integrate-on-tip is the experiment AND the fix; the load-bearing closing RED is clean-rebase-but-checks-red (a green pre-integration catch going red post-integration), with a disjoint-trunk LandClean degenerate guard. Flips to RESOLVED-IN-CODE on the green #12 build. The catch.Outcome/ledger token stays byte-identical across the orthogonal Land seam (the invariant under test).

New clashes opened: NONE at the target level. Two registered notes, both self-labeled non-target-level and chair-resolved in-band: (1) Game's #14 scope (type-only is mis-scoped → ADOPTED as a binding scope-lock: #14 is "typed Trace + STREAM", its gate asserts the staged SSE sequence with monotonic T); (2) CI/CD's one-lane-vs-per-card-rebase → folded into sub-brick 12e (one serialized lane).

Decisions (the marching orders):
1. NEXT BUILD (#12, 5/6 converge; Game's #14 dissent is build-order, chair-resolved): integrate-on-tip — a typed orthogonal `Land` verdict {LandClean | LandConflict | LandChecksRed} on CycleResult, computed by rebasing fixRev onto trunk tip and re-running testCmd on the REBASED tree (reusing runOracleAt's worktree+checks path incl. the Background-ctx cleanup discipline), threaded exactly like #11's Reason (NEVER folded into catch.Outcome — the mint token stays the 4 economy outcomes), AND surfaced as its own data-state row on the card. Built as ONE serialized merge-queue lane. Via tdd-rygba.
2. PREREQUISITE sub-bricks IN ORDER: [12a] trunk-tip rebase primitive in internal/pipe (rebase fixRev onto tip in a throwaway worktree, detect textual conflict from rebase exit — the missing capability); [12b] typed Land enum replacing the Unintegrated const, populated from 12a, catch.Outcome/ledger UNCHANGED; [12c] integrated-checks re-run on the rebased tree (clean rebase + green → LandClean; clean rebase + red → LandChecksRed; conflict short-circuits to LandConflict before checks); [12d] surface the Land row (present.go gains a Land branch, separate data-state, LandClean → no chrome, Conflict/ChecksRed → one actionable row with a true detail; RenderVerdict stays the shared renderer); [12e] serialize the lane (one lane; lock the no-fan-out invariant in code/comment).
3. ACCEPTANCE FIXTURES (real git repos, end-to-end through the seam, each asserting catch.Outcome IDENTICAL to the pre-integration run): [closing RED] TestRunCatchCycle_cleanRebaseButChecksRedYieldsChecksRed (trunk advanced clean-merge but rebased suite fails → LandChecksRed); TestRunCatchCycle_landsConflictWhenFixDivergesFromTip (trunk advanced a conflicting hunk on the anchored file → LandConflict, short-circuits before checks); [degenerate guard] TestRunCatchCycle_landsCleanOnNonConflictingTip (trunk advanced a DISJOINT file → LandClean — proves the verdict constrains, fires only on real breakage); TestResolve_cleanLandRendersNoIntegrationChrome; TestReviewCard_rendersConflictLandStateAsActionableWithoutClaimingClean.
4. RANKED ROADMAP: [#12 THIS ROUND] integrate-on-tip (sound base, all pricing/trust gates on it); [#11.5 fast-follow AFTER #12] rename-cliff (recurs on the rebased worktree #12 introduces — do NOT preempt #12); [#13] multiset survivors (refine an already-sound token); [#14 type+STREAM, near fast-follow] typed Trace + stream the four beats (first felt tempo; unblocks pacing/meters); [#16] first economy stock + two-agent Board (meters come ON here); [#17] catch PRICING against an integrated base (gated on #12 + #15 K-concurrent benchmark).
5. BLOCKERS: (a) the mint's base is unsound until #12 — pricing/trust are fictional until Land is the tree that actually integrates; (b) Land is invisible on screen until 12d; (c) the rebase MUST be one serialized lane, never N concurrent rebases (12e); (d) #14's four beats are produced and dropped — the felt loop waits on #12 so it streams beats that mean something.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (8th consecutive round): 5/6 lenses (UX, Systems, TDD, CI/CD, Refactoring) advocate #12 integrate-on-tip as #1, arriving from independent vantage points on the SAME structural reason — the catch is minted on an unsound pre-integration base, and #11's orthogonal-seam pattern is the proven template the Land verdict must reuse. The sole dissent (Game: #14 typed-trace-first) is build-order, chair-resolved against #12 on the panel's standing no-fake-XP principle (streaming honest beats that culminate in a verdict minted on a base that may never integrate makes the loop FEEL honest while staying economically fictional); #14 follows #12 closely with the Game Designer's scope contribution preserved as a binding type+STREAM lock. No new target-level clash. The next event is a BUILD — 12a→12e in order, gated on the five fixtures with clean-rebase-but-checks-red as the closing RED — not another deliberation round.

## Round 13 — the felt loop: typed + STREAMED trace, the council's cleanest mandate (6/6) — 2026-06-04

Trigger: the Round-12 #12 wave BUILT and SHIPPED green (integrate-on-tip; typed Land verdict; Clash C resolved-in-code). The council reconvenes for the fifth consecutive build-evidence wave under the full-prototype goal: confirm/refine the immediate next build, rank the slices, flag blockers. Build state re-verified: 13 green packages.

Panelists present: all six. No new lens.

New evidence on the table (verified by reading code this round):
- #12 was the SECOND reuse of #11's orthogonal-seam pattern (Reason, then Land), now a ratified design rule. A cycle now does 3 SERIAL full-suite runs (runOracleAt base + runOracleAt fix + integrateOnTip's check.Run) and produces ~6 genuinely-timed beats (settled base → oracle base → settled fix → oracle fix → catch → land, appended at pipe_cycle.go:73-75/94-96/109/115).
- BUT the felt loop is dead at the seam: CycleResult.Trace is a flat []string of fmt.Sprintf lines (no T, no Kind); resolve.go reads Outcome/Reason/Land/After and NEVER res.Trace; live.go flushes ONE in-flight→resolved patch via a single Stream poll. The human watches a 100ms spinner snap to a verdict over seconds of real work, feeling none of the beats. #12 minted a NEW Land beat and lengthened the wait, so MORE honest tempo is now discarded — the gap widened, it did not close.
- Residuals carried (de-prioritized by the lenses that own them): #13 SET-not-multiset survivor keying (catch.go) mis-files a 2-same-operator-survivor→kill-1 as NoCatch — a narrow lower-edge corner (PartialCatch handles cross-operator shrink), under-credits a corner not the spine; #11.5 rename-cliff coarseness (honest-but-coarse, recurs on the rebased tree, no consumer until the loop is felt); the integrated-cost multiplier (3 suites/cycle) is unquantified — #15 benchmark.

Per panelist (ALL SIX advocate #14):
- UX: #12 gave the card a second honest row (Land) but deepened the felt-time problem — the only motion is a time-filling spinner, the exact "motion that doesn't report a state change" the lens forbids. Build #14: type Trace as []TraceEvent{T,Kind,Msg}, surface it through Resolution, stream each beat as a REAL state change terminating in the verdict. Fixture: assert the staged SSE sequence (monotonic T, Kind order), card stays in-flight through intermediate beats, terminal frame verdict == PresentVerdict; a sibling Outdated fixture asserts the oracle-fix beat is ABSENT (reports the real path, not a fixed animation).
- Game design: ~5-6 genuine beats over seconds of real work, all discarded; the human feels none of the integration drama #12 just made real. Build #14 type+STREAM (binding scope-lock, two rounds, NEVER type-only): emit each beat on a channel, drain it in the existing via.Stream 100ms poll into a new c.Beats cell. The streaming infra ALREADY EXISTS (proven by #12) — #14 adds the per-beat write, not new infra. Gate: beats arrive as SEPARATE flushes over wall-clock, not one batch — the gate must FAIL a type-only refactor.
- Systems: the mint is sound (catch.Outcome byte-identical, Land orthogonal). #13's multiset lossiness is REAL but narrow (under-credits a corner, ShouldRecord==false by farm-denial). #14 is #1 because every downstream economy primitive — meters (#16), integrated-cost pricing (#17), the K-concurrent benchmark (#15) — needs a TIMED typed event stream as its substrate; the integrated-cost multiplier is currently invisible. #13 is a fast-follow that unblocks nothing.
- Pragmatic TDD: #12's REDs CONSTRAIN. But the current green (single terminal-frame await, live_test.go:42) proves the verdict transition fires, NOT that any tempo is delivered — a type-only Trace refactor would green VACUOUSLY against that test. The gate that makes #14 real: collect the ORDERED SSE frame list and assert ≥4 staged beats before the verdict, strictly-monotonic T, terminal verdict == PresentVerdict. A typed Trace that still snaps FAILS it.
- CI/CD & Delivery: #12 encoded "Landed ≠ Merged" honestly AND confirmed the cost worry — 3 serial full suites/cycle, re-run per SSE connect. The single lane is synchronous (no queue over K branches), the integrated-cost multiplier is unquantified (#17 pricing has no cost denominator). #14 type+STREAM the existing beats with the terminal beat carrying the Land verdict; #14's timed stream is the measurement substrate the #15 benchmark needs.
- Refactoring: the felt loop is dead — no tempo exists to refactor against. #14 is a clean characterization-then-retype refactor over the 6 existing append sites + streaming through live.go's Stream loop. It is #1 NOW: the first felt tempo and the only thing that unblocks pacing/meters (#16). #11.5/#13 are accuracy refinements on already-honest paths with no consumer until the loop is felt — non-urgent.

Clashes touched: NONE at the target level. #14 closes a missing-CAPABILITY gap (felt tempo), not a doc contradiction (catalogue holds at 12). The Game Designer's two-round type+STREAM scope-lock is now adopted verbatim by all six as the gate definition — convergence, not a clash. #13 (mint-denominator fidelity, adjacent to the mutation 0-survivors-ambiguous / coverage-map findings) and #11.5 (rename-similarity cliff) are sequenced as fast-follows, not contested as alternative #1s. D/A/E/H/I remain TBD (Board/meters, #16).

Verdicts updated: none of the §3 clashes move (F/G/C already resolved-in-code; D/A/E/H/I gated on #16). #14 is a capability build, not a clash settler — its value is unblocking the downstream economy primitives that DO settle A/D/H.

New clashes opened: NONE. 6/6 on #14, zero new target-level clashes — exceeds the Round-7 bar.

Decisions (the marching orders):
1. NEXT BUILD (#14, 6/6 — the cleanest mandate of the council): typed + STREAMED Trace. Type CycleResult.Trace as []TraceEvent{T,Kind,Msg}, surface it through Resolution, and stream each beat to the LiveCard as its own SSE patch AS the pipe transition lands, terminating in a verdict frame matching surface.PresentVerdict. BINDING SCOPE-LOCK (panel-ratified): type+STREAM, gate on the staged SSE sequence, NEVER type-only — a typed Trace that still resolves in one terminal snap FAILS the gate.
2. PREREQUISITE sub-bricks IN ORDER: [SB1] characterize-then-retype — lock the existing 6-beat ordering with a characterization test on CycleResult.Trace, THEN retype Trace from []string to []TraceEvent{T,Kind,Msg}, stamping T at the REAL transition (after the work) with Kind ∈ {settle-base, oracle-base, settle-fix, oracle-fix, catch, land}; [SB2] emit each beat on a nil-safe chan<- TraceEvent at the real pipe boundary (a nil channel preserves the batch API for existing tests; the oracle-fix beat is conditionally absent on the Outdated/LostViaRename branch — the stream reports the real path); [SB3] carry Trace through Resolution, PresentVerdict UNCHANGED, Land stays orthogonal; [SB4] live.go drains the beats channel per-beat into a new c.Beats via.StateTabStr cell inside the existing via.Stream poll (adds the per-beat write, not new infra), terminal frame writes Verdict+Land as today; [SB5] surface.RenderBeats renders the streamed Kinds as a 3rd orthogonal row (one row never speaks for another).
3. ACCEPTANCE FIXTURES: [A, load-bearing] extend live_test.go — capture the FULL ordered tc.SSE() frame list, assert ≥4 staged beats in pipe Kind-order with strictly-monotonic T before the verdict frame, card stays in-flight through intermediate beats, terminal verdict == PresentVerdict + data-state=land-clean (a type-only Trace flushing all beats at the end FAILS); [B, separate-flush proof] a gated/slow cycle so beats arrive as SEPARATE flushes over wall-clock, killing the batch-in-one-snap cheat; [C, real-path] an Outdated-anchor fixture asserts the oracle-fix beat is ABSENT; [D, beats⊥verdict] reuse the clean-rebase-but-checks-red pair so earlier beats show a green catch while the terminal land beat is LandChecksRed — beats and Land independently felt.
4. RANKED ROADMAP: [#14 THIS ROUND] typed+STREAMED Trace (first felt tempo; the substrate every downstream primitive needs); [#13 fast-follow] multiset survivor accounting (narrow corner, RED ready: 2 same-operator survivors → kill 1 → currently NoCatch); [#15] integrated-cost benchmark (3-serial-suite multiplier; #14's timed stream is the measurement substrate); [#16] meters/Board pacing (needs #14's timed stream); [#11.5] rename-cliff fidelity (non-urgent); [#17] pricing against integrated cost (blocked on #15).
5. BLOCKERS: (a) the felt loop is unobservable+untuneable until #14 — #16's meters/Board have nothing to pace against; (b) the integrated-cost multiplier is invisible on the wire until #14's timed stream exposes it; (c) the gate must assert SEPARATE staged flushes + terminal-verdict match, or #14 greens vacuously as a type-only refactor.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (9th consecutive round, 6/6 — the cleanest mandate of the protocol): all six lenses independently land on #14 typed+STREAMED Trace as #1 NOW, with the identical mechanism and the Game Designer's two-round type+STREAM scope-lock now unanimous as the gate definition. The convergence is causal: #12 added a Land beat and lengthened the wait, so MORE honest tempo is discarded — the felt-loop gap widened. #13/#11.5 are fast-follow accuracy refinements explicitly sequenced behind #14 by the lenses that own them (build-order under an agreed frame, not dissent). The next event is a BUILD — SB1 (characterize-then-retype) → SB5 (the beat row), gated on the staged-SSE-sequence fixtures with the separate-flush proof as the load-bearing RED — not another deliberation round.

## Round 14 — the economy is logged but never SHOWN: render the first retrospective stock — 2026-06-04

Trigger: the Round-13 #14 wave BUILT and SHIPPED green (typed + STREAMED Trace; the felt loop — RenderBeats accrues settle-base→…→catch→land live over the seconds of real oracle+rebase work). The council reconvenes for the sixth consecutive build-evidence wave under the full-prototype goal. Build state re-verified: 13 green packages; the watchable single-card wire is real and honest (beats/verdict/land rows). RISKS meta-finding in scope: the build order de-risked the DESIGN pipe thesis; the 8 scoring/trust-integrity risks live in the VISION trust-economy thesis — the groundbreaking, under-de-risked half — still largely unbuilt.

Panelists present: all six. No new lens.

New evidence on the table (verified by reading code this round; FIVE lenses independently grep-confirmed the same fact):
- The economy is fully LOGGED and never SHOWN. `internal/ledger/ledger.go` defines `Records()` (line 82); `CatchRecord` captures every mint-time fact (SelfFlagged, WouldHaveShipped, ReasonTag, before/after inventories); `Append` enforces farm-denial (refuses any non-Catch outcome, ShouldRecord==Catch only); `live.go` appends `res.Record` on mint. BUT `Records()` is called from NO surface or app code outside tests, and `grep Focus|Trust|Treasury|Board` as rendered state → 0. The ledger is WRITE-ONLY at the surface.
- This is WHY economy Clashes A/D/H have stayed un-adjudicable across every round: there is no number on screen to argue guilt-vs-diagnostic (A), shop-vs-context-switch-tax (D), or Ledger framing (H) over.
- `live.go` is single-instance by construction (liveState is a package var, "one Lead, one card", re-runs the whole cycle per tab); Via mounts compositions zero-value per tab with no constructor injection — so a two-agent Board is NOT additive, it forces a per-session keying rewrite of the liveState global. That coupling, not the stock, is the big-bang risk in #16.

Per panelist:
- UX (→ #16 stock): #14's RenderBeats proved "felt loop" honest on the SAME pure-render seam (state→h.H, no economy import). Clashes A/D/H stay un-adjudicable with no rendered meter. #1, smallest brick: a RetrospectiveLedger row — RenderCatches/RenderStock reading Records(), a calm append-only tally of PAST catches, meters OFF — a tally of things that already happened, so it cannot induce guilt (Clash A) by construction, and the first thing on screen that outlives one card. Fixture forbids any data-state="meter"/gauge node.
- Game design (→ #16 stock): #14 gave the felt BEAT but a beat is a drum hit, not a GAME — no cross-card accumulation. #1, smallest honest brick: render ONE real stock — the lifetime confirmed-catch count + reason tally from Records() — as a fourth row that climbs the instant a catch mints. Honest by construction: ShouldRecord==Catch-only means it can only count real mints, never fake XP. Negative case: a NoCatch/NoOracleSignal cycle leaves the stock put.
- Systems (→ #16 stock): the mint is sound and persisted but the ledger is WRITE-ONLY at the surface — logged, never read back, never summed, never spent. #1: render ONE honest stock — Confirmed Catches count from Records() filtered ShouldRecord==Catch — the first time the mint becomes a HELD quantity, not just an append. Pure read-side, no pricing (a count equals len(Records()), no inferred weight).
- Pragmatic TDD (→ #16 stock): #14 adds no model-inferred value, only timing of real work; the ledger is still write-only. #1: a pure projection ConfirmedCatches(recs) over the ledger, rendered as ONE stock. RED that CONSTRAINS: (1) N Catch records → Stock==N (a derivation, not a counter); (2) a no-op-churn/NoCatch/NoOracleSignal/PartialCatch run appends nothing → Stock unmoved (farm-denial rendered); (3) re-Open the Log → identical N (pure function of persisted facts, no hidden state).
- CI/CD & Delivery (→ #15 benchmark, the lone dissent): #14's beat row is a live latency tape of the 3 serial full-suites, but it times ONE lane. live.go OnConnect re-runs the ENTIRE 3-suite cycle per SSE connect with NO queue/cap/cost meter — N tabs = 3N concurrent full-suites; integrateOnTip per-tip re-runs multiply under a moving-tip Board (8N). #1: #15 — a benchmark over RunCatchCycle measuring integrated cost = suite-runs × K-concurrent, asserting the 3-serial-suites/cycle invariant, so #16's Board has a measured safety ceiling before meters come on. #17 pricing has no cost denominator without it.
- Refactoring (→ #16 stock): the Board/queue/second-agent is what forces the per-session liveState rewrite (the real big-bang), NOT the stock. #1, ONLY the smallest brick: a read-only StockCard calling Records() and rendering ONE stock via a new pure surface.RenderStock, copying the RenderBeats shape — no second agent, no queue, no per-session keying. It reuses the single-writer ledger + proven render shape, so it cannot sprawl liveState. The Board stays deferred.

Clashes touched: A (guilt-vs-diagnostic — PRE-EMPTED by construction this brick: retrospective append-only tally, meters OFF, no live gauge; the fixture forbids any data-state="meter"/gauge node — but formal A adjudication deferred to a post-#16 round with the stock on screen); D (1:N shop-vs-tax — still UN-adjudicable: the brick is single-card/single-agent by design; D needs a second agent, deferred with #16's later bricks behind #15); H (Trust-Ledger framing — partially touched: the stock is the first rendered projection of the CatchRecord ledger, making H ARGUABLE for the first time; full framing adjudication deferred until the stock + reason-tally render). All three move from un-rulable (panel-unanimous "no rendered meter") to rulable-next-round.

Verdicts updated: A/D/H remain TBD but their gating blocker — "no rendered economy surface to argue over" — is the thing #16's first brick removes; the next round (stock on screen) is where A/D/H first become adjudicable. No §3 clash flips this round (this is a capability build that UNBLOCKS the economy clashes rather than settling one).

New clashes opened: NONE at the target level. The single divergence (#15-vs-#16 build-order) is resolved by scoping — #16's first brick runs ZERO additional concurrent cycles (read-only render over persisted data on the single-instance wire), so #15's safety ceiling is not yet load-bearing; #15 converts from a blocker into a HARD prerequisite the moment #16's later bricks add a second agent or a queue.

Decisions (the marching orders):
1. NEXT BUILD (#16, scoped to its SMALLEST honest brick; 5/6 converge): render ONE retrospective economy STOCK — a read-only "Confirmed Catches" count + reason/self-flag/would-have-shipped tally derived PURELY from `ledger.Records()`, via the proven RenderBeats/RenderVerdict/RenderLand pure-render seam. NOT the two-agent Board, NOT a queue, NOT pricing/weights/meters, NOT a per-session liveState rewrite. It closes the write→read loop #14 left open. Retrospective (a tally of facts that already happened), meters OFF — pre-empts Clash A by construction.
2. PREREQUISITE sub-bricks IN ORDER (each via tdd-rygba): [SB1] pure projection — `ConfirmedCatches(recs []ledger.CatchRecord) Stock` (Count + per-ReasonTag tally + SelfFlagged/WouldHaveShipped sub-counts), a total function of the records, no in-memory counter; [SB2] pure render — `surface.RenderStock([]ledger.CatchRecord) h.H` in a new stock.go mirroring beats.go (h.Class+h.Data("state","stock")), empty slice → calm zero/empty row with NO meter/gauge/percentage affordance, marker disjoint from beats/verdict/land; [SB3] read-only mount — a Stock row in LiveCard.View calling RenderStock over liveState.log.Records() RE-DERIVED on connect (do NOT push-increment over SSE, do NOT key liveState per-session — keeps the single-instance wire untouched).
3. ACCEPTANCE FIXTURES: [projection] N Catch records → Stock.Count==N + tallies match; re-Open the Log → identical Stock (pure function of persisted facts); [farm-denial rendered] Append refuses NoCatch/NoOracleSignal/PartialCatch/no-op-churn (ShouldRecord==Catch-only) → Records() unchanged, Stock.Count stays put (can never inflate); [render contract] empty → calm zero row, NO gauge/meter/percentage node, NO data-state="meter" (the anti-guilt contract), marker disjoint from beat/verdict/land; [live-wire] Append 2 Catch records, mount LiveCard, connect → rendered stock reads 2, beats/verdict/land rows unaffected, liveState NOT keyed per-session.
4. RANKED ROADMAP: [#16 smallest brick THIS ROUND] retrospective Confirmed-Catches stock, read-only, single-user, meters-OFF; [#15] integrated-cost benchmark (instrument runOracleAt+integrateOnTip with an atomic counter; assert 3-serial-suites/cycle; K=1..3 concurrent → suite-runs=3N + per-tip re-runs) — HARD prerequisite before any CONCURRENT/N-agent Board or pricing; [#13] multiset SET-keying fast-follow (narrow corner); [#17] pricing/weights on the stock — BLOCKED on #15 + the stock existing; [#16 later bricks] queue + concurrency cap + second agent + per-session liveState keying (the big-bang rewrite) — BLOCKED on #15's measured ceiling, explicitly deferred.
5. BLOCKERS: (a) the economy is write-only at the surface — #16's first brick is the read side; (b) A/D/H stay un-adjudicable until a number renders — this brick makes A pre-empted-by-construction and H arguable, D still needs a second agent; (c) the second-agent Board forces the liveState per-session rewrite — deferred behind #15's measured ceiling; (d) #17 pricing has no cost denominator until #15.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (10th consecutive round): 5/6 lenses converge on #16's smallest honest brick — render ONE retrospective Confirmed-Catches stock read-only from the ledger, closing the write→read loop the felt loop (#14) left open and putting the first auditable economy number on screen. The lone dissent (CI/CD #15 integrated-cost benchmark) is build-order, chair-resolved by scoping: the stock adds ZERO concurrency, so #15's safety ceiling is not yet load-bearing — #15 is ratified #2 and the hard prerequisite for the concurrent Board (#16 later) and pricing (#17). No new target-level clash; the retrospective meters-OFF framing pre-empts Clash A by construction, and A/D/H become adjudicable for the first time once the stock renders. The next event is a BUILD — SB1 projection → SB2 render → SB3 read-only mount — not another deliberation round.

## Round 15 — cost must be a COUNTED invariant before it is a measured ceiling: the integrated-cost benchmark — 2026-06-04

Trigger: the Round-14 #16 smallest brick BUILT and SHIPPED green (the first economy STOCK rendered — surface.RenderStock, a read-only ledger.ConfirmedCatches projection). The council reconvenes for the seventh consecutive build-evidence wave. Build state re-verified: 14 green packages; the watchable single-card wire now SHOWS the economy (stock/beats/verdict/land rows).

Panelists present: all six. No new lens.

New evidence on the table (verified by reading code this round; the cost hazard is worse than the "3 serial suites" shorthand):
- A catch cycle (pipe_cycle.go) = runOracleAt(base) + runOracleAt(fix when Same/Moved) + integrateOnTip; each oracle run is `mutation.Run` firing `runTests` ONCE PER MUTANT (runner.go:243) across maxWorkers=8 WITHIN the cycle. So a cycle ≈ (M_base + M_fix + 1) full-suite executions, M of them CPU-saturating.
- `LiveCard.OnConnect` (live.go:93 → ResolveStreaming) fires the WHOLE cycle UNCONDITIONALLY per SSE connect with ZERO dedup — grep confirms no sync.Once / cache / queue / Semaphore / SetLimit in live.go. N tabs = N×(M_base+M_fix+1) suite-execs concurrently TODAY; a moving-tip Board is the ≈8N regime `integrateOnTip`'s own comment (pipe_cycle.go:163-166) names in PROSE and leaves unenforced.
- The only existing measurement, `BenchmarkRunManySites` (runner_bench_test.go), times ONE oracle run on a 30-site fixture and reports survivors + ns/op — it measures neither the CYCLE nor any K-concurrent regime and asserts NO invariant: a vanity number, not a constraint.
- `liveState` is a package var keyed by NOTHING (live.go:37, set once in NewServer); every tab's LiveCard reads the SAME cfg+log. A second agent forces a per-session keying rewrite of that global — the real big-bang. live_test.go has 4 tests but ALL drive ONE connect; NONE pins two-concurrent-connect behavior (the exact thing the rewrite changes).

Per panelist (ALL SIX advocate #15):
- UX: #16 gave the economy its first pixel (calm retrospective stock, pre-empts A). But a Board without a cost ceiling is a "thundering herd I'd be designing blind." #15 is invisible AS A SCREEN but is the gate that makes a CALM Board designable — it turns "how many cards can breathe at once" from guess to a measured K_max the queue caps at.
- Game design: the stock now accumulates (the tycoon's "you've earned N"), but it's one workbench, no shop. Clash D is unanswerable with one card. WANTS the Board but "will not build a tycoon shop on an unmeasured tick budget" — 3N→8N serial suites with no number means he can't tell shop from slideshow. #15 gives the per-action cost a management game's whole feel depends on.
- Systems: the catch is now a SHOWN unit of account but not SPENT — no conversion, no scarce resource traded. Pricing (#17) is unblocked on the stock axis but STILL has no cost denominator. Pricing a catch without the integrated cost-to-mint = a denominator-free conversion = the cheat the lens exists to prevent. #15 exposes the 3N→8N multiplier as a measured number.
- Pragmatic TDD: the stock proved the projection pattern green and bought ZERO cost-safety. The honest first brick is NOT a wall-clock bench (a vanity number) — it is an atomic suite-exec COUNTER threaded through runTests + a TEST asserting RunCatchCycle fires the suite EXACTLY (M_base+M_fix+1) times. "Cost must be a counted invariant before it is a measured ceiling, and before any Board rewrite can be characterized."
- CI/CD & Delivery: per-cycle cost is ~(2M+1) suite-execs, fired per connect with no global queue/cap; integrateOnTip's doc NAMES the O(N²)/8N regime as unenforced prose. #15 (the measured ceiling + cost denominator) is his standing #1; building the second card before this number exists ships the 8N multiplier the code already warns about.
- Refactoring: the second agent is a big-bang rewrite of the untested liveState global. "I refuse to rewrite an untested global." #15's K-concurrent-connect harness IS the characterization fixture the rewrite needs — measure-before-refactor AND test-before-refactor collapse into one build. The Board stays gated until #15 gives both the cost number and the green two-connect baseline.

Clashes touched: A (pre-empted in the small by #16's retrospective meters-OFF stock; full adjudication needs a live meter on a loop); H (arguable-in-the-small — calm retrospective, not a dread-gauge — un-resolvable at scale until a Board exists); D (still UN-adjudicable — one card, one package-var liveState; #15 makes a CALM Board designable but does not itself touch D — D moves only at the Board brick). No §3 clash flips (a measurement/instrumentation build that gates the slices which DO settle A/D/H).

Verdicts updated: none flip; #15 is the gating prerequisite that makes the Board (which settles D and A-on-a-loop) safe to build — it converts the unenforced 8N prose warning into a measured, regression-guarded ceiling.

New clashes opened: NONE. 6/6 on #15, zero new target-level clashes. The two divergences (TDD: counted-invariant-before-wall-clock; Refactoring: the K-connect harness is the rewrite's characterization snapshot) are build-order refinements INSIDE the agreed brick, both ADOPTED into the sub-brick ordering.

Decisions (the marching orders):
1. NEXT BUILD (#15, 6/6 — scoped as an INSTRUMENTED-INVARIANT brick, NOT a vanity timer): the K-concurrent integrated-cost benchmark — counted suite-execs first, characterization second, wall-clock ceiling third.
2. PREREQUISITE sub-bricks IN ORDER (each via tdd-rygba): [SB1, RED today] thread an atomic suite-exec counter through the cost path (a counter hook at the `mutation.runTests` chokepoint — the single point every oracle suite-run passes through — surfaced as a test seam, e.g. an injectable testCmd that counts or an Options/callback hook); a test runs RunCatchCycle on a deterministic fixture (known M_base/M_fix on the anchored line) and asserts suite-execs == M_base + M_fix + 1 EXACTLY (deterministic, no sleeps) — converts the unquantified multiplier into a pinned invariant; [SB2, RED today] a test fires K=3 LiveCard.OnConnect cycles concurrently via via.WithTestServer, gated on a SYNC BARRIER (not time.Sleep), asserts suite-execs == K×(M_base+M_fix+1) (proving the uncapped per-connect fan-out) AND captures the Refactoring characterization snapshot — all K connects observe the SAME liveState cfg/log and all K appends land in the one Log (the shared-global behavior the future per-session rewrite must preserve-or-deliberately-change); [SB3] BenchmarkConcurrentCycle K∈{1,2,4,8} on a fixed fixture, b.ReportMetric on suite-execs, p50/p95 wall-clock/cycle, ratio-vs-K=1, emitting the K_max where per-cycle wall-time crosses a declared budget.
3. ACCEPTANCE FIXTURES: [counted invariant] TestRunCatchCycle_firesExactlySuiteExecsPerCycle (== M_base+M_fix+1; RED today — no counter exists); [uncapped fan-out + characterization] TestLiveCard_perConnectFanOutIsUncapped (K=3 behind a sync barrier → counter==K×cycle AND all K read identical cfg/log AND all K records in the one ledger); [ceiling] BenchmarkConcurrentCycle (stable printed K_max + ns/cycle the Board cap and #17 pricing both cite as the denominator; optionally fail loud if a K=1 cycle exceeds a declared budget — turning the pipe_cycle.go:163 prose warning into a measured ceiling).
4. RANKED ROADMAP: [#15 THIS ROUND] instrumented-invariant integrated-cost benchmark (counter → characterization → wall-clock ceiling; the K_max + cost denominator + two-connect green baseline); [#16-Board-brick-1] the SMALLEST honest Board slice — re-key liveState from a package var to per-session state behind a SINGLE BOUNDED QUEUE enforcing the K_max cap #15 measured, a behavior-preserving refactor characterized by #15's two-connect snapshot, NOT an N-agent big-bang; [#17] pricing/conversion (the first SPENT scarce resource against the cost-to-mint denominator #15 establishes); [#16-Board-later] the full multi-card shop that adjudicates Clash D + A-on-a-loop, gated behind brick-1's re-key+queue+cap; [#13] multiset corner; [#11.5] rename-cliff fidelity.
5. BLOCKERS: (a) the integrated cost is unquantified + uncapped — #15 is the measured gate; (b) a concurrent Board ships the 8N multiplier the code already warns about until #15 sets K_max; (c) the second agent rewrites an untested global — #15's K-connect harness is its characterization snapshot; (d) #17 pricing is denominator-free until #15.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (11th consecutive round, 6/6): all six lenses advocate #15, the K-concurrent integrated-cost benchmark, scoped (TDD's sharpening, unanimous) as an instrumented-invariant brick — a counted suite-exec invariant BEFORE a measured wall-clock ceiling — with the Refactoring lens's observation that the K-concurrent-connect harness IS the characterization snapshot the eventual per-session liveState rewrite must preserve, folded into SB2. No new target-level clash. The verified cost model (a cycle ≈ (M_base+M_fix+1) suite-execs, fired uncapped per SSE connect, the ≈8N Board regime named in unenforced prose) makes #15 the gate that both the Board (#16 later, settles Clash D) and pricing (#17) are blind without. The next event is a BUILD — SB1 counter+invariant → SB2 fan-out+characterization → SB3 wall-clock K_max — not another deliberation round.

## Round 16 — make the lane scarce before you price it: the bounded-queue cap — 2026-06-04

Trigger: the Round-15 #15 wave BUILT and SHIPPED green (the integrated cost is now a COUNTED INVARIANT — a cycle fires exactly M_base+M_fix+1 suites, K concurrent → 3K, pinned — plus a measured K_max ceiling from BenchmarkConcurrentCycle). The council reconvenes for the eighth consecutive build-evidence wave. Build state re-verified: 14 green packages; the watchable wire SHOWS the economy and now has MEASURED cost.

Panelists present: all six. No new lens.

New evidence on the table (verified by reading code this round):
- #15 measured the multiplier on an UNCAPPED engine: K=1→348ms/3execs, K=2→348ms/6 (flat), K=4→405ms/12 (~16% degradation, 20-core box). `cost_test.go` pins K cycles→3K suite-execs (cost_test.go:54, the uncapped fan-out as a passing FACT) and the 2-connect shared-ledger snapshot (cost_test.go:87, the characterization the re-key must preserve).
- `LiveCard.OnConnect` (live.go:93-110) fires `go func(){ ResolveStreaming(...) }()` per SSE connect UNCONDITIONALLY — NO semaphore/queue/cap. The 3K test DOCUMENTS the unbounded fan-out; it does not CONSTRAIN it. So today there is no RED that fails when K_max is exceeded.
- `liveState` is a package var (live.go:37, set once in NewServer); every LiveCard reads the SAME cfg+log. A second agent forces a per-session re-key — but the cap needs NO re-key.
- The stock is SHOWN (read-only) but never SPENT — no sink, no conversion (Game/Systems). Pricing (#17) is the conversion, but pricing against an UNCAPPED lane has no opportunity cost: the degenerate strategy (Systems) is "spam connects → 3N free suites → farm cheap catches against an un-contended budget." Price needs contention; contention needs the cap.

Per panelist:
- UX (→ #16-board-queue): #15's ceiling tells me a calm Board can run ~2-3 cards before the felt tempo degrades. But the stock is inert (shown, never spent) = a gauge you can't act on = noise by my own standard. #1: a bounded semaphore enforcing K_max in OnConnect on the CURRENT single-instance wire, BEFORE per-session re-key. Cap the 3N fan-out first; defer NATS (an in-process channel is the smaller honest step; NATS drags §15/§19 scrutiny for zero pixels). Validate the cap holds WITHOUT freezing an admitted card's beat row.
- Game design (→ #17-pricing, the lone outlier): the stock only ever CLIMBS — a balance with a faucet, no drain. #1: pricing — the smallest honest SINK (a logged DEBIT in the same JSONL so balance = credits − debits stays a pure projection; over-budget Spend rejected). But concedes the framing requires a contended budget.
- Systems (→ #16-board-queue): #15 gave the denominator; now the economy needs its CONVERSION — but pricing against ONE uncapped lane is decoration (no opportunity cost). The degenerate strategy: farm cheap catches against an un-contended budget. #1: the bounded queue/semaphore — the FIRST scarce/contended resource, the precondition pricing needs to bite. Pricing is #2, right after. Defer NATS.
- Pragmatic TDD (→ #16-board-queue): the 3K test DOCUMENTS the fan-out, it does not CONSTRAIN it — there is no RED that fails when K_max is exceeded. #1: a bounded acquire/release wrapping the OnConnect goroutine — the one slice that converts a passing-fact into a constraint. Deterministic RED (no sleeps): K_max+1 OnConnects whose cycle is stubbed to block on a barrier; assert peak in-flight == K_max while the +1 blocks on acquire. Keep cost_test.go:87 green (behavior-preserving). Defer NATS.
- CI/CD & Delivery (→ #16-board-queue): #15 turned my prose warning into a number; any cost is load-dependent (348ms@K=1 vs degrading@K=4), so pricing would denominate against a figure that drifts under fan-out. #1: the bounded in-process semaphore (buffered chan of size K_max) gating OnConnect — the serialization point the 1:N economy rests on — BEFORE the re-key. Connects beyond K_max QUEUE, never drop (the merge-queue invariant). Defer NATS.
- Refactoring (→ #16-board-queue): #15 gave the re-key its characterization snapshot — but the cap and the re-key are SEPARABLE refactors with separate characterizations; conflating them is the big-bang risk. #1: the in-process semaphore enforcing K_max on the CURRENT wire — it needs NO re-key (liveState stays a package var); the per-session re-key is the NEXT slice, alone, with cost_test.go:87 as its green baseline. Two commits, each individually behavior-preserving. NATS in the same brick mixes an external-broker rewrite into a green-throughout refactor — defer.

Clashes touched: D (still UN-adjudicable — one card; the cap is its prerequisite, not its settler — a second agent needs the re-key, which needs the cap first); A (pre-empted in the small by #16's retrospective stock; full adjudication needs a Board); the FARM-THE-FAUCET exploit (Systems) — capping the lane closes it and establishes the contention pricing (#17) will denominate against. No §3 clash flips (a concurrency-cap build that makes the lane scarce, the precondition for the slices that settle A/D and for honest pricing).

Verdicts updated: none flip; the cap converts the unenforced 8N prose warning (#15 measured) into a real, regression-guarded constraint, and makes the cost denominator STABLE for pricing.

New clashes opened: NONE. 5/6 on #16-board-queue, 1/6 (Game) on #17-pricing — a within-arc build-ORDER preference, not a target-level clash, and self-defeating without the cap (Systems demonstrates Game's own exploit). Zero new target-level clashes. NATS-defer is unanimous across all five #16 advocates.

Decisions (the marching orders):
1. NEXT BUILD (#16-board-queue, 5/6; Game's pricing-first dissent chair-resolved as build-order, ratified roadmap #3): an in-process bounded semaphore (buffered chan of size K_max) gating `OnConnect`'s ResolveStreaming goroutine on the CURRENT single-instance wire — BEFORE any per-session re-key and BEFORE pricing. NATS DEFERRED to a dedicated round (the cap is a chan, not a broker; an external broker drags the §13.3 rewrite + §15/§19 egress/auth scrutiny into a ~10-line behavior-preserving refactor, and earns its place only at per-session demux/second-agent).
2. PREREQUISITE sub-bricks IN ORDER (each via tdd-rygba): [1a] write the RED at the OnConnect LAYER (NOT the Resolve layer — cost_test.go:54 drives app.Resolve directly, one layer below the cap, and stays an uncapped per-cycle FACT): a NEW deterministic-concurrency test drives K_max+1 concurrent OnConnect cycles whose cycle is stubbed to block on a barrier (a started/release channel pair, NO time.Sleep), asserts peak concurrent in-flight == K_max while the +1 blocks on acquire — fails today (peak == K_max+1); [1b] introduce K_max into LiveConfig (explicit default; 0 = unbounded for back-compat) threaded through setLiveState so the semaphore size is wired, not magic; keep the 2-connect sequential snapshot (cost_test.go:87) green UNCHANGED (sequential connects never contend → the cap is behavior-preserving for K≤K_max); [1c] add the buffered-channel semaphore + wrap the OnConnect goroutine: acquire before the cycle, release (defer) after the result is delivered — connects beyond K_max BLOCK on acquire then proceed when a slot frees (queued, never dropped); [1d] AUDIT -race + no goroutine leak (every acquire has a matching release on BOTH the success and the cycle-error branch at live.go:103; the Stream nil-channel drain is untouched — the felt loop is not frozen for an admitted card while a queued one waits).
3. ACCEPTANCE FIXTURES: [cap holds] K_max+2 concurrent OnConnects with a barrier-blocked cycle → atomic in-flight gauge peaks EXACTLY at K_max (never K_max+1), surplus block on acquire, release → all complete (deterministic, no time.Sleep, green under -race); [no dropped work] with the real cycle and K_max=2, fire 8 concurrent OnConnects against one ledger → EXACTLY 8 catch records land (queue serializes, never sheds); [felt-loop not frozen] an admitted card still receives beat frames while a queued connect waits on acquire (the semaphore gates ADMISSION, not the Stream tick); [behavior-preserved] cost_test.go:87 (2 sequential connects → recs==2) stays green UNCHANGED; [farm-the-faucet closed] K connects beyond K_max do NOT fire 3K concurrent suites (≤3·K_max in-flight) — the exploit backpressures, establishing the contention pricing denominates against.
4. RANKED ROADMAP: [#16-board-queue THIS WAVE] in-process bounded semaphore (K_max) on OnConnect, single-instance, NATS deferred — cap-only, no re-key, no second agent, no pricing; [#16b per-session re-key, NEXT wave alone] liveState package var → per-session keyed state so two DISTINCT cards can exist, characterization baseline already in hand (cost_test.go:87); [#17 pricing, after #16b] the SINK/debit (Spend record + Stock.Balance = credits − debits as a pure JSONL projection, over-budget Spend rejected) — NOTE: ledger.Log.Append gates on ShouldRecord (catch-only) and will REJECT a non-catch Spend record; pricing must extend the append contract or use a sibling record kind; [#16c second agent / per-session demux, LATER] the round where adopting NATS is reconsidered on its merits; [#13 multiset], [#11.5 rename-cliff].
5. BLOCKERS: (a) the lane is uncapped → pricing has no opportunity cost and the farm-the-faucet exploit is open until the cap; (b) the cost denominator drifts under load until the cap makes it stable; (c) the re-key and second agent are downstream of the cap (the cap needs no re-key); (d) #17's Spend record needs the ledger append contract extended.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (12th consecutive round): 5/6 lenses converge on #16-board-queue — an in-process bounded semaphore enforcing the measured K_max on the OnConnect cycle, the smallest honest Board slice (cap-only, no re-key, NATS deferred), turning the uncapped 3N fan-out into the first contended/scarce resource. The lone dissent (Game: pricing-first) is build-order, chair-resolved against by the panel's own reasoning (Systems demonstrates pricing-against-an-uncapped-lane is the farm-the-faucet exploit; price needs contention, contention needs the cap) — pricing is ratified roadmap #3, after the per-session re-key. No new target-level clash; NATS-defer unanimous. The chair sharpened the brick against the real code: the cap lives at the OnConnect layer (live.go:98-110, the per-connect goroutine spawn), so the RED is a NEW OnConnect-layer barrier test, not a mutation of the Resolve-layer cost fact. The next event is a BUILD — 1a OnConnect-layer RED → 1c semaphore → 1d audit, behavior-preserving (cost_test.go:87 green throughout) — not another deliberation round.

## Round 17 — the economy's missing half: the SINK (pricing) — 2026-06-04

Trigger: the Round-16 #16-board-queue wave BUILT and SHIPPED green (a bounded admission semaphore caps concurrent cycles; the catch lane is now the first contended/scarce resource; farm-the-faucet closed). The council reconvenes for the ninth consecutive build-evidence wave.

Panelists present: all six. No new lens.

New evidence on the table (verified by reading code this round):
- #16-board-queue capped the FAUCET RATE (liveState.sem, queued-never-dropped, slot released on every exit incl. cycle-error). But the TANK is undrained: `ConfirmedCatches` only does `s.Count++` (stock.go:24); Balance is credits with NO subtraction term; `ledger.Append` HARD-REFUSES any non-catch via the catch-only `ShouldRecord` gate (ledger.go:67-69). A capped faucet over an undrained tank is a slower-filling tank, not an economy. The stock is SHOWN but never SPENT.
- `Append(r CatchRecord)` and `Records() ([]CatchRecord, error)` are HARD-TYPED to CatchRecord (ledger.go:19,82) — admitting a debit is not free; it forces a discriminated record on the same JSONL or a parallel reader. The farm-denial invariant ("refusing to record a non-catch outcome") is load-bearing and must NOT be diluted.
- liveState is STILL a package var; two DISTINCT cards do not exist — Clash D un-adjudicable until the per-session re-key + a real second session.

Per panelist:
- UX (→ #17): the stock row is structurally inert (only climbs) — a gauge the Lead cannot act on = noise by my own standard; AND #16's scarcity is invisible (View still renders exactly 4 rows). #1: the smallest Spend slice — a sibling DebitRecord with its OWN append path (Append's catch-only gate UNTOUCHED) + a balance row + ONE keyboard Spend affordance on the card that exists today. Gives the Lead the FIRST ACTION on the stock, no re-key, no second-agent plumbing.
- Game design (→ #17): a meter that only goes up is a score, not an economy; #16 gave cost a feel but the tank never drains. #1: the SINK — a sibling debit kind via Log.AppendSpend, balance = credits − debits as a pure JSONL projection, over-budget rejected; ONE spend verb mapped to one logged fact. No fake XP.
- Systems (→ #17): #16 gave the first scarcity; now the conversion is finally honest to build (the lane is contended so a spend trades against something). #1: a sibling DebitRecord{Amount,Reason} (NOT shoehorned through CatchRecord/ShouldRecord), balance = credits − debits, over-budget rejected-not-logged. The degenerate strategy (mint-then-refund / self-deal) is punished because debits are logged against a REAL credit balance derived only from oracle-confirmed catches: you cannot spend what you did not catch.
- Pragmatic TDD (→ #17): #16's scarcity is genuinely TESTED (peak≤cap, no-drop, errored-cycle-releases). The economy has a faucet and no drain; a Spend is rejected by the catch-only gate today. #1: a sibling debit kind (RecordKind: Catch|Spend) keeping Append's gate INTACT for catches (farm-denial unweakened), balance = credits − debits, over-budget rejected, pure JSONL projection (no live counter), replay-auditable. RED: 3 catches + Spend{2} → Balance==1, identical on fresh re-read.
- CI/CD & Delivery (→ #16b, dissent): #16 gave the serialization point, but it serializes N connects onto ONE head; no second head exists (every OnConnect reads the SAME liveState). #1: the per-session re-key — package var → per-session keyed map (sessionID → {cfg, *Log, sem}), the ONLY structural unlock for a real Board/merge-queue (N capped heads). Pricing is economy DOWNSTREAM of having more than one thing to spend against. Smallest sub-slice: the map + a single derived sessionID, byte-identical behavior. DEFER NATS.
- Refactoring (→ #16b, dissent): #16 gave the characterization net (the shared-liveState snapshot) the re-key has needed for rounds. #1: re-key package-var liveState → per-session keyed map indexed by a connect-derived session key, re-routing only readLiveState/cycleSem/setLiveState, NO behavior change yet (single-session path keeps the snapshots green by construction). DEFER NATS (an in-process sync.Map is the smaller honest step; the broker waits for #16c's real second producer). Pricing adds a new ledger record kind — a separable concern.

Clashes touched: D (touched, DEFERRED — un-adjudicable until two distinct cards coexist, which #16b enables structurally and #16c exercises; all six agree D is blocked on the re-key, not on pricing); A (pre-empted in the small; the balance row + a real drain start to give the Lead an action, but full adjudication needs the Board); H (the SPENT economy makes the Trust-Ledger framing more concrete). The NATS-decision juncture is touched and ruled DEFERRED by unanimous panel position. No §3 clash closes this round; #17's sibling-kind design opens no clash (keeps farm-denial intact).

Verdicts updated: none flip; #17 installs the economy's missing SINK so the stock can finally drain (the first non-climbing transition), and the lane being contended (#16) makes a Spend trade against something real.

New clashes opened: NONE (all six report none). The only divergence is SEQUENCE (#17 vs #16b) over an undisputed two-brick frontier.

Decisions (the marching orders):
1. NEXT BUILD (#17-pricing — 4/6 advocate it directly; 2/6 (CI/CD, Refactoring) advocate #16b-rekey as a build-ORDER preference, NOT a target-level clash — both concede #17's necessity, scope, and the sibling-kind/gate-intact design; CHAIR-ADJUDICATED CONVERGED per the loop's "a real recorded dissent the chair resolves is valid", and sequenced #17-first because it adds NEW behavior with a hard RED (the first stock drain) on the ONE card that exists today, while #16b is a pure relocation whose payoff is two bricks away at #16c): the smallest honest Spend slice. Add a SIBLING debit record kind, project Balance = credits − debits as a pure JSONL replay, reject over-budget Spend WITHOUT appending it. KEEP `ledger.Append`/`ShouldRecord` CATCH-ONLY (byte-identical); debits travel a separate guarded `AppendSpend` (Amount>0, Balance>=Amount). NATS NOT adopted (orthogonal).
2. PREREQUISITE sub-bricks IN ORDER (each via tdd-rygba): [1] a record-kind discriminator on the SAME JSONL — a `Kind` field DEFAULTING to catch so existing catch-only logs re-read byte-identical (replay-compat RED: an existing log still yields the same Stock); [2] `Log.AppendSpend(amount, reason)` SEPARATE from Append — validates Amount>0 AND Balance>=Amount against the current projection, appends one Spend line on success, returns an error and writes NOTHING on over-budget (RED: over-budget → error + log byte-length unchanged on re-read); Append+ShouldRecord stay catch-only and regression-pinned; [3] the Balance projection — `Balance(recs) = ConfirmedCatches(recs).Count − sum(Spend.Amount)`, a pure function over the records in the stock.go idiom (RED: 3 catches + Spend{2} → Balance==1, identical on fresh re-read); [4] surface — a 5th `RenderBalance` row on the LiveCard reading Balance read-only (degrade-to-empty on error, like RenderStock) + ONE keyboard-triggered Spend verb that calls AppendSpend and re-renders the balance over SSE (the first DRAIN motion; over-budget rejected → row unchanged). NO re-key, NO second-agent plumbing — rides the package-var card.
3. ACCEPTANCE FIXTURES: [balance projection] 3 catches → Count==3; AppendSpend{2} → Balance==1; Balance recomputed from a FRESH re-read of the JSONL == 1 (pure replay); [over-budget rejection + byte-immutability] Balance==1, AppendSpend{5} → non-nil over-budget error, log byte-length unchanged, Balance stays 1; [farm-denial regression] Append(non-catch CatchRecord) still refuses; a Spend mis-routed through Append is still refused (debits ONLY via AppendSpend); [backward-compat replay] an existing catch-only JSONL (no Kind field) re-reads to the identical Stock (discriminator defaults to catch — no migration, no silent drop); [surface drain motion] append 3 catches → balance row reads 3; trigger Spend(2) → row re-renders to 1 over SSE (first non-climbing transition); Spend(5) → rejected, row unchanged at 1, no new ledger line.
4. RANKED ROADMAP: [#17-pricing THIS WAVE] the SINK (sibling debit kind, AppendSpend guard, Balance projection, balance row + spend verb); [#16b-rekey #2 immediate next] behavior-preserving package-var liveState → per-session keyed map (sessionID → {cfg,*Log,sem}), single derived session so behavior is byte-identical, snapshots green by construction — the structural unlock for a real Board; [#16c second session] a real second session so two DISTINCT cards coexist with isolated ledgers + independent caps — the first point Clash D is exercised; [#18 NATS broker, deferred] adopt the external bus ONLY when #16c proves a real second producer needs cross-process fan-out (§13.3 + §15/§19); [#19 spend taxonomy, deferred] broaden beyond Catch|Spend after the single verb proves the projection + over-budget guard; [#13 multiset], [#11.5 rename-cliff].
5. BLOCKERS: (a) the tank never drains until #17's sink; (b) ledger.Append is catch-only by construction — debits need a sibling kind + AppendSpend, never a relaxed gate; (c) Clash D stays blocked on the re-key (#16b) + a real second session (#16c); (d) NATS waits for a real second producer.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (13th consecutive round, 4/6 + chair-adjudicated): 4/6 lenses (UX, Game, Systems, TDD) advocate #17-pricing — the economy's missing SINK — as #1; 2/6 (CI/CD, Refactoring) advocate the #16b per-session re-key, a build-ORDER preference over an undisputed two-brick frontier (both concede #17's necessity, scope, and design; neither opens a target-level clash). Per the loop's recorded-dissent rule the chair ratifies #17-first — it adds NEW behavior with a hard RED (the first stock drain) on today's single card, while #16b is a pure relocation whose payoff is two bricks away (#16c) — and ratifies #16b as the immediate #2 with its spec preserved verbatim. The load-bearing ruling: KEEP ledger.Append catch-only (farm-denial intact); admit debits via a sibling kind + a guarded AppendSpend; Balance = credits − debits as a pure replay, over-budget rejected-not-logged. No new target-level clash; NATS unanimously deferred to #18. This is the FIRST round under ≥5/6 — recorded honestly as a 4/6 order-only dissent the chair resolved, not a manufactured supermajority. The next event is a BUILD — discriminator → AppendSpend guard → Balance projection → balance row + spend verb — not another deliberation round.

## Round 18 — the structural unlock for ≥2 cards: re-key liveState to a per-session registry — 2026-06-04

Trigger: the Round-17 #17 wave BUILT and SHIPPED green (the economy's SINK + visible drain — earn→hold→spend→drain complete on ONE card; the ledger is now thread-safe). The council reconvenes for the tenth consecutive build-evidence wave.

Panelists present: all six. No new lens.

New evidence on the table (verified by reading code this round, cross-checked against ../via):
- #17 closed the full economy loop on ONE card. But all six lenses name the SAME residual: LiveCard.Spend's "dispatch a unit of agent work" (`AppendSpend(1,"dispatch")`) is a DANGLING VERB — a logged debit with ZERO downstream, because there is no second agent to dispatch to.
- Root cause is structural and unanimous: `liveState` is a process-wide package var (live.go:47), read identically by View (CtxR, :102), Spend (Ctx, :130), OnConnect (Ctx, :149) via readLiveState()/cycleSem() — one cfg / one *ledger.Log / one sem for the whole process. Two DISTINCT cards cannot coexist → Clash D (1:N shop vs context-switch tax) is literally un-exercisable.
- The re-key seam is verified to need NO new Via capability: `CtxR.ID()` (ctx.go:128) and `Ctx.ID()` (ctx.go:207) both return the per-tab wire id, and `CtxR.Session()` (ctx.go:160) exists — so View is NOT key-starved (correcting the Refactoring lens's stated blocker). A connect-derived tab-id key threads symmetrically through all three readers. (Session.id is unexported, sess.go:26 — a stable-across-tabs key, if wanted later, would be minted via sess.Put/Get; the public tab id is sufficient and simplest for the unlock.)

Per panelist (ALL SIX advocate #16b-rekey-alone):
- UX: #17 made one card actionable but the dispatch verb dispatches to nothing. The Board (two triageable cards + an attention queue) needs ≥2 cards. #1: re-key liveState to a sync.Map registry keyed by a connect-derived key, ONE seeded entry, behavior-preserving (cost_test.go:87 stays green); NATS deferred.
- Game design: the loop is felt on one workbench; the SHOP is imaginary (spend dispatches into a void). Clash D is unanswerable with one card. #1: re-key to a sync.Map keyed by Ctx.ID() (tab id) so two tabs = two cards, each its own *ledger.Log + sem. The inverse-of-the-shared-ledger-snapshot RED is the structural proof Clash D can finally be set up (lands in #16c).
- Systems: #17 gave the economy its conversion; the loop is complete but degenerate (one economy because one liveState singleton). #1: a sessionID-keyed registry with a SEPARATE per-session ledger path — the economic verdict is ISOLATED economies, not a shared Treasury: per-session ledgers make each balance non-transferable so the faucet stays the sole credit source (matching the #16-board-queue farm-denial; a shared balance makes "one session farms another's budget" trivially expressible). The boundary is only EXPRESSIBLE after #16b and DECIDED in #16c.
- Pragmatic TDD: #17 made the ledger thread-safe, de-risking concurrent per-session ledgers. The re-key needs a test that CONSTRAINS its new behavior — the only snapshot (cost_test.go:87) stays green under a single key, proving nothing changed. Wanted the inverting isolation test shipped WITH #16b. (Chair-adjudicated to #16c — see dissent.)
- CI/CD & Delivery: the per-session registry is the structural change that lets N capped heads (a real merge-queue/Board) exist, each its own cap + ledger. #1: a sync.Map[sessionKey]→{cfg,*Log,sem} threaded through OnConnect/View/Spend; the second session's isolation (each its own LedgerPath so the JSONL isn't shared on disk) lands in #16c. Defer NATS until a real second cross-process producer.
- Refactoring: the re-key is the big-bang flagged for rounds; the characterization snapshot now exists. #1: introduce the sync.Map behind readLiveState/cycleSem/setLiveState, seed exactly ONE entry under a single key so behavior is byte-identical — the second config is the NEXT slice. Keep them separate commits. Defer NATS.

Clashes touched: D (UN-ADJUDICABLE today — liveState is a singleton; #16b is the structural prerequisite that makes D adjudicable, #16c exercises it once two real cards exist; no clash RESOLVED in #16b, clash D UNBLOCKED for #16c). A/H (pre-empted/concrete in the small; full adjudication needs the Board). The economic-isolation sub-question (per-session ledger vs shared Treasury) becomes EXPRESSIBLE only after #16b, DECIDED in #16c.

Verdicts updated: none flip; #16b is a pure behavior-preserving refactor unblocking the second agent (which settles D) and making the economic-boundary question expressible.

New clashes opened: NONE (all six report none). The only divergences: (a) a build-SCOPE split (isolation test in #16b vs #16c) — adjudicated as scope, not a clash, in favor of re-key-alone; (b) Systems' per-session-vs-shared-ledger economic question — ruled a scoping decision folded into #16c (un-expressible at #16b's target). Clean 6/6 convergence.

Decisions (the marching orders):
1. NEXT BUILD (#16b, CLEAN 6/6 — the council's first unanimous round, NOT chair-adjudicated like R17's 4/6): re-key `liveState` from a process-wide package var to an in-process `sync.Map` registry keyed by a connect-derived session key, threaded through the three readers (View, Spend, OnConnect) + setLiveState/cycleSem. BEHAVIOR-PRESERVING: seed ONE entry under a single key so the existing single-card behavior stays byte-identical and the preservation suite (cost_test.go:87 + spend + cap) stays GREEN with zero edits. NATS DEFERRED to a dedicated round (in-process sync.Map is the smaller honest step; the broker waits for a real second cross-process producer at #16d).
2. PREREQUISITE sub-bricks IN ORDER: [SUB-1] registry type — replace the liveState package-var struct with `var liveReg sync.Map` (sessionKey → *liveEntry{cfg, log, sem}); [SUB-2] seed one entry — setLiveState(cfg, log) stores ONE *liveEntry under a fixed default key (sem built per-entry exactly as today); [SUB-3] thread the key — readLiveState()→readLiveState(key) and cycleSem()→cycleSem(key) look up liveReg by key, FALLING BACK to the default key when the derived key isn't registered (so behavior is preserved with one seeded entry); update the three callsites; [SUB-4] derive the key from the PUBLIC tab id — ctx.ID() in Spend/OnConnect (*Ctx) and r.ID() in View (*CtxR), both verified to exist symmetrically; [SUB-5] green oracle — the full suite passes UNCHANGED (zero test edits == proof of behavior preservation), plus a focused internal registry test pins the keyed lookup (a seeded key resolves to its entry; an unknown key falls back to the default).
3. ACCEPTANCE FIXTURES: [preservation, zero edits — the refactor oracle] cost_test.go:87 (2 sequential connects → one shared ledger, recs Len==2), the spend drain/no-op cases, and cap_internal_test all stay GREEN unchanged; [registry unit, the new keyed-lookup constraint] an internal test: after setLiveState seeds the default key, readLiveState(defaultKey) returns the entry (hit) and readLiveState("unregistered") falls back to the same entry (fallback) — both branches covered without a second live card; [DEFERRED to #16c, written now as its entry gate, NOT landed in #16b] the inverting isolation RED — two connects under TWO DISTINCT keys, each with its OWN LedgerPath, each mints a catch → keyA.Balance==1 AND keyB.Balance==1 (NOT 2 shared), a Spend on keyA drains ONLY keyA → Clash D made executable + the economic-isolation verdict made testable.
4. RANKED ROADMAP: [#16b THIS WAVE] re-key to the sync.Map registry, one seeded key, behavior-preserving; [#16c] real second session — distinct keys end-to-end, the inverting isolation RED, adjudicate Clash D + rule per-session-isolated-ledger vs shared-Treasury (presumptive default: isolated, per farm-denial); [#16-board] an attention queue ranking ≥2 cards (needs the leverage/dependency-graph signal — see the leverage-needs-dependency-graph finding); [#16d dispatch downstream] give Spend's "dispatch" a real second agent/producer (the dangling verb gets a consequence; the FIRST real cross-process producer that would justify NATS); [#18 NATS] deferred to a dedicated round, after #16d; [#13 multiset], [#11.5 rename-cliff]; trust-economy thesis bricks (calibration/the-bet/Focus/tiers/Ship-Quality) remain post-pipe (8/15 risks live there).
5. BLOCKERS: (a) liveState is a singleton → two cards can't coexist → Clash D un-exercisable until #16b; (b) the dispatch verb dispatches into a void until #16d gives it a second agent; (c) the economic-isolation boundary is un-expressible until #16b, decided at #16c; (d) NATS waits for a real second cross-process producer.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (14th consecutive round, CLEAN 6/6 — the council's strongest convergence, a genuine unanimous result, NOT order-adjudicated like Round 17's 4/6): all six lenses ratify #16b-rekey-alone — re-key the process-wide liveState package var to an in-process sync.Map registry keyed by a connect-derived tab id, seeded with ONE entry so the slice is byte-identical (the preservation suite stays green unchanged as the refactor oracle). The structural unlock for ≥2 cards, the prerequisite that makes Clash D adjudicable at #16c. The scope split (TDD: isolation test in-commit) is adjudicated to #16c on TDD's own green-discipline grounds — the isolation RED needs two distinct keys threaded end-to-end, which is #16c's new behavior, and smuggling it into a behavior-preserving commit breaks the refactor/feature boundary; that RED is the mandatory acceptance gate OPENING #16c. The economic-isolation question (isolated ledger vs shared Treasury) is folded into #16c, not a new clash. NATS unanimously deferred. The next event is a BUILD — SUB-1 registry → SUB-4 key derivation → SUB-5 suite green unchanged + a registry-lookup test — not another deliberation round.

## Round 19 — the second session: isolation is a test, the shop-vs-tax feel is a human session — 2026-06-04

Trigger: the Round-18 #16b wave BUILT and SHIPPED green (liveState re-keyed to the per-session sync.Map registry, behavior-preserving). The council reconvenes for the eleventh consecutive build-evidence wave.

Panelists present: all six. No new lens. CLEAN 6/6 convergence on #16c-second-session-isolation, zero new target-level clashes.

New evidence on the table (verified by reading code + the vendored ../via this round):
- #16b made a second card STRUCTURAL (liveReg holds *liveEntry per key, each its OWN *ledger.Log from cfg.LedgerPath) but changed nothing a Lead can SEE: setLiveState hardcodes defaultSessionKey, NewServer mounts ONE route (via.Mount[LiveCard](app,"/")), lookupLiveEntry falls every connect back to default, cmd boots one -base/-fix/-file/-line target.
- The economic substrate for ISOLATION is already structural: two entries = two ledger files = two independent balances; nothing in the credit/debit path crosses entries.
- ROUTING (the unanimous diagnosis): ctx.ID() is a per-TAB id (server-minted, random), NOT a Lead-chosen session selector — so #16b's ctx.ID() keying can't aim a connect at a chosen session. A connect must reach a SPECIFIC session via a Lead/URL-controlled key. VERIFIED in ../via: a struct field tagged for URL decode is written into the per-connection component INSTANCE (render.go:17 reflect.New(d.typ); :39/40 decodePathParams/decodeQueryParams), and the action handler retrieves the SAME persisted instance by tab id (action.go:109 getCtx(tabID)) — so the key, decoded once at the initial render, PERSISTS and is readable in View AND in the action handlers (OnConnect/Spend), resolving the lenses' worry that the action ctx exposes only ctx.ID().

Per panelist (ALL SIX advocate #16c-second-session-isolation, identical mechanism): re-key the session selector from ctx.ID() to a Lead/URL-controlled key on the LiveCard instance; add a registerSession(key,cfg,log) that Stores a distinct *liveEntry (own ledger + sem) so cmd can seed ≥2 targets each with its own LedgerPath; route View/OnConnect/Spend through the card's key (fallback to default when empty, preserving the single-card wire). The testable deliverable is the inverting isolation RED (distinct keys → distinct ledgers → keyA.Balance==1 AND keyB.Balance==1, Spend on keyA leaves keyB) — every lens specifies it identically. UX/Game/CI/CD/Refactoring add: deliver a watchable two-card surface a human opens side by side; TDD + all are HONEST that the shop-vs-tax FEEL is a human session, not a green test. Refactoring also flags the two stale "liveState" prose comments (cost_test/cap_internal) to fix in-scope.

BUILD-VERIFIED ROUTING DECISION (a correction to the chair's presumptive /s/{key} path route, grounded in ../via): checkPathParams (composition.go:194) PANICS if a path-tagged field has no matching {seg} in the route — so mounting LiveCard (with a path key field) at BOTH "/" and "/s/{key}" is impossible, and dropping "/" would force editing the preservation suite (which connects to "/"), breaking the zero-edits proof. A QUERY param (query:"key") is route-segment-FREE (checkPathParams only validates path slots; querySlots decode via decodeQueryParams the same way and persist per-tab identically) — so "/" stays mounted (key="" → default, preservation suite untouched) and the Lead selects a session via /?key=a vs /?key=b. The build uses query:"key", not the path route, for this verified back-compat reason.

Clashes touched: D (1:N shop vs context-switch tax) — NOT resolved by this slice, BY DESIGN: it is the first headline goal a green test CANNOT close. #16c makes D WATCHABLE for the first time (two live cards) + instruments dwell/idle/rework as RAW OBSERVATION ONLY (explicitly NOT a pass/fail oracle — the flaky-vs-intermittent and catch-weight findings warn against laundering watched telemetry into a score). The economic boundary (isolated ledger vs shared Treasury) is RATIFIED in favor of per-session ISOLATED ledgers (R18 farm-denial default — a shared balance would let one session farm another's budget); the isolation RED enforces it. The shared-Treasury farm exploit is pushed to #16d (where dispatch makes it reachable; the isolated-ledger schema commitment forecloses it).

Verdicts updated: none flip; #16c is the SETUP that makes Clash D adjudicable by a human and locks the isolated-ledger economic boundary.

New clashes opened: NONE (6/6 report none). The OnConnect/Spend-only-have-ctx.ID() concern is the shared diagnosis (resolved by the persisted-instance key, build-verified), not a clash. Build-order (TDD: RED-first) is ordering inside the agreed brick.

Decisions (the marching orders):
1. NEXT BUILD (#16c, CLEAN 6/6): register a SECOND keyed session + route to it by a Lead-controlled QUERY key, so two DISTINCT, ISOLATED cards become reachable side by side (/?key=a, /?key=b), each its own ledger/balance/sem. THE FIRST SLICE WHOSE HEADLINE GOAL (Clash D) IS NOT CLOSEABLE BY A GREEN TEST — stated plainly.
2. PREREQUISITE sub-bricks IN ORDER (the testable core via tdd-rygba): [a] add `Key string \x60query:"key"\x60` to LiveCard; route View/OnConnect/Spend through c.Key (fallback to defaultSessionKey when empty); the "/" mount is unchanged (no key → default), so the preservation suite stays green; [b] registerSession(key, cfg, log) → liveReg.Store(key, &liveEntry{...}) with its OWN ledger.Open(LedgerPath) + sem; keep setLiveState seeding defaultSessionKey for the single-target wire; [c] the isolation RED (internal test, swap resolveCycle to a no-real-oracle fake that mints one Catch Record): register keyA{ledgerA} + keyB{ledgerB}; connect /?key=keyA (+SSE → mints to ledgerA) and /?key=keyB (→ ledgerB); assert ledgerA.Balance==1 AND ledgerB.Balance==1 (NOT 2 shared); fire Spend on the keyA client → ledgerA→0, ledgerB stays 1; [d] cmd/agntpr grows ≥2 review targets (a repeatable -session flag) each registered with a distinct LedgerPath — WIRING, verified by build/vet; [e] fix the two stale "liveState" prose comments in cost_test/cap_internal (now in-scope).
3. ACCEPTANCE FIXTURES: [isolation RED, the testable half] as in [c]; [same-key-still-shares, preservation] cost_test.go:87 (2 sequential same-default-key connects → one ledger, Len==2) stays GREEN UNEDITED + registry_internal_test fallback contract green; [back-compat wire] the "/" route + single-target cmd still serves the one default card; [routing unit] a connect with ?key=keyB resolves Spend/OnConnect/View to keyB's entry (c.Key=="keyB"), proving the URL — not ctx.ID() — is the selector.
4. HUMAN EXPERIMENT (NOT a green test): a Lead opens /?key=a and /?key=b in two tabs, lets both catch cycles run live, triages both (watches the streamed beat rows, in-flight→resolved verdicts, draining balances), and JUDGES whether sitting in front of two live cards feels like a SHOP (1:N leverage) or a context-switch TAX (thrash). The build delivers: (1) testable isolation [green RED], (2) a watchable two-card surface [/?key=a + /?key=b], (3) dwell/idle/rework as logged RAW observation. The VERDICT on D is the watched session, never a go-test assertion.
5. RANKED ROADMAP: [#16c THIS WAVE] second keyed session + query routing + cmd ≥2 targets + the isolation RED; [#16d dispatch-consequence] give Spend's dispatch a real consequence (the first cross-process producer — only here does a bus earn its keep, and the shared-Treasury farm exploit becomes reachable, foreclosed by the isolated-ledger schema); [NATS] deferred until #16d; [triage/attention-queue UI] deferred (leverage-needs-a-dependency-graph makes it half-uncomputable today); [multi-Lead/co-review] far-deferred (multiuser-rewrites-the-scoring-spine); [#13 multiset], [#11.5 rename-cliff].
6. BLOCKERS: (a) the Lead sees one card until #16c routes a second; (b) routing must be Lead/URL-controlled (query key), not ctx.ID(); (c) Clash D's verdict is a human session, not a test — the build only SETS IT UP; (d) the dispatch verb stays dangling until #16d; (e) NATS waits for a cross-process producer.
NO VISION/DESIGN text changed (the 12-contradiction reconciliation pass remains queued per RISKS sequencing step 5).

CONVERGED (15th consecutive round, CLEAN 6/6): all six lenses ratify #16c-second-session-isolation — register a second keyed session, route to it by a Lead-controlled key, isolate the ledgers — with the identical inverting isolation RED. The headline goal (Clash D) is, for the FIRST time, NOT closeable by a green test: the build delivers the testable isolation + a watchable two-card surface + raw instrumentation, and the shop-vs-tax verdict is an explicitly-defined human session. Build-verified correction: routing uses a query:"key" param, not the chair's presumptive /s/{key} path route, because checkPathParams would panic on a path key at "/" and break the preservation suite — query keys preserve "/" with zero test edits. Economic boundary ratified as per-session ISOLATED ledgers (farm-denial); the shared-Treasury exploit deferred to #16d. NATS deferred. The next event is a BUILD — [a] key the LiveCard via query → [b] registerSession → [c] the isolation RED → [d] cmd ≥2 targets — not another deliberation round.

## Round 20 — the dangling verb gets a consequence: spend funds a logged work-order (queued, not run) — 2026-06-04

Trigger: the Round-19 #16c wave BUILT and SHIPPED green (a second keyed live session; LiveCard.Key query routing; registerSession/AddSession per-key isolated ledgers + sem; cmd -session flag + validateSessions rejecting key/ledger-path collisions before any open; the inverting isolation RED green; preservation suite UNEDITED; whole suite green under -race). The council reconvenes for the twelfth consecutive build-evidence wave.

Panelists present: all six. No new lens. CLEAN 6/6 convergence on the TARGET (#16d dispatch-consequence); a single intra-slice sub-clash (queued vs executing) chair-adjudicated by scoping — NOT a frontier-level target clash.

Progress / shared diagnosis: #16c put a second card on screen (two isolated economies side by side) and in doing so made the hollow center LOUDER — every lens independently names the SAME blocker: LiveCard.Spend does AppendSpend(1,"dispatch") with ZERO downstream. The Lead spends a hard-won catch and the only on-screen consequence is the Balance row ticking down one. An action whose sole feedback is your own number shrinking reads as a meter you feed, the opposite of a tycoon shop. Two empty shops instead of one. The full loop earn→hold→spend→drain has no second half; "drain" buys nothing.

Per panelist (ALL SIX advocate #16d-dispatch-consequence, IN-PROCESS, identical outer boundary):
- UX: smallest VISIBLE slice — on a successful Spend append a "dispatched" record to the SAME ledger + render a distinct data-state="dispatch" row (RenderDispatch(n) reading the ledger like Stock; RenderDispatch(0) reads calm not empty-chrome); the existing Balance.Write fan-out already re-renders. Spend visibly MOVES a catch from balance into a dispatched tally — spend→see-a-thing.
- Game design: make Spend produce a state-changing GOOD; advocated the EXECUTING cycle (spend→run a real in-process catch cycle that mints back, closing the reinvestment loop). [Sub-clash dissent — re-sequenced to #16e.]
- Systems / Economy: one debit ⇒ exactly ONE funded work-order, same session only, CONSERVED (debits == dispatched count, per-account, always). Guardrail (not a clash): the consequence MUST stay in-process + same-account this round; cross-process would be a target clash — no lens proposed it.
- Pragmatic TDD: make the debit and its consequence ONE atomic ledger fact — `ledger.AppendDispatch(reason)` under the same `mu` (reuses the AppendSpend guard, no new overspend path) writing the debit AND a paired kind:"workorder" line carrying a monotonic id + producer field; `PendingDispatches()` projects open orders (skipped by Records like spends). Forces the P0 event-log producer field NOW, EARNED by a real second line-kind, not speculatively.
- CI/CD & Delivery: enqueue ONE work-order (id, key, status=queued, producer tag) atomically with the debit under the balance lock; one new SSE row. Carry producer+status from line one so the later cross-process step already has the schema the P0 finding demands. Defends as TARGET-level (not order): #16d must NOT spawn a cross-process executor or stand up NATS this slice.
- Refactoring: advocated REUSING the existing RunCatchCycleStreaming under the existing per-key sem to RUN the dispatched cycle — the first in-process producer, no new process/NATS/ledger-kind beyond a counter. [Sub-clash dissent — re-sequenced to #16e.]

Clashes touched: the DANGLING VERB (the headline R17 open item) — CLOSED: Spend now buys a funded, projectable, conserved work-order. Clash D (1:N shop vs context-switch tax) — made WATCHABLE-WITH-CONTENT (each card shows earn+spend+dispatched-tally) but its verdict stays a human side-by-side session, NOT a green test. Event-log-concurrency P0 — PRE-PAID (producer+status FIELDS forced onto the new workorder line-kind now, earned by a real second line-kind) while a single in-process writer keeps the monotonic seq honest; full multi-producer reconciliation is NOT discharged (deferred to #16e when a real second producer exists). NATS-deferral — CONFIRMED (the bus earns its keep only once an order crosses a process boundary). Shim-enforcement / sandbox-egress / secret-scrub — kept correctly DORMANT (no agent RUNS this round). Farm-denial extends from the balance side to the work side via per-account conservation. Seeds leverage-needs-a-dependency-graph: the work-order is the first node that graph will need.

Verdicts updated: none flip; #16d installs the consequence the spend has lacked since R17 and converts the economy from bookkeeping into a purchase the Lead can SEE.

New clashes opened: NONE at the frontier. One real intra-#16d SUB-CLASH (target-level, adjudicated by scoping): the consequence is a passive QUEUED work-order (UX, Systems, Pragmatic-TDD, CI/CD — 4 lenses) vs a work-order that RUNS an in-process catch cycle, spend-to-earn (Game design, Refactoring — 2 lenses). RULING: the QUEUED non-executing order wins THIS round — (i) strictly smaller honest slice with a clean green RED ("an order was funded + projected" is assertable without a second producer goroutine racing the mint on the same per-key entry); (ii) running a cycle introduces exactly the genuine second in-process producer the P0 single-seq finding warns about — funding-without-running pre-pays the schema fields while honestly deferring seq reconciliation; (iii) the 4-lens framing already carries the producer field the runner will need, so #16e is a clean follow-on, not rework. The 2-lens runner is NOT rejected — re-sequenced to #16e with its spec preserved. All six agree on the #16d TARGET and all six independently guarded the same outer boundary (no cross-process/NATS), so convergence holds 6/6.

§3-style verdicts for the wave:
1. NEXT BUILD (#16d, CLEAN 6/6 on target): a successful Spend atomically funds exactly ONE logged work-order in the SAME isolated ledger and renders a Dispatched/Queue row; the dispatched unit does NOT execute code this round. IN: (a) `ledger.AppendDispatch(reason)` — under the same `mu` as the balance guard, refuses on balance<1, else writes the debit AND a paired kind:"workorder" line carrying a monotonic work-order id + producer field + status=queued; (b) `PendingDispatches()`/`Dispatched()` projection (workorder lines skipped by Records like spends); (c) `surface.RenderDispatch(n)` — distinct data-state="dispatch" row, one row never speaking for another, RenderDispatch(0) reads calm; (d) LiveCard.Spend calls AppendDispatch and broadcasts BOTH the drained Balance cell AND the new Dispatch row over the existing SSE fan-out; (e) per-key isolation carried through the consequence (A's order never lands in B's queue). EXPLICITLY DEFERRED: the order RUNNING a cycle (#16e); any cross-process executor; NATS/JetStream; triage/attention-queue ranking; the producer+commit-status multi-producer RECONCILIATION (only the FIELDS are pre-paid; single-writer seq stays honest); all of shim/egress/secret-scrub (activate only when an order executes code).
2. ACCEPTANCE FIXTURES (hard RED, internal/app + ledger + surface, preservation suite UNEDITED): (1) ledger core — mint 2 catches into one keyed isolated ledger, AppendDispatch twice → Balance==0, PendingDispatches==2, the two workorder lines carry DISTINCT monotonic ids + a producer field + status=queued, debit-count==workorder-count (conservation); (2) atomicity under -race — balance=1, fire AppendDispatch from N goroutines → exactly ONE succeeds, Balance==0, PendingDispatches==1 (debit + workorder never tear; extends the concurrent spend test); (3) over-budget — a Spend at balance 0 funds NO order and emits NO dispatch frame; (4) surface/integration — connect, drive a catch in, POST Spend → the streamed View contains the dispatch row at count 1 AND the balance row drained to 0 in the SAME render, RenderDispatch(0) is calm; (5) isolation — Spend on A funds an order ONLY in A, B's count stays 0. NOT test-closeable / human session: Clash D's worth-it verdict.
3. RANKED ROADMAP: [#16d THIS WAVE] queued in-process work-order + Dispatch row; [#16e execute-the-order] run the dispatched work-order through the existing RunCatchCycleStreaming under the per-key sem (spend-to-earn; the 2-lens runner re-sequenced here) — the FIRST real in-process second producer, forcing the P0 single-seq reconciliation honestly with a live producer in hand; [#16f cross-process producer] order crosses an OS-process boundary — where NATS/JetStream first earns its keep AND where shim/egress/secret-scrub activate together (gate behind security mitigation); [#13 multiset]; [#11.5 rename-cliff]; [triage/attention-queue ranking ≥2 cards] still blocked on the dependency-graph signal the work-order begins to supply; trust-economy thesis bricks (calibration/the-bet/Focus/tiers/Ship-Quality) — 8/15 risks live here, post-pipe.
4. BLOCKERS: (a) the only hazard inside #16d is ATOMICITY — the debit and the workorder line MUST be written under the single `mu` that already guards balance, else a crash/race mints a free order or burns a catch for nothing (covered by RED #2 under -race); (b) pre-paying producer+status fields does NOT discharge the P0 multi-producer seq reconciliation (deferred to #16e); (c) cross-process security trio + full fan-out reconciliation stay dormant until an order executes (#16e) and crosses a process (#16f); (d) triage ranking stays half-uncomputable until the dependency graph the work-order seeds matures.

CONVERGED (16th consecutive round, CLEAN 6/6 on target): Round 19's #16c put a real second keyed live session on screen and made the hollow center louder — every lens reports the SAME blocker, the dangling verb (Spend doing AppendSpend(1,"dispatch") with zero downstream, a debit that buys nothing and re-renders only its own shrinking number). All six independently name #16d dispatch-consequence as #1 and all six independently scope it IN-PROCESS (no cross-process executor, no NATS) and all six pre-emptively flag the cross-process/NATS framing as the one thing that WOULD be a target clash — since no lens proposed it, that shared guardrail confirms the boundary rather than breaking convergence. The chair adjudicates the single real sub-clash — passive queued work-order (4 lenses) vs a work-order that RUNS a catch cycle, spend-to-earn (2 lenses) — in favor of the QUEUED order this round: strictly smaller honest slice, clean green RED (a funded, conserved, projected order), pre-pays the event-log producer+status FIELDS the P0 finding demands while a single in-process writer keeps the monotonic seq honest; the executing cycle is not rejected but re-sequenced to #16e where a genuine second producer can be confronted with a live producer in hand. The slice is fully test-closeable — conservation (debits==orders), atomicity under -race (debit + workorder never tear), over-budget no-op, per-key isolation carried through the consequence, and the streamed View showing the dispatch row at 1 with balance drained to 0 in the same render — with the explicit caveat that Clash D (1:N value vs context-switch tax) is the one thing #16d still cannot assert green: it gives the two cards real content to watch, but the verdict is a human side-by-side session. Spend finally buys a thing the Lead can see; the shop stops being a meter you feed. The next event is a BUILD — AppendDispatch (atomic debit+workorder) → PendingDispatches projection → RenderDispatch row → LiveCard.Spend dual broadcast — not another deliberation round.

---

## 6. The validating slices these clashes are waiting on

From the build plan — the slices most likely to produce verdicts:

1. **Pipe + one review round-trip** (DESIGN §17 + a single comment →
   revision cycle) → informs Clash B framing baseline, the core
   review-vs-chat feel.
2. **Mutation-as-`question:`-thread** (D §29.4) → informs Clash B, F, and
   the central "does it ship good software" question.
   **BUILT & TESTED** (`internal/mutation`, slice 2025) — logic layer
   only (produces `Finding`s, not yet rendered as UI threads). Verdicts
   logged in Clash B & F and the round below.
3. **Two-agent Board with queue-to-zero loop** (VISION §11) → informs
   Clash A, D, E, H — the whole "does it feel like a shop" thesis.
4. **A real refactor through the Invariant View** → informs Clash G.

When any of these is built and tested, return to §3, fill the verdicts,
and log a round in §5.
