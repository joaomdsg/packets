# agntpr — Vision

> The agentic coding experience as a management game you actually want
> to play. You don't write the code. You run the shop.
> Companion to `DESIGN.md` (the how); this is the *what* and the *why*.

## 0. One line

**agntpr** turns you from a coder into a *lead*: you command a fleet of
Claude Code agents, review their work like pull requests, and ship — and
because the real dynamics of that job already are a management game, we
lean all the way in and make it *feel* like one. A tycoon / management
sim where the operation is your codebase and the workers are agents.

## 1. The reframe: you are the lead, not the coder

Every other agentic tool keeps you in the coder's chair with an AI
helper. agntpr moves you up a level. Your job becomes the *actual* job
of a senior engineer running a team:

- decide **what** should be built and **why**,
- review **what came back**, sharply and fast,
- keep **many things moving** without dropping any,
- spend a **finite budget** of attention and tokens wisely.

That is a management sim. So we stop pretending it's an editor and build
the best version of the management sim.

## 2. Why a game — the honest version

This is **not** gamification-as-lipstick (no fake XP bars bolted onto a
form). The game *is* the work, because the work has every property of a
good tycoon game already:

| Tycoon game property | The real agentic-coding dynamic |
|----------------------|----------------------------------|
| A scarce resource you ration | **Your review attention** — the true bottleneck |
| An economy | **Tokens / compute** = gold; spend to produce |
| Parallel workers | **Agent fleet** — N changesets in flight |
| A throughput goal | **Shipped, reviewed PRs per session** |
| Bottleneck management | You can't review faster than you can read |
| Triage under pressure | Which red-checks / blocked agents do I touch first? |
| Mastery curve | Better tasks + conventions → less rework → more flow |

The "fun" is the genuine satisfaction of a well-run shop: agents humming,
a clean review queue, green checks, things landing. We make that
*legible and tactile* instead of buried in terminals and tabs.

## 3. The world

| Game object        | What it really is |
|--------------------|-------------------|
| **You — the Lead** | the reviewer/orchestrator; the only human |
| **Agents**         | Claude Code harness instances, each on a task |
| **Workshops**      | the per-session Docker containers they work in |
| **Work Orders**    | tasks you assign (the issue/intent) |
| **Deliverables**   | changesets = reviewable PRs (revisions) |
| **The Board**      | fleet dashboard: every agent + its state |
| **The Treasury**   | token/compute budget = the economy |
| **The Ledger**     | event log / timeline; every action, replayable |

Tone is *competent and calm*, not cartoonish — think a beautifully
instrumented control room / strategy-game HUD, not a clicker with
confetti. Juice is earned and tasteful (§8).

## 4. Core loop A — The Craft: review-async

The unit of work you love. An agent finishes; you review its PR.

1. **Plan gate (rev 0.5).** Before code, the agent posts a *plan* you
   review like a draft PR — approve the approach or redirect it. The
   cheapest place to steer. (agntpr's original plan phase, reborn as a
   review gate.)
2. **The deliverable arrives.** A coherent changeset: auto-written PR
   description + an annotated diff. **No chat window** — the diff *is*
   the conversation.
3. **Self-flagged weak spots.** The agent annotates its own diff —
   *"⚠ unsure about this error path"* — so your eyes land where they
   matter first. The author confesses its doubts; review inverts.
4. **You comment, intent-tagged.** Conventional Comments, machine-read:
   `blocking:` (must fix) · `nit:` (optional) · `question:` (answer, no
   edit) · `suggestion:`. The agent *acts* by tag, never guesses
   severity.
5. **Review macros.** One key for the common asks — `add test`,
   `extract`, `handle error`, `does this match CONVENTIONS.md?` — expand
   into precise turns.
6. **The author revises.** Edits return as a new revision with a one-line
   "what changed since last round" (like a force-push note); threads
   re-anchor or go outdated (DESIGN §28).
7. **Watch it earn trust.** RED→GREEN tests stream into the checks panel
   live — you *see* it verify itself.
