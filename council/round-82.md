# Round 82 — producer-auth, slice 2: retire the unauthenticated HTTP POST /claim — 2026-06-11

Trigger: R81 bound the authenticated NATS ingress. Per the maintainer's
"NATS-only, retire HTTP /claim" decision, this slice removes the unauthenticated
HTTP claim edge so a claim can arrive ONLY through the grant-confined boundary.

## The change

- Removed the `POST /claim` handler from `internal/app/live.go` (and its
  `maxClaimBodyBytes` cap and the now-unused `encoding/json` import). A claim now
  reaches a session ONLY by an authenticated producer publishing to its
  grant-confined claim subtree (R81), which the host's claim consumer drains. The
  per-message size bound is now NATS's max-payload; the cage verifier remains the
  fail-closed check that a claim's revisions resolve.
- This is a security improvement: previously anyone who could reach the HTTP port
  could inject a claim into any registered session. That edge is gone.

## Test migration (the lifecycle tests keep their meaning, lose the HTTP edge)

The HTTP `/claim` post was the test ingress in four files. Migration:

- New `publishClaim(t, key, target)` helper submits a claim via
  `ledger.PublishClaim` on the shared fabric — the in-process equivalent of an
  authenticated producer publishing the same encoded `ClaimRecord` over the NATS
  socket (the ingress `gc`/`board_inflight`/`inproc_verifier` tests already used).
- `claim_consumer`, `claims_internal`, `lazy_consumer` now publish via that helper
  instead of `http.Post(.../claim)`; their consume→verify→mint assertions are
  unchanged, so the route→publish→consume→verify→mint coverage is preserved minus
  the retired HTTP hop.
- Deleted `claim_route_internal_test.go`: its tests asserted HTTP-edge behaviors
  (202-accepted, 400-oversized-body, 404-unregistered-key, JSON-field validation)
  of a route that no longer exists. The "accepted as in-flight, mints nothing"
  invariant they protected is already covered by the `board_inflight` PublishClaim
  tests.

RED test: `TestPostClaim_isRetiredFromTheUnauthenticatedHTTPSurface` — POST /claim
must now be a client error (≥400), never 202.

## Scope held

`/bundle` is still unauthenticated (R83 next). The in-process `publishClaim`
helper exercises the consumer, not the authenticated cross-process publish over
TCP — that end-to-end (external NATS client → grant-confined publish → consume) is
fabric's tested contract (`fabric/listen_test.go`) plus R81's connect proof; this
round does not add a full external-publish integration test, and says so.

## Verdict

Full suite green with `-race`. The unauthenticated HTTP claim edge is gone;
claims arrive only through the authenticated boundary or the host's in-process
publish. Next: R83 authenticates the HTTP `/bundle` blob upload against the same
grant table.

## New clashes opened / resolved

Resolved: claim submission is NATS-only (R78→R82). The HTTP claim edge — an
unauthenticated injection point into any registered session — is removed. Open:
`/bundle` remains an unauthenticated HTTP surface until R83.
</content>
