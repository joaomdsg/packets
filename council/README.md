# The packets Design Council

> A living record of the expert panel that shaped `VISION.md` and
> `DESIGN.md` — who they are, what they argued, and **where they still
> disagree**. Most clashes are settled not by argument but by **evidence
> from a built, tested slice**. Come back after each slice and fill in the
> verdicts.

Layout of this directory:

- This `README.md` — how to use the council, the panelist seeds (§1), the
  contribution ledger (§2), the bold-swings scoreboard (§4), the
  reconvene template (§5), the validating slices (§6), and the round
  index.
- [`clashes.md`](clashes.md) — §3, the open clashes (the heart). Resume
  these after builds.
- `round-NN.md` — one file per round (see the index below).

## How to use this document

1. **Before building** — read the open clashes ([`clashes.md`](clashes.md)).
   Each names the experiment that would settle it; let that shape what the
   slice measures.
2. **After building/testing a slice** — find the clashes it touched, fill
   in the `Verdict (post-build)` field with what happened, and mark the
   clash `RESOLVED` / `STILL OPEN` / `NEW QUESTION`.
3. **To re-summon a panelist** — the persona seeds (§1) are written so a
   future session can re-instantiate the same voice via the Agent tool.
   Original agent IDs are session-scoped and dead; use the seeds.
4. **To run a new round** — use the round template (§5) and add a
   `round-NN.md`.

## Round index

- [Round 3](round-03.md) — first build evidence (mutation oracle slice)
- [Round 4](round-04.md) — latency + the question-thread artifact
- [Round 5](round-05.md) — de-risking the right thesis: economy primitive
  vs rendered surface
- [Round 6](round-06.md) — the next gate: economy-primitive-first vs
  surface/integration-first
- [Round 7](round-07.md) — CONVERGED: ordering & mint-scope closed around #1
- [Round 8](round-08.md) — from clashes to a committed BUILD SEQUENCE
- [Round 9](round-09.md) — the #1 wave shipped: green economy unit → first
  real mint
- [Round 10](round-10.md) — the #2 wave shipped: first real mint + the
  rendered card
- [Round 11](round-11.md) — the thesis catches the project lying: confident
  falsehood on rename
- [Round 12](round-12.md) — integrate-on-tip makes the catch's base sound
- [Round 13](round-13.md) — the felt loop: typed + streamed trace (6/6)
- [Round 14](round-14.md) — the economy is logged but never shown: render
  the first stock
- [Round 15](round-15.md) — cost as a counted invariant before a measured
  ceiling
- [Round 16](round-16.md) — make the lane scarce before you price it: the
  bounded queue
- [Round 17](round-17.md) — the economy's missing half: the sink (pricing)
- [Round 18](round-18.md) — the unlock for ≥2 cards: per-session registry
- [Round 19](round-19.md) — the second session: isolation is a test
- [Round 20](round-20.md) — spend funds a logged work-order (queued, not run)
- [Round 21](round-21.md) — execute the order: the first second producer,
  identity-gated
- [Round 22](round-22.md) — fuel: a work-source backlog so the loop
  compounds
- [Round 23](round-23.md) — legible compounding: split stock by provenance
- [Round 24](round-24.md) — the fleet-read board: ordered by activity, never
  faked leverage
- [Round 25](round-25.md) — the faucet refills: from-catch candidate supply
- [Round 26](round-26.md) — the thesis is PROVEN: golden-replay demo +
  hit-rate
- [Round 27](round-27.md) — the spine is real but parallel: weighing the
  NATS-first pivot
- [Round 28](round-28.md) — the fork resolves: one authoritative log
  (JetStream)
- [Round 29](round-29.md) — #4 NATS→SSE browser bridge shipped: the economy
  streams to the browser off the stream
- [Round 30](round-30.md) — #5 cross-session board aggregator shipped: the
  fleet streams off the one stream (GET /fleet)
- [Round 31](round-31.md) — #6 opens: cross-process producer boundary —
  converged on sequencing, one clash open (cross-boundary trust model)
- [Round 32](round-32.md) — #6 trust model resolved (claim-submission,
  host-side mint); council CONVERGED on #6's design → [`06-plan.md`](06-plan.md)