8. **Approve → land.** Squash + push/PR. The order is done.

The whole loop is **keyboard-native** (`j/k` hunks, `c` comment, `r`
resolve, `a` approve) — GitHub+Vim muscle memory, server-driven by
Via+Datastar.

## 5. Core loop B — The Operation: fleet management

The unlock no chat-IDE can reach. **Chat is 1:1 by nature; review is
1:N.** Once reviewing one Claude feels like reviewing a PR, reviewing
five is just a queue — and *that queue is the game*.

**The Board** — your fleet at a glance, each agent a card:

```text
┌─ THE BOARD ─────────────────────────────────────── ◷ 14:32 ─┐
│  ⚙ auth-refactor      EDITING    rev3   ░░░░▓ tests…         │
│  ✦ rate-limiter       AWAITING   rev2   ● needs review (2)   │
│  ⚙ docs-pass          PLANNING   —      plan ready ●         │
│  ⛔ migrate-db         BLOCKED    rev1   permission: drop tbl │
│  ✓ flaky-test-fix     LANDED     —      PR #412 merged       │
│                                                              │
│  Treasury ▓▓▓▓▓░░░ 312k / 500k tokens   ·   Queue: 2 to review│
└──────────────────────────────────────────────────────────────┘
```

The management game, for real:

- **Attention triage.** Multiple agents want you at once — *whose*
  review unblocks the most? The interface ranks the queue by impact
  (blocked-downstream, idle-cost, staleness), but you choose.
- **You are the bottleneck — and the game makes that visible.** Agents
  idle while awaiting review *cost* (tokens spent, context going stale).
  The Board surfaces the cost of your inattention, gently.
- **Dispatch.** Assign new work orders; clone a winning approach across
  repos; pause a runaway; rein in budget.
- **States** (the card lifecycle): `Planning → Awaiting-plan-approval →
  Editing → Checks → Awaiting-review → Revising → Landed` (or
  `Blocked` / `Errored`). One glance = whole shop status.

## 6. The economy — attention & tokens

Two scarce resources, both real, both made legible:

- **Attention (the true scarcity).** You can't review faster than you
  read. The Board's job is to spend your attention optimally: surface
  the highest-leverage review next, batch nits, never make you hunt.
- **Treasury (tokens/compute).** Visible budget per session/day. Agents
  spend it; fan-out (DESIGN §18) spends faster; idle-awaiting-review
  leaks it. You *feel* the cost, which makes good task-scoping and quick
  reviews intrinsically rewarding — the economy teaches the skill.

No fake currency. The "gold" is your actual Anthropic spend, shown as a
resource you steward. Running a tight, productive shop is *literally*
cost-efficient.

## 7. Progression & mastery (the curve, not a grind)

Mastery is real, so we reflect it — without grindy XP:

- **Conventions compound.** Every `blocking:`/`nit:` you give can be
  offered as a durable rule (→ `CONVENTIONS.md`, fed to all agents).
  Your shop literally gets better: fewer nits, less rework, more flow.
  *That* is leveling up — earned competence, not a points bar.
- **Macros & templates** you define become your personal toolkit.
- **The Ledger** lets you scrub any session's timeline — learn from how
  a changeset evolved; build trust in the fleet.
- **Throughput, honestly shown.** "Reviewed-and-landed this session" as
  a quiet scoreboard — satisfying because it's true output, not vanity.

## 8. Feel — "10/10" without gimmicks

The juice that makes it *groundbreaking* rather than another dashboard:

