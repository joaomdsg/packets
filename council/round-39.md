# Round 39 — scoping slice B (governor hardening) after SHA transport — CONVERGED, auth-before-flood-defense — 2026-06-10

Trigger: slice A (producer SHA transport: POST /bundle → internal/ingest →
refs/producers/<key>/* → cage) is end-to-end. Slice A introduced a NEW ungoverned
surface — POST /bundle has only a per-call 32 MiB cap, is unthrottled, has no
per-producer aggregate quota, and ingested objects are never GC'd. This round
scopes slice B.

Grounded state: VERIFY path is governed (per-claim 120s deadline, per-producer
token bucket Burst 12/0.1s, process-wide concurrency sem ≤4). The INGEST/storage
surface is not. CRITICAL grounding: the live HTTP producer surface (POST /claim,
POST /bundle) is only SESSION-KEY-GATED, NOT authenticated — fabric's
ProducerGrant (User/Pass) is the NATS path and is NOT wired into the live server
(it uses an embedded fabric).

Panelists: Security/Sandboxing (lead), Systems/Economy, Pragmatic TDD, CI/CD.

## Per panelist

- 🛡️ Security: priority-1 is the unbounded-/bundle disk-fill DoS — wants a
  /bundle per-producer rate limit (B1) + a per-producer aggregate storage quota
  (B2); global disk ceiling (B3) and GC/retention (B4) deferrable; verify-path
  governor is sufficient (a burst is bounded by the concurrency cap — no gap).
- ⚙️ Systems: the unbounded store is OPS/SECURITY, NOT economy — objects are
  off-ledger, so the two-scores ledger + single-minter invariants hold regardless
  of store size. The economy-relevant piece is only the GC RETENTION RULE: prune
  refs/producers/<key>/* for targets already RESOLVED (minted OR rejected), keep
  exactly those backing an in-flight claim (ClaimsInFlight is the source of
  truth). A resolved target's objects are dead evidence; a pending one's must
  survive.
- 🧪 Pragmatic TDD: a BYTE-measured quota is BRITTLE — git compression/gc/version
  nondeterminism makes the limit and the test data-dependent. GC-by-resolved is
  the cleanest TDD entry: a PURE selectPrunableRefs(allRefs, inFlightTargets) +
  a thin idempotent `git update-ref -d` step; zero flake. Rate-limit reuses the
  proven tokenBucket but needs new HTTP-handler plumbing. (If a total bound is
  ever needed, count bytes-ACCEPTED at upload — known + deterministic — never
  git's on-disk size.)
- 🚀 CI/CD: per-producer storage quota is the top OPS priority (a buggy/hostile
  producer fills disk fastest); background GC over inline; "ship quota now, defer
  GC" — but acknowledged quota-without-GC is harsh on a legit heavy producer.

## Chair adjudication — CONVERGED

1. Slice B is OPS/SECURITY hardening, not economy (Systems). The verify-path
   governor is SUFFICIENT (Security) — no work there.
2. THE KEY CATCH: the flood-defenses (rate-limit B1, quota B2) PRESUPPOSE a
   producer-AUTH boundary the live HTTP surface does NOT have — /claim and
   /bundle are session-key-gated, not authenticated. Governing "a producer"
   against a malicious flood is premature when anyone holding a registered
   session key IS "the producer". Building per-producer flood limits now would
   harden a threat model whose identity layer is absent. So flood-defense is
   GATED ON producer auth (its own future slice/council).
3. AVOID the brittle byte-quota (TDD). 
4. BUILD NOW the one B item valuable INDEPENDENT of the auth/threat debate:
   GC-by-resolved. It is economy-safe (Systems' exact retention rule), TDD-clean
   (TDD's pure-selection sketch), good hygiene that bounds the live working set
   to in-flight claims, and correct regardless of whether a producer is
   authenticated. The unbounded-storage DoS is LOGGED in RISKS.md and deferred
   with its dependency (auth) named.

## Decision — next build (GC-by-resolved)

A pure selection + thin git I/O, retention rule = keep in-flight, prune resolved:
- PURE: `selectPrunableProducerRefs(producerRefs []ref{name,target-identity},
  inFlight map[identity]bool, minted map[identity]bool) []refName` — a ref is
  prunable iff its claim identity is NOT in-flight AND is resolved (minted or
  has a rejection verdict). NEVER prune an identity still in flight (its objects
  back a pending verify). Pure over data → deterministic unit test.
- THIN I/O: delete the selected refs via `git update-ref -d` (then optionally
  `git gc --prune` later — defer the actual object reclamation; deleting the ref
  is the economy-safe step, gc is ops).
- Trigger: a host-side maintenance call (background tick or post-verdict hook) —
  wiring, decided when built. The PURE rule + ref-delete is the load-bearing,
  TDD-able slice now.
- Load-bearing RED: given producer refs for identities {A in-flight, B minted,
  C rejected}, selectPrunable returns {B, C} and NEVER A; and the ref-delete
  removes B,C from the store while A survives (the pending claim still verifies).

DEFERRED (with dependency named): /bundle rate-limit + per-producer quota
(GATED ON producer auth on the HTTP surface), global disk ceiling, TTL-reap of
uploaded-but-never-claimed objects, durable-across-restart accounting.

## New clashes opened / resolved

- No formal clash. Recorded a SEQUENCING decision: producer-AUTH on the live
  HTTP surface precedes per-producer flood governance (rate-limit/quota). That
  auth slice will want its own design round (where does auth live — extend the
  session-key gate to credentials, or wire the NATS ProducerGrant path into the
  live server?).
