# Round 9 — the #1 wave SHIPPED: from a green economy unit to its first real mint, and the ranked march to a usable prototype — 2026-06-04

Trigger: the Round-8 marching orders BUILT — for the first time the
council reconvenes on post-build evidence, not a byte-identical tree.
Charge: confirm/refine the next build, produce a ranked roadmap from #1
through a USABLE prototype, flag prototype blockers. Build state: 9 green
packages (`env -u GOROOT go test ./internal/...` → 9 ok); the R8 wave
(`internal/catch`, `internal/reanchor`, `internal/refactor`) landed
across 3 real commits.

Panelists: all six. No new lens.

New evidence (verified in code):

- `internal/catch` is a real Go type — `Detect(before, after LineState)
  Outcome`, `Outcome ∈ {Catch | NoCatch | NoOracleSignal | PartialCatch}`
  (catch.go:50-70). Identity key enforced as specced: `len(beforeInv)==0
  → NoOracleSignal` (52); `!setsEqual(beforeInv, afterInv) → NoCatch` (55,
  inventory-change rule); before-survivors empty → NoCatch;
  after-survivors empty → Catch; strict-subset → PartialCatch. `LineState`
  SET-keyed, dedup'd per line (catch.go:35-43, deliberate v1
  simplification).
- Denominator owned by ONE pure function; refusal arms TESTED:
  `TestDetect_refusesCatchWhenFixChangesOperatorInventory`,
  `_refusesCatchForNoOpChurn`, `_refusesCatchWhenNewSurvivorAppears`,
  `_refusesCatchWhenInventoryShrinks`, + real-oracle
  `TestDetect_mintsCatchAcrossRealOracleRevisionsWhenTestStrengthened`
  (catch_test.go:178). The catch CONSTRAINS.
- `internal/reanchor` ships LostViaRename as DISTINCT (reanchor.go:35,75)
  alongside Same/Moved/Outdated; the refactor trace
  (`internal/refactor/trace_test.go`) locks carnage as expected-PASS:
  40-file rename orphans all 40 threads, neutral rename → LostViaRename
  != Catch, extract-module re-mutated as net-new.
- BUT `catch.Detect` has ZERO prod callers — `grep catch. internal/`
  (non-test) returns only a COMMENT in reanchor.go. The mint is struck
  and never opened: `Detect` is dead code, nothing mints a Catch.
- Still ZERO integration primitives (`grep rebase|merge-base|integrate|
  onto` non-test → 0); orchestrator.go:37,49 diffs an immutable baseRev.
  Still ZERO surface (`grep ListenAndServe|event-stream|Flusher` → 0).
  ZERO economy callers beyond catch itself.

## Per panelist

- UX: 3 commits, #1 brick green and type-committed. Headline acute: 8
  rounds, zero pixels — catch.go emits an Outcome no surface renders,
  translate emits into the void, orchestrator emits TurnOutcome w/ nothing
  to play it. Build #3 pipe + #4 Via/SSE card as ONE wave: four outcome
  states + a designed coalesced in-flight beat, meters OFF first screen,
  the MOST-COMMON 'N considered, 0 survived — tested' screen designed as
  a calm WIN. NEW: the enum collapses two opposite 'quiet' meanings —
  NoOracleSignal (blind) vs NoCatch-already-constrained (verified-strong)
  — that a naive surface renders identically; the pipe must carry WHY it
  is quiet, not just the Outcome string.
- Game design: a tycoon game w/ no screen is a spreadsheet. Nobody has
  FELT a catch. FLIP confirmed: pipe-first (#3, pure wiring, zero backend
  — orchestrator emits TurnOutcome, translate emits activity.agent +
  turn.ended), capture the replay, THEN the surface. NEW: the pipe emits
  a raw firehose w/ NO designed TEMPO — un-coalesced it reads as log-spew
  or a frozen card and the catch beat DROWNS; #3 must PERSIST the trace so
  #4 has an honest replay to tune cadence against.
- Systems: `Catch` is a real type, identity-key enforced. But the economy
  object is an ISLAND: zero callers, `Detect` is dead code. Every
  property I red-team is proven ONLY against hand-built LineState
  literals, NEVER against two real revisions through settle→diff→mutation
  →Detect. Next build is the smallest driver minting ONE real Catch from
  settle A→settle B, validated by the farm fixture FOR REAL
  (agent-authors-killing-test → Catch; agent-edits-line → NoCatch).
  Acceptance must be 'a real Catch minted from two real settles', not
  'catch package green'. Also: LineStateAt reads inventory from current
  src but survivors from a passed-in res — these can desync across
  re-anchoring, and reanchor is not wired into LineStateAt yet.
