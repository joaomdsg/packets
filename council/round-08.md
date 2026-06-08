# Round 8 — from validated clashes to a committed BUILD SEQUENCE: ranking the slices to a working full prototype — 2026-06-04

Trigger: new charge — commit to building agntpr as a WORKING FULL
PROTOTYPE, advancing one validated green-tested slice at a time. Round 7
converged on #1 and said "next event is a BUILD," yet the tree is
byte-identical (only test-infra commits since: testify migration +
CONVENTIONS compliance). The council must now CHART and CONFIRM the build
sequence to a usable prototype. Build state unchanged: 6 green backend
packages (mutation/review/settle/diff/translate/orchestrator); `grep
Catch|Trust|Focus|Treasury|Ledger internal/` → ZERO typed events; `grep
fetch|rebase|merge-base|integrate|onto` (non-test) → ZERO integration
primitives; no http/SSE/template surface.

Panelists: all six. No new lens.

New evidence (verified in code):

- `mutation.Result` = `{Findings []Finding, MutantsConsidered int}`; each
  Finding keyed only by `(Line, Original, Mutated)` STRINGS
  (generate.go:24-26, runner.go:40-57) — no stable mutant identity. The
  tri-state catch maps cleanly onto this as a pure differential over
  `Run()` — no oracle rewrite.
- `translate.go:80-95` emits `activity.agent {thinking|editing|tool}` +
  `turn.ended`; `orchestrator` emits `TurnOutcome{Minted,SHA,Added,
  Deleted,Diff,Secrets}` — a live heartbeat + a settle beat w/ NO surface.
- `diff.go:42` `--no-renames` hardcoded (rename → delete+add → anchors
  evaporate); `orchestrator.go:37,49` baseRev immutable; `thread.go:40`
  a `question:` on EVERY survivor; `review/thread.go:27` `Render()` is
  string concat.

## Per panelist

- UX: 7 rounds, 0 pixels — render-dissent SHARPENED by the prototype
  goal: a "full WORKING prototype" is unusable until a human can SEE one
  card, so the roadmap MUST end at a usable surface, not another economy
  primitive. Ratify the catch (concession holds), but build the §17 pipe
  + Via/SSE surface as a PAIR right behind it — FOUR outcome states + a
  designed IN-FLIGHT state + a designed empty/zero "N considered, 0
  survived — tested" state (the MOST COMMON screen, undesigned, reads as
  "broken"). Meters OFF the first screen.
- Game design: a usable prototype is one you can sit in front of. The
  catch is the right spine but minting it before there's a Board is
  building scoring before the game. FLIP the order: pipe-to-a-live-card
  FIRST (zero new backend, only wiring), catch as build #2 layered onto a
  loop w/ a human in the chair, so the first catch is WITNESSED. NEW:
  in-flight has no defined TEMPO — raw passthrough reads as log-spew or a
  frozen card; it needs a designed CADENCE (debounce/coalesce), tunable
  only against a live replay.
- Systems: RATIFY #1 a 4th time, but it converts from spec to a real Go
  TYPE this round or it is not built: one typed `Catch{Anchor,
  BeforeInventory, AfterInventory, Outcome}` where Outcome is a PURE
  function of the two operator-inventory sets. Build the identity key as
  a real type FIRST inside this brick so one function owns the
  denominator. The §17 surface is buildable now on today's
  single-revision set (zero new backend), build it in parallel, but do
  NOT mint an inferred catch-weight alongside it.
- Pragmatic TDD: substrate confirmed — tri-state catch is a pure
  differential ON TOP of `Run()`, no rewrite. Stop deliberating; build #1
  via tdd-rygba w/ fix-edits-anchored-line as the RED proving "same
  mutant killed" incoherent, carry re-anchoring in-scope, ship
  no-oracle-signal as first-class. Caveat: a green RED→GREEN proves the
  transition FIRES, not that it CONSTRAINS — the degenerate suite
  (agent-authored killing test, no-op churn must-not-mint) earns its
  keep. CI/CD's rebase dependency documented as an open gap, settled by
  the trunk-moved variant.
