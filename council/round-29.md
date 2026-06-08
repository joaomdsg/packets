# Round 29 — #4 NATS→SSE browser bridge SHIPPED: the economy streams to the browser off the authoritative stream — 2026-06-08

Trigger: build-evidence round logged after the swap (Round 28) unblocked the
first cross-process consumer. No re-convene — this records the slice that
lands roadmap #4.

Panelists present: none re-convened; evidence the next round can argue over.

New evidence on the table:

- `internal/bridge.Watch(ctx, f, session, instance) <-chan ledger.Projection`
  subscribes to a session's minted subtree via `fabric.Subscribe` and emits a
  freshly-folded projection per committed event — history first (a late
  subscriber sees current state), then live tail. Re-folds via
  `ledger.ReplayProjection` so the canonical fold is reused, not duplicated.
- `internal/bridge.Handler` serves that feed as a plain `text/event-stream`
  endpoint: a JSON snapshot frame (balance/catches/orders/queued) per event,
  flushed, with `Cache-Control: no-cache`. Teardown rides the request context
  — client disconnect cancels Watch's subscription and feeder goroutine (the
  no-leak chain verified against net/http: handler return → cancelCtx →
  Watch teardown, including the blocked-write path).
- Mounted in the running binary: `NewServer` registers `GET /stream` on the
  Via app over the default session's fabric, so a browser connects
  end-to-end. The method-qualified pattern sidesteps the Go 1.22 ServeMux
  precedence conflict with Via's `GET /` mount.

This is the bridge UX Round 28 deferred until AFTER the swap: the rendered
view now rides the subscribable stream, so it cannot disagree with the real
ledger, and a future cross-process producer's events drive the same render.
It is deliberately a separate, plain SSE path — distinct from the in-process
Via reactivity at `/` and `/board`, which is framework-coupled and not
cross-process.

## Clashes touched

None re-litigated. The bridge is the read-side substrate the trust-economy
render bricks (Clash A's silent-vs-badge framing, Clash H's outward Ledger)
will eventually ride — but those stay blocked on log-derivable inputs, not on
transport.

## Verdicts updated

None flip. The thesis stays PROVEN; this is leverage on the proven economy.

## New clashes opened

None at target level. One honest scope note carried forward: `/stream` serves
only `defaultSessionKey`. Per-session and cross-session streaming is roadmap
#5 (the cross-session board aggregator) — the bridge primitive is already
session-parameterized (`Watch`/`Handler` take session+instance), so #5 is a
keying/fan-in slice over this foundation, not new transport.

## Decisions

1. Roadmap #4 (NATS→SSE browser bridge) is LANDED for the default session,
   browser-reachable end-to-end, `-race` green.
2. NEXT BUILD is #5, the cross-session board aggregator: key the stream by
   session (or aggregate the fleet) so the board reflects every live card off
   the one stream — reusing the session-parameterized bridge primitive.
3. The OS-process boundary (#6, the cross-process producer + security trio)
   stays gated; the SSE bridge is read-side and single-process, so it shipped
   ungated.
