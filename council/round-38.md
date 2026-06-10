# Round 38 — slice A (SHA transport): the object-ingestion threat model — CONVERGED, host-pull rejected on SSRF — 2026-06-10

Trigger: round 37 sequenced the next slice as A (SHA transport) but required its
OWN security round on the object-ingestion threat model before building. Today
`internal/cage/materialize.go` does `git clone --local -- hostRepo`: the host
must already HOLD a producer's commits, and nothing transports them. This round
settles HOW a cross-process producer's commits reach the host, the threat model,
and the thinnest TDD-able first increment.

Panelists: Security/Sandboxing (lead), Systems/Economy, Pragmatic TDD, CI/CD &
Delivery. (Autonomous council per the standing steering directive.)

## The transport fork (three options weighed)

- (i) PRODUCER-PUSH — a git-over-HTTP/SSH receive endpoint on the host.
- (ii) HOST-PULL — the claim carries (url, ref); the host `git fetch`es it at
  verify time.
- (iii) BUNDLE-OVER-CHANNEL — the producer ships a `git bundle` over the
  existing authenticated channel; the host unbundles + validates offline.

## Per panelist

- 🛡️ Security (lead): MUST enforce — recompute every ingested object's hash
  (git does this natively on unbundle/index-pack; `git fsck --strict`
  post-ingest), per-producer ref NAMESPACING (`refs/producers/<id>/*`, never a
  shared mutable ref another producer's claim resolves against — this breaks
  "move the judge"), no cross-tenant READ, host refs immutable to producers, and
  resource caps (per-push size, per-producer aggregate quota, object-count
  ceiling, ingest time limit) against pack bombs. For storage: one shared store
  with per-producer ref namespacing is acceptable for v1 (per-producer object
  stores are a defense-in-depth upgrade).
- ⚙️ Systems: the content-address invariant is the economy's spine — a claim
  names SHAs; never trust a producer's "this is commit X" label, only the bytes
  that hash to X. A SHARED store is ECONOMY-safe: catch identity is
  {BeforeRev,FixRev,Path,Line} and Append's dedup gate means two producers
  converging on the same revs+anchor is one catch, not a double-mint; producer
  provenance is stamped but NOT part of identity, and sessions isolate who mints.
  So per-producer isolation is a SECURITY concern, not an economy-correctness
  one. Ingestion must stay ORTHOGONAL to the claim/minted subtrees (a separate
  step/record), so the two-scores ledger and single-minter invariants are
  untouched.
- 🧪 Pragmatic TDD: decompose A into (a) host-side INGEST+VALIDATE (unbundle into
  a producer-namespaced area, recompute SHAs, enforce namespace + caps, reject
  mismatches) and (b) the WIRE endpoint. (a) is the load-bearing slice and is
  cleanly OFFLINE-testable with real `git bundle create` + `git index-pack`/
  `fsck` over temp dirs (the existing cage harness pattern) — no network, no
  Docker, deterministic. (b) is wiring (build/manual) behind a thin
  `ObjectIngestor` seam. Load-bearing RED: ingest rejects objects outside the
  producer namespace / on SHA mismatch / over a byte cap.
- 🚀 CI/CD: pushed HOST-PULL (ii) as the smallest deployable surface (one `git
  fetch`, no new daemon, fetch failure maps to the permanent/transient
  distinction). Flagged producer-push (i) as a heavy git-server surface and
  bundle (iii) as moderate.

## Chair adjudication — CONVERGED on (iii) bundle-over-channel

HOST-PULL (ii) is REJECTED despite its ops simplicity: it makes the TRUSTED host
issue an outbound `git fetch` to a PRODUCER-CONTROLLED url at verify time — a
textbook SSRF (cloud-metadata endpoints, internal services, internal git hosts)
and a reintroduction of host-side egress to arbitrary destinations, the exact
surface #6c's design fought to eliminate (R34: "the only network in the system is
the trusted-side prefetcher"; the cage runs `--network=none`). CI/CD's
"smallest attack surface" claim missed the SSRF/egress vector.

PRODUCER-PUSH (i) is rejected for v1 as too large a protocol surface (pack
negotiation, receive-pack, a running daemon).

BUNDLE-OVER-CHANNEL (iii) wins: the producer SENDS a `git bundle` over the
existing ProducerGrant-authenticated channel (NOT inside the small claim payload
— a bundle is large; a separate authenticated upload, BEFORE the claim), and the
host ingests + validates it OFFLINE. No host egress, no SSRF, no new git daemon.
A malformed / oversized / out-of-namespace / SHA-mismatched bundle is a PERMANENT
failure that reuses the `ledger.ErrClaimUnverifiable` machinery built last slice.
It also IS Pragmatic TDD's load-bearing offline-testable slice.

## Decisions — slice A design (converged)

- TRANSPORT: bundle-over-authenticated-channel. Producer uploads a `git bundle`
  of its commits (authed by ProducerGrant) as a SEPARATE step before the claim;
  the claim still carries only SHAs (unchanged, stays small/64KiB-capped).
- STORAGE: one shared host object store, per-producer ref namespacing
  `refs/producers/<producerID>/*`. Host branches never writable by a producer.
- THREAT MODEL (MUST enforce): unbundle into the producer namespace ONLY; git
  recomputes object hashes on index-pack (+ `fsck --strict`); reject objects/refs
  outside the namespace; per-upload byte cap + per-producer aggregate quota +
  object-count ceiling + ingest time limit; no cross-tenant read; reject (durably,
  permanent) on any violation.
- INGESTION IS ORTHOGONAL to the claim/minted subtrees — the two-scores ledger
  and single-minter invariants are untouched.

## Thinnest first build increment (next tick)

A new `internal/ingest` package: `IngestProducerObjects(ctx, store, producerID,
bundle []byte, caps) error` — verify the bundle (`git bundle verify`/
`index-pack`), unbundle ONLY into `refs/producers/<producerID>/*`, enforce the
byte/object caps, return typed errors (`ErrOutOfNamespace`, `ErrBundleInvalid`,
`ErrCapExceeded`) that the caller maps to a permanent reject. Offline TDD with
real `git bundle` + temp dirs (no network, no Docker). Load-bearing RED:
`TestIngestProducerObjects_rejectsObjectsOutsideTheProducerNamespace` +
`_rejectsAnOversizedBundle` + `_acceptsAValidBundleIntoTheNamespace`. The wire
upload (how the bundle bytes arrive, ProducerGrant-gated) is wiring, deferred
behind an `ObjectIngestor` seam.

## New clashes opened / resolved

- **Clash K — SHA-transport mechanism: RESOLVED (R38).** bundle-over-channel,
  shared store + per-producer ref namespacing. Host-pull rejected on SSRF/egress;
  push-daemon rejected as too heavy for v1. Per-producer object-store isolation
  noted as a defense-in-depth upgrade if/when needed.
