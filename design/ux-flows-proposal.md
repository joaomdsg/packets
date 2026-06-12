# Packets — UX Flows & Information Architecture Proposal

> Status: proposal for debate. No source touched. Every recommendation is
> tied to a real handler, View, render helper, or class hook in the repo as it
> stands today.
>
> Author lens: interaction rigor of Linear/Stripe/Figma, held against
> Packets' sacred ethos — calm, honest, no fabricated urgency, "one row never
> speaks for another," accessibility non-negotiable, progressive enhancement.

## 0. How to read this

The surfaces audited, with their authoritative files:

- The session card at `/` — `internal/app/live.go` (`LiveCard.View`,
  `OnConnect`, `Spend`, `FundChosen`, `PlaceOrder`), `internal/app/authoring.go`
  (`renderAuthoring`, `composeSurface`, `AnalyzeDraft`),
  `internal/app/onboarding.go`, `internal/app/prep_bench.go`,
  `internal/app/review_badge.go`, `internal/app/supply.go`.
- The economy rows — `internal/surface/{stock,balance,bandwidth,dispatch,beats,card,land}.go`.
- The fleet board at `/board` — `internal/app/board.go`.
- The review surface at `/review` — `internal/app/review_surface.go`.
- Settings at `/settings` — `internal/app/settings_card.go`.
- The shared nav — `internal/app/nav.go`.

A guiding constraint I adopted from VISION §13.1 and the in-code comments:
the two economies are deliberately separated **in time and meaning**, not
just on screen. Balance (catches) funds backlog Spends; Bandwidth (attention)
funds authored live orders. The audit and IA below treat keeping those two
legible-yet-unconfused as a first-order goal, because the current card stacks
them as two near-identical rows with near-identical microcopy.

---

## 1. AUDIT — where a real Lead gets lost

### 1.1 First-run / onboarding

What exists: `onboardingHint` (`onboarding.go`) renders a calm three-line
`<section data-state="empty">` ahead of the all-zero economy rows, gated on
`stock.Count == 0`. It is honest and well-reasoned, but as the *entire*
first-run experience it has real gaps:

- It explains the *catch→balance→spend→reinvest* loop in prose, but on a
  fresh load the card is simultaneously running the catch cycle in
  `OnConnect`. The Lead sees onboarding prose, an empty beat row, an
  "Oracle running…" verdict (`present()` default), and three zero rows — four
  competing things, none of which point at a single next action.
- The hint never mentions the *other* half of the app: bandwidth, authoring
  live orders, the board, or settings. A first-run Lead has no idea `/board`,
  `/review`, or `/settings` exist except via the tiny nav crumb.
- There is no acknowledgement that the very first useful action (running a
  live order) requires an API key. The key warning only appears *inside*
  `composeSurface` (`compose__needs-key`), which is gated behind
  `bandwidth > 0` — and a brand-new session has zero bandwidth, so the Lead
  literally cannot see the key prompt on the surface where they'd act on it.
  The onboarding-to-first-live-order path is broken at step zero.
- Disappearing-onboarding cliff: the moment `stock.Count` ticks to 1, the
  hint vanishes forever. There is no "you've done one loop, here's what's
  next" scaffolding — the Lead is dropped from guided to bare.

Verdict: onboarding explains the *balance* economy and nothing else, and
omits the prerequisite (key) for the *bandwidth* economy it never mentions.

### 1.2 Settings / key flow

`settings_card.go` is clean and honest: a `data-state` status line plus a
masked input and "Save key". Save is a silent no-op on empty (good — won't
clobber). Problems are about *flow*, not the widget:

- Discoverability: the only route to `/settings` is the nav crumb (`nav.go`)
  or the `compose__needs-key` link — which, as noted, is unreachable on a
  fresh session. A Lead with no key and no bandwidth has no signposted path
  to settings from the place they need it.
- No validation or feedback beyond "configured". A pasted-wrong key (typo,
  trailing space already trimmed, revoked key) reads "configured — live
  orders can run", then the *next* live order fails silently inside
  `runLiveOrder` (`AppendStatus(id,"failed")`) with no surfaced reason. The
  Lead sees an order go to "failed" on the card with zero explanation. This
  is the single worst feedback-latency gap in the app: the error is hours of
  cognitive distance from its cause.
- No "test this key" affordance, no masked-preview of the stored key
  (last-4), no indication of *where* the key is stored or that it persists
  across restarts (it does — `loadStoredTokenIntoEnv`).
- Returning to settings after save: the input still shows a placeholder, the
  status says configured. Fine, but there's no "replace key" vs "key set"
  distinction — the same control reads identically whether you're arming for
  the first time or rotating.

