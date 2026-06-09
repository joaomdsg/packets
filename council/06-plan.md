# Roadmap #6 — converged design: the cross-process producer boundary

The council converged on this design across rounds 31–32. It is the PLAN,
not an authorization: building it remains hard-gated on the security trio
and explicit maintainer sign-off (round 28, reaffirmed round 32).

## The trust model (round 32, 5/5)

Claim-submission with a host-side single minter. A separate sandboxed agent
process is a WORKER, not a minter: it publishes unverified CLAIMS; the
trusted host re-runs the mutation oracle and mints. **Only the verifier
mints** — enforced structurally (the producer has no capability to write a
minted event), not by policy. This keeps today's in-process mint path
(`oracle(work) → CatchRecord → ledger.Append`) literally unchanged; only the
provenance of the input work moves across the boundary.

## Authz schema

- New subject status token `claim`:
  `packets.session.<key>.events.<inst>.claim.>`.
- A producer authenticated for (key, inst) may PUBLISH only to its own
  `claim` subtree — nothing else.
- The authoritative `...minted.>` subjects are reserved to the HOST
  identity; producers cannot publish them. Closes subtree-jumping AND
  mint-without-verification in one control.

## Build sequence

- **#6a — listen-mode + authn + authz (one unit, TDD-able now, default
  off).** Add an ADDITIVE listen option to `fabric.Start`; the `DontListen`
  in-process default stays byte-identical and is the regression oracle (the
  #1–#5 suite + the state-equivalence lock green is the gate-to-merge).
  Producer authentication + the authz schema land together (no
  authenticate-now-authorize-later open-mint window). RED tests (real
  loopback listener): forged/unauthenticated-producer rejection;
  claim-subtree confinement (A credentialed for A is rejected publishing to
  B, and rejected publishing any `...minted.*`); authorized claim
  happy-path; negative subject-shapes (wildcards, parent-subject, sibling
  escapes).
- **#6a' — the claim→verify→mint loop.** Host subscribes to `claim`
  subjects; re-runs the SINGLE mutation-oracle implementation (the same code
  path as the in-process cycle — transport/trigger differ, the verifier does
  not) against the claim's content-addressed artifact, verified against the
  TIP at mint time; mints via the existing host Append path on success,
  rejects on failure or non-reproducibility. Determinism is a hard
  requirement (pin revs, mutant seed/ordering, the anchored survivor-set).
  Idempotency over (session, target, oracle-result-hash) defeats
  replay/double-mint. Verification fans out (stateless, parallel); only the
  mint/append serializes — reuse the existing single tip-append lane. Lock
  with an EQUIVALENCE test: same oracle verdict → identical projection
  whether the work was generated in-proc or arrived as a claim.
- **#6b — full-history secret scan (SHIPPED).** Scope clarified to
  SCAN-AND-BLOCK, not history rewrite: rewriting an agent's git history is
  destructive and is not what a boundary gate does — the gate detects and
  refuses to open while a secret exists anywhere in reachable history.
  `settle.ScanHistory` walks `git log -p --all` and reuses the existing
  per-settle rule set + parser (no fork), finding a secret even when a later
  commit removed it from the working tree. Noted limitations (acceptable for
  a gate, follow-ups if needed): merge-commit diffs are skipped (git log
  default); `--all` scans all refs, so against a repo with fetched remotes it
  flags secrets the local tree never had (desirable for a gate, but worth an
  operator note).
- **#6c — HARD-GATED: expose a bound socket to an untrusted producer.**
  Behind the security trio, in order: netns + host egress proxy
  (default-deny, cheapest blast-radius cap) → seccomp/LSM (pin the
  syscall/file surface) → out-of-container permission broker (highest-trust,
  last; inherits §15/§19 scrutiny). The kernel members are NOT pure-Go
  unit-testable — integration only (attempt the forbidden syscall → assert
  EPERM; never assert a config flag). Broker approvals are modeled as fabric
  EVENTS (subscribable, causal — the broker waits on the approve event), per
  UX, so the approval UX is calm, keyboard-native, and renderable on the
  board.

## Carried invariants / riders

- Single-minter: CatchRecords originate from exactly one host-side call
  site; no second mint code path anywhere; the producer subtree has no
  Append capability.
- Confirmed-only hit-rate: claims-in-flight may render as gray "pending"
  (bets placed), but the hit-rate reads ONLY host-minted CatchRecords —
  never two scores on screen.
- Claim rate-limit/quota so verification compute (the new scarce resource an
  adversary could exhaust) can't be starved.
- The in-process default is the regression oracle; no unit test may claim to
  verify the kernel trio.

## Refinement (post-R32, maintainer decision)

The maintainer confirmed the target is running MULTIPLE UNTRUSTED agents. That
sharpens two things the council left implicit:

- Verification executes the agent's code (the oracle compiles + runs its
  tests/mutants). For untrusted agents this MUST run sandboxed — it cannot run
  in the trusted host process. So the verify step is part of #6c (gated), not a
  free in-process call.
- Therefore #6a' is built with the verifier as a SEAM (an injected
  interface/func): the safe scaffold — claim schema, host-side claim consumer,
  and mint-on-verdict reusing the existing single mint path — lands now with a
  stub verifier in tests; the real sandboxed verifier slots into that seam at
  #6c. #6c (the sandbox that runs verification on untrusted code + the agent
  isolation) is now the required critical path, gated on the maintainer's
  runtime choice.

Progress: #6a shipped (listen + authn/authz). #6a' scaffold underway — the
claim schema (ledger.ClaimRecord / PublishClaim / DecodeClaim on the claim
subtree) has landed, with the invariant that a claim mints nothing on its own.

## What the council did NOT decide

Whether and when to authorize the gated boundary, and the target deployment
(does the host control a container runtime + netns + egress proxy + broker,
or is this still single-host prototype where the trio is premature and the
work stops at #6a/#6a'/#6b). Those are the maintainer's calls.