- CI/CD & Delivery: still ZERO integration primitives. "Landed" is not
  done; only "Merged" through real CI is — this prototype has no Merged
  state. Concede #1 (economy spine), do NOT block; but the catch is
  minted on PRE-INTEGRATION coordinates — re-anchoring (§28) maps WITHIN
  the branch, NOT across a moved trunk. Build integrate-on-tip
  {clean|conflict|checks-red} on the rebased tree to convert
  Landed→Merged; fixture (B) clean-rebase-but-checks-red is the killer
  proving a green pre-integration catch can be a red post-integration
  regression. Under the N-agent goal this is a SERIALIZATION POINT —
  build ONE merge-queue lane, not N rebases. Price the catch only against
  an integrated base.
- Refactoring: ratify #1, but its re-anchor prerequisite is where
  refactors go to die, so the concurrent refactor trace is the acceptance
  bar that prerequisite is built against — NOT optional garnish.
  `--no-renames` (diff.go:42) downgrades a 30-file rename to delete+add →
  threads orphan, transition computed against a vanished line → silent
  no-oracle-signal at best, phantom catch at worst. A clean refactor w/
  green tests is the SAFEST skim, yet today's tree treats it as MAXIMAL
  noise. Build the RED baselines NOW, SAME wave as the re-anchor
  sub-brick, BEFORE the §17 surface.

