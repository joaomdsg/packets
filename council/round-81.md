# Round 81 — producer-auth boundary, slice 1: bind the authenticated NATS listener — 2026-06-11

Trigger: the maintainer opened the R78-held "producer-auth + bundle GC" gate and
chose, on interview: auth = the EXISTING NATS `ProducerGrant` (not bolted-on HTTP
auth); producer identity == session key; claim ingress moves to NATS-only; the
bundle blob stays HTTP authenticated against the same grant table; GC by a
post-verdict hook; and the full flood-defense sequence in scope. This round is
slice 1: wire the authenticated listener into the live server.

## What already existed (so this slice is wiring, not invention)

`fabric.StartListening(ctx, dir, addr, grants...)` already boots the fabric with
a TCP listener AND per-producer auth: the in-process host maps to `hostUser`
(full access, IN_PROCESS-only) via `NoAuthUser`, while each `ProducerGrant`
authenticates and may publish ONLY to
`packets.session.<session>.events.<instance>.claim.>` and subscribe only to its
reply inbox. The grant-confinement is already exhaustively fabric-tested
(`fabric/listen_test.go`). It was simply never wired into the live server, which
booted the in-process-only `fabric.Start`.

## The slice (TDD)

- `LiveConfig` gains `ListenAddr string` + `Grants []fabric.ProducerGrant`. Empty
  ListenAddr keeps the in-process-only fabric (the default — every existing test
  and single-process run needs no socket and no auth surface, so they are
  untouched).
- `NewProducerGrant(sessionKey, user, pass)` is the sanctioned grant constructor:
  it binds the grant to `ledgerInstance` (the one instance every economy uses) so
  a producer's claims are both publishable under its grant AND consumable by the
  session's claim consumer. Callers need not know the internal instance token.
- `startLiveFabric` now takes the addr + grants and calls `StartListening` when an
  addr is configured, else `Start`. `NewServer` threads `cfg.ListenAddr`/`cfg.Grants`.
- `cmd/packets`: `-producer-listen <host:port>` binds the socket; repeatable
  `-producer key:user:pass` authorizes producers. `parseProducerSpec` (pure,
  unit-tested) splits on the first two colons so a password may contain colons,
  and requires all three fields. `-producer` without `-producer-listen` fails fast
  (no point authorizing a producer with no socket to reach).

RED tests:
- `TestNewServer_bindsAnAuthenticatedListenerForGrantedProducers` (app, internal):
  a configured ListenAddr binds a real socket (`liveFabric.Addr()` non-empty); the
  granted producer's credentials connect; a WRONG credential is refused at connect;
  and the in-process host economy still reads its own ledger (host path unchanged).
- `TestParseProducerSpec_*` (cmd): grant confined to the session key,
  colon-bearing password preserved, half specs rejected.

## Scope held

This slice ONLY binds the authenticated ingress; it does NOT yet move claim
submission off the unauthenticated HTTP `POST /claim` (slice 2 / R82) nor
authenticate `/bundle` (R83). Both still work as before — the socket is additive.
So the boundary is bound and credential-enforced, but the unauthenticated HTTP
claim path still exists until R82 retires it; this round does not over-claim the
gate is closed.

## Verdict

`internal/app`, `cmd/packets`, `internal/fabric` green with `-race`. The
authenticated producer ingress is now bound by the live server and reachable with
grant credentials, with the in-process host minting unaffected. Next: R82 retires
the unauthenticated HTTP `POST /claim` so claims arrive ONLY through this boundary.

## New clashes opened / resolved

Resolved: the producer-auth boundary is the NATS `ProducerGrant` listener (a
maintainer decision, R78→R81), wired into the live server without disturbing the
in-process host or the default in-process-only fabric. Open (next slices): the
unauthenticated HTTP `/claim` (R82) and `/bundle` (R83) still bypass the boundary.
</content>