- [Round 33](round-33.md) — #6c sandbox design on plain Linux containers,
  CONVERGED 5/5 (Security lens added)
- [Round 34](round-34.md) — #6c verification-flow mechanics: SHA-pinned
  transport, read-only dep cache, host-derived verdict; egress proxy dropped
- [Round 35](round-35.md) — #6c slice C3b: the producer claim lifecycle
  (in-flight / verified-lost) surfaces on the board + live fleet stream, not
  the host card; lie-green fixed; opened Clash J (the gray-bets visual)
- [Round 36](round-36.md) — #6c slice C4: the gray bets-vs-confirmed visual —
  settling Clash J (stylesheet vs CSS-free semantic distinction)
- [Round 37](round-37.md) — sequencing the next #6 slice: CONVERGED 4/4 on
  permanent-vs-transient verify failure first (reject an unverifiable claim
  instead of lingering), then SHA transport, then governor hardening
- [Round 38](round-38.md) — slice A (SHA transport) threat model: CONVERGED on
  bundle-over-authenticated-channel + per-producer ref namespacing; host-pull
  rejected on SSRF/egress (Clash K resolved)
- [Round 39](round-39.md) — scoping slice B (governor hardening): verify-path is
  sufficient; build GC-by-resolved now (economy-safe); DEFER flood-defenses
  (rate-limit/quota) until producer AUTH exists on the live HTTP surface
- [Round 40](round-40.md) — direction after #6c feature-complete: the live
  boundary is maintainer-gated; next NON-GATED slice = the non-ASCII re-anchor
  correctness fix; the Board is the next major thread (its own round)
- [Round 41](round-41.md) — scoping the Board: the loop reaches the edge of
  autonomous-safe work — high-value Board increments need maintainer product/UX
  taste (or hit the refused fabricated-leverage trap); remaining work is gated or
  taste-dependent → surface the direction fork to the maintainer
- [Round 42](round-42.md) — maintainer re-delegated direction to the council:
  build the per-session RECENT-DISPATCHES legibility surface (honest caught/missed
  per work-order; no fabricated rank/treasury, CSS-free, non-gated)
- [Round 43](round-43.md) — UX/UI VISUAL-DESIGN direction opens (maintainer-
  authorized): CSS now in scope; converged on a calm control-room visual language
  + flows/nav (deferred) + guardrails; first slice = base stylesheet on existing
  hooks via AppendToHead, no markup change
- [Round 44](round-44.md) — first nav / drill-and-return flow: a shared nav
  header (packets home + fleet breadcrumb) + clickable board rows → /?key=<key>
  → back; honest raw keys, no menus/JS/keyboard yet (keyboard nav = R45+)
- [Round 45](round-45.md) — per-state visual polish: color the verdict + land
  states in the calm honest palette (pure CSS on existing data-state hooks,
  selector-coverage tested); keyboard nav deferred to R46
- [Round 46](round-46.md) — first-run onboarding affordance: a fresh session
  (stock.Count==0) renders a calm guide to the core loop instead of a dead screen
  of zeros; chosen OVER keyboard nav because the latter is browser-side behavior
  our server-render tests can't verify ("prove it for real"). Server-rendered +
  fully vt-testable; keyboard nav deferred to R47+
- [Round 47](round-47.md) — the Spend control: render a balance-gated trigger
  bound to the (already-tested) Spend action, closing the core loop from the UI
  (catch → balance → spend → work-order → reinvest). Shown iff balance>0; retracts
  live over SSE when the last catch is spent. Fully vt-testable (markup + fire +
  drain). keyboard nav deferred to R48+
- [Round 48](round-48.md) — per-order dispatch round-trip on the live card:
  surface this session's RecentDispatches (WO#id path:line caught/missed) below the
  counts, so the Lead watches the order they funded resolve on the SAME surface
  they spent on (reuses the board's renderDispatches; rides SSE; omitted when no
  orders). keyboard nav + "nothing-to-fund" affordance deferred to R49+
