# Round 37 — sequencing the next #6 slice after the claim-lifecycle UI — CONVERGED 4/4 — 2026-06-10

Trigger: slice C (the producer claim-lifecycle UI: in-flight / verified-lost on
`/board` + live `/fleet`, C1–C4) is done. A grounding scan of the real state
surfaced three candidate next slices; the council was convened (per the standing
autonomous-steering directive) to pick + order them.

Grounded state (verified in code, not assumed):
- The hardened cage IS the production default (`cmd/packets` wires
  `StartCageClaimConsumers` with the real `DockerRunner`); `InProcVerifier`
  survives only for the differential equivalence lock. → "flip default to
  sandboxed" is effectively already done.
- `internal/cage/materialize.go` does `git clone --local --no-hardlinks --
  hostRepo`: the HOST must already hold the producer's commits. NO transport
  exists for a cross-process producer's SHAs to reach the host object store
  (R34: "build-time plumbing, noted").
- `internal/ledger/claim.go` `ConsumeClaims` (post-C3a) treats a verify ERROR as
  TRANSIENT (ack, no verdict, stays in-flight, resubmittable); a clean no-catch
  writes a durable rejection. So a claim for unresolvable/unknown SHAs ERRORS and
  lingers IN-FLIGHT FOREVER.

Panelists: Systems/Economy, Security/Sandboxing, Pragmatic TDD, CI/CD & Delivery.

## Candidates

- **C — permanent-vs-transient verify failure:** distinguish an unresolvable /
  malformed claim (permanent → durable rejection, leaves in-flight) from a cage
  flake (transient → stay in-flight, resubmittable).
- **A — SHA transport:** a mechanism for a producer's commits to reach the host
  object store (authz-gated push/fetch) so Materialize can clone real
  cross-process commits.
- **B — farm governor hardening:** caps/quotas/admission beyond the built
  per-claim deadline + per-producer token bucket + process-wide concurrency sem.

## Per panelist — all four chose C → A → B

- ⚙️ Systems: C is the correctness blocker (the two-scores ledger is incoherent
  if claims vanish into transient-limbo); A is the unblock dependency that makes
  C's permanent case real; B is defensive hardening on already-bounded resources.
- 🛡️ Security: C is a hard DoS fix — an unresolvable-claim flood = unbounded
  in-flight growth at zero cage cost (the cage never even starts). C must precede
  A because A's object-injection failures (malformed pushed objects) MUST land in
  C's permanent-rejection path, not linger. A's own threat model: per-producer
  namespacing, recompute-the-SHA (content-address invariant), no cross-tenant
  read, reject-on-mismatch.
- 🧪 Pragmatic TDD: C is the only candidate testable-now with high confidence and
  no flake — a fake runner + a real inline git repo + an unknown-SHA claim, no
  Docker/network. A drags in a network push endpoint (integration-heavy). Named
  the load-bearing RED test (below).
- 🚀 CI/CD: C is defensive plumbing with ZERO deploy surface (~30 LOC); ship it
  first to prove the cross-process loop is survivable (bad work → durable
  rejection, not silent poisoning). Then A's THINNEST first increment: a
  ProducerGrant-authed push/fetch of a producer ref into a producer-keyed subdir,
  + one fetch line in Materialize — not a full git server.

## Chair adjudication — CONVERGED

Order: **C → A → B.** C is a correctness + DoS hole (not a feature) and the
foundation A's failure modes reuse; it is the smallest, least-flaky,
highest-confidence slice and has no deploy surface. Build C next.

## Design constraint discovered (load-bearing for C)

`ConsumeClaims` is in package `ledger`; it CANNOT import `cage` to inspect a cage
error type (`cage` already imports `ledger` — that would cycle). So the
permanent-vs-transient signal must cross the `ledger.Verifier`
(`func(ClaimRecord) (*CatchRecord, error)`) seam via a **sentinel error defined
in `ledger`** that `cage` wraps:
- `ledger` defines an exported sentinel (e.g. `ErrClaimUnverifiable`) meaning
  "this claim can never verify — reject it durably."
- `cage.CageVerifier` wraps `Materialize`'s unresolvable-revision error with that
  sentinel (and `internal/cage/materialize.go` should return a typed/distinct
  error for "host cannot resolve revision" vs a clone/IO failure).
- `ConsumeClaims`: on verify error, `errors.Is(err, ledger.ErrClaimUnverifiable)`
  → publish a durable `ClaimVerdict{Rejected:true}`; otherwise keep the current
  transient behavior (ack, no marker, stays in-flight).

Load-bearing RED test (TDD): `TestConsumeClaims_unresolvableTargetIsDurablyRejectedNotLeftInFlight`
— a verifier returning `(nil, fmt.Errorf("...: %w", ledger.ErrClaimUnverifiable))`
drives `ClaimsRejected()==1` / `ClaimsInFlight()==0`, while a verifier returning
a plain `(nil, error)` stays `ClaimsInFlight()==1` / `ClaimsRejected()==0`. A
second cage-level test (real inline git repo + fake runner): `CageVerifier` over
an unknown-SHA claim returns an error that `errors.Is(ErrClaimUnverifiable)`.

## Decision

Next slice = **C, permanent-vs-transient verify failure** (the durable rejection
of an unverifiable claim). Then A (SHA transport, thin push/fetch first), then B
(governor hardening). The transport threat model (per-producer namespacing,
SHA re-derivation, no cross-tenant read) is recorded for A.

## Verdict (post-build, 2026-06-10)

Slice C (permanent-vs-transient) BUILT and shipped (commit 92b72cf):
`ledger.ErrClaimUnverifiable` + the ConsumeClaims reject-on-permanent branch;
`cage.ErrUnresolvableRevision` wrapping Materialize's unresolvable-revision
failures; `CageVerifier` maps it to the ledger sentinel. An unresolvable/malformed
claim is now durably rejected (leaves in-flight) instead of lingering forever — the
unbounded-in-flight hole is closed. The Audit caught a real misclassification: a
context cancellation/deadline during `rev-parse` was being branded permanent;
fixed so a `ctx.Err()` surfaces transiently (a valid-but-slow claim, or host
shutdown, stays retryable). Two-scores intact. Full `-race -p 1` gate green.

## New clashes opened

NONE. A (SHA transport) — the next slice — will need its OWN security round on the
object-ingestion threat model (per-producer namespacing, recompute-the-SHA
content-address invariant, no cross-tenant read, reject-on-mismatch) BEFORE it is
built. Convene that round next tick.