- Pragmatic TDD: deliberation debt paid — the catch constrains. Now build
  the §17 pipe so the catch runs at settle on a real two-revision
  sequence: nothing has run before→after end-to-end through the
  orchestrator. Killer RED: the SAME pipe on a turn editing the anchored
  line (>=→>) must yield NoCatch, not a phantom catch — proving the
  inventory-change rule survives the REAL reanchor path. NEW logged: the
  set-not-multiset keying (catch.go:39) under-credits a line where a fix
  kills one of two same-operator survivors (reports NoCatch despite real
  progress) — RISKS.md before any pricing.
- CI/CD & Delivery: the catch is real, typed, green. But my seam is WORSE:
  the catch is now minted on PRE-INTEGRATION coordinates IN SHIPPED CODE —
  'Landed != Merged' is a property of catch.go + orchestrator.go
  (immutable baseRev, no rebase path). I do not block #3 — I block #3
  shipping a FAKE 'land'. Fold integrate-on-tip into #3's land as a
  tri-state {clean|conflict|checks-red}, or make 'land' return an explicit
  Unintegrated state. Settle the rebase question w/ trunk-moved (A) and
  clean-rebase-checks-red (B) BEFORE the catch is PRICED. NEW: the roadmap
  lists integration LATE — backwards under 1:N; it is the serialization
  point and must not be retrofitted after the Board trains a fake 'land'.
- Refactoring: wave-1 baselines green — rename is LostViaRename, the
  inventory rule kills the phantom catch. But we built TWO correct bricks
  and NO edge between them: `grep catch.Detect` (non-test) empty, Detect
  takes a `line` int the CALLER must have re-anchored, Detect does NOT
  call reanchor. The ratified safety property — 'lost via rename →
  NoOracleSignal, NEVER a phantom Catch' — is asserted in two SEPARATE
  test packages and enforced by NO single function. Build the one-function
  JOIN CatchAcross(repoDir, anchor, beforeRev, afterRev) — reanchor first,
  NoOracleSignal-by-construction on LostViaRename/Outdated, Detect only on
  Same/Moved — BEFORE the pipe.

Clashes touched: F (the unit FLIPS toward RESOLVED-IN-CODE — tri-state +
identity-key shipped green w/ tested refusal arms; residual: FARM half
asserted only against literals, Detect has never seen two real revisions);
G (refactor trace now EXECUTABLE evidence — moves to RESOLVED for the
carnage-baseline question, exposes the unjoined-packages gap); C
(unchanged verdict but SHARPENED — catch minted on pre-integration coords
in SHIPPED code; ordering stands as 'precede catch PRICING not minting',
fixtures A+B); A (silent-vs-badge + NoOracleSignal-vs-Catch
discrimination becomes #4's acceptance bar); B (capture-at-mint from R8
NOT in the Catch type — no record persisted, un-backfillable data lost on
every mint, of which there are zero); D/E/H/I (render-camp — roadmap
REACHES them at #4/#8).

## Verdicts updated

- Clash F: → RESOLVED-IN-CODE on the UNIT (tri-state + identity-key +
  tested refusal arms, all green), RE-SCOPED OPEN on the LOOP — 'oracle
  runs every settle' and 'farm denied in the wild' asserted only against
  LineState literals; Detect has never adjudicated two real revisions.
  Flips fully on the green #3 pipe (real Catch from two real settles +
  agent-edits-line → NoCatch end-to-end).
- Clash G: → RESOLVED on the carnage-baseline question (refactor trace
  executable). Residual reclassified into the new composition gap, NOT a
  fresh clash: asserted safety lives in two unjoined packages, closed by
  the reanchor→catch JOIN.
- Clash C: unchanged verdict (catch minted pre-integration), now a
  PROPERTY OF SHIPPED CODE rather than a warning; both settling fixtures
  (trunk-moved A, clean-rebase-checks-red B) carried to #5; integrate-on
  -tip folded into #3's land as a tri-state or explicit Unintegrated.
- Clashes A, D, E, H, I: remain TBD; roadmap terminates at #4 (surface) +
  #8 (two-agent Board) that unblock them, w/ named experiments.

New clashes opened: NONE at target level — #1 shipped green, council
converges 6/6 on the §17 pipe as next build. The build-ORDER split (UX:
pipe+surface one wave; field: pipe then surface) and SCOPING refinements
(Systems: real-mint not generic pipe; Refactoring: reanchor→catch JOIN as
prereq; CI/CD: no fake land) are scheduling/scope sub-disputes INSIDE the
agreed brick — per the Round-7 bar they do not block convergence.

## Decisions

