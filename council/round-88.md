# Round 88 — order-loop slice A: configure the Anthropic key from the UI — 2026-06-11

Trigger: a LIVE order runs an agent through the harness, which needs an
`ANTHROPIC_API_KEY`. Today that key reaches the harness only as a host env var
exported before boot — a Lead working from the browser has no way to supply it,
so the whole web order loop is dead without shell access. This opens a sequenced
program (the order loop A–E); slice A is the setup surface that makes a key
configurable from the UI and durable across restarts.

The program (each slice lands green + independently usable):

- A — token setup (this round).
- C — place a LIVE order from the UI (prompt + base/tip → funded `Target`).
- E — watch the in-flight order's transcript (scrolling beats + state + verdict).
- B — harness-driven authoring assist (highlights, insights, clarifying
  questions in Monaco).
- D — attention-bandwidth economy (earn-by-unblock, latency-weighted, redeemed
  against logged unblock events).

## The change

`internal/tokenstore` owns one concept: the on-disk home of the single Anthropic
key. `Save` writes it owner-only (0600), trimming stray padding so a pasted
key's newline never reaches the harness; `Load` treats absence as the
unconfigured state (not a failure); `Configured` reports presence only.

`SettingsCard` (mounted at `/settings`) is the setup surface. `View` reports the
configured/unconfigured STATE in the calm palette (dim "not yet" vs the balance
hue for a live capability) and renders a masked input + save button. The token
value is NEVER rendered back — the card reports presence, not the secret.

`SaveToken` persists the key AND injects it into `ANTHROPIC_API_KEY` in the host
process env, which is precisely what makes it reach the agent without a restart:
`RunProcess` inherits the server's env, and `RunContainer` passes the key through
BY NAME (`-e ANTHROPIC_API_KEY`, never a value in argv — the R75 hygiene holds).
An empty save is a silent no-op so it can't disarm a configured key.

`NewServer` binds the store beside the ledger (`<ledger>.token`) and, at boot,
injects any saved key into the env — but only when the env carries none, so an
explicitly-exported key (the pre-UI workflow) always wins. The shared nav gains a
`/settings` link so the surface is reachable from every page.

## The tests

- `internal/tokenstore` (external pkg): absent → empty+unconfigured;
  save→load round-trip; whitespace trimmed; re-save overwrites (never appends);
  the saved file is owner-only `0600`.
- `internal/app`: the card reports unconfigured + renders the bound input and
  save action; a save persists the key, injects it into the env, and flips the
  card to configured; the token value never appears in the rendered page; the
  nav links to `/settings`.

## Verdict

Full repo green with `-race`. A Lead can now configure the Anthropic key from the
browser; it persists owner-only across restarts and reaches both harness runners
without leaking the value into argv or the DOM. The order loop's first
prerequisite is met.

## New clashes opened / resolved

Resolved: the harness key is now UI-configurable and durable, closing the
"no key without shell access" gap that blocked the web order loop. No new clash;
slice C (place a LIVE order from the UI) is next.