- [Round 49](round-49.md) — preview what the Spend funds: the Spend control names
  the actual next target ("Spend a catch → fund alpha.go:8") via a pure helper, so
  the Lead makes an informed choice (honest — same nextUnconsumedTarget the action
  funds). REJECTED the "nothing-to-fund" affordance as near-unreachable (supply
  refills from own catches). Lesson: verify a slice's target state is reachable
  before building. keyboard nav deferred to R50+
- [Round 50](round-50.md) — color the dispatch round-trip outcome: caught→confirmed
  hue, missed→loss hue (a data-outcome hook on renderDispatches, shared card+board),
  extending R45's per-state honest color to the round-trip; queued/running stays
  neutral. Closes the card/board VISUAL+FLOW thread opened at R43 — the survey found
  remaining work is marginal polish OR a bigger thread (menus, order diagnostics,
  fleet actions) needing a maintainer steer; loop will surface that, not manufacture
  marginal slices.
- [Round 51](round-51.md) — ORDER-DIAGNOSTICS thread (slice 1): persist the
  oracle's per-order verdict as a ledger fact (kind "woverdict"), projected into
  DispatchView.Verdict — so a missed order can show WHY, not just "done, not caught".
  Maintainer gave the council creative latitude; a 3-voice council chose this over a
  reinvest-rhythm loop (marginal) and a session-menu (reachability-blocked: boot-time
  consumer snapshot). Diagnostic only (two-scores: no balance/catch). R52 renders it.
- [Round 52](round-52.md) — ORDER-DIAGNOSTICS thread (slice 2): RENDER the per-order
  verdict on card + board (shared renderDispatches appends a dim board-row__dispatch-why
  span on a resolved order: "done missed no-catch"), so a miss is diagnosable
  (no-catch vs lost-via-rename vs no-oracle-signal), not undifferentiated. Calm dim
  secondary detail; omitted when none persisted. The diagnostics loop now reads
  end-to-end: spend → dispatch → resolve → SEE WHY.

Session-scoped agent IDs from rounds 1–2 (dead, provenance only): UX
`a985fda4…`, Game design `af9d2f4c…`, Systems `a494dd62…`, TDD `afcf847e…`,
CI/CD `a5b74ebb…`, Refactoring `a172b669…`.

## 1. The panelists (persona cards)

Each card: their lens, north star, what they reliably push for, signature
bold swing, and a **re-summon seed** (the essence of the prompt that
produces this voice).

### 🎨 The UX Designer

- **Lens:** product UI/UX; Linear / Vercel / Raycast sensibility.
- **North star:** _calm, power-user-first, keyboard-native._ A meter you
  can't act on right now is noise.
- **Reliably pushes for:** fewer surfaces, information gated to the moment
  it's actionable, motion that reports a real state change.
- **Bold swings:** Time-travel review (bidirectional Ledger scrubber); the
  Disagreement Replay (calibration you _feel_, not read).
- **Re-summon seed:** _"World-class product designer. Constraint: pure-Go
  Via framework over SSE, no SPA, Monaco only as a plugin island. Calm
  control-room aesthetic, keyboard-native. Allergic to gauges that induce
  guilt or can't be acted on. Render mechanics without building a cockpit
  of meters."_

### 🎮 The Game Designer

- **Lens:** management/tycoon games; Factorio / Frostpunk / Two Point
  Hospital / Slay the Spire.
- **North star:** _the work already is a game; make it feel like one
  without lying._ Honest hooks, not dark patterns. Queue-zero is a win
  you're allowed to walk away from.
- **Reliably pushes for:** killing dead-air, an ethical "just one more,"
  framing that flatters competence, a real mastery curve, beats that
  punctuate.
- **Bold swings:** The Trust Ledger (calibrated delegation is the game);
  Delegation Tiers (Ascension — opt-in shrinking safety net).
- **Re-summon seed:** _"World-class tycoon/management game designer.
  packets is an HONEST game (mechanics must map to real dynamics, no fake
  XP/confetti, no engagement dark patterns). Obsessed with moment-to-moment
  feel, pacing across 30s/30m/session loops, and whether a mechanic
  flatters or grades the player."_

### ⚙️ The Systems / Economy Designer

