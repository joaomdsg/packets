# Round 64 — review answer flow: ALL BELLS AND WHISTLES (maintainer steer) — 2026-06-10

Trigger: after the editable-answering thread shipped + the A1 in-flight indicator,
the maintainer directed: "continue with the review user flows polish with ALL BELLS
AND WHISTLES." So: the full polished review-answer experience, ambitious.

Full six convened, grounded in the via mechanics (StateTab, OnConnect/Stream,
data-ignore-morph, datastar bindings, the rerunWithOverlay seam, the findings cache).

## Convergence

- UX: editable MONACO answer pane (Go syntax, matching the read-only pane), pre-seeded
  test stub, synced to the answertest signal + morph-safe; keyboard-native (Cmd/Ctrl+
  Enter submit, focus-on-open, Esc); calm live transitions (running → vanish-on-kill /
  "still open, try again" on weak); stacked layout (read-only source above, editable
  answer below).
- Game: the vanishing question IS the reward (no XP/confetti/streaks/progress-bars —
  explicit dark-pattern traps to AVOID). Honest flourishes worth shipping: surface the
  MUTATION (>= → >) so the reviewer knows what to constrain; a quiet count tick-down;
  auto-scroll to the next question after a kill; calm "no open questions" walk-away.
  Not-killed = teach, never scold ("your test didn't break the >= → > boundary").
- Refactoring: clean async = action records pendingAnswer on liveEntry (under
  findingsMu) + an OnConnect/Stream drains it, race-safe (capture locals under lock
  before `go`, per R54); editable Monaco synced one-way (onChange → dispatch input on
  the bound textarea) inside data-ignore-morph; reuse ONE findings cache + ONE
  rerunWithOverlay seam.
- Systems/CICD: async FIREWALL stays (the background verdict calls ONLY setFindings,
  never ledger); one-re-run-at-a-time per session (atomic in-flight guard) so re-submits
  can't queue 50 oracle runs; reuse the cycle semaphore for compute; the flaky-fence
  holds (error/Undetermined → don't clear, retryable).
- TDD: server contracts (cache, the kill/weak verdict) are vt-testable with a stubbed
  rerunWithOverlay; the editable Monaco + keyboard + datastar transitions are CLIENT
  islands → browser-verify, NOT vt (asserting them via vt would be test-theater).

## Key testability finding (shapes the sequencing)

vt's Fire() DISCARDS the action response body and Reload() starts a FRESH tab — so a
post-action StateTab "note" is NOT cleanly vt-assertable unless /review becomes a
streaming surface (SSE + AwaitFrame). BUT the synchronous @post response already
patches the browser (verified: the empty state appears after a killing submit). So the
USER-FACING polish needs no async-streaming refactor; only invisible "free the request"
would, and its value is marginal for a single-user prototype while its race risk is
real. DECISION: deliver the user-facing bells & whistles on the synchronous flow
(browser-verified on :3000); DEFER the async-streaming refactor (invisible, marginal,
risky). "All bells and whistles" = the rich user experience, which this delivers.

## Slice plan (build over next ticks; browser-verify each on :3000)

- B1 — EDITABLE MONACO answer pane: replace the textarea with an editable Monaco (Go,
  vs-dark) inside a data-ignore-morph wrapper, synced one-way to the hidden bound
  textarea (onChange → dispatch input → datastar updates answertest). The textarea
  stays the bound source + no-JS fallback. Server test asserts the editable mount +
  bound textarea; the editor behavior is browser-verified.
- B2 — KEYBOARD: Cmd/Ctrl+Enter submits from the editor; focus the answer editor on
  load; Esc blurs. Client island, browser-verified.
- B3 — TEACH-ON-NOT-KILLED + MUTATION SURFACING: a calm "still open — your test didn't
  break <mutation>; try a tighter assertion" after a weak answer (in the @post response
  render; browser-verified), and surface the mutation operator in the question. Honest
  teach, no scold.
- B4 — CALM TRANSITIONS + COUNT + auto-scroll-to-next + empty-state polish.

Guardrails: diagnostic-only/off-economy (firewall), calm/no-fake-reward (no XP/
confetti/progress-bar/streak — Game's explicit avoid-list), reachability-grounded,
server-tested where the contract allows + browser-verified for client islands. The #6
boundary stays gated. Async-streaming refactor DEFERRED (documented, low-value/high-
risk for single-user).

## B1 DONE — editable Monaco answer pane (the maplibre bridge)

The first editable-Monaco attempt FAILED on a stubborn datastar sync: Monaco synced
to a hidden data-bind textarea, but the answertest signal never reached the action.
The maintainer pointed at the via maplibre plugin for inspiration — the canonical
client-JS-island↔datastar pattern. Its bridge (events.go): the client lib fires a DOM
CustomEvent, and a `data-on:<event>` handler ASSIGNS signals from evt.detail INLINE,
then `@post`s — never data-bind. Re-applied to the answer pane:
- The editor + submit sit in ONE data-ignore-morph wrapper with
  `data-on:viaanswer="$answerfile=evt.detail.file;$answerline=evt.detail.line;
  $answertest=evt.detail.test;@post('/_action/AnswerQuestion')"`.
- answerEditorJS mounts an editable Monaco (Go, vs-dark); a submit-button click OR
  ⌘/Ctrl+Enter dispatches the viaanswer CustomEvent carrying the editor's value.
- Dropped the bound textarea entirely.
VERIFIED end-to-end on :3000 + by debug instrumentation: the POST body carries the
full test (answertest=125 chars), the action receives it, and the re-run returns
findings=0 / err=nil — the mutant is KILLED. The editable Monaco + ⌘/Ctrl+Enter +
running indicator all work. Tests updated (review_answer_internal_test.go) to the
data-on bridge; full -race gate green.

KNOWN FOLLOW-UP (connect-cycle vs answer): the live "/" card's connect cycle also
writes the findings cache (setFindings); if it completes AFTER an answer, it
re-populates the survivor, visually undoing the answer. In normal use the cycle
settles before the reviewer opens /review + answers, so it sticks; rapid overlap (or
a re-connecting "/" tab) can clobber. Coordination (answer-wins / pause cycling
during an answer, or the council's per-session in-flight guard extended to cover the
cycle) is the next slice. The answer MECHANISM is proven correct regardless.

## New clashes opened / resolved

None — a clean convergence on the full polished flow + a testability-driven sequencing.
The editable-Monaco datastar-sync blocker is RESOLVED via the maplibre CustomEvent +
data-on bridge (data-bind was the wrong tool). Async-streaming refactor still deferred.
</content>
