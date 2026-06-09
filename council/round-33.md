# Round 33 — #6c sandbox design on plain Linux containers — CONVERGED 5/5 — 2026-06-09

Trigger: the maintainer chose the runtime for untrusted agents — PLAIN LINUX
CONTAINERS (not VMs) — so the council convenes on the #6c sandbox design (the
gated piece: isolate each untrusted agent AND safely run verification, which
executes untrusted code). RISKS §15/§19 explicitly asks for a council round to
adopt this boundary. Decision round.

Panelists: a NEW Security/Sandboxing lens added for this round (per the
reconvene template), plus CI/CD, Systems/Economy, Pragmatic TDD, Refactoring.
Game design and UX did not re-poll (UX's broker-approvals-as-fabric-events
rider stands from R31).

## Per panelist

- 🛡️ Security/Sandboxing (CONVERGE, conditional): hardened plain containers
  clear the bar for THIS threat model because the host alone mints from
  host-recomputed evidence, so a container escape short of a kernel 0-day is
  economically inert (worst case: DoS or a junk claim that mints nothing). The
  residual is the shared host kernel — unclosable in-container — so design the
  execution substrate as a SEAM with hardened-container as impl #1 and
  gVisor/microVM a drop-in swap if the threat escalates. NON-NEGOTIABLE
  hardening: user namespaces/rootless, --cap-drop=ALL, no-new-privileges,
  seccomp default-DENY allowlist, netns with no egress, read-only rootfs +
  ephemeral scratch, pids/mem/cpu limits, NO docker.sock / host mounts /
  --privileged, ephemeral one-shot container per verification. A single
  enforced launch path + a fail-closed config-conformance test (a hardened
  container misconfigured is a plain container).
- 🚀 CI/CD (CONVERGE, two hard conditions): #6c is a build farm — ephemeral
  one-shot workers, fan-out verify / fan-in serial mint on the EXISTING single
  append lane. (1) caps + per-agent quotas + admission control ship in the
  SAME increment as the farm (an ungoverned farm fed by untrusted agents is
  self-DoS); (2) NO shared writable cache across trust domains —
  content-addressed read-only caches + per-claim ephemeral only, deps via the
  egress allowlist (a shared writable cache reopens cross-tenant poisoning).
- ⚙️ Systems/Economy (CONVERGE, one hard ask): the verdict must be
  HOST-DERIVED, never a PASS the box asserts — the box emits work + trace, the
  host re-derives the economic fact. The oracle is host-owned, mounted
  read-only, content-pinned; the agent supplies only the unit under test, never
  the judge. Determinism (pin toolchain, seed, freeze clock, no network) so
  flakiness can't be re-rolled; content-addressed claims dedup re-verification;
  charge-on-submission so claim-spam spends the spammer's own quota. The
  content-addressed dedup store must be HOST-WRITE-ONLY (poisoning it = mint
  many counterfeits).
- 🧪 Pragmatic TDD (CONVERGE, one hard line): the seam + claim→mint wiring is
  already unit-proven with a stub. Enforcement is kernel-enforced, so it is
  testable ONLY by attempting the forbidden thing in a REAL container and
  observing the kernel deny it (forbidden syscall → EPERM; real outbound
  connect → blocked; real fork-bomb → capped; exit-0-with-garbage-oracle-output
  → no mint). BANNED lie-green: mocking the sandbox in enforcement tests,
  asserting a config/profile string as proof, trusting a self-reported exit
  code. Determinism tested by repetition-under-perturbation; flake ≡ reject.
  The merge gate is the real-host escape-attempt + determinism suite, NOT
  unit-green — keep #6c hard-gated until that suite is wired in.
- 🔧 Refactoring (CONVERGE, one binding condition): #6c is additive — the real
  sandboxed verifier implements the EXISTING ledger.Verifier seam; the
  in-process default stays byte-identical (the regression oracle). ONE oracle
  artifact, two invocation sites (in-proc + in-container) — never a second
  oracle (the drift/counterfeit vector). Extend the equivalence lock with a
  DIFFERENTIAL test: in-proc verdict ≡ sandboxed verdict → identical
  projection, GREEN on the full corpus BEFORE any untrusted claim reaches the
  sandbox.

## Clashes touched

- #6c-sandbox-on-plain-containers — RESOLVED 5/5: hardened plain containers +
  the defense-in-depth trio are an acceptable, deliverable, behavior-preserving
  boundary for the stated threat model, behind an isolation-substrate seam that
  keeps a VM/gVisor upgrade a config swap.

## Verdicts updated

None flip. The R32 trust model (only the host mints) is REAFFIRMED and
SHARPENED: the verdict itself is host-derived, not box-reported.

## New clashes opened

NONE at target level. The convergence carries a set of compatible, binding
design conditions (below) — not open clashes.

## Decisions — the converged #6c design

The full design is recorded in [`06-plan.md`](06-plan.md). Binding conditions
(all 5/5):

1. Isolation as a SEAM — hardened plain container is impl #1; gVisor/microVM a
   drop-in swap. The non-negotiable hardening set is the launch contract,
   enforced by a single launch path + a fail-closed config-conformance test.
2. Separate ephemeral one-shot verification container per claim, host-
   parameterized — never the agent's own box.
3. The verdict is HOST-DERIVED from host-checkable evidence; the box never
   asserts an authoritative PASS. The oracle is host-owned, read-only,
   content-pinned; one oracle artifact, two invocation sites (no second
   oracle).
4. Build order (trio, blast-radius-first): (1) netns + host egress proxy
   (default-deny) → (2) seccomp/LSM profile → (3) out-of-container permission
   broker (approvals as fabric events).
5. The farm ships WITH its governor: caps + per-agent quotas + admission
   control; fan-out verify, fan-in serial mint on the existing lane; NO shared
   writable cache (host-write-only content-addressed + per-claim ephemeral).
6. Determinism + content-addressed dedup; flake ≡ reject.
7. The DIFFERENTIAL equivalence lock (in-proc ≡ sandboxed verdict) green before
   any untrusted claim is verified in the sandbox.
8. Enforcement proven ONLY by real-container attack fixtures in a real-host CI
   gate; never config-string assertions or self-reported exit codes. #6c stays
   hard-gated until that suite is green.

CONVERGED on the #6c design. This is the trace-forward the round was charged
to produce; building it is the maintainer's authorization, slice by slice,
with the real-host CI gate as the definition of done.
