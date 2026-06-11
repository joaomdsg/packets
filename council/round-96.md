# Round 96 — order-loop slice B-ii: wire the authoring assist into the card — 2026-06-11

Trigger: the authoring-assist ENGINE landed (R95) but nothing ran it — the Lead
still authored a live order's prompt blind. This slice wires the engine into the
card: a producer reads the draft on demand and the card surfaces its structured
read — summary, readiness, clarifying questions, and the flagged spans decorated
in Monaco — so the Lead sharpens the goal before placing the order.

## The change

`analyzeDraft` is the seam the assist runs through (default `runAnalysisProcess`,
which shells `claude -p <prompt> --output-format text` and returns the raw reply;
process I/O, verified by build + manual run like `RunProcess`; tests swap it). It
runs in plain text output and settles nothing — analyzing a draft must touch
neither the working tree nor the economy.

`LiveCard.AnalyzeDraft` reads the authored `OrderPrompt` (the compose textarea's
bound value), runs the analysis harness on `assist.AnalysisPrompt(draft)`, parses
the reply through `assist.ParseAnalysis`, and caches the result on the session
entry (`draftAnalysis`, off-ledger like `findings`). An empty draft is a silent
no-op; a failed run or unreadable output degrades to a calm "analysis
unavailable" state with the place control intact. FIREWALL: it writes only the
off-economy cache, never the ledger — analyzing mints nothing.

`renderAuthoring` replaces the bare compose control: the textarea now carries an
"Analyze draft" button beside "Place order" (both read the same `OrderPrompt`),
and beneath it `renderAnalysis` surfaces the producer's read — the summary, a
readiness hook (`ready|blocked`, colored in the calm palette), the clarifying
questions to answer before re-analyzing, and a Monaco authoring island that
decorates the analyzed draft with each flagged span (offsets → positions via
`getPositionAt`, hover = the producer's note). The island is progressive
enhancement: a loader/parse failure leaves the server-rendered summary + questions
intact.

## The tests

`internal/app` (NOT parallel — shared globals): a stubbed producer reply renders
the summary, the blocked readiness hook, and every clarifying question; the
highlight payload + a ready hook reach the Monaco island; unreadable output
degrades to the calm unavailable state with the place control surviving; an empty
draft never spawns a producer; the analyze control renders when bandwidth funds
authoring.

## Verdict

Full repo green with `-race`, vet clean. The authoring loop is closed: type a
draft → analyze → read the producer's summary/readiness/questions with the flagged
spans highlighted in Monaco → sharpen → place. The order loop A→C→E→D→B is
complete.

## New clashes opened / resolved

Resolved: the producer now reads the draft and the card surfaces its structured
insights + Monaco-anchored highlights, closing slice B — the harness-driven
authoring assist the Lead asked for. Open: the assist is on-demand (a button),
not live-as-you-type, and answering a question means editing the draft + re-
analyzing rather than an inline interactive reply; a future slice could debounce
analysis on pause and thread the questions back into the draft inline.
