# Round 10 — the #2 wave SHIPPED (first real mint + the rendered card): from green islands to a watchable prototype — wiring the seam — 2026-06-04

Trigger: the Round-9 marching orders BUILT — second consecutive
post-build wave. The Round-8 brief snapshot is now STALE by 4 commits (it
claims zero economy/surface — wrong). Charge holds: under the
full-prototype goal, confirm/refine the next build, rank the slices to a
USABLE prototype, flag prototype blockers. Build state: 11 green packages
(`env -u GOROOT go test ./internal/...` → 11 ok); the R9 wave landed the
reanchor→catch JOIN, the §17 first-real-mint pipe, and the Via/SSE
surface card.

Panelists: all six. No new lens.

New evidence (verified in code):

- `internal/pipe.RunCatchCycle` (pipe_cycle.go:45) mints ONE typed catch
  from TWO REAL settles end-to-end: settle→throwaway worktree→mutation×2
  →reanchor→CatchAcross→Detect. `CatchAcross` is the JOIN (pipe.go:23).
  Land returns `Unintegrated` (pipe_cycle.go, LandState const) — honest,
  never fake-merged; CI/CD's non-negotiable HELD in shipped code.
- `internal/surface/card.go` renders all four `catch.Outcome` states +
  `Tested` (separate const, card.go:20) + in-flight, each a distinct
  `data-state` marker (present(), card.go:53-71). Tested reads "Tested —
  ship it" (card.go:67-68); NoOracleSignal distinct from Catch
  (card.go:59-61). Tested over httptest SSE incl.
  `_streamsEachVerdictAsLivePatch` and
  `_hostileVerdictCannotBreakOutOfTheStateAttribute`;
  `_carriesNoEconomyMetersOnTheFirstScreen` locks meters OFF.
- BUT `grep ListenAndServe|Flusher|event-stream|via.New|via.Run`
  (non-test) → ZERO. No `cmd/`. The card is exercised only by the via
  test client; NOTHING serves it — a human still cannot open a browser
  and watch a catch.
- The pipe→card EDGE does not exist: zero prod caller joins
  `CycleResult.Outcome` → `ReviewCard.Sig`. Two islands tested apart.
