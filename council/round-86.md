# Round 86 — producer-GC tail: the global disk ceiling (and why the TTL-reap is subsumed) — 2026-06-11

Trigger: continuing the deferred bundle-storage item (4) after the gate closed
(R81–R85). RISKS.md item (4) was "a TTL-reap for uploaded-but-never-claimed
objects + a GLOBAL disk ceiling as defense-in-depth."

## Why the TTL-reap is now subsumed (no code — an honest assessment)

When item (4) was written, GC was unbuilt. It is now: the periodic sweep
(`StartProducerGC`) and the post-verdict hook (R84) reap a producer's WHOLE
namespace whenever `ClaimsInFlight()==0` — which is precisely the
uploaded-but-never-claimed state (the producer has no in-flight claim). So
never-claimed objects are already reclaimed at the sweep cadence; there is no
unbounded leak for a TTL-reap to fix.

A TTL would only ADD a GRACE PERIOD (keep freshly-uploaded objects for N minutes
before reaping), which is the OPPOSITE direction: it would relax the immediate
idle-prune that R39 deliberately accepts (and that the upload→claim self-heal
relies on). Adding a grace window changes accepted GC semantics, so it is a
maintainer design call, not a mechanical defense — explicitly NOT built here.

## The genuinely-additive piece: a global disk ceiling (TDD)

The per-producer quota (R85) bounds ONE flooder; it does not bound MANY producers
collectively filling the store. R86 adds a global ceiling:

- `bundleGlobalCeilingBytes` caps the SUM of retained bytes across all producers.
- Retained accounting moves under one `bundleAcctMu` so each guard's `retained`
  and the global aggregate `bundleGlobalRetained` stay mutually consistent (uploads
  are infrequent — one accounting mutex is ample and dodges any per-guard/global
  lock-ordering hazard). `reserve(n)` now checks BOTH the per-producer quota and
  the ceiling, and returns whether the GLOBAL limit was the binding one.
- The handler maps a per-producer overflow to 413 (this producer's fault) and a
  global overflow to 503 (host at capacity, not this producer's fault).
- GC frees the aggregate too: `resetBundleRetained` (called on prune, R84)
  subtracts the producer's bytes from the global total, so reclamation frees both
  the quota and the ceiling.

RED tests:
- `TestBundleGuard_globalCeilingBoundsTheSumAcrossProducers` — a second producer
  within its own quota is refused (flagged global) when the aggregate would exceed
  the ceiling; freeing the first producer's bytes makes room.
- `TestPostBundle_refusesWhenTheGlobalCeilingIsReached` — the HTTP 503 path.
- The existing per-producer quota/reserve test updated for the `(ok, global)` shape.

## Scope held

The limits stay sensible package-var defaults (128 MiB/producer, 1 GiB global),
tunable in code; exposing them as CLI flags is a trivial follow-on if a deployment
needs it, not built speculatively. Item (4)'s TTL-reap is intentionally NOT built
(subsumed + a semantics change requiring a steer, as above).

## Verdict

Full `internal/app` green with `-race`; build clean. Producer bundle storage is
now bounded per-producer (quota) AND host-wide (ceiling), with GC freeing both.
The producer surface is fully defended; the only remaining (4) sub-item is the
opt-in TTL grace window, which needs a maintainer decision.

## New clashes opened / resolved

Resolved: a global disk ceiling bounds the cross-producer aggregate the
per-producer quota cannot; GC frees it on reclamation. Open (needs a steer, not a
defect): whether to add a TTL grace window before reaping freshly-uploaded
objects — it would relax R39's accepted immediate idle-prune.
</content>
