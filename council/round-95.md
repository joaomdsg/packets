# Round 95 — order-loop slice B-i: the authoring-assist engine — 2026-06-11

Trigger: the order loop is complete and economically coherent (A→C→E→D), but the
Lead authors a live order's prompt blind — no producer reads the draft as it is
written. Slice B is the harness-driven authoring assist the Lead asked for: a
producer analyzes the draft live, flags spans, and asks the clarifying questions
worth answering before the work is dispatched. This round lands the ENGINE — the
typed contract between a harness run and the authoring surface — ahead of the UI
wiring (B-ii), so the analysis is a tested value, never raw agent output the card
parses inline.

## The change

`internal/assist` models the producer's read of a draft. `Analysis` is the typed
result: a one-line `Summary`, a `Ready` verdict (is the goal clear and verifiable
enough to run unattended), `Highlights` (byte-offset spans with a note + severity,
so the editor can anchor a decoration on exactly that range), and the clarifying
`Questions`.

`AnalysisPrompt` builds the instruction a producer harness runs: it carries the
draft and pins the EXACT JSON shape the parser decodes, so the prompt and parser
are one contract (the engine matches the Lead's choice — a Claude Code harness
run, not a separate API path).

`ParseAnalysis` extracts the one JSON object the agent printed (tolerant of
surrounding prose and ```json fences, tracking string literals so a brace inside a
JSON string never throws off the balance) and validates it against the draft: a
highlight whose range is inverted or out of bounds is DROPPED rather than returned
as a range Monaco can't anchor (an end exactly at `len(draft)` is valid). Output
with no JSON object is an error, never a silently empty analysis.

## The tests

`internal/assist` (external pkg): the parser extracts the JSON block from noisy
output and from a fenced code block; errors when no object is present; drops
out-of-bounds/inverted highlights while keeping the valid ones; accepts a
highlight ending exactly at the draft length; the prompt carries the draft and
names every field of the contract; and a prompt-shaped reply round-trips through
the parser without loss.

## Verdict

Full repo green with `-race`. The authoring-assist contract is a tested value: a
harness run's output becomes a validated `Analysis` the UI can render against,
with every editor-unsafe highlight already filtered. The engine is ready to wire.

## New clashes opened / resolved

Resolved: the producer's draft analysis is a typed, validated contract
(`assist.Analysis`) rather than raw agent text parsed at the call site. Open:
B-ii — wire the engine into a live Monaco authoring surface (run the analysis
harness on the draft, render highlights as decorations + a clarifying-questions
panel + the readiness signal) feeding `PlaceOrder`.
