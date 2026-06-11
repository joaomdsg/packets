# Round 93 — order-loop slice D-iii: wire the block/unblock emission — 2026-06-11

Trigger: the bandwidth earn (R91) and meter (R92) were live but earned nothing in
the real loop — no event emitted a block or an unblock. This slice wires the
emission to the most concrete logged responsiveness signal: a surfaced review
question is the producer asking for the Lead's input (a block); the Lead's
killing answer clears it (an unblock).

## The change

`liveEntry.recordQuestionBlocks` logs an attention block for each newly-surfaced
review question (the fix oracle's surviving mutants), idempotent per question id
via a `blockedQ` set — a later connect cycle re-finding the same survivor never
re-blocks, so the interval is anchored to when the question FIRST appeared. The
ledger publish runs outside `findingsMu` so I/O never holds the lock; a failure is
best-effort (a missed block only forgoes a future award).

`liveEntry.recordQuestionUnblock` logs the clearing when the Lead's answer kills
the mutant — closing the interval and earning the latency-weighted award.

Wiring: `OnConnect` calls `recordQuestionBlocks(res.Findings)` right where it
caches the cycle's questions, and `AnswerQuestion` calls `recordQuestionUnblock`
on the killing-answer branch (right after `markResolved`).

The balance FIREWALL holds: an unblock moves only the bandwidth meter, never the
catch balance. The existing killing-answer test (balance untouched) still passes;
a new end-to-end test proves the same answer EARNS bandwidth on the second meter.

## The tests

`internal/app`: `recordQuestionBlocks` logs one block per question id (a re-find
never re-blocks); clearing a surfaced question earns bandwidth; an unblock with no
prior block earns nothing; and end to end through the real `AnswerQuestion`
action, a killing answer earns bandwidth while the catch balance stays untouched.

## Verdict

Full repo green with `-race`. The attention economy now turns in the real loop:
answer a producer's question and the bandwidth meter rises, weighted by how fast
you cleared it — grounded in logged block→unblock pairs.

## New clashes opened / resolved

Resolved: the bandwidth meter is now fed by real, logged responsiveness events,
respecting the catch-balance firewall. Open: D-iv (spend bandwidth to dispatch a
live order) makes the earned attention actually FUND autonomous work; the
order-scoped review path (/review?wo=) is a follow-on emission point.