- `CycleResult.Trace` is a flat `[]string` of `fmt.Sprintf` lines
  (pipe_cycle.go:36,52,88) — NO timestamps, NO event types.
  Capture-at-mint (R6/R7/R8/R9 #4) is ABSENT: `CycleResult` carries
  Outcome/Path/Line/Land/Trace but NO persisted Catch record, no
  self-flag, no would-have-shipped proxy. `grep selfflag|would_ship|
  persist|ledger` (non-test) → 0. Every real mint discards un-backfillable
  data; zero integration primitives (`grep rebase|merge-base|integrate`
  non-test → 0).

## Per panelist

- UX: ten rounds and the card EXISTS and is HONEST — every state I fought
  for is green, incl. the in-flight beat and the calm-win "Tested — ship
  it." But a tested component is not a prototype; a prototype is something
  a human OPENS. Build the SSE-served single-card surface wired to a real
  RunCatchCycle next: the server + the pipe→card edge, not new render
  logic. NEW: the bare `Outcome` cannot express Tested vs in-flight vs
  blind-NoOracleSignal — the seam must map pipe state and carry WHY a card
  is quiet, or the calm-win screen collapses into the blind-silence screen
  (the R9 two-quiet-meanings collision, now at the wire). Meters off first
  screen.
- Game design: we minted the coin, type-checked the card, and STILL
  nobody has watched a catch happen — 9 rounds, zero pixels; an exhibit
  behind glass, not a prototype. Boot a real Via SSE server, let a human
  watch one recorded catch land as a single beat. NEW (un-met R9 demand):
  the trace is an UNTIMED, UNTYPED `[]string` — a replayable trace without
  time is a log, not a replay. Fix Trace to `[]TraceEvent{T,Kind,Msg}` so
  cadence can be tuned — feel lives on the time axis.
- Systems: 11 green pkgs, the mint is no longer an island at the catch-pkg
  level. But TWO NEW islands replaced the old one: pipe↔surface have NO
  edge, and capture-at-mint is STILL un-built — every mint discards the
  denominator. Build the end-to-end SEAM + the CatchRecord in ONE wave:
  one runnable cmd + SSE server streaming `CycleResult.Outcome` into the
  card, AND persist a typed `CatchRecord{Outcome, Anchor, BeforeInv,
  AfterInv, SelfFlagged, WouldHaveShipped}` as the mint's only durable
  artifact — data-only, NO weight/pricing. One scarce object, one
  conversion, one durable record, recorded BEFORE any stock renders. The
  seam must carry the quiet-discriminator, not reconstruct it surface-side.
- Pragmatic TDD: the mint is open and it CONSTRAINS — eight rounds of
  "beautiful odometer, no trip counter" is finally a trip counter that
  counted a real trip (Clash F's loop half RESOLVED-IN-CODE end-to-end).
  But we mint a catch and drop it on the floor: RunCatchCycle returns a
  string and persists NOTHING. PULL capture-at-mint FORWARD as the
  immediate build — the only un-backfillable thing the live mint is
  destroying. SAME build: close the farm-untested-end-to-end gap — drive
  no-op-churn and agent-authored-killing-test fixtures THROUGH
  RunCatchCycle, not literals; the test that earns its keep is "no-op
  churn writes NOTHING to the log." Persist WITH the WHY (MutantsConsidered
  + a reason tag), or we store outcomes we can never audit for theater —
  the exact failure the oracle was built to prevent, at the persistence
  layer.
- CI/CD & Delivery: a real mint and card shipped and my non-negotiable
  HELD (land is Unintegrated, not fake-merged). But honesty as a
  placeholder is where prototypes lie to first users; only integration
  through real CI is done, and the merged state is still a string
  constant. Next build is integrate-on-tip as ONE serialized merge-queue
  lane replacing the Unintegrated const: rebase onto tip in a throwaway
  worktree reusing runOracleAt — abort→Conflict, integrated checks
  non-zero→ChecksRed, else Clean. Load-bearing RED: clean-rebase-but-
  checks-red (disjoint lines, cross-file break). I do NOT block the
  surface; I block it being PRICED against a fake integration verdict.
  NEW cost-sharpening: the cycle already runs the oracle twice, integrate
  adds a third run, the queue re-runs every card per tip — the
  K-concurrent benchmark must run on the INTEGRATED cost before pricing.
- Refactoring: we did the right builds — #1, the refactor baselines, the
  pipe, the card — and they're green; my Clash-G trace stands (40-file
  rename orphans 40/40, neutral rename → LostViaRename != Catch,
  extract-module invisible). But six green islands are not a prototype.
  The single highest-leverage move is the end-to-end WIRE (cmd + via app +
  SSE-fed card driven by a real settle) — it needs NO new economy and NO
  integrate-on-tip, only built+green parts. NEW: the card is anchored to a
  SINGLE line's verdict and has NO designed state for "anchor lost to a
  rename" — a renamed file's card sits in in-flight FOREVER, lying about a
  terminal lost state. The refactor task-type breaks the SURFACE state
  machine, not just the oracle; it needs a fifth designed terminal state.
  Wire it, then give lost-via-rename its state.