Clashes touched: F (identity half — ratified a 4th round AND converted to
a required TYPE: `BeforeInventory`/`AfterInventory` pure-function
denominator becomes the brick's contract, inventory-change rule the
load-bearing RED); C (integrate-on-tip remains the experiment, ordering
re-stated "precede catch PRICING not minting," settled by fixtures A +
B); G (refactor trace re-ratified concurrent; rename_neutral_move
stresses F's identity key); D (Game nominates human-dwell, CI/CD
nominates integrated-checks-RED — opposite experiments on the same card,
both deferred to #7/#5); A (silent-vs-badge becomes #4's acceptance bar);
B (self-flag + would-have-shipped captured at mint, data-only); E/H/I
(render-camp — unblock once surface + loop exist; roadmap now REACHES
them).

## Verdicts updated

- Clash F: remains PARTIALLY RESOLVED but #1 is now BUILD-READY and
  TYPE-COMMITTED — the identity key becomes a real Go type owned by one
  pure function THIS build, not prose; tri-state + partial-catch +
  no-oracle-signal-as-first-class stand. Verdict flips on the green #1.
- Clash G: still TBD; experiment re-CONFIRMED concurrent and re-scoped as
  the acceptance bar for the re-anchor sub-brick (SAME wave, not after).
- Clash C: gains both settling fixtures (trunk-moved A; clean-rebase-
  checks-red B); ordering refined — integrate-on-tip precedes catch
  PRICING, not MINTING.
- Clashes A, D, E, H, I: remain TBD; roadmap terminates at the surface +
  two-agent loop that unblock them, w/ named experiments.

New clashes opened: NONE at target level — #1 held a 4th round, HARDENED
into a typed contract. The build-ORDER split (UX+Game pipe-first vs field
catch-first) is a scheduling sub-dispute, NOT a new clash; per the
Round-7 bar it does not block convergence. Game's in-flight-CADENCE is an
additive requirement folded into surface build #4.

## Decisions

1. NEXT BUILD (#1, 6/6 ratify the brick; build-order dissent recorded,
   chair-resolved): the tri-state confirmed-catch oracle as a typed
   two-revision differential over `Run()` — `Catch{Anchor,
   BeforeInventory, AfterInventory, Outcome}`, Outcome ∈ {Catch | NoCatch
   | NoOracleSignal | PartialCatch}, survivor-SET non-empty→empty on the
   same anchored line, NEVER "same mutant killed." The identity key
   (denominator = line's current operator inventory per revision) becomes
   a REAL TYPE owned by one pure function THIS build, w/ the
   inventory-change rule (fix edits L + changes inventory → ill-typed →
   NoCatch). Built via tdd-rygba; fix-edits-anchored-line the load-bearing
   RED. First economy object + first adversarial acceptance entry.
2. #1 PREREQUISITE sub-brick, BUILD FIRST: from-base re-anchoring
   (§28/§14), "lost via rename" a distinct state (→ NoOracleSignal, never
   a phantom Catch). Document the OPEN gap: re-anchoring does NOT survive
   an integration rebase; the catch is minted on pre-integration coords.
3. CONCURRENT with #1 (no shared prereq, today's green tree), SAME wave:
   the adversarial refactor trace — internal/refactor/testdata/
   {rename_40, rename_neutral_move, extract_module}, each w/ a GREEN
   unchanged suite; RED baselines: orphanedThreadCount>0; survivor-set
   ill-typed across rename (lost-via-rename != Catch); extract-module
   re-mutated as net-new. Settles Clash G; de-risks #1's sub-brick.
4. CAPTURE AT MINT (cheap, un-backfillable, data-only, IF it does not
   delay the definition): self-flag bit + would-have-shipped proxy on the
   Catch record — NO weight, NO pricing (guards inflationary Ledger /
   forbidden catch-weight V§13.5).
5. RANKED ROADMAP to a USABLE PROTOTYPE (each w/ its experiment):
   [#1-sub] re-anchoring [rename re-anchors / lost-via-rename distinct];
   [#1] tri-state catch [three-case + degenerate suite]; [#2 concurrent]
   refactor trace [RED carnage baselines]; [#3] §17 pipe end-to-end on
   ONE real changeset [one comment → one revision, mutation at settle,
   thread anchors — the single-user happy loop dispatch→settle→review→
   land]; [#4] Via/SSE single-card surface, FOUR outcome states +
   designed in-flight/streaming state w/ designed cadence, against
   today's single-revision oracle [silent-vs-badge reads "tested, ship
   it" not guilt; streaming-no-flash], METERS OFF first screen; [#5]
   integrate-on-tip {clean|conflict|checks-red} [trunk-moved (A) +
   clean-rebase-checks-red (B)] — if (A) shows the transition fails a
   rebase, #5 precedes catch PRICING; [#6] single-lane merge queue
   wrapping #5 [throughput-to-zero]; [#7] two-agent Board loop
   instrumented for idle/dwell [Clash D real N-ceiling via
   rework-vs-concurrency].
6. BLOCKERS before the prototype is reachable: (a) re-anchoring (§28) is
   a HARD prereq of #1 AND known-incomplete (does not survive rebase) —
   refactor trace #2 quantifies the carnage in the same wave; (b) the
   survivor-set has no identity key until #1 makes it a type; (c) the
   single-user happy loop (#3) MUST exist before any economy stock is
   rendered — meters (Focus/Trust/Treasury) stay OFF the first screen;
   (d) catch PRICING (not minting) is gated on #5 if the trunk-moved
   fixture shows the transition is rebase-dependent.
7. RISKS.md carries the four code-level risks from R5-7 (oracle latency;
   thread.go churn miscalibration; orchestrator.go immutable-baseRev;
   survivor-set no-identity-key); run the K-concurrent-settle benchmark
   before any catch pricing. NO VISION/DESIGN text changed
   (12-contradiction reconciliation pass queued per RISKS step 5).

CONVERGED (4th consecutive round): 6/6 lenses ratify the tri-state catch
oracle as the #1 brick — now type-committed — and 6/6 affirm the roadmap
REACHES a usable §17 pipe + Via/SSE surface (four outcome states +
designed in-flight) rather than terminating at an economy primitive. Sole
dissent is build-ORDER (UX+Game: pipe-before-catch), a scheduling
sub-dispute the Round-7 bar does not let block; chair resolves it
catch-first because the catch is the only un-backfillable primitive and
the pipe+surface follow at #3-#4. Next event is a BUILD — re-anchoring
sub-brick + tri-state catch (#1) + refactor trace (#2) in one wave — not
another round.