### 1.3 The author → analyze → place → watch → review loop

This is the spine of the product and it has the most going on.

Author (`composeSurface`): a Monaco editor, "Analyze draft", "Place order",
an "analyzing…" indicator, a readiness reflection, an inline-decoration
payload, a needs-key note. Strong bones. Issues:

- Button hierarchy is flat: "Analyze draft" and "Place order" are visually
  peer buttons (`compose__analyze`, `compose__place`) with no primary/
  secondary distinction. Place is the consequential, irreversible-ish action
  (it spends bandwidth and dispatches a real agent); Analyze is the cheap,
  repeatable one. The hierarchy is inverted from their weight.
- The whole authoring block is gated on `bandwidth > 0` in `View`. So the
  composer — arguably the headline feature — is invisible until the Lead has
  earned bandwidth by clearing review questions. A new Lead cannot even *see*
  how to author an order. That's a severe discoverability and "what is this
  app" failure. (Contrast: the Spend button is gated on `balance > 0`, which
  is at least reachable from the connect-cycle catch.)
- Placing an order gives no confirmation and no optimistic feedback in
  `PlaceOrder`. It writes the bandwidth/dispatch cells and returns; the Lead's
  draft text stays in the editor (it's never cleared), there's no toast, no
  "order WO#N placed", no scroll-to or focus-move to the filling row. The
  only signal is the dispatch counts moving and (eventually) an
  `order-filling` div appearing further down the card. The act of placing —
  the emotional peak of authoring — is silent.
- Analyze feedback latency: `AnalyzeDraft` shells `claude` synchronously and
  can take many seconds. The only in-flight signal is the client-side
  `compose__analyzing` indicator toggled by the editor's debounce JS — which
  is decoupled from the *actual* server round-trip. If the producer run is
  slow, the indicator can read "idle" while the server is still working, or
  the reverse. There is no server-driven analyzing state.
- Failure copy is decent ("the producer run failed — try again") but it
  replaces the whole analysis panel; a Lead who had a good analysis and then
  edits into a transient failure loses their prior read entirely.

Watch (`order-filling`, `order-activity`, `order-transcript` in `View` +
`OnConnect` Stream poll): genuinely nice — a live latest-move line plus a
scrolling transcript plus cycle beats. This is the best-realized "calm but
alive" surface in the app. Remaining gaps:

- These live elements render *in the middle of the card* (after dispatch row,
  before the recent-dispatch list). With everything else on the card also
  present, the Lead doesn't know to look there after placing. No focus
  management routes attention to the filling region.
- `aria-live="polite"` is on the whole `main` (good), but the transcript
  appends many lines rapidly — a polite region re-announcing the entire
  economy on every 100ms FillBeats tick is a screen-reader firehose. The live
  region is too coarse; the transcript should be its own bounded live region
  (or `aria-live="off"` with a summarizing status line that updates rarely).

Review (`review_surface.go`): the dispatch→review tie is well built — a
filled order's `Questions` count drills to `/review?wo=N` with a Monaco diff
of base→fix and the surviving-mutant threads, with an editable answer pane.
Gaps:

- Two different "review" experiences live on one route with different
  capabilities, and the difference is invisible until you're there:
  `/review?key=` (session findings, editable answer flow) vs `/review?wo=N`
  (order findings, *also* editable now per the `renderAnswerForm(.,woID)`
  path, plus a diff). A Lead arriving from a board link doesn't know which
  they'll get.
- The answer flow is powerful but unexplained: "write a test that kills the
  mutant" assumes the Lead knows Go test conventions and the package context.
  No starter scaffold, no example, no indication of which package/file the
  test compiles into (`answerTestFilename` is fixed but invisible).
- The reward for answering is *the question vanishing* (by design, off-
  economy). But the bandwidth award from `recordQuestionUnblock` happens on
  the *session* path only, and it's silent — the Lead's bandwidth meter rises
  on the `/` card they're not currently looking at. The single most important
  feedback in the bandwidth economy (you cleared a question → you earned
  attention → you can now author) is delivered on a different surface than
  the one the Lead is on. The loop's payoff is invisible at the moment it's
  earned.

### 1.4 The fleet board

`board.go` is a dense, honest projection. It's the most information-rich
surface and the least designed. Issues:

- It's a wall of `span`s per row: stock, bets cluster, balance, activity,
  misses, hit-rate, backlog, optional questions, optional land, optional
  dispatches, optional retire. ~10 data points per row with near-zero visual
  hierarchy — every span is the same weight. This is the opposite of the
  "skim the header, inspect the payload" packet metaphor.
- Ordering is by `Queued` desc, tie-broken by registration `seq`. The code
  is scrupulous that this is *activity, not priority* (no leverage rank yet).
  But the Lead has no way to know the sort axis — there's no column header,
  no "sorted by queued work" label. A sort with no legend reads as arbitrary.
- It is NOT live (`/board` is a request-scoped GET; correctly not
  `aria-live`). But a "fleet board" that doesn't update while you watch
  contradicts the VISION's "living board." The `/fleet` SSE endpoint exists
  (`bridge.FleetHandler`) but the human-facing `BoardCard` doesn't consume
  it. So the board is a snapshot the Lead must manually refresh — a major
  expectation mismatch.
- Create/retire are real but bare: a text input + "Create session" with no
  validation feedback (an invalid token is a silent no-op in `CreateSession`),
  and "retire" with no confirm and no undo.
- No empty state: a fresh server with only the default session shows one
  cryptic row of zeros and a create box. No "your fleet" framing.

### 1.5 Navigation / IA

`nav.go`: home ("packets" → `/board`), then `fleet · settings` crumbs, then
on a card the raw session key. Honest and minimal. Issues:

- "packets" home points to `/board`, but the conceptual home / where work
  happens is `/` (the card). A Lead clicking the wordmark expecting "home"
  lands on the fleet roll-up, not their session. The mental model is muddled:
  is `/board` home or is `/`?
- The crumb is not a real breadcrumb (no hierarchy/position). `fleet ·
  settings › <key>` mixes peer links (fleet, settings) with a trailing
  position (key) using two different separators. `/review` is *not in the nav
  at all* — the only way to reach it is a badge/board link, and once there,
  there's no nav item showing you're "in review" or how to get back to the
  card you came from.
- No keyboard nav (the code says "a later slice"). For a product whose VISION
  is emphatically keyboard-native (`j/k`, `c`, `r`, `a`), the absence of any
  key affordance is the largest gap between vision and surface.
- No indication of *which economy* a surface serves. Nothing tells the Lead
  that `/review` is where bandwidth is earned and `/` is where it's spent.

### 1.6 Empty / loading / error / degraded states (cross-cutting)

- Empty: onboarding (`/`) and the two `/review` empties are handled. `/board`
  and `/settings` have none.
- Loading: the connect cycle has beats + an "Oracle running…" verdict (good).
  Analyze has a client-only indicator (fragile). Place has *nothing*. The
  answer re-run has a `data-show="$answering"` running line (good). Board
  create/retire have no loading state.
- Error: the worst category. Live-order failures, bad keys, harness-missing,
  over-budget spends, exhausted backlog, invalid session keys — **all are
  silent no-ops by design** ("never an error to the Lead"). The intent (calm,
  no alarms) is right, but the execution has crossed from "calm" into
  "opaque": the Lead clicks Spend with balance 0 and *nothing happens*, clicks
  Place with no key and the order silently fails later, types a bad session
  key and the create box just clears. Calm ≠ no feedback. A control-room is
  calm *because* every state is legible, not because failures are hidden.

### 1.7 Cognitive load & discoverability summary

The `/` card, fully populated, renders in order: nav, onboarding (maybe),
stock row, balance row, bandwidth row, spend button (maybe), authoring block
(maybe), bench (maybe), dispatch row, filling/activity/transcript (maybe),
recent dispatches (maybe), beats row, verdict row, questions badge (maybe),
land row. That's up to **15 stacked regions**, many with overlapping
vocabulary ("confirmed", "balance", "bandwidth", "dispatched", "caught",
"reinvested", "questions"). The "one row never speaks for another" principle
keeps them honest but also keeps them *flat* — there is no grouping that says
"these three are the economy ledger; these two are your action zone; this is
the live feed." The Lead must read every row every time.

---

## 2. INFORMATION ARCHITECTURE — the ideal structure

### 2.1 The four surfaces, re-rolled

Keep the four routes. Re-cast their *jobs* so each owns one verb and one
economy phase (VISION §13.1's "separate the economies in time"):

- `/` — **The Flow (act).** Your one session. Where you SPEND: spend a catch
  to fund backlog work, author+place a live order with bandwidth, and WATCH
  it fill. The live, breathing surface.
- `/review` — **Inspect (earn).** Where you EARN bandwidth: clear surviving-
  mutant questions. Frame it explicitly as the bandwidth faucet.
- `/board` — **The Fabric (orient).** Cross-session standup: where is work
  moving, what's blocked, what's mine to do next. Read-only orientation, made
  live off `/fleet`.
- `/settings` — **Setup (arm).** Key + (future) preferences. A prerequisite
  surface, reachable from everywhere it's needed.

### 2.2 The two economies, made legible without confusion

The core IA move: **stop rendering balance and bandwidth as two identical
sibling rows.** Today `RenderBalance` and `RenderBandwidth` are byte-for-byte
parallel (`"Balance: N"` / `"Attention bandwidth: N"`). They read as the same
thing twice. Instead, give each a *direction* and a *source/sink verb* in its
own microcopy and grouping:

- Balance (catches): a SPENDABLE stock. Source = the oracle confirming
  catches. Sink = funding backlog work-orders (Spend / bench). Group it
  visually with the Spend button and the bench — its sink is right there.
- Bandwidth (attention): an EARNED allowance. Source = clearing review
  questions (`/review`). Sink = authoring live orders (the composer). Group
  it with the authoring block — its sink is right there.

So the card's body splits into two labelled zones instead of a flat stack:

```text
LEDGER (look back, calm, never actionable)
  · confirmed-catch stock
  · this session's recent dispatches + outcomes
  · the live verdict / land / beats feed

ACT (forward, the only buttons on the page)
  Catches → Spend         [Balance: N]  → bench list + Spend button
  Attention → Author      [Attention: N] → composer (+ key note)
```

Each meter sits with the action it funds, so the Lead never has to remember
"which meter does what" — the answer is one line below the number. This is
the Stripe move: put the number next to the thing it pays for.

### 2.3 Wayfinding

- Make the wordmark "packets" go to `/` (your flow / home where work
  happens), not `/board`. Add an explicit "fleet" item for the board.
- Real breadcrumb with one separator and a clear trail:
  `packets / fleet / <session-key> / review` — where the current surface is
  the non-link tail. Put `/review` *in* the trail when you're on it (today
  it's orphaned).
- On every surface, a one-line "what this surface is for" subhead tied to its
  economy phase (e.g. on `/review`: "Clear a question to earn attention
  bandwidth."). This is the cheapest legibility win available and it directly
  fixes the "payoff is invisible / economies confused" problems.
- Add a persistent, tiny economy strip in the nav (read-only): `◆ N catches ·
  ◇ N attention` so both meters are always visible regardless of surface —
  the Lead never loses sight of their two resources when they navigate from
  `/review` (where attention rises) back to `/` (where it's spent). This is
  the single fix that makes the cross-surface bandwidth payoff legible.

---

## 3. KEY USER FLOWS (step-by-step, with states)

### Flow A — First run → configure key

Entry: Lead opens `http://localhost:3000/` (`/`).

Happy path (proposed):

1. Card loads. `OnConnect` starts the cycle. Onboarding section renders
   (today gated on `stock.Count==0` in `onboardingHint`).
2. NEW: onboarding becomes a 3-step checklist, not three prose lines:
   - "① Arm your key — live orders need an Anthropic key. [Set key →]"
     (links `/settings`; shown REGARDLESS of bandwidth, fixing the §1.1
     unreachable-key gap — this needs the key note moved OUT of the
     `bandwidth > 0`–gated `composeSurface`).
   - "② Watch this card confirm a catch — it's running now." (live, ties to
     the beats/verdict already streaming).
   - "③ Spend a catch to fund work, or clear a question to earn attention."
   Each step shows a done-check as its precondition is met (key configured;
   stock>0; first spend/answer). This replaces the onboarding cliff with a
   graduating checklist.
3. Lead clicks "Set key →" → `/settings`.

Settings states:

- Unconfigured (today: `settings__status` data-state). Add: input is
  auto-focused; helper microcopy "Stored locally beside the ledger; persists
  across restarts."
- Saving: NEW — a `data-show` "saving…" line driven by an in-flight signal
  (mirror the review `$answering` pattern) so the silent `SaveToken` gets a
  beat of feedback.
- Configured: status flips (already SSE via `Saved.Write`). Add a masked
  preview "key set · sk-ant-…last4" and a "Replace key" affordance distinct
  from first-arm. Add a "Run a test ping" button (calls a tiny harness no-op)
  so a *bad* key is caught HERE, not hours later in a failed order — this is
  the highest-value error-latency fix in the app.
- Error (disk/save fail): today silent. Surface a calm inline "couldn't save
  — try again", same idiom as analysis-unavailable.

Where it breaks today: the key prompt is unreachable on fresh sessions
(gated behind bandwidth); a bad key is invisible until a later order fails.

### Flow B — Author → analyze → place → watch → review

Entry: `/` with bandwidth > 0 (PROPOSED: composer always visible, disabled-
with-reason when bandwidth==0 or no key — see §4).

Happy path with states:

1. Compose: editor focused (`ed.focus()` already). NEW: a primary "Place
   order" button (filled) and a secondary "Analyze draft" (quiet). Today both
   are peer `<button type=button>` — invert weight.
2. Debounced auto-analyze fires (`onDidChangeModelContent` 900ms). State:
   `compose__analyzing` → pending → analyzing. NEW: drive the analyzing state
   from the SERVER round-trip (AnalyzeDraft writes an `analyzing` cell true
   on entry, false on resolve) so the indicator reflects reality, not just
   the client debounce.
3. Analysis lands: inline Monaco decorations (already works via the payload +
   MutationObserver), summary + clarifying questions panel
   (`renderAnalysisPanel`), readiness reflection beside Place
   (`compose__readiness`). Keep. NEW: when a prior good analysis exists and a
   re-run fails, KEEP the prior analysis visible and show the failure as a
   non-destructive inline note, not a wholesale panel replacement.
4. Place: `PlaceOrder`. NEW feedback (the missing peak):
   - optimistic: on click, clear the editor, move focus to the filling
     region, show "Order placed — WO#N, filling now" as a transient status in
     an `aria-live` slot.
   - the bandwidth meter drain (already broadcast via `BandwidthMeter.Write`)
     should animate down, paired with the new status.
5. Watch: `order-filling` + `order-activity` (latest move) + `order-transcript`
   (scrolling). Keep — this is the best part. NEW: make the transcript its
   own bounded `aria-live="polite"` region with a coalesced summary
   ("agent editing auth.go — 14 events"), and set the *outer* main to
   `aria-live="off"` during a fill so the whole economy doesn't re-announce
   on every 100ms tick (§1.3 firehose fix).
6. Resolve: order goes done; recent-dispatch row shows caught/missed +
   verdict + a "N open questions" drill. Keep.
7. Review its questions: click the drill → `/review?key=&wo=N`. Diff +
   threads + answer pane. Keep, with §3 Flow-C improvements.

Where it breaks today: composer invisible at bandwidth 0; place is silent;
analyze indicator is client-only; failure is destructive to prior analysis.

### Flow C — Earn bandwidth by clearing review questions

Entry: `/review?key=<session>` (from the card badge `reviewQuestionsBadge`,
the board `board-row__questions`, or a per-order `board-row__dispatch-questions`).

Happy path with states:

1. Surface loads. NEW subhead: "Clear a question to earn attention bandwidth
   — N open." (Today: "N open — surviving mutants the tests didn't catch:".
   Keep the honesty, add the *why it matters*.)
2. Read: text threads + read-only Monaco with glyph-margin hovers (works),
   opened ON the first question (`revealLineInCenter` — nice).
3. Answer: editable Monaco answer pane. NEW: pre-seed a starter test
   scaffold (package decl + a `func TestAnswer_...` stub anchored to the
   file/line) so the Lead isn't staring at an empty editor needing to know
   the package. Show which file it compiles into (`answerTestFilename`).
4. Submit (⌘/Ctrl+Enter or button). State: `$answering` running line (good).
5. Resolve:
   - Kill: the question vanishes (`markResolved`), and
     `recordQuestionUnblock` awards bandwidth. NEW + critical: show a calm
     earned-moment HERE — "Question cleared — +attention bandwidth" — because
     today the award is silent and on a different surface. This is the
     bandwidth economy's only payoff signal; it must be felt where it's
     earned. Pair with the nav economy strip (§2.3) updating live.
   - Weak: question stays. Today: silent (no-op re-run, question remains).
     NEW: "Mutant survived — your test didn't constrain line N. Try again."
     A weak answer that produces no change is currently indistinguishable
     from a transient failure; name the difference.
6. Empty: "No open questions…" (handled, both session and order paths).

Where it breaks today: the earned-bandwidth payoff is invisible and
off-surface; weak vs failed answers are indistinguishable; empty editor has
no scaffold.

### Flow D — Fleet / multi-session

Entry: `/board` (nav "fleet").

Happy path with states:

1. Board loads. NEW empty/standup framing: "Your fleet — N sessions. M to
   inspect." A standup header (VISION §12.6) instead of a bare create box.
2. Rows: re-designed as scannable cards (§5) with a clear sort legend
   ("sorted by queued work"). Each row's key drills to `/?key=`.
3. Live: consume `/fleet` SSE so rows update as work moves (today static).
   Mark the region live ONLY once it's actually streaming (honesty rule).
4. Create: input + button. NEW: inline validation — an invalid token shows
   "use letters, digits, dashes" instead of silently clearing
   (`CreateSession` no-op today). A duplicate shows "already running".
5. Retire: NEW confirm-inline ("retire <key>? this drops the live view") +
   the honest note that the ledger persists. Today one-click, no confirm.
6. Blocked roll-up: `board__land-summary` ("M of N blocked from landing") is
   good — keep, elevate it into the standup header.

Where it breaks today: not live; no sort legend; no create/retire feedback;
no standup framing.

---

## 4. INTERACTION PATTERNS & BEST PRACTICES to adopt

Mapped to actual hooks/handlers.

### 4.1 Button hierarchy & labels

- Establish primary/secondary/tertiary. Primary (filled, one per zone):
  `compose__place` "Place order", `spend-action`, `settings__save`,
  `board-create__btn`. Secondary (quiet): `compose__analyze`,
  `bench__item`. Tertiary (text): `board-row__retire`. Today they're all
  visually equal `h.Button`s — this is purely a class-hook + stylesheet job;
  add `data-variant="primary|secondary|tertiary"` to each so the calm
  stylesheet can weight them without server logic changes.
- Keep `spendButtonLabel`'s "Spend a catch → fund path:line" — it's exemplary
  microcopy (names what you buy). Apply the same pattern to Place:
  "Place order → run the agent now (−1 attention)".

### 4.2 Disabled-with-reason over hidden (the big one)

Today the composer (`bandwidth > 0`), the spend button (`balance > 0`), and
the authoring block all *vanish* when their resource is zero. Vanishing
destroys discoverability — a Lead can't learn an action exists if it's only
visible once they've already earned the right to it. Replace "omit when zero"
with "render disabled + a one-line reason":

- Composer at bandwidth 0: render it, disabled, with "Earn attention by
  clearing a review question to author live orders. [Inspect →]".
- Spend at balance 0: render it disabled with "No catches to spend yet — the
  oracle mints them as it confirms."
- This is the Linear/Stripe pattern: the affordance teaches the precondition.
  It also fixes the unreachable-key bug (the composer's `compose__needs-key`
  note becomes reachable).
- Caveat to honor the ethos: a disabled control must never *nag*. Disabled =
  quiet/greyed with a calm reason, never a red alarm. And keep the
  *no-op-on-click* server guards exactly as they are (defense in depth).

### 4.3 Inline validation

- `CreateSession` / settings / answer: validate client-side where cheap
  (token charset, non-empty) and echo the server's honest no-op reason as a
  calm inline note. Never a modal, never a toast-storm.

### 4.4 Optimistic / loading / empty / error patterns

- Optimistic: Place clears the editor + shows "WO#N placed" immediately
  (server confirms via the dispatch/bandwidth cells already written).
- Loading: every async action gets a server-driven in-flight state. Pattern
  to copy everywhere: the review answer's `data-indicator="answering"` +
  `data-show="$answering"` running line. Apply to Analyze (server-driven),
  Place, SaveToken, CreateSession.
- Empty: add to `/board` and `/settings` (none today).
- Error: convert the silent no-ops into calm legible states. NOT alarms —
  one-line, in the surface's own idiom (like `analysis__unavailable`). The
  rule: a failure the Lead caused or can fix MUST say so; a transient backend
  hiccup may stay quiet-but-retryable.

### 4.5 Progressive disclosure

- The `/` card's 15 regions (§1.7) collapse into the two-zone IA (§2.2):
  Ledger (look-back, default-collapsed detail) + Act (forward). The
  per-reason stock spans, self-flagged/would-ship tallies
  (`RenderStock`) are detail — disclose under a "ledger detail" toggle, not
  always-on. The Lead skims two numbers and two buttons; expands for the
  audit trail.
- Board rows: header line (key, state, hit-rate) always; the bets/misses/
  backlog cluster on expand.

### 4.6 Readiness / analyzing feedback

- `compose__readiness` (ready/caution) is good — keep its "guide not gate"
  framing (placing stays allowed). Make the analyzing state authoritative
  (server-driven, §3 Flow-B step 2). Show a subtle "last analyzed Ns ago" so
  a stale analysis after further edits is legible (today the decorations can
  silently lag the text).

### 4.7 Focus management for SSE live regions

- After Place: move focus to the filling region heading (a `tabindex="-1"`
  target) so keyboard/SR users land on the thing that just started.
- Scope live regions tightly: transcript = its own bounded polite region with
  a coalesced summary; demote the outer `main` to `aria-live="off"` during a
  fill (§1.3). The current single coarse polite region on `main` (live.go
  `View`) will over-announce.
- `/board` must stay non-live UNTIL it actually consumes `/fleet` — then mark
  it polite. Never label a static GET as live (the code already gets this
  right; preserve it).

### 4.8 Keyboard navigation

The vision is keyboard-native; the surface has none (except Monaco's
⌘/Ctrl+Enter answer submit). Minimum viable, no framework:

- Global: `g f` → fleet, `g r` → review, `g s` → settings, `g h` → flow (`/`).
- On `/`: `a` focus composer, `s` Spend, `e` focus next bench item.
- On `/review`: `j/k` move between threads, `Enter` focus answer editor,
  ⌘/Ctrl+Enter submit (exists). This directly honors VISION §4's `j/k`/`c`/
  `r`/`a` muscle-memory promise. Implement as a tiny progressive-enhancement
  script; the page must still work with it stripped (every key action has a
  visible control behind it).

---

## 5. PRIORITIZED, CONCRETE CHANGES (implementer-ready)

Each: the file/helper to touch, the UX principle, and the microcopy. P0 =
fixes a broken flow; P1 = major legibility; P2 = polish.

### P0 — broken flows

1. Make the API-key prompt reachable on fresh sessions. Principle:
   discoverability / don't gate the prerequisite behind its own outcome.
   Where: move the `compose__needs-key` note OUT of `composeSurface`
   (`authoring.go`) and into `onboardingHint` (`onboarding.go`) as step ①, and
   render it whenever `!tokenStore.Configured()` regardless of bandwidth.
   Copy: "① Arm your key — live orders need an Anthropic key. [Set key →]".

2. Surface bad-key / failed-live-order errors. Principle: error legibility;
   collapse feedback latency. Where: `runLiveOrder` (`live.go`) on the
   `AppendStatus(id,"failed")` path — capture the harness error reason into an
   off-economy per-order cache (sibling to `orderFindings`) and render it on
   the recent-dispatch row + as a settings-side "last live order failed:
   <reason>". Add a "Test key" ping button to `SettingsCard`. Copy:
   "WO#N couldn't run — <reason>. Check your key in settings."

3. Give Place visible feedback. Principle: feedback for the consequential
   action. Where: `PlaceOrder` (`live.go`) — write a transient status cell;
   clear the editor (client, via the existing CustomEvent bridge); move focus
   to the filling region. Copy: "Order placed — WO#N, filling now."

4. Make the earned-bandwidth payoff visible where it's earned. Principle:
   reward the event at its moment. Where: `AnswerQuestion` (`review_surface.go`)
   success path + `renderAnswerForm` — on a kill, render an earned-moment
   line. Copy: "Question cleared — +attention bandwidth. Author a live order
   on your flow. [Go →]". Add the nav economy strip (§2.3) so the meter rise
   is also seen.

5. Composer / Spend: disabled-with-reason instead of hidden. Principle:
   affordance teaches precondition. Where: `LiveCard.View` (`live.go`) — drop
   the `bandwidth > 0` / `balance > 0` omit-guards; always render, add
   `disabled` + a reason note when zero (keep the server no-op guards). Copy
   (composer 0): "Earn attention by clearing a review question. [Inspect →]".
   Copy (spend 0): "No catches to spend yet."

### P1 — major legibility

6. Re-group the card into Ledger / Act zones with directional meter copy.
   Principle: put the number next to what it pays for; separate economies by
   meaning. Where: `LiveCard.View` ordering + `RenderBalance` / `RenderBandwidth`
   (`surface/`). New copy: Balance → "Catches to spend: N" grouped with bench
   + Spend; Bandwidth → "Attention to author with: N" grouped with composer.

7. Add per-surface "what this is for" subheads tied to economy phase.
   Principle: wayfinding. Where: `ReviewCard.View`, `BoardCard.View`,
   `SettingsCard.View`, `LiveCard.View`. Copy examples in §3.

8. Fix the nav model. Principle: predictable home + real breadcrumb. Where:
   `nav.go` — wordmark → `/`; explicit "fleet" link; single-separator
   breadcrumb `packets / fleet / <key> / review` with the current tail as
   text; include `/review` in the trail; add the read-only economy strip.

9. Button hierarchy via `data-variant`. Principle: visual weight matches
   action weight. Where: every `h.Button` across `live.go`, `authoring.go`,
   `board.go`, `settings_card.go`, `review_surface.go`. No logic change —
   just a hook the stylesheet weights. Invert `compose__place` (primary) vs
   `compose__analyze` (secondary).

10. Server-driven analyzing state. Principle: in-flight truth. Where:
    `AnalyzeDraft` (`authoring.go`) — write an `analyzing` cell true on entry,
    false on resolve; bind `compose__analyzing` to it instead of (only) the
    client debounce. Keep client debounce as the trigger.

11. Tighten live-region scoping. Principle: accessible live updates without a
    firehose. Where: `LiveCard.View` — make `order-transcript` its own
    bounded `aria-live="polite"` region with a coalesced summary; set outer
    `main` `aria-live="off"` during an active fill. Keep landmarks.

12. Board: sort legend + standup header + go live off `/fleet`. Principle:
    a sort needs a legend; a "living board" must live. Where: `BoardCard.View`
    (legend + header copy "Your fleet — N sessions, M to inspect, K blocked")
    and wire it to consume `/fleet` (then mark polite). Copy for sort:
    "sorted by queued work".

### P2 — polish

13. Onboarding becomes a graduating checklist (not a one-shot prose block
    that vanishes at stock==1). Where: `onboardingHint` — keep showing a
    slimmer "next step" rail past the first catch until the Lead has done one
    full loop. Principle: no onboarding cliff.

14. Review answer scaffold + weak-vs-failed distinction. Where:
    `answerEditorJS` (pre-seed a `func TestAnswer_...` stub + show the target
    filename) and `AnswerQuestion` (distinct copy for survived vs error).
    Copy (survived): "Mutant survived — your test didn't constrain line N."

15. Progressive disclosure of stock detail. Where: `RenderStock` — collapse
    per-reason / self-flagged / would-ship spans under a "ledger detail"
    toggle; show count + reinvested by default. Principle: skim then inspect.

16. Settings polish: masked last-4 preview, "Replace key" vs "Set key",
    saving state, "stored locally, persists" helper. Where: `SettingsCard.View`
    + `SaveToken`.

17. Create/retire feedback. Where: `CreateSession` / `RetireSession`
    (`board.go`) — inline validation copy + inline retire-confirm.

18. Minimal keyboard nav (`g f/r/s/h`, `/`-page `a`/`s`, `/review` `j/k`).
    Where: a new progressive-enhancement script attached via `AppendToHead`,
    every binding shadowing a visible control. Principle: keyboard-native
    vision, PE-safe.

---

## Microcopy quick-reference (current → proposed)

- Balance row: "Balance: N" → "Catches to spend: N".
- Bandwidth row: "Attention bandwidth: N" → "Attention to author with: N".
- Composer needs-key: keep wording, RELOCATE to onboarding step ① + show at
  bandwidth 0.
- `/review` lead: "N open — surviving mutants the tests didn't catch:" →
  "N open — clear one to earn attention bandwidth."
- Answer success (new): "Question cleared — +attention bandwidth. [Author →]".
- Answer weak (new): "Mutant survived — your test didn't constrain line N."
- Place (new): "Order placed — WO#N, filling now."
- Spend (keep): "Spend a catch → fund <path>:<line>". Place label →
  "Place order → run the agent now (−1 attention)".
- Board sort (new): "sorted by queued work".
- Settings helper (new): "Stored locally beside the ledger; persists across
  restarts."

---

## Executive summary (the direction, 10 lines)

1. Packets' bones are excellent and its ethos (calm, honest, one-row-per-truth)
   is worth protecting — the problems are legibility and reachability, not taste.
2. The biggest sin is "calm" drifting into "opaque": silent no-ops hide
   failures (bad keys, failed orders, over-budget spends) the Lead must see.
3. Discoverability is broken by "hide when zero": the composer and key prompt
   are invisible until you've already earned the right to them — flip to
   disabled-with-reason so affordances teach their preconditions.
4. The two economies read as one thing twice; fix by giving each meter a
   direction and seating it beside the action it funds (balance↔spend,
   attention↔author).
5. The bandwidth payoff (clearing a review question) is invisible and
   delivered on the wrong surface — surface it where it's earned, plus a
   persistent nav economy strip.
6. Place — the emotional peak of authoring — is silent; add optimistic
   confirmation, editor clear, and focus-to-fill.
7. The `/` card's 15 flat regions need two zones (Ledger=look-back,
   Act=forward) and progressive disclosure of audit detail.
8. The fleet board must actually live (consume `/fleet`), show its sort
   legend, and open with a standup header instead of a bare create box.
9. Accessibility hardening: tighter live-region scoping (transcript firehose),
   focus management on async actions, and the keyboard-native nav the vision
   promises but the surface lacks — all progressive-enhancement-safe.
10. Sequence: P0 fixes broken flows (reachable key, surfaced errors, Place
    feedback, visible bandwidth payoff, disabled-with-reason), P1 the
    legibility/IA re-grouping and nav, P2 the polish — none of it betrays the
    control-room calm; it makes the calm legible.
