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
| Merge-queue-as-integrator          | CI/CD     | low-risk, standard practice | _TBD_ → roadmap #6: single-lane queue wrapping integrate-on-tip (#5); experiment = throughput-to-zero on K branches; designed-in to avoid O(N²)/8N contention |
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