- **A living Board.** Cards breathe — editing shimmer, a check flipping
  green, a card sliding into the review queue. SSE makes it alive
  (Via's home turf).
- **Tactile review.** Keyboard-first, zero-latency comment→dispatch,
  threads that visibly route to the agent and come back.
- **Calm, not noisy.** Control-room aesthetic. One sound for "needs
  you," none for routine. Respect attention — the scarce resource.
- **Earned moments.** A landed PR, the queue hitting zero, the treasury
  staying green at end of day — small, real, satisfying punctuation.
- **Plugins as rich surfaces** (Via plugin model): **Monaco** for
  diff/edit, a **timeline graph** for the Ledger, a **fleet map** for
  the Board. Each a JS island, Go-driven.

## 9. Concept integration map

Every brainstorm idea, placed:

| Concept                    | Lives in |
|----------------------------|----------|
| Review-async loop          | §4 The Craft |
| Fleet / 1:N management     | §5 The Kingdom |
| Plan-as-first-revision     | §4.1 plan gate |
| Conventional Comments      | §4.4 intent tags |
| Confidence pre-seeding     | §4.3 self-flagged |
| Review macros              | §4.5 |
| Keyboard-native review     | §4 (end) |
| Session dashboard          | §5 The Board |
| Conventions auto-fed       | §7 progression |
| Timeline scrubber          | §6/§7 The Ledger |
| Agent "speaks PR"          | §4.2/§4.6 |
| Monaco / Via plugins       | §8 surfaces |
| Token economy / tempo      | §6 economy |

## 10. Anti-gimmick guardrails

What keeps this a *tool that happens to feel like a game*, not a toy:

1. **Every mechanic maps to a real dynamic** (the §2 table). If a game
   element doesn't reflect a true constraint, it's cut.
2. **Power-user first.** Keyboard, density, speed. The game layer never
   slows the expert down — it *organizes* their attention.
3. **No dark patterns.** We optimize for *your* throughput and spend,
   not engagement-for-its-own-sake. Hitting "queue zero" and closing the
   shop is a *win*, not a loss.
4. **The work is the source of truth.** Real diffs, real commits, real
   PRs, real tokens. The game is a lens on reality, never a substitute.

## 11. Smallest slice that *feels* like the vision

To prove the experience (not just the pipe of DESIGN §17): the
**two-agent Board**.

- Run **2 agents** on 2 work orders in 2 containers.
- A **Board** with 2 living cards cycling real states over SSE.
- The **Craft loop** on one of them: open its deliverable, leave one
  `blocking:` comment, watch it revise, approve → land.
- A visible **Treasury** ticking real tokens.

If two cards on a board, a real review that round-trips, and a budget
that moves can make someone go *"oh — I'm running a shop,"* the vision
is validated. Everything else is scale and polish.

## 12. Panel synthesis — adopted mechanics

A UX, a game designer, and a systems designer reviewed §§1–11. They
converged hard on three flaws and one unifying idea. These are now part
of the vision; they supersede the rougher first-pass economy in §6 and
progression in §7.

### 12.1 The real core loop has a hole: dead air → the Prep Bench

The first-pass loop (dispatch → wait → review) leaves you staring at
compute for ~90s. That idle shape is the make-or-break feel risk
("bigger than the event-translation layer"). Fix: the **Prep Bench** —
during agent compute, your job is *sharpening the next work orders*
(splitting vague tasks, writing acceptance criteria, pre-seeding review
checklists). The honest core loop is continuous:

```text
review what landed → sharpen the bench → dispatch → review what landed …
```

You're never idle, because there's always high-leverage upstream work to
prepare — and better-scoped tasks mean less rework, so the bench is the
*opposite* of busywork. (Factorio: the factory runs while you build the
next thing.)

### 12.2 Economy: rank by leverage, cost only at decisions

The §6 "idle tokens leaking per card" idea is **cut** — it's
structurally guilt-inducing ("a Slack with 14 red badges dressed as a
strategy game"). Replaced by:

- **Leverage is the only per-card pressure signal.** A card's pull on
  your eye = how much reviewing it *unblocks* (downstream agents waiting,
  plan-gates expiring), shown as a thin left-edge accent. Queue
  auto-ranks by leverage; you can override. Eye pulled by
  **consequence**, never by **cost**.
- **Cost is felt at the moment of a choice**, not as a constant drain:
  the projected cost-to-land appears when you *dispatch* or *fan out*;
  fan-out visibly multiplies a card's burn. Treasury is one ambient
  aggregate bar + an honest end-of-day number — never per-card and
  accusatory.
- **Burn-signature as a real signal:** a card editing 20 min at high
  tokens/min with no revision settling = *thrashing*; surface that so
  you can pause/re-scope. That's the highest-value token decision there
  is.

### 12.3 Scoring: net quality, confirmed catches only

Volume scoring ("PRs landed") breeds rubber-stamping. Replaced by a
**Ship Quality** composite computed from the existing `events`/`threads`
tables:

```text
+ landings
− rework ratio        (revisions after first review ÷ landings)
− regression hits     (green→red later, same files)
+ confirmed catches   (a `blocking:` comment the revision PROVED real —
                       a test flipped / a branch got handled)
− stamp ratio         (large diffs landed with zero comments + near-zero
                       review time vs. self-flag density)
```

**Meta-principle:** every point must be redeemable against a real, logged
engineering event. **Reward outcomes, never actions** — confirmed
catches, not comments emitted (else you get comment-theater).

### 12.4 The Trust Ledger — calibrated delegation is the game

The unifying bold swing (two panelists invented it independently). The
deep skill of agentic coding is *delegation under uncertainty*: how much
to verify vs. trust. Make that the spine.

- Every approval is implicitly a **bet** at a chosen depth: **deep**
  (read it), **review** (full diff), **skim** (trust it, glance only at
  self-flagged spots).
- The **Trust Ledger** records outcomes and surfaces your **calibration**
  per agent / task-type / subsystem: *"you skim docs-passes accurately
  (94%), but you've been over-trusting auth (2 escaped bugs)."*
- When calibration says you're **over-trusting** a risky area, the
  system can *force a deep review* — the anti-dark-pattern: a tool that
  protects you from your own complacency.

> Other tools level up *the AI*. agntpr levels up *the human's judgment
> about the AI* — the durable skill of the next decade. The endgame
> fantasy isn't "command 20 agents"; it's *"I know exactly which of my
> 20 I never need to read closely, and I'm right."*

### 12.5 Focus — attention as the central spent resource (a budget, not a gate)

Model the human's finite daily bandwidth as **Focus**: each review action
costs Focus ∝ diff complexity; a Deep Review costs more and earns full
catch-credit + suppresses rework; a Glance costs ~nothing but marks the
landing **unverified** (which feeds the §12.3 penalties if it bounces).

**Hard rule:** Focus is a budget you *allocate*, **never a gate that
blocks you** (that's a fake mobile-energy mechanic). You can always
glance; Focus just measures whether your scarce real attention landed
where it counted. The win condition writes itself: **queue zero,
treasury green, Focus spent on the diffs that mattered.**

### 12.6 Session arc: standup → loop → close-out

Give the session a shape (Frostpunk/FTL act structure), which also
supplies the *ethical* "just one more" hook — the pull is "one more
landable PR before close-out," a natural stopping point, not infinite
scroll.

- **Standup:** the queue, treasury, what's stale, what landed overnight.
- **The loop:** §12.1.
- **Close-out:** what shipped, what's parked, treasury spent, the one
  CONVENTIONS rule this session earned. **Queue-zero is the win screen,
  and it's allowed to be the end.**

### 12.7 Earned concurrency — the honest mastery curve

Don't drop a novice onto a 6-card Board (the context-switch tax crushes
them; reviewing N live diffs may not scale past 2–3 without it). Start
effectively **serial** (1 agent, learn the Craft loop deeply); let
concurrency **unlock as calibration improves** (§12.4) — not an arbitrary
gate, but because the tool has *evidence* you can handle N: *"you've been
skim-approving accurately — the Board can run 3."* The number of agents
you can run *well* is the honest skill expression.

### 12.8 Time-travel review — the Ledger as a bidirectional scrubber

The Ledger (§7) is not a read-only log — it's a **control surface**.
Scrub the timeline and the diff/threads/checks reconstruct that exact
moment (the `events` log is the source of truth; the client is a pure
projection — DESIGN §13.3). The swing: drop a comment at a *past* point
("at rev1, don't go down this path") and the agent **re-runs forward as
a new branch of revisions**. You review the agent's *decision tree* and
prune it retroactively — the management-sim "rewind and try the other
build order," made real. No PR tool or chat IDE can do this; it falls
straight out of the replayable event spine.

### 12.9 The "brief" rail — fix the no-chat fallback

"The diff IS the conversation" (§4) orphans the quick non-anchored ask
("no, the *refresh-token* path"). Promote DESIGN's top-level/task thread
to a **first-class, always-visible rail** (`t` to focus it) — call it
*the brief*, not chat, to keep the framing. It absorbs the ambiguity
that line-anchored threads can't, killing the dead-air-on-misunderstand
failure.

### 12.10 Confidence routing, done honestly

Agent self-flagged weak spots (§4.3) are a **routing hint + a calibration
input measured against outcomes** — never blind trust (a confidently
wrong agent flags nothing). The *mismatch* between flag-density and your
actual review-time is the rubber-stamp detector (§12.3 stamp ratio).
Decoration and signal — not the spine of where you look.

## 13. Round-2 hardening (expanded panel)

Round 2 added a pragmatic-TDD, a CI/CD-delivery, and a refactoring
expert, and re-seated the original three for cross-critique. The
convergence was strong; these refinements supersede the rougher edges of
§12 where they conflict.

### 13.1 The economy, unified (and time-separated)

Five "meters" were really **2 stocks + 1 router + 1 exchange rate + 1
receipt**:

```text
TREASURY (tokens) ─spend▶ AGENT ─produces▶ DELIVERABLE
                                              │
            TRUST LEDGER (exchange rate) ─────┤ prices how much
            per agent×task-type×subsystem     │ FOCUS this needs
                                              ▼
   LEVERAGE ranks the queue ──▶ YOU spend FOCUS at a chosen DEPTH
   (order: what's next)            (deep/review/glance — a BET)
                                              │
                                              ▼   CI settles later
                              LAND ▶ MERGE ▶ SHIP QUALITY (receipt)
                                              │
                    earned concurrency ◀──────┘ (more agents = more
                                                 Treasury throughput)
```

- **Trust is a Focus discount.** High verified calibration on a lane
  *refunds* attention — you've earned the right to skim it. That single
  conversion is the spine of the whole economy: *good calibration buys
  back attention.*
- **Leverage = order; Trust = depth.** One sorts the queue, the other
  prices each item. Never let both pull the eye.
- **Ship Quality is downstream-only** — a receipt you cannot spend.
  You can't optimize what you can't spend.
- **Separate the economies in time, not on screen** (the IA rule that
  keeps it calm): **Standup owns Ship Quality** (looking back),
  **the Board owns Leverage only** (*the Board shows a sort, never a
  score*), **Close-out owns Focus** (the day's allocation grade). No
  live draining Focus bar — it's a post-hoc lens on how well attention
  landed.

### 13.2 The Trust Ledger points *outward* — a scouting report, not a mirror

A calibration gauge that grades *you* is Sunday-night dread. Reframe it
as a **scouting report on your agents** (Football Manager / XCOM):
*"`docs-bot` ships clean — 94% first-pass. `auth` runs hot — last 2
skims bounced; recommend a deep read."* Identical data, opposite feel —
you're the smart one reading the roster.

- **Reward the event, not the average.** The dopamine is the
  **"Caught it."** beat when a `blocking:` is *proven real*; calibration
  is just the quiet ledger those accrue into.
- **Force-deep = watching your back, not a hall monitor.** Rare,
  framed as intel (*"`auth` is your blind spot this week — want the deep
  diff?"*), gated hard on real escaped-bug evidence.
- **Endgame = promotion.** An agent/lane with sustained accurate-skim
  history earns **`trusted`** status — it auto-lands low-risk work and
  only surfaces when *it* is unsure. Trust is spent as *the right to
  stop looking* — the actual senior-engineer fantasy.

### 13.3 Delegation Tiers (keep a mastered shop from going slack)

Slay-the-Spire Ascension, applied to delegation depth. You **opt into** a
tier that shrinks your safety net: Tier 1 = forced-deeps active; Tier 3
= no forced-deeps, more concurrency, leaner treasury; Tier 5 = `trusted`
work auto-lands with no glance, you see only self-flags. Highest
*sustained* tier = the honest one-number mastery badge — re-earned if a
miscalibrated tier burns you. Solves late-game ennui without a vanity
grind (it's redeemable against real escaped-bug evidence).

### 13.4 Prep Bench: seeded + pre-flighting the next diff

The Bench (§12.1) only kills dead-air if it's **never empty**. The system
keeps it stocked with 20-second cards even when your backlog is thin:
decompose suggestions, acceptance-criteria stubs, convention prompts
harvested from the live diff — and the killer one: **pre-stage the
self-flagged spots of the agent currently computing**, so the instant its
diff lands you're already pointed at line 47. Onboarding's "second agent"
is a **scripted prep track**, not a second repo — which resolves the
collision between earned-concurrency (start serial) and the Bench
(needs parallel work).

### 13.5 Exploit defenses (the system defines the units)

Red-teaming surfaced two degenerate strategies and one meta-rule:

- **Confirmed-catch farming** (breed weak rev1s to harvest catches) →
  weight a catch by *how hidden it was*: `(1−P(agent self-flagged it)) ×
  (1−P(predictable from the spec))`. Catching your own trap scores ~0.
- **Trust laundering** (build calibration on trivial lanes, spend it on
  risky ones) → risk-tier is **derived from the diff, not the work-order
  title**; calibration is **partitioned by risk-tier** (no upward
  aggregation); trust has a **half-life** (decays without fresh verified
  outcomes — earned concurrency can be *lost*).
- **Meta-rule:** *the system, not the user, defines the units the score
  is denominated in.* The user spends; the user never defines what their
  spending means.

### 13.6 Mutation testing — the independent oracle (resolves "is it real TDD?")

The sharpest round-2 finding: *"confirmed catch = a test flipped" is
farmable because the agent writes the test that flips.* RED→GREEN proves
**sequence, not constraint** (a tautological test shows a pristine arc).
The fix:

- **Diff-scoped mutation** on changed lines at each settle. The checks
  panel shows `mutants killed 9/11 — 2 survived (lines 41, 58)` instead
  of a coverage stat — a heat-map of where green is decorative.
- **Redefine confirmed-catch:** a `blocking:` is confirmed only if a
  mutation on the relevant line **survived before and is killed after** —
  an oracle the agent didn't author.
- **Surviving mutants become auto-`question:` threads** anchored on the
  exact line: *"I changed `>` to `>=` and every test still passed — is
  this covered?"* The tool hands the reviewer the lines where green is a
  lie. (Bold swing: mutation-driven adversarial review.)
- **Plan gate = a behavioral test-list contract** (named cases, not
  prose); landed tests map back to plan items (unplanned coverage and
  unfulfilled contract both flagged). Capture *why* RED was red
  (stub-`NameError` vs asserted failure). Review tests first, code
  collapsed. Green only counts if the test ran in the *settled* revision.

### 13.7 Refactoring as a first-class task-type

The diff-first model is hostile to behavior-preserving change (40
identical hunks break anchoring; the stamp-ratio penalty punishes the
*correct* fast skim of a clean refactor). Fixes:

- **`kind: refactor`** work orders carry a **behavior-preservation
  contract**: touch no test assertions, add no behavior; the proof is
  *"test suite unchanged & still green."*
- **The Invariant View** replaces the hunk diff: three panels — *the
  proof* (tests changed: 0; 412 green; API surface unchanged), *the seam*
  (the one declared transformation), *the exceptions* (any hunk that
  isn't a pure instance of the transformation, promoted to the top —
  that's the whole review). Threads anchor to the **transformation
  step**, not a line.
- **Refactor is the safest thing to skim** when tests are unchanged and
  green — the suite *is* the characterization harness. **Invert the
  stamp penalty** for clean refactors; **force-deep keys on
  assertion-touched / API-changed, not on diff size.**
- Bold swing: **Characterization Gate + equivalence replay** — pin a
  characterization suite green, transform, replay; review the *behavior
  diff*; time-travel (§12.8) binary-searches to the exact step where
  equivalence broke.

### 13.8 Delivery: "Landed" ≠ done; only "Merged" is

The design stopped at the container wall. At fleet scale that ships a
wall of individually-green changesets that don't compose.

- **Land into a merge queue, not into main.** The queue is the
  integration point (rebase onto real tip, run real CI, merge if green).
  *agntpr produces PRs; the merge queue integrates them — don't reinvent
  it.* Card lifecycle gains `Queued → Integrating → Merged | Bounced`;
  a Bounced PR returns to its card for a cheap rebase-retry.
- **Two-tier checks:** *Container checks* (fast, in-box, **advisory**,
  drive the live RED→GREEN feel) vs. *Pipeline* (real downstream CI,
  post-land, via webhook). Never conflate container-green with mergeable.
- **Flaky quarantine before any score.** A failure scores as a
  regression only if it reproduces on rerun, isn't quarantined-flaky,
  and bisects to this session's files. Every *penalty*, like every
  point, must redeem against a real, settled event.
- **Eventually-consistent scoring.** Ship Quality finalizes *overnight*
  as CI settles and back-applies at the next **Standup** — which is what
  gives the session arc real day-to-day continuity.
- **DORA on the dashboard, honestly:** lead time (`task.create →
  Merged`), **change-fail rate** (replaces the noisy raw regression-hit),
  deploy frequency (merges/session), MTTR.

### 13.9 Fleet-wide collision awareness — ships *with* concurrency

Named the #1 risk by two experts independently. **Earned concurrency
without cross-session collision awareness is unlocking the gun before the
safety.**

- The orchestrator computes a **fleet-wide file-overlap graph** across
  all active sessions; overlapping cards show a shared-file accent +
  projected-conflict warning.
- **Disjointness becomes a *scheduling* primitive** (§18 generalized
  from one submit to the whole Board): prefer dispatching work orders
  onto disjoint regions; merge disjoint changesets in parallel batches,
  serialize overlapping ones. Keeping footprints disjoint by design keeps
  integration near-linear instead of O(N²).
- Coordination tools: **rebase queue** (land the cross-cutting refactor
  first; dependents auto-rebase + re-check), **time-boxed refactor
  freeze**, **land-time rebase gate** (approve gated on rebased-onto-tip).

### 13.10 Two new audits

- **Shadow Review** (kills survivorship bias): spend Treasury to
  background-deep-review a landing you only glanced — *most often exactly
  where calibration is high but untested*. A clean shadow makes trust
  *evidence-backed*; a shadow that finds a miss triggers the forced-deep
  *before* a production regression. Converts idle tokens into insurance
  against your own complacency.
- **Disagreement Replay** (calibration you *feel*): at Standup, re-present
  a glance-approved diff that later bounced — cold, with your own
  scroll/dwell trace replayed skipping the line that mattered. Turns the
  Trust Ledger from a scoreboard into a coach. Strictly retrospective,
  opt-in, reuses data already captured.
- **Time-travel, disambiguated** (§12.8 refined): scrubbing is
  **read-only** (sepia "HISTORICAL" ribbon); a past-comment mints a
  **named alternate branch** that appears as a second Board card (an A/B
  bake-off), resolved by a deliberate *pick winner*. Rule: *live
  (non-sepia) cards are current; everything sepia is history.*

---

### Status

Vision captured and twice panel-hardened. The economy (§13.1), trust
model (§13.2–13.3), test-integrity (§13.6), refactoring (§13.7), and
delivery (§13.8–13.9) are now coherent and defensible. Architectural
deltas these imply are recorded in **DESIGN §29**. **Open question to
converge next: which slice do we build first** — DESIGN §17's "prove the
pipe" or §11's "two-agent Board"?
