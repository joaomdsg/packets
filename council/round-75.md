# Round 75 — the AGENT CONTAINER thread opens (maintainer steer) — 2026-06-11

Trigger: maintainer lifted the gate — "move the goal post to the container
orchestration for the harness instances." The live harness runs as a HOST
subprocess today (`harness.RunProcess`); the GOAL's explicit north-star is the
agent running IN A CONTAINER. Full council (Security, CI/CD, Refactoring, Systems,
TDD) convened, grounded in `internal/sandbox` (the verification cage's
DockerRunner), `internal/harness` (Supervisor/RunProcess), `internal/app` (the
runHarness seam + runLiveOrder).

## Ground truth

- `sandbox.DockerRunner` is the VERIFICATION cage: `conform` ENFORCES
  `--network=none`, `--cap-drop=ALL`, seccomp, `--read-only`, non-root uid,
  pids/mem caps — for UNTRUSTED oracle code. The agent needs the OPPOSITE profile
  (egress to the API, a WRITABLE repo, baked toolchains, the API key) → it is a
  SEPARATE runner, not a tweak of the cage.
- `harness.Supervisor.Run(ctx, io.Reader)` reads the stream-json from any reader →
  it is the I/O boundary, UNCHANGED by where the agent runs. `RunProcess` only
  differs in HOW it builds the reader (`exec.Cmd` + StdoutPipe).

## Convergence (5/5)

- SECURITY: the agent container is a TRUST/ISOLATION boundary (trusted Claude Code
  harness in a box), NOT the cage's CONTAINMENT boundary (untrusted oracle in
  lockdown) — different threat model, opposite config. First-slice-safe on a TRUSTED
  LOCAL repo: full egress to the Anthropic API, API key via env, but STILL hardened —
  `--cap-drop=ALL`, seccomp, non-root, `--security-opt=no-new-privileges`, pids/mem
  caps, `--read-only` rootfs + a WRITABLE bind of ONLY the repo workdir, NO
  docker.sock. INVARIANT: the container can edit only the user's own repo; the HOST
  re-derives every verdict (the cage oracle certifies the agent's revision — the
  economy firewall is unchanged). DEFER: multi-tenant isolation, egress allowlist
  (full egress OK for trusted-repo slice 1), secret rotation.
