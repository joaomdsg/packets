# Round 91 — order-loop slice D-i: the attention-bandwidth earn — 2026-06-11

Trigger: slice D introduces a SECOND economic meter — attention bandwidth, the
scarce resource that funds dispatching autonomous work, earned by how
responsively the Lead unblocks producers. The model is the riskiest part of the
program (the council holds economic units to a "redeem against a logged fact"
bar), so it lands first as a self-contained, tested ledger primitive — the EARN
side only, no UI or gating yet (D-ii surfaces the meter; D-iv spends it).

## The change

Two new logged event kinds on the canonical minted subtree: `block` (a producer
needs the Lead's input — a raised question, an order awaiting review) and
`unblock` (the Lead cleared it). Each carries an id and a wall-clock stamp, so the
clear LATENCY is a logged fact, never an inference. `Log.AppendBlock` /
`AppendUnblock` log them; unlike a spend they gate on nothing (a block is a fact,
not a debit).

`Projection.Bandwidth` folds the earn: the sum of awards across every CLEARED
block (a block id with a matching unblock). The award redeems against exactly one
logged block→unblock pair and folds BOTH axes the Lead chose:

- a throughput base (`bandwidthBase = 1`) — you cleared a block at all;
- a latency bonus — `+2` within 2 min (fast), `+1` within 15 min (medium), `+0`
  beyond (slow).

So a fast clear is worth 3, medium 2, slow 1. An OPEN block (no unblock) earns
nothing; a duplicate unblock for the same id never re-pays (the first clearing
wins); a skewed (negative) latency floors at the base. The fold takes the first
block and first unblock stamp per id, so the interval and the award are stable
across replay.

## The tests

`internal/ledger` (external pkg): zero with no events; an open block earns
nothing; a cleared block earns base + the right latency bonus (table across
fast/medium/slow + the tier boundaries); bandwidth sums across cleared blocks; a
duplicate unblock does not double-count; a negative latency floors at the base.

## Verdict

Full repo green with `-race`. The attention economy's source is a sound,
tested projection: every bandwidth credit redeems against one logged block→unblock
pair, weighted by a latency the log proves. No model judgment enters the unit.

## New clashes opened / resolved

Resolved: the attention-bandwidth EARN is grounded and tested, clearing the
soundness bar before any UI or gating depends on it. Open: D-ii (surface the
meter on the card) and D-iv (spend bandwidth to dispatch) build on this; the
real block/unblock emission points (D-iii) wire it to questions/reviews.