- **Lens:** game systems & economies; Factorio logistics / RimWorld
  incident economy / Into the Breach perfect-information.
- **North star:** _one scarce resource, one conversion, one loop that
  punishes the obvious cheat._ Meters aren't an economy until they trade.
  The system defines the units; the user only spends.
- **Reliably pushes for:** collapsing meters into roles, red-teaming for
  degenerate strategies, tying every reward to a logged downstream fact.
- **Bold swings:** The Focus meter (attention as the central spent
  resource); the Shadow Review (spend tokens to audit untested trust).
- **Re-summon seed:** _"World-class game systems/economy designer. Hostile
  to 'meter soup' — insists on stocks vs rates vs scores. Always asks:
  what's the degenerate strategy, and what logged fact redeems each point?
  Every exploit is an attempt to control the denominator."_

### 🧪 The Pragmatic TDD Expert

- **Lens:** test-driven development, Kent-Beck-pragmatic (not dogma).
- **North star:** _tests must constrain behavior, not just exist and
  pass._ Ceremony decoupled from constraint is theater.
- **Reliably pushes for:** an independent oracle (mutation testing),
  test-list-as-contract at the plan gate, reviewing tests before code,
  distinguishing "RED for the right reason" from a birth-cry.
- **Bold swing:** Mutation-driven adversarial review (surviving mutants
  become `question:` threads — "green is a lie here").
- **Re-summon seed:** _"World-class pragmatic TDD practitioner. Core
  question: does this mechanic produce well-tested code or
  confidently-green test-theater? Knows RED→GREEN proves sequence not
  constraint, and that mutation score is the only non-gameable signal.
  Knows when a test earns its keep and when it's ceremony."_

### 🚀 The CI/CD & Delivery Expert

- **Lens:** continuous delivery, DORA, trunk-based dev, merge queues,
  progressive delivery, flaky-test management.
- **North star:** _"Landed" ≠ done; only "Merged" through real CI is._
  Don't reinvent integration — feed a merge queue.
- **Reliably pushes for:** an explicit integration point, two-tier checks
  (container-advisory vs real-pipeline), flaky quarantine before scoring,
  honest DORA metrics, controlling superlinear integration cost.
- **Bold swing:** Speculative integration preview (background-rebase onto
  tip + CI before you approve).
- **Re-summon seed:** _"World-class CI/CD & delivery expert. Sees the seam
  where the design hands N changesets to downstream CI and walks away.
  Worries about stale-base collisions across the fleet, scoring humans on
  noisy/flaky downstream truth, and O(N²) merge-queue cost."_

### 🔧 The Refactoring Expert

- **Lens:** large-scale refactoring; Fowler / Feathers "Working
  Effectively with Legacy Code."
- **North star:** _behavior-preserving change keeps codebases alive — and
  it's the safest thing to skim when tests are trustworthy._
- **Reliably pushes for:** refactor as a first-class task-type with a proof
  (tests unchanged & green), reviewing the invariant not the hunks,
  transformation-anchored threads, cross-session collision safety.
- **Bold swing:** Characterization Gate + mechanical-equivalence replay
  (review the proof of invariance, time-travel to where it broke).
- **Re-summon seed:** _"World-class refactoring expert (Feathers/Fowler).
  Tests whether the diff-first/anchored/fan-out model supports healthy
  refactoring or punishes it as churn. Knows a 40-file rename breaks
  line-anchoring and that a clean refactor with green unchanged tests is
  the safest possible skim."_

## 2. Contribution ledger

Where each panelist's ideas landed in the docs (Round 1 / Round 2).

- **UX** — R1: leverage-not-cost, brief rail (V §12.2/§12.9). R2:
  time-separated economy, outward Trust render, time-travel
  disambiguation, Disagreement Replay (V §13.1/§13.2/§13.10).
- **Game design** — R1: Prep Bench, session arc, earned concurrency
  (V §12.1/§12.6/§12.7). R2: outward Trust Ledger, Delegation Tiers,
  seeded+pre-flight Bench (V §13.2/§13.3/§13.4).