- CI/CD: image = a base + node/go/python + git + the `claude` CLI (the GOAL's "bakes
  in node/go/python"), one-shot `docker run --rm -v <repo>:/work:rw`. PUSH-BEFORE-
  TEARDOWN: a BIND mount (not a copy) → the agent's commits land directly in the host
  repo, so teardown loses nothing (durable-remote push is a separate later concern —
  no remote in the local prototype). TEST via a FAKE-CLAUDE IMAGE (entrypoint edits
  the mounted repo + emits stream-json) through the real runner → end-to-end in CI,
  no API key (mirrors the host-subprocess fake-claude integration test).
- REFACTORING: the agent runner is SEPARATE from DockerRunner (conform's flags are
  cage-fixed). Clean seam: `RunProcess` → `RunContainer` (SAME signature
  `func(ctx,repoDir,prompt,onActivity)([]Turn,error)`) builds a `docker run` argv
  instead of a `claude` argv and pipes ITS stdout to `Supervisor.Run` (UNCHANGED);
  the `runHarness` seam swaps to it, so `runLiveOrder` is UNCHANGED. Extract the
  shared docker reap/orphan/name primitive rather than duplicate DockerRunner's.
- SYSTEMS: the FIREWALL is UNCHANGED — the container is a transparent PRODUCER of the
  same git revision; the host still settles → the cage-oracle re-derives → the
  single-minter mints. No new degenerate strategy (the catch anchor is Lead-specified
  R70; the agent can't reach the host oracle/ledger — only the bind-mounted repo).
  The agent's revision is UNVERIFIED until the oracle certifies it (already how
  runLiveOrder works via resolveCycle).
- TDD: a PURE docker-argv builder (encodes the security profile as data, unit-tested
  like ClaudeArgs) + a fake-claude-IMAGE integration test (real `docker run`, no API
  key); test-theater = mocking docker. `runLiveOrder`/`Supervisor` untouched.

## Clashes resolved

- "Both are containers" (Security/CI-CD): the cage = containment (untrusted, locked
  down, host re-derives); the agent box = isolation (trusted harness, egress+write).
  Don't share `conform`'s profile — separate hardened profiles over a shared
  docker-exec primitive.
- repo mount model (Systems "scratch" vs CI/CD "bind the host repo"): for slice 1,
  BIND the host `cfg.RepoDir` writable at `/work` (parity with RunProcess, which runs
  claude in cfg.RepoDir today; no loss, no copy). A disposable-scratch + diff-back
  model is a later isolation refinement.

## Slice plan (AGENT-CONTAINER thread; tdd-rygba; gate green; docs fresh)

- SLICE 5a-i (NEXT — BUILD): the PURE docker-argv builder
  `containerArgs(image, repoDir, prompt string) []string` (or a Spec) — encodes the
  agent security profile (egress-allowed + cap-drop/seccomp/non-root/no-new-privileges/
  pids+mem caps + read-only rootfs + writable `<repo>:/work` bind + workdir /work +
  API-key env + the image running `claude` with ClaudeArgs(prompt)). Unit-tested
  data→data (pins each hardening flag + the writable-repo bind + egress-NOT-disabled).
  No Docker needed.
- SLICE 5a-ii: `harness.RunContainer(ctx,repoDir,prompt,onActivity)([]Turn,error)` —
  wiring that runs `docker run <containerArgs>` and pipes stdout to Supervisor.Run;
  proven by a FAKE-CLAUDE-IMAGE integration test (no API key), mirroring
  runprocess_integration_test.go. Extract the shared docker reap/name primitive.
- SLICE 5b: WIRE the runHarness seam to select RunContainer (a LiveConfig field /
  `-container` flag); runLiveOrder UNCHANGED. + the real agent image (Dockerfile
  baking node/go/python + claude); a real run is manual/API-key-gated.
- SLICE 5c+ (deferred): egress allowlist, multi-tenant isolation, push-to-durable-
  storage, the internal package mirror.

## Build record — slice 5a-i SHIPPED (the pure agent-container argv builder)

`internal/harness/container.go`: `ContainerSpec{Image,RepoDir,Prompt,SeccompPath,
User,EnvPassthrough,PidsLimit,Memory}` + pure `ContainerArgs(spec) []string` — the
hardened-but-egress-allowed `docker run` argv. Hardening PINNED by tests: --cap-drop=ALL,
--security-opt=no-new-privileges, seccomp=<path>, --read-only + --tmpfs=/tmp, pids/memory
caps, --user=<host uid:gid> (so repo writes are host-owned). Egress ALLOWED (test asserts
NO --network=none — the discriminator vs the cage). The repo is the ONLY writable bind
(`<repo>:/work`, test asserts exactly one -v, no :ro) at -w /work; secrets pass by NAME
(`-e ANTHROPIC_API_KEY`, test asserts ALL -e values are bare, no =VALUE → no argv leak);
no docker.sock; the command tail is `claude` + the reused `ClaudeArgs(prompt)`. tdd-rygba:
Red → Yellow (strengthened to exactly-one-writable-mount + all-env-bare, closing two
security false-greens) → Green → Blue (all flags + security props pinned; pure) → Audit
(clean; valid docker argv; full suite green).

NOTE FOR SLICE 5a-ii (audit-surfaced, real): with --read-only rootfs + only /tmp & /work
writable, the agent's tools (claude/git/go/node) need a writable $HOME + caches
(~/.claude, ~/.cache, ~/.config, GOCACHE/GOMODCACHE, npm) or they hit EROFS. The runner
(5a-ii) / the agent image must set HOME + cache env to a writable path (/tmp or /work) or
add a tmpfs HOME. Deferred to 5a-ii by design (the pure argv builder's job is the
hardened argv; the env/image is wiring).

## Build record — slice 5a-ii SHIPPED (RouteEnv: writable HOME/cache for the read-only rootfs)

Addresses the 5a-i audit finding (the agent's tools EROFS on a read-only rootfs without
a writable HOME/cache). `ContainerSpec` gains `RouteEnv []EnvVar{Name,Value}` — host-set
NON-secret routing (HOME, GOCACHE, npm cache, … → the writable /tmp), rendered by
`ContainerArgs` as `-e NAME=VALUE` AFTER the by-name secret passthrough. tdd-rygba:
Red → Yellow (caught TWO real false-greens: the exact-match `==`/`NotContains` secret
checks missed a `ANTHROPIC_API_KEY=<value>` leak → switched to `HasPrefix` prefix-checks
that genuinely catch a leak) → Green → Blue (all 5a-i security tests intact; RouteEnv-
leak is out-of-contract documented) → Audit (clean; full suite 20/20). The secret stays
by-NAME bare; RouteEnv is non-secret NAME=VALUE — kept distinct + prefix-guarded.

## Build record — slice 5a-iii SHIPPED (the RunContainer runner)

`sandbox.MaterializeSeccompProfile` EXPORTED (rename) so the agent reuses the cage's
known-good seccomp deny-list. `internal/harness/runcontainer.go`: pure `agentSpec`
(unit-tested — API key by-name, HOME/GOCACHE/XDG/npm routed to the writable /tmp [the
EROFS fix], caps, image) + `RunContainer(ctx,repoDir,prompt,onActivity)([]Turn,error)`
— SAME signature as RunProcess (so the runHarness seam can swap). Wiring: head →
materialize seccomp (defer cleanup) → agentSpec(user=host uid:gid) → ContainerArgs →
`docker run --name packets-agent-<hex>` with StdoutPipe → the UNCHANGED Supervisor.Run
→ deadlock-safe kill(rm -f)+drain+reap on error (mirrors RunProcess) + cmd.Cancel by
name (mirrors DockerRunner). tdd-rygba: Yellow tightened the HOME/GOCACHE assertions to
require the writable /tmp prefix (a wrong route to "/" would EROFS); Blue + Audit
confirmed valid argv, deadlock-safe teardown, no orphaned seccomp refs, full suite
green. RunContainer is wiring (build/vet); its end-to-end proof is the fake-claude-IMAGE
integration test — SLICE 5a-iv (next).

## New clashes opened / resolved

Resolved: the agent container is a trust-isolation boundary (separate hardened
profile, egress+writable), distinct from the cage's containment; the firewall is
unchanged (host re-derives verdicts); the RunProcess→RunContainer seam leaves
Supervisor + runLiveOrder untouched. The maintenance-mode hold (R74) is superseded.
</content>
