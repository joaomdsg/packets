# Round 90 — order-loop slice E: watch the in-flight transcript — 2026-06-11

Trigger: a placed live order (R89) ran with only a single updating "latest move"
line on the card. To WATCH a run unfold — the payoff of placing an order — the
Lead needs the accruing transcript: every beat, in order, scrolling in place.
This slice adds it, completing the usable place-and-watch loop.

## The change

`liveEntry` gains `activityLog` — the accruing transcript of the agent's beats
this fill, in stream order. `addActivityBeat` now records BOTH the single
latest-line (unchanged) AND appends to the transcript, capped at
`maxActivityLog` (300, oldest dropped) so a long run can't grow the per-session
buffer without bound. It is bracketed to the fill lifecycle like the latest-line
(reset in `startFill`, cleared in `endFill`) so a resolved order's transcript
does not linger. `activityTranscript` returns a copy — the run history the card
scrolls.

`View` renders the transcript while an order is in flight: a bounded,
scroll-in-place region (`data-state="transcript"`) with one line per beat, below
the existing latest-move line and above the verdict/land rows already on the
card. It re-renders each Stream tick (the same poll that drives the latest-line),
and is omitted until there is a beat — no empty pane, no spinner. The CSS bounds
its height (`max-height` + `overflow-y:auto`) so a long run scrolls rather than
pushing the card.

The place-and-watch loop now reads end to end on one surface: author a task
(R89) → it funds and dispatches → watch the agent's transcript scroll as it runs
→ the verdict/land rows resolve in place.

## The tests

`internal/app`: the transcript accumulates each streamed beat in order
(asserted mid-run from the harness stub); `endFill` clears it so a resolved
order's transcript does not linger; the card renders the scrolling transcript
region with each beat while an order is in flight.

## Verdict

Full repo green with `-race`. A Lead watches a live order's run unfold beat by
beat on the card they placed it from — the usable place-and-watch loop (slices
A→C→E) is complete.

## New clashes opened / resolved

Resolved: an in-flight live order's run is now legible as a scrolling transcript,
closing the "place an order but can't watch it work" gap. No new clash; the
fancy authoring assist (slice B) and the attention-bandwidth economy (slice D)
are next.
