# Round 34 — #6c verification-flow mechanics on plain containers — CONVERGED, transport clash reconciled — 2026-06-09

Trigger: the #6c cage is built + proven (no-network, pids cap, seccomp,
read-only, cap-drop, non-root, one-shot). The remaining piece — running the
oracle on an agent's work inside the cage — needs concrete mechanics the
maintainer asked the council to settle: code transport, offline deps, base
image, and the verdict-evidence format. The crux: the oracle runs `go test`
mutation-testing over a git repo, and `go test` normally fetches deps over the
network, colliding with the cage's --network=none.

Panelists: Security/Sandboxing, Systems/Economy, Pragmatic TDD, Refactoring,
CI/CD & Delivery.

## Per panelist

- 🛡️ Security (CONVERGE): keep --network=none; host provides everything,
  including a READ-ONLY module cache — don't trade a proven binary invariant
  (no socket exists) for a configured egress allowlist. Base image
  digest-pinned, host-built, minimal, GOTOOLCHAIN=local, oracle in the image.
  Transport: ranked patch > shared-repo > egress; flagged that whichever
  transport, anything outside the work paths (go.mod/toolchain/harness) is the
  "move the judge" vector and must be rejected host-side.
- ⚙️ Systems (CONVERGE): the cage emits a mutation TRANSCRIPT, never a PASS;
  the host RE-DERIVES the survivor-set delta + anchor binding + class + mint
  decision. work-id = H(base-image-digest ‖ revs ‖ work ‖ toolchain ‖ seed ‖
  clock) is the determinism/dedup key; catch-id stays (revs,anchor,reason).
  Read-only host-pinned dep cache (writable shared = poisoning = mint-many).
  Wanted content-addressing so verify-A/mint-B is impossible by construction.
- 🧪 Pragmatic TDD (CONVERGE): pick the choice with the FEWEST nuisance axes
  (every axis is a flake source). Determinism is a testable contract
  (verify N× under perturbation → identical survivor-set); flake ≡ reject, on
  the HOST's observation. ONE oracle, two sites; the differential equivalence
  lock (in-proc ≡ sandboxed on a real fixture → identical projection) is the
  gate, green before any untrusted claim. Lie-green trap: a cage that exits 0
  but emits evidence proving the claim false MUST be rejected — the host
  re-derives from the survivor-set, never the exit code.
- 🔧 Refactoring (CONVERGE): the cage run must be a behavior-preserving
  RELOCATION of the in-proc RunCatchCycle — same repoDir bytes + revs + testCmd
  → same verdict. So the host MATERIALIZES a byte-identical repoDir and the
  SAME `packets` binary runs over it offline; a patch-apply is a second
  materialization path that can diverge (whitespace/EOL/mode/metadata → tipRev
  differs) — reject it. Ship the same binary into the image, never a forked
  oracle; the equivalence lock is the tripwire.
- 🚀 CI/CD (CONVERGE): this is the standard offline build farm — host checks
  out commits on the trusted side, bind-mounts them read-only; a curated
  read-only content-addressed GOMODCACHE (GOPROXY=off) populated by a
  TRUSTED-SIDE prefetch (go mod download + go.sum/policy verify = an admission
  gate, the only network in the system). Cage stays --network=none → the
  egress-proxy slice dies. Ephemeral workers, fan-out verify / fan-in serial
  mint, caps+quotas+admission together, verify-at-tip, no shared writable cache.

## Clashes touched

- CODE TRANSPORT — the one clash. patch-in-claim (Security, Systems: content-
  addressed, no TOCTOU) vs host-checks-out-the-agent's-commits (TDD,
  Refactoring, CI/CD: byte-identical relocation; patch-apply forks
  materialization). RECONCILED by the chair (below).
- DEPS / EGRESS — RESOLVED 5/5: read-only host-pinned module cache + trusted-
  side prefetch; cage stays --network=none; the #6c egress-proxy slice is
  DROPPED. The egress allowlist "boundary" moves to the trusted-side prefetcher.
- VERDICT EVIDENCE — RESOLVED: host re-derives from an emitted transcript;
  box asserts nothing; incomplete/non-deterministic → reject.
- ONE ORACLE / EQUIVALENCE LOCK — RESOLVED: same `packets` binary, two sites;
  differential lock green before any untrusted claim.

## Chair adjudication of the transport clash

The two camps reconcile: the claim carries IMMUTABLE COMMIT SHAs, and the host
checks those exact SHAs out of a host-controlled object store. A git commit SHA
IS an immutable content-address, which delivers BOTH sides' requirements:
- Security/Systems get content-addressing + no TOCTOU — a SHA can't be
  repointed, so verify-A/mint-B is impossible, and each claim pins its OWN
  base+fix SHAs so a shared repo can't let one agent move another's base
  (missing/unfetchable SHA → reject).
- TDD/Refactoring/CI-CD get the byte-identical relocation — the host checks out
  the real commits (no patch reconstruction), and the SAME RunCatchCycle runs
  over the same repoDir bytes the in-proc call would see.
Patch-apply is REJECTED (a divergent second materialization path). How the
agent's commits reach the host's object store (a push endpoint / shared repo,
gated by the #6a authz) is build-time plumbing, noted.

## Verdicts updated

None flip. R31-33 reaffirmed and sharpened: the verdict, like the mint, is
host-derived; the host controls the full denominator (code via SHA, deps via
the read-only pinned cache, toolchain via the pinned image, seed/clock pinned)
and folds it into one content address.

## New clashes opened

NONE at target level. New trusted-side surface flagged for its own rigor: the
PREFETCHER (network + write to the curated cache = the highest-value target;
pin it, sum-verify everything it writes, treat its fetch allowlist as the
security boundary the dead egress-proxy would have been).

## Decisions — the converged verification flow

Recorded in [`06-plan.md`](06-plan.md). Build sequence (suite green at each
step; no untrusted claim accepted until the equivalence lock + lie-green trap
are green):

1. Extract a `packets verify-catch` subcommand wrapping RunCatchCycle over an
   arg-supplied repoDir — pure refactor, characterization-locked to the in-proc
   verdict.
2. Host repoDir materialization (checkout the claim's base+fix SHAs from the
   object store) + trusted-side prefetch (go mod download + go.sum/policy verify
   into the curated read-only GOMODCACHE).
3. Real cage-exec Verifier behind the seam (NOT default): the same `packets`
   binary in the digest-pinned image, RO worktree + RO GOMODCACHE,
   GOPROXY=off, --network=none.
4. Host re-derives the verdict from the emitted transcript (never the exit
   code); incomplete/non-deterministic → reject; the lie-green-trap test.
5. The differential equivalence lock (in-proc ≡ sandboxed → identical
   projection) green on a real corpus — THEN flip the default to sandboxed.
6. The farm governor: caps + per-agent quotas + admission control,
   verify-at-tip, fan-out verify / fan-in serial mint on the existing lane.

CONVERGED on the verification-flow design. The #6c egress-proxy slice is
dropped; the security boundary it would have held moves to the trusted-side
prefetcher. Building proceeds slice by slice with the equivalence lock as the
gate before any untrusted code is judged.
