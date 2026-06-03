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
- **Verdict (post-build):** _TBD_

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
- **Verdict (post-build):** _TBD_

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
- **Verdict (post-build):** _TBD_

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
| Mutation-driven adversarial review | TDD       | high conviction | **validated**: weak→finding/strong→silent; survivor→`question:` thread artifact built (`internal/review`, data layer); latency benchmarked (~30 ms/mutant warm). Pending: UI rendering + harness wiring |
| Trust Ledger (calibrated delegation)| Game     | spine, framing-risk (Clash H) | _TBD_ |
| Merge-queue-as-integrator          | CI/CD     | low-risk, standard practice | _TBD_ |
| Focus as central resource          | Systems   | adopted, render-risk (Clash A) | _TBD_ |
| Speculative integration preview    | CI/CD     | high value, infra cost | _TBD_ |
| Characterization Gate + replay     | Refactor  | high value, scoped to refactors | _TBD_ |
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
