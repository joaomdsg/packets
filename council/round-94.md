# Round 94 — order-loop slice D-iv: spend bandwidth to dispatch — 2026-06-11

Trigger: the attention-bandwidth meter earned (R91–R93) but funded nothing — a
live order still cost a catch (slice C reused the existing economy). This slice
makes the earned attention actually FUND autonomous work, completing the
two-meter division the Lead chose: catches fund the backlog Spend; bandwidth
funds the live orders you author.

## The change

The bandwidth meter gains a SINK. `BandwidthSpendRecord` (kind `bwspend`) is a
debit against earned bandwidth, mirroring `SpendRecord` for the catch balance;
the projection folds it as `earned − spent`. `Log.AppendBandwidthSpend` refuses
any amount the meter can't cover (no overdraft).

`Log.AppendLiveDispatch` funds a UI-authored live order from bandwidth: under the
lock it refuses its own target (the distinct-work guard) and any dispatch the
bandwidth can't cover, then publishes the bandwidth debit + the queued
work-order in one write (the meter can never fund more orders than it held). The
catch balance is untouched — a live order is funded by attention, not a catch.

`LiveCard.PlaceOrder` now calls `AppendLiveDispatch` and announces the drained
meter over a new `BandwidthMeter` broadcast cell. The compose control is gated on
`bandwidth > 0` (the attention to fund a live order), not balance — so the two
controls track their two meters: Spend appears with balance, the order composer
with bandwidth.

## The tests

`internal/ledger`: a bandwidth spend debits the meter; an overdraft is refused
(meter untouched); `AppendLiveDispatch` spends one bandwidth, queues the
prompt-carrying order, and leaves the catch balance untouched; it refuses with no
bandwidth and refuses its own target. `internal/app` (slice C, re-funded): a
placed order now draws down bandwidth (3 earned → 2), an empty prompt leaves the
meter untouched, and the composer renders when bandwidth funds it.

## Verdict

Full repo green with `-race`. The attention economy is closed: you EARN bandwidth
by clearing producers' questions fast, and you SPEND it to dispatch the
autonomous work you author. Both meters are live and each funds its own move.

## New clashes opened / resolved

Resolved: attention bandwidth now funds live-order dispatch — the spend-to-earn
loop the Lead described, grounded end to end in logged facts. The A→C→E→D loop
is complete and economically coherent. Open: slice B (the harness-driven Monaco
authoring assist) is the remaining ambition; the live-dispatch cost is a flat 1
(a future slice could scale it by the order's scope).
