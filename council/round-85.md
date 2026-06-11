# Round 85 — producer-auth gate, final slice: bundle flood-defenses — 2026-06-11

Trigger: the auth boundary (R81–R83) makes a bundle upload attributable to a
producer (== session key), and GC-by-resolved (R84) reclaims its objects. This
slice adds the per-producer flood-defenses RISKS.md sequences after the auth
boundary — completing the gate.

## The change

A per-producer `bundleGuard` (keyed by session key, server-lifetime) on
POST /bundle, with two limits the authenticated identity now makes enforceable:

- **Upload rate-limit:** a `golang.org/x/time/rate` token bucket
  (`bundleRatePerSec`/`bundleBurst`) — a producer that floods past its burst gets
  429 before the ingest path, so it cannot hammer the host's git work. Checked
  after auth + the no-repo guard.
- **Aggregate retained-byte quota:** `bundleQuotaBytes` bounds the bytes a
  producer may have RETAINED at once. `guard.reserve(len)` is taken BEFORE
  ingesting (an over-quota upload is refused 413 without doing the work); a failed
  ingest calls `guard.release(len)` so a rejected upload never permanently consumes
  quota. The quota counts bytes-ACCEPTED (deterministic), never git's on-disk size.
- **GC frees the quota (couples to R84):** when `pruneProducerIfIdle` reclaims an
  idle producer's namespace, it calls `resetBundleRetained(key)` — so reclaimed
  objects free the quota that backed them, closing the loop with the post-verdict
  hook and the periodic sweep.

The limits are package vars (not consts) so tests shrink them deterministically;
the per-producer guard registry is cleared in `resetConsumersForTest` and
`bundleServer` so server-lifetime state never leaks across tests.

RED tests:
- `TestBundleGuard_reserveRefusesOverTheQuotaAndResetFrees` — reserve/release/reset
  accounting against a small quota (pure).
- `TestPostBundle_throttlesAProducerPastItsBurst` — the 3rd rapid upload past a
  burst of 2 gets 429.
- `TestPostBundle_refusesAnUploadOverTheRetainedQuota` — with a 1-byte quota, a real
  bundle is refused 413 before ingest.

## The gate is complete (R81–R85)

The RISKS.md producer fix sequence is now fully built: (1) producer auth on the
ingress — NATS `ProducerGrant` for claims (R81/R82), HTTP Basic vs the same grant
table for the bundle blob (R83); (2) per-producer rate-limit + aggregate quota
(R85); (3) GC-by-resolved frees the working set + quota (R84, post-verdict hook +
periodic sweep). The unauthenticated HTTP claim edge is gone.

## Scope held

(4) the TTL-reap for uploaded-but-never-claimed objects + a GLOBAL disk ceiling
(defense-in-depth across producers) remains deferred — it is a different,
cross-producer policy layer, not a per-producer defense, and the periodic sweep
already bounds the never-claimed case at its cadence. TLS for the HTTP/NATS
transports stays a deployment concern. Both noted, not silently dropped.

## Verdict

Full repo green with `-race`. The producer surface is authenticated, GC'd, and
flood-defended end to end. The R78 "producer-auth + bundle GC" gate is closed.

## New clashes opened / resolved

Resolved: per-producer flood-defenses (rate + quota) are enforceable now that the
boundary attributes uploads to a producer, and GC frees the quota on reclamation
(R84↔R85 coupling). Open (deferred, cross-producer): TTL-reap of never-claimed
objects + a global disk ceiling.
</content>
