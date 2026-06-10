# Round 62 — MONACO REVIEW UI thread opens (maintainer steer): nail the user flows — 2026-06-10

Trigger: the maintainer steered explicitly — "steer to Monaco integration. I can't
wait to see the review UI working. make sure the council nails the user flows." A
genuine, NON-GATED steer (only the #6 live boundary is hard-gated). This opens the
Monaco review-editor thread that R59/R60 had flagged as "client-side / untestable /
no WithPlugins".

Panelists present: the full six (UX, Game, Systems, TDD, CI/CD, Refactoring),
re-summoned from README §1, each grounded in the via framework + the R56 review model.

## New evidence — reachability OVERTURNS the R59/R60 assumption

A grounded scout of go-via/via@v0.5.0 found that R59's "no WithPlugins / untestable"
claim was WRONG:
- via HAS a `Plugin` interface + `via.WithPlugins` (a working echarts CDN-plugin
  precedent: plugins/echarts/plugin.go AppendToHead(h.Script(h.Src(cdn)))).
- `h.DataIgnoreMorph()` (h/datastar.go) makes a DOM subtree SURVIVE SSE re-renders —
  Datastar's morph skips `[data-ignore-morph]`. So a Monaco editor div is never
  clobbered by a live re-render. This was the blocker; it's solved.
- Testability boundary (the discipline the council adopts): a vt SERVER-render test
  CAN assert the island container + data-ignore-morph + the DATA PAYLOAD the server
  feeds the editor (a `<script type="application/json">` of anchored threads). It
  CANNOT assert Monaco rendering/typing — that's the client island, deliberately
  NOT unit-tested (asserting it through vt would be test-theater).

## Per panelist — and the convergence (UNANIMOUS on shape + sequencing)

- UX: entry = card "N open questions" badge → /review; Monaco shows the fix file
  READ-ONLY with threads anchored inline (glyph-margin + zone widgets) at File:Line,
  keyboard thread→thread nav; the thread VANISHING next cycle is the calm reward.
  Read-only first. Slice 1 = the island scaffold + JSON payload (testable now).
- Game: the review session is a "clear the board" beat; the honest reward is the
  thread vanishing (NO count-badge inflation, NO confetti); empty review = a
  walk-away win, not a nag. Read-only is the honest loop (the kill happens in the
  REAL suite). Defer editable.
- Systems: read-only keeps the two-scores firewall TRIVIAL (answering a question
  mints NOTHING; questions stay diagnostic, off-ledger). Editable-in-browser
  reintroduces a write→re-run path with a real degenerate-strategy surface (trivial
  tests "killing" mutants), cage compute cost, and a diagnostic-vs-minted firewall to
  design — a DEEP thread needing a maintainer steer. Read-only first.
- TDD: the testable SERVER CONTRACT for slice 1 = the structured thread DATA PAYLOAD
  fed into the island (a load-bearing RED: strip the payload → the editor has no
  data → test fails). Monaco's rendering is the client island we do NOT fake-test.
  Read-only gives the cleanest first RED. Explicit test-theater boundary: never
  assert a Monaco class/decoration/keystroke through vt.
- CI/CD: a CDN script on the SERVED page is the Lead's browser, NOT the cage (the
  #6 egress kill was the cage at --network=none — a separate boundary, no conflict).
  Still, lean VENDOR (embed.FS + a /static handler, pinned version) for
  reproducibility/offline — but that decision belongs to slice 2 (when Monaco
  actually loads). Slice 1 (payload only) needs no script load → no CI impact (vt
  asserts a string, never fetches).
- Refactoring: EXTEND /review's ReviewCard + the sessionOpenThreads projection — do
  NOT fork the data model. One projection feeds BOTH the server text and the JSON
  payload (no drift). Incremental order: slice 1 = payload + island scaffold; slice
  2 = load Monaco + read-only decorations; slice 3 = editable (later). Inline the
  island in View now; extract a MonacoPlugin only if a 2nd surface needs it.

## Decisions

- **READ-ONLY first; editable-in-browser is a DEEP fork DEFERRED to an explicit
  maintainer steer** (cage compute cost + degenerate-strategy gating + the
  diagnostic-vs-minted economy firewall must be designed first — Systems + TDD +
  CI/CD all flagged it). The loop will NOT build editable autonomously.
- **One projection, no fork** — the JSON payload and the server text both come from
  sessionOpenThreads (Refactoring).
- **Test the server data contract; never fake-test the client editor** (TDD).
- Slice ORDER: (1) payload + island scaffold [THIS ROUND]; (2) load Monaco read-only
  + anchored decorations [CDN-vs-vendor decided then, lean vendor]; (3) editable
  [gated on maintainer].

## SLICE 1 (built this round)

ReviewCard.View, when a session has open questions, now emits a `review-editor`
island container (`id="review-editor"`, `data-ignore-morph` so Monaco's future DOM
survives SSE) carrying a `<script type="application/json" id="review-threads-data">`
payload of the threads `[{file,line,tag,body}]`, projected from the SAME
sessionOpenThreads the server text uses. Omitted entirely when there are no
questions (nothing to scaffold over an empty set; the calm empty state stands).
encoding/json HTML-escapes <,>,& so an oracle message can't break out of the script.
Tests (review_island_internal_test.go): payload present + VALID JSON + correct
thread count/field values when questions exist; island + payload omitted when none.
Pure server contract — zero client-JS this slice, so zero test-theater. Blue clean,
full -race gate green.

## New clashes opened / resolved

- The R59 "Monaco untestable / no WithPlugins" claim is RESOLVED as a misread — via
  hosts client-JS islands and the server data contract is testable. The editable-vs-
  read-only question is settled (read-only first; editable gated on maintainer).
</content>