- **Systems** — R1: leverage formula, net-quality scoring, Focus, exploit
  framing (V §12.3/§12.5). R2: unified economy, exploit patches, CI-truth
  loop, Shadow Review (V §13.1/§13.5/§13.10; D §29.3).
- **TDD** — joined R2: mutation oracle, test-list contract, confirmed-catch
  redefinition (V §13.6; D §29.4).
- **CI/CD** — joined R2: merge queue, two-tier checks, "Merged not Landed",
  flaky quarantine, DORA (V §13.8; D §29.1/§29.2/§29.5/§29.9).
- **Refactoring** — joined R2: refactor task-type, Invariant View, fleet
  collision guard (V §13.7/§13.9; D §29.6/§29.7).

## 4. The bold swings scoreboard

The signature bets and their status.

- **Mutation-driven adversarial review** (TDD, high conviction) —
  **VALIDATED.** Confirmed-catch minted end-to-end through the §17 pipe
  (`pipe.RunCatchCycle`, #3): two real settles → worktree mutation×2 →
  reanchor → CatchAcross → real Catch; edited-anchor → NoOracleSignal. Now
  rendered as a distinct live state (`surface.ReviewCard`, #4) and watchable
  over HTTP/SSE (R10 #10). Pricing pending integrate-on-tip (#12).
- **Trust Ledger (calibrated delegation)** (Game, spine) — _TBD_,
  framing-risk (Clash H).
- **Merge-queue-as-integrator** (CI/CD, low-risk) — **R12 #12 SHIPPED.**
  `pipe.integrateOnTip` rebases the fix onto a real tip + runs checks on the
  integrated tree → typed Land {clean|conflict|checks-red}, one serialized
  lane. Clash C resolved-in-code. Remaining: the single-lane queue over K
  branches + the #15 K-concurrent benchmark before pricing.
- **Focus as central resource** (Systems, adopted) — _TBD_, render-risk
  (Clash A).
- **Speculative integration preview** (CI/CD) — _TBD_, high value, infra
  cost.
- **Characterization Gate + replay** (Refactor, scoped to refactors) —
  roadmap #2: adversarial refactor trace as RED baselines (rename_40 /
  neutral_move / extract_module). **R11 #11: Clash G surface-honesty half
  RESOLVED-IN-CODE** — a renamed/edited anchor renders a true terminal
  (`surface.LostViaRename`/`AnchorEdited`) instead of the false "no mutable
  operator" (residual: rename-cliff coarsening → #11.5).
- **Time-travel review** (UX) — _TBD_, distinctive, value-unproven (Clash I).
- **Delegation Tiers (Ascension)** (Game) — _TBD_, late-game depth,
  premature.
- **Shadow Review (anti-survivorship)** (Systems) — _TBD_, elegant,
  token-cost unclear.
- **Disagreement Replay (coaching)** (UX) — _TBD_, cheap, depends on
  dwell-tracking.

## 5. Reconvene template (for future rounds)

Copy this block into a new `round-NN.md` and fill it in.

```markdown
# Round N — <focus> — <date>

Trigger: <what prompted it — e.g. "after building the mutation-thread slice">
Panelists present: <which personas, + any new lens added>
New evidence on the table: <build/test results that inform the debate>

Per panelist:

- <persona>: <their take this round, 1-3 lines>

Clashes touched: <A–I + any new>
Verdicts updated: <which clashes moved, and to what>
New clashes opened: <…>
Decisions: <what changed in VISION/DESIGN as a result>
```

## 6. The validating slices these clashes are waiting on

From the build plan — the slices most likely to produce verdicts:

1. **Pipe + one review round-trip** (DESIGN §17 + a single comment →
   revision cycle) → informs Clash B framing baseline.
2. **Mutation-as-`question:`-thread** (D §29.4) → informs Clash B, F.
   **BUILT & TESTED** (`internal/mutation`) — logic layer only.
3. **Two-agent Board with queue-to-zero loop** (VISION §11) → informs Clash
   A, D, E, H — the "does it feel like a shop" thesis.
4. **A real refactor through the Invariant View** → informs Clash G.

When any is built and tested, return to [`clashes.md`](clashes.md), fill
the verdicts, and log a round.
