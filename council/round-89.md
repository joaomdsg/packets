# Round 89 — order-loop slice C: place a LIVE order from the UI — 2026-06-11

Trigger: a LIVE order (a prompt-carrying target the harness runs) could only be
baked at boot via the `-live` CLI flag. With the key now UI-configurable (R88),
the missing piece of a usable web loop is AUTHORING an order from the browser —
type a task and place it. This slice adds that.

## The change

`LiveCard.PlaceOrder` is the runtime counterpart of `-live`: it reads the
free-form task the Lead authored (the `OrderPrompt` signal), resolves the repo's
CURRENT HEAD as the base (so the agent works the live tree), and funds a
`Target{BaseRev: head, Prompt: prompt}` through the same `AppendDispatch` path a
Spend uses — one catch debited, the order queued. The existing drain routes a
prompt-carrying order to the live harness (`runLiveOrder`), so the authored task
actually runs. It mirrors Spend's SSE announce (drained balance + risen dispatch)
and runs the order in the background.

Three honest no-ops guard it: an empty prompt is not an order; an unresolvable
repo HEAD (`repoHead` reports false) never dispatches a treeless order; an
over-budget balance is refused by the ledger. None surfaces as an error to the
Lead.

`renderCompose` adds the control to the card when there is balance to fund an
order: a prompt textarea bound to `OrderPrompt` + a place-order button. Below it,
when no API key is configured, a calm note links to `/settings` — the authored
order would fail without a key, so the Lead is pointed at the fix rather than
left to place a dead order.

The economy gate is UNCHANGED — placing a live order costs one catch, exactly
like any dispatch. (Slice D replaces that gate with the attention-bandwidth
model; this slice deliberately reuses the existing economy so the loop is usable
today.)

## The tests

`internal/app`: placing an order funds + dispatches the authored prompt against
the live HEAD and spends one catch; an empty prompt is a no-op (balance
untouched); the card renders the authoring control (bound input + action) when
funded.

## Verdict

Full repo green with `-race`. A Lead can author a task in the browser and place
it as a funded live order that runs through the harness — the loop's entry point
is live. Watching it run (the transcript) is slice E, next.

## New clashes opened / resolved

Resolved: live orders are now composable at runtime from the UI, not only baked
at boot — the web loop has an entry point. No new clash; slice E (watch the
in-flight transcript) is next.
