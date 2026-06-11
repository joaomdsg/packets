# Round 83 — producer-auth, slice 3: authenticate the HTTP /bundle blob — 2026-06-11

Trigger: claims are now NATS-only behind the authenticated boundary (R82), but the
git-bundle upload stays on HTTP (a 32 MiB binary blob is ill-suited to NATS
messaging — the maintainer's "HTTP /bundle, auth vs grant table" decision). This
slice authenticates it against the SAME grant table.

## The change

`POST /bundle` now requires HTTP Basic credentials matching a `ProducerGrant` for
the request's session key (producer == session key): `bundleAuthorized` finds a
grant with `Session == key && User == user` and compares the password with
`crypto/subtle.ConstantTimeCompare` (so a prober cannot time-recover it; the
user/session equality checks are not secret). The check runs BEFORE the registry
lookup, so an unauthenticated prober cannot use the endpoint as an oracle for
which session keys exist.

Auth is gated on `len(cfg.Grants) > 0`: with producers configured (a real
cross-process deployment, `-producer`/`-producer-listen`), the bundle channel
demands grant credentials; with no grants (in-process / single-user runs and the
existing tests) the endpoint stays open exactly as before. One credential
authority (the grant table), two transports (NATS for claims, HTTP for the blob).

## Tests

RED: `TestPostBundle_requiresGrantCredentialsWhenProducersAreConfigured` — with a
grant configured, no credentials / wrong password / unknown user all get 401, and
the granted producer's credentials get 202 with the commit ingested into
`refs/producers/<key>/*`. The existing no-grant bundle tests (`bundle_route`) stay
green — proving the open in-process path is unchanged.

## Scope held

The grant table is reused as-is; no new credential store. Aggregate byte quota +
per-producer rate-limit on the (now authenticated) bundle channel are R85. TLS /
transport confidentiality stays a deployment concern (Basic is cleartext without
it), as with the NATS path — out of scope here, noted.

## Verdict

Full `internal/app` suite green with `-race`; build clean. Both producer ingresses
— NATS claims (R81/R82) and the HTTP bundle blob (R83) — now authenticate against
one grant table. Next: R84 wires the GC-by-resolved post-verdict hook; R85 adds
the bundle flood-defenses the auth boundary now makes attributable.

## New clashes opened / resolved

Resolved: the HTTP bundle upload is authenticated against the grant table when
producers are configured, closing the last unauthenticated producer ingress. The
in-process/no-grant path stays open by design (no boundary to enforce when there
are no external producers).
</content>
