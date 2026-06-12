# Round 98 — order-loop slice B-iv: the draft IS the Monaco editor — 2026-06-12

Trigger: the authoring surface was split — the Lead typed in a plain textarea while
a SEPARATE read-only Monaco panel mirrored the analyzed draft with the flagged
spans. That diverged from the ask ("the producer highlights parts of the text,
native to Monaco"): the highlights belong in the editor you write in, not a copy.
This slice makes the draft a single editable Monaco editor with inline highlights.

## The change

The textarea + read-only mirror are gone. `composeSurface` now renders ONE editable
Monaco editor as the draft source. Its value reaches the server through the proven
CustomEvent bridge (the review answer-form / maplibre pattern that works without
data-bind and survives morphs): the analyze + place buttons dispatch a CustomEvent
carrying the editor's value, and the wrapper's `data-on:viaanalyze` /
`data-on:viaplace` handlers assign it to `$orderprompt` INLINE before `@post`ing —
so `AnalyzeDraft` and `PlaceOrder` read the live editor content unchanged (their
handlers, firewall, and tests are untouched).

The interactive subtree (`.compose__live`, `data-ignore-morph`) holds the editor +
buttons + indicator, so the editor's DOM, the Lead's text, the cursor, and the JS
listeners all survive SSE re-renders. The re-rendering bits — the readiness
reflection and the highlights payload — sit OUTSIDE the shield. `authoringEditorJS`
decorates the flagged spans INLINE in the editor and reapplies them whenever a fresh
analysis payload arrives (a `MutationObserver` on the payload element), so the
highlights update live without remounting the editor or losing the draft. A
debounced `onDidChangeModelContent` dispatches the analyze bridge, so the producer
keeps reading as the Lead types.

## The tests

`internal/app` (NOT parallel — shared globals): the authoring control renders the
editable editor (`authoring-editor`) whose value is lifted into the order-prompt
signal at place time (`$orderprompt=evt.detail.draft` + `/_action/PlaceOrder`); the
editor carries the live debounced re-analysis bridge (`viaanalyze`) and the
analyzing indicator. The analyze/place/readiness/firewall server tests are
unchanged and still green (the bridge sets the same signal the tests inject).

## Verdict

Full repo green with `-race`, vet clean. The draft is now one editable Monaco
editor with the producer's flagged spans highlighted inline as you write — the
"native to Monaco" authoring the Lead described, with no second surface and the
server contract (and its firewall) unchanged.

## New clashes opened / resolved

Resolved: the authoring surface is a single editable Monaco editor with inline,
live-updating highlights — closing the textarea/mirror split. Open: the editor
bridge + decoration observer are browser-verified, not unit-tested (as with every
Monaco island here); byte vs UTF-16 offset alignment is exact only for ASCII drafts
(a future slice could pass rune-accurate offsets); and an end-to-end browser run of
the full author→analyze→place loop is the remaining unautomatable check.
