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
- **#6c — HARD-GATED: the sandbox (runtime = plain Linux containers).**
  Designed and converged 5/5 in council round 33. The remaining piece:
  isolate each untrusted agent AND safely RUN VERIFICATION (the oracle
  compiling + running untrusted code). Design contract:
  - **Isolation as a SEAM.** Hardened plain container is impl #1; gVisor /
    microVM is a drop-in swap if the threat escalates — kept an interface, not
    a rewrite.
  - **Non-negotiable hardening (the launch contract):** user
    namespaces/rootless, --cap-drop=ALL, no-new-privileges, seccomp
    default-DENY allowlist, netns with no egress, read-only rootfs + ephemeral
    scratch, pids/mem/cpu limits, NO docker.sock / host mounts / --privileged,
    ephemeral one-shot container per verification. A single enforced launch
    path + a FAIL-CLOSED config-conformance test (a misconfigured hardened
    container is a plain container).
  - **Verdict is HOST-DERIVED.** A separate ephemeral one-shot verification
    container per claim, host-parameterized — never the agent's own box. The
    oracle is host-owned, mounted read-only, content-pinned; the agent supplies
    only the unit under test, never the judge. The box emits work + trace; the
    host re-derives the economic fact and mints. The box NEVER asserts an
    authoritative PASS. ONE oracle artifact, two invocation sites (in-proc +
    in-container) — never a second oracle.
  - **Cage build status (SHIPPED, slices 1-4):** the launch contract +
    fail-closed conform gate; and break-out proofs — egress (--network=none),
    pids cap, and a seccomp syscall-deny profile — each real-container,
    differential, mutation-verified. read-only/caps deferred (confounded by the
    non-root user; need a root-differential). The kernel members are NOT
    pure-Go unit-testable — integration only (attempt the forbidden op → assert
    denial; never a config flag).
  - **Egress proxy — DROPPED (round 34):** the verification flow keeps the cage
    at --network=none and feeds deps from a host-provided READ-ONLY module
    cache, so no in-cage egress (and no egress proxy) is needed. The egress
    allowlist boundary it would have held moves to the trusted-side prefetcher.
  - **Out-of-container permission broker:** approvals as fabric events, per UX
    — still part of #6c, after the verification flow.
  - **Verification flow (converged round 34):** see the dedicated section below.
  - **Farm ships WITH its governor:** ephemeral one-shot workers, fan-out
    verify / fan-in serial mint on the EXISTING single append lane; caps +
    per-agent quotas + admission control in the SAME increment; NO shared
    writable cache across trust domains (the module cache is host-write-only,
    read-only-mounted, content-addressed; per-claim worktree ephemeral).
  - **Determinism:** pin toolchain/seed/clock, no network; content-addressed
    claims dedup re-verification; charge-on-submission; **flake ≡ reject**.
  - **Equivalence lock extended:** a DIFFERENTIAL test (in-proc verdict ≡
    sandboxed verdict → identical projection) green on the full corpus BEFORE
    any untrusted claim is verified in the sandbox.
  - **Testability / definition of done:** enforcement is proven ONLY by
    real-container attack fixtures in a real-host CI gate (forbidden syscall →
    EPERM; real outbound connect → blocked; real fork-bomb → capped;
    exit-0-with-garbage-output → no mint). BANNED lie-green: mocking the
    sandbox in enforcement tests, asserting a config/profile string, trusting a
    self-reported exit code. #6c stays hard-gated until that suite is wired and
    green.

## #6c verification flow (converged round 34)

How the host runs the oracle on an agent's work inside the cage:

- **Transport — SHA-pinned shared object store.** The claim carries immutable
  commit SHAs; the host checks those exact SHAs out of a host-controlled object
  store and materializes a byte-identical repoDir. A git SHA is an immutable
  content-address → no TOCTOU, verify-A/mint-B impossible, each claim pins its
  own base+fix. Patch-apply rejected (a divergent second materialization). How
  the agent's commits reach the store (a push endpoint gated by #6a authz) is
  build-time plumbing.
- **Deps offline — read-only, content-addressed module cache.** A curated
  GOMODCACHE populated by a TRUSTED-SIDE prefetch (`go mod download` +
  go.sum/policy verify = an admission gate, the only network in the system),
  mounted read-only into the cage with `GOPROXY=off`. Cage stays
  --network=none. Cache-miss is a hard fail, never a network fallback.
- **Base image — fixed, digest-pinned, host-built, minimal, GOTOOLCHAIN=local**,
  with the SAME `packets` binary shipped in (one oracle, two sites — never a
  forked copy).
- **Verdict evidence — host re-derives, box asserts nothing.** The cage emits a
  mutation transcript (baseline-green + per-mutant killed/survived + output
  digests + mutant→line map + determinism markers); the host recomputes the
  survivor-set delta, the anchor binding (the mutant maps to the claimed line
  in the rev the host authorized), the class, and the mint decision. An
  incomplete or non-deterministic transcript → reject (never default-PASS).
- **Determinism / idempotency.** work-id = H(base-image-digest ‖ revs ‖ work ‖
  toolchain ‖ seed ‖ clock); same work-id ⇒ same transcript ⇒ ≤1 catch per
  anchor (replay is a cache hit). flake ≡ reject, on the host's observation.

Build sequence (suite green at each step; NO untrusted claim accepted until the
equivalence lock + lie-green trap are green):

1. `packets verify-catch` subcommand wrapping RunCatchCycle over an
   arg-supplied repoDir — pure refactor, characterization-locked to in-proc.
2. Host repoDir materialization (checkout the claim's base+fix SHAs) +
   trusted-side prefetch into the read-only GOMODCACHE.
3. Real cage-exec Verifier behind the seam (NOT default): same `packets` binary
   in the pinned image, RO worktree + RO cache, GOPROXY=off, --network=none.
4. Host re-derives the verdict from the transcript (never the exit code);
   incomplete/non-deterministic → reject; the lie-green-trap test.
5. The differential equivalence lock (in-proc ≡ sandboxed → identical
   projection) green on a real corpus — THEN flip the default to sandboxed.
6. The farm governor (caps/quotas/admission, verify-at-tip).

New trusted-side surface flagged for its own rigor: the PREFETCHER (network +
write to the curated cache) is the highest-value target — pin it, sum-verify
everything it writes, treat its fetch allowlist as the security boundary the
dropped egress proxy would have been.

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