1. NEXT BUILD (#3, 6/6 converge on the §17 pipe): the pipe AS THE CATCH'S
   FIRST REAL MINT — a minimal single-user driver settle A → settle B →
   diff → mutation.Run×2 → re-anchor B→A → CatchAcross → emit ONE typed
   Catch + resolve/keep the question: thread, on ONE real temp git repo.
   Acceptance bar (chair-set, Systems): 'a real Catch minted from two real
   settles', NOT 'catch package green'. Built via tdd-rygba.
2. #3 PREREQUISITE sub-brick, BUILD FIRST (Refactoring + Systems,
   chair-adopted): the reanchor→catch JOIN CatchAcross(repoDir, anchor,
   beforeRev, afterRev) — reanchor first; LostViaRename/Outdated →
   NoOracleSignal BY CONSTRUCTION (never reaches Detect); Detect only on
   Same/Moved against the RE-ANCHORED after-LineState. Load-bearing RED:
   rename_neutral_move yields a phantom Catch via a naive caller, GREEN
   when CatchAcross short-circuits to NoOracleSignal. Fuses the two
   green-but-unjoined bricks into one TYPED guarantee, the single entry
   point #3 and #4 require.
3. CI/CD BINDING into #3 (chair-adopted, non-negotiable): #3's land step
   returns a tri-state {clean|conflict|checks-red} on the rebased tree, OR
   an explicit Unintegrated placeholder — NEVER a fake 'merged'. 'Landed
   != Merged' is a property of shipped code; #3 must not bake it deeper.
4. CAPTURE-AT-MINT, WITH #3 (Systems' Clash-B blocker, data-only, no
   weight/pricing): persist the Catch record w/ a self-flag bit +
   would-have-shipped proxy. There is currently NO Catch record persisted;
   the un-backfillable data must be captured on the first real mint.
5. RANKED ROADMAP to a USABLE PROTOTYPE (each w/ its experiment):
   [#3-prereq] reanchor→catch JOIN [neutral-rename phantom RED →
   NoOracleSignal]; [#3] §17 pipe / first real mint [two-revision farm:
   agent-strengthens-test → Catch; agent-edits-line → NoCatch end-to-end;
   catch as one beat in a persisted replayable trace; land returns
   tri-state/Unintegrated]; [#4] Via/SSE single-card surface, FOUR outcome
   states + designed in-flight/coalesced state, meters OFF first screen
   [zero-survivor reads 'tested'; NoOracleSignal visibly DISTINCT from
   Catch; 50-event burst coalesces no-flash-no-freeze; one-comment→one-
   revision loop]; [#5] integrate-on-tip {clean|conflict|checks-red}
   [trunk-moved A; clean-rebase-checks-red B] — if (A) flips the outcome,
   #5 precedes catch PRICING; [#6] single-lane merge queue wrapping #5
   [throughput-to-zero, batch+bisect]; [#7] capture-at-mint persisted
   [self-flag + would-have-shipped, no weight]; [#8] first economy stock
   rendered + two-agent Board instrumented for idle/dwell [Clash D
   N-ceiling via rework-vs-concurrency]; [#9] catch PRICING [gated on
   #5(A) + K-concurrent-settle benchmark, against an integrated base].
6. BLOCKERS before the prototype is reachable: (a) the reanchor→catch
   JOIN is a HARD prereq of #3 — until CatchAcross exists the ratified
   safety property is enforced by NO prod function and a pipe author can
   mint a phantom catch; (b) catch.Detect has ZERO callers — the economy
   is an island until #3 opens the mint; the degenerate/farm suite is
   unproven against real revisions; (c) the single-user happy loop (#3)
   MUST exist and persist its trace before any meter renders —
   Focus/Trust/Treasury stay OFF the first screen; (d) 'land' must never
   be a fake merged — fold integrate-on-tip or return Unintegrated; (e)
   catch PRICING gated on #5 if the trunk-moved fixture shows the
   transition is rebase-dependent.
7. RISKS.md additions/standing: log the set-not-multiset under-crediting
   gap (catch.go:39 — a fix killing one of two same-operator survivors
   reports NoCatch); log LineStateAt inventory/survivor desync across
   re-anchoring; the four R5-7 code-level risks stand; run the
   K-concurrent-settle benchmark before any catch pricing. NO
   VISION/DESIGN text changed (12-contradiction reconciliation pass queued
   per RISKS step 5).

CONVERGED (5th consecutive round): the #1 wave SHIPPED green — 6/6 ratify
the result — and 6/6 advocate the §17 pipe as next build, refined to the
catch's FIRST REAL MINT on a reanchor→catch JOIN prerequisite, w/ land
returning tri-state/Unintegrated (never fake merged) and capture-at-mint
alongside. The roadmap REACHES a usable Via/SSE surface (four outcome
states + designed in-flight) and a two-agent Board. Sole dissent is
build-ORDER/scoping inside the agreed brick (UX: pipe+surface one wave;
chair keeps #3 then #4 so the surface tunes against a real persisted
trace and a real mint). No new target-level clash. Next event is a BUILD —
the reanchor→catch JOIN, then the §17 pipe minting one real Catch from two
real settles — not another round.