Clashes touched: F (the LOOP half FLIPS to RESOLVED-IN-CODE —
RunCatchCycle mints a real Catch from two real settles and the
inventory-change rule survives the real reanchor path; residual is the
farm-denial-end-to-end gap — no-op-churn and agent-authored-killing-test
still asserted only against literals, closed by this wave's log-assertion
fixtures); A (silent-vs-badge + NoOracleSignal-vs-Catch becomes
CONCRETELY runnable the moment a human watches the served card); C
(untouched in verdict — land stays honestly Unintegrated; integrate-on
-tip moves toward the #12 build w/ fixtures A+B; pricing gated
downstream); G (carnage baselines stand green but the punishment of
behavior-preserving change is about to become USER-VISIBLE — the surface
has no honest terminal state for a renamed anchor); B (capture-at-mint
finally lands the self-flag + would-have-shipped + reason-tag columns the
Catch type has lacked since R6 — data-only); D (single-lane vs N-rebase
named as the first concrete N-ceiling mechanic, deferred to #13); E/H/I
(render-camp — roadmap now ONE build from the watchable surface).

## Verdicts updated

- Clash F: → RESOLVED-IN-CODE on BOTH the unit and the loop (carried from
  the R9 #3-pipe verdict, re-confirmed on the 11-green tree). Residual is
  no longer oracle but ECONOMY-PLUMBING: the Catch is minted but NOT
  PERSISTED (capture-at-mint, this wave) and the farm-denial is asserted
  on literals not through RunCatchCycle (closed this wave by asserting on
  the event LOG). NEW persistence-layer concern logged (TDD): a
  tautological-tests line (real fix kills nothing) is indistinguishable
  from already-strong at the NoCatch token — the CatchRecord MUST carry
  MutantsConsidered + a reason tag or the Ledger stores un-auditable
  outcomes.
- Clash A: gains its first RUNNABLE acceptance experiment — a served card
  whose silent/Tested vs catch render is watched by a human (reassurance
  vs guilt), no longer a thought experiment; flips off pure-TBD toward
  "buildable-this-wave."
- Clash C: unchanged verdict (catch minted pre-integration, land honestly
  Unintegrated); integrate-on-tip sharpened into the #12 build w/ fixtures
  A (trunk-moved) + B (clean-rebase-checks-red, the load-bearing RED), and
  the cost-sharpening (third integrated run + per-tip re-runs) folded into
  the #15 benchmark gate before pricing.
- Clash G: carnage baselines remain RESOLVED-in-code; a NEW surface-level
  residual logged (no terminal lost-via-rename state → renamed card lies
  in-flight forever) — NOT a fresh clash, an additive surface-state
  requirement scheduled at #11.
- Clashes D, E, H, I: remain TBD; roadmap terminates ONE build from the
  surface (#10) and reaches the two-agent Board (#16) + pricing (#17).

New clashes opened: NONE at target level — the #2 (R9) wave shipped green
and the council converges on the end-to-end wire as next build. The
build-ORDER/scoping splits (TDD: capture-at-mint as literal #1; CI/CD:
integrate-on-tip as #1; Game: timed-trace-first) are scheduling
sub-disputes INSIDE the agreed wave — per the Round-7 bar they do not
block convergence. The chair folds capture-at-mint into the #10 wave
(Systems pairs seam + record), keeps integrate-on-tip at #12 because the
pipe already returns honest Unintegrated (CI/CD blocks pricing, not the
wire), and sequences the timed trace to #14 (the wire ships a live
first-watch on the current trace; tempo TUNING needs the time axis). UX's
two-quiet-meanings, TDD's audit-the-WHY, Refactoring's lost-via-rename are
additive sharpenings folded into #10/#11.

## Decisions

1. NEXT BUILD (#10, the converged wire): a runnable cmd/agntpr (or
   internal/app) that boots a real Via app over a live HTTP/SSE server
   (the first prod ListenAndServe/Flusher), mounts surface.ReviewCard, and
   on a real settle drives orchestrator→RunCatchCycle, feeding the result
   into the card over SSE — a human opens a browser and WATCHES one verdict
   go in-flight → resolve to the real Outcome of two real settles. NO new
   render logic, NO new oracle: the SSE server + the pipe→card EDGE. Built
   via tdd-rygba. Acceptance: a real Outcome from a real two-settle pipe
   reaches a rendered SSE frame, no flash/no-freeze on
   in-flight→resolved.
2. #10 PREREQUISITE sub-brick A, BUILD FIRST: the pipe→card PRESENTER — a
   pure mapping from pipe state → card verdict token that carries WHY a
   card is quiet (Tested vs in-flight vs NoOracleSignal vs the
   catch.Outcome). CycleResult.Outcome ALONE cannot express the surface's
   non-Outcome states (surface.Tested is a separate const; in-flight is
   the empty verdict). Map state, do not forward the bare enum. Test the
   pure function before the server.
3. #10 PREREQUISITE sub-brick B, BUILD FIRST (capture-at-mint, deferred 4
   rounds, chair-pulled into this wave, data-only): a typed
   CatchRecord{Outcome, Anchor, BeforeRev, AfterRev, BeforeInventory,
   AfterInventory, MutantsConsidered, ReasonTag, SelfFlagged bool,
   WouldHaveShipped bool} appended to an event log on every real mint — NO
   weight, NO pricing (guards forbidden catch-weight V§13.5). The live
   mint returns a string and persists NOTHING; un-backfillable data must
   be captured on the first real served mint. ReasonTag carries the WHY so
   the Ledger is auditable for theater. Build the no-op-churn-writes-
   NOTHING assertion (on the LOG) first — the farm-denial claim still
   untested in the wild.
4. ADVERSARIAL ACCEPTANCE for #10, all end-to-end through the wired binary
   against the live SSE stream: (a) strengthen-test → in-flight→catch
   "Caught" as ONE discrete transition; (b) edit-anchored-line → quiet,
   NEVER "Caught" over the wire; (c) no-op-churn → NoCatch + NO CatchRecord
   appended (assert on the log); (d) zero-survivor → "tested" DISTINCT
   from operator-free "no-oracle-signal"; (e) CatchRecord byte-identical on
   replay; (f) 40-file rename → renamed anchors render a TERMINAL lost
   state (expected-RED today: card has none — locked as a visible
   baseline); (g) meters-off lock holds end-to-end.
5. RANKED ROADMAP to a USABLE PROTOTYPE (each w/ its experiment): [#10 +
   prereqs 2,3] the end-to-end wire + presenter + CatchRecord [watch a
   real catch resolve over SSE; farm-denial on the log; calm-win vs blind
   discrimination]; [#11] fifth TERMINAL surface state orphaned/lost-via-
   rename [40-file rename → terminal lost, not in-flight-forever]; [#12]
   integrate-on-tip {clean|conflict|checks-red} replacing the Unintegrated
   const [trunk-moved (A) — catch survives the rebase or pricing is
   hard-gated; clean-rebase-checks-red (B), the load-bearing RED]; [#13]
   single-lane merge queue wrapping #12 [throughput-to-zero on K branches;
   first N-ceiling mechanic, Clash D]; [#14] timed/typed Trace
   ([]TraceEvent{T,Kind,Msg}) persisted as a replay artifact [50-event
   burst coalesces to one legible beat at honest tempo — Game's tempo
   demand]; [#15] K-concurrent-settle benchmark on the INTEGRATED cost
   [third-run + per-tip re-runs; MUST precede pricing]; [#16] first
   economy STOCK rendered + two-agent Board instrumented for idle/dwell
   [rework-vs-concurrency, Clash D — meters come ON here, never before];
   [#17] catch PRICING against an integrated base [gated on #12(A) + #15;
   redeemed only against the logged CatchRecord's objective columns, never
   a model-inferred catch-weight V§13.5; unblocks H/E/I].
6. BLOCKERS before the prototype is reachable: (a) NO prod SSE server
   exists — the prototype is not watchable until #10's seam stands one up;
   the single gate between 11 green packages and "usable"; (b) the
   pipe→card edge does not exist — CycleResult.Outcome cannot express
   Tested/in-flight, so without the presenter the calm-win screen
   mis-renders as blind silence; (c) capture-at-mint is un-backfillable
   and the live mint destroys it on every run — must land this wave or the
   Ledger is born without its only objective columns; (d) the surface has
   NO terminal state for a lost-via-rename anchor — a renamed card lies
   in-flight forever (cleared at #11); (e) land must NEVER be fake-merged
   — stays Unintegrated until integrate-on-tip (#12), and catch PRICING is
   gated behind #12(A) + the #15 benchmark; (f) the trace carries no time
   axis — cadence cannot be tuned until #14, though the wire can ship a
   live first-watch without it.
7. RISKS.md additions/standing: log the persistence-layer audit gap
   (CatchRecord must carry MutantsConsidered + ReasonTag or
   NoCatch-tautological is indistinguishable from NoCatch-already-strong);
   log the integrated-cost multiplier (oracle×2 + integrated run + per-tip
   queue re-runs); the set-not-multiset under-crediting gap and the four
   R5-7 code-level risks stand; run the K-concurrent-settle benchmark on
   the integrated cost path before any catch pricing. NO VISION/DESIGN text
   changed (12-contradiction reconciliation pass queued per RISKS step 5).

CONVERGED (6th consecutive round): the R9 wave SHIPPED green — 6/6 ratify
the result (a real Catch minted from two real settles + a card rendering
all four outcomes + Tested + in-flight, land honestly Unintegrated) — and
the council converges on the END-TO-END WIRE as next build: an
SSE-served, runnable surface feeding a real RunCatchCycle into the card,
so the first real catch becomes a beat a human WITNESSES instead of a row
in a Go test. Four lenses (UX, Game, Systems, Refactoring) advocate the
wire as #1 outright; the chair folds capture-at-mint (TDD's #1, paired by
Systems) into the SAME wave as un-backfillable and pulls the presenter
sub-brick first, and keeps integrate-on-tip at #12 (CI/CD's #1) because
the pipe already returns honest Unintegrated and CI/CD blocks pricing, not
the wire. No new target-level clash. The roadmap REACHES a usable Via/SSE
prototype, a merge queue, and a priced catch against an integrated base.
Next event is a BUILD — the presenter + CatchRecord + the SSE-served wire
(#10) — not another round.
