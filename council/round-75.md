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

## Build record — slice 5a-iv SHIPPED (the container path PROVEN end-to-end)

A fake-`claude`-image integration test proves `RunContainer` end-to-end with a REAL
`docker run`, NO API key. `fakeAgentImage` builds a tiny image FROM the CI-built
`packets-cage:dev` (no extra base pull — the flake source) whose `claude` edits the
bind-mounted `/work` repo + emits stream-json; an `export_test.go` `SetAgentImage`
seam points RunContainer at it. The test: a real container spawns → the agent edits
`/work/feature.go` → stream-json → the HOST's `SettleTurn` commits (the firewall: only
the host mints) → a minted revision whose diff includes `feature.go` + the editing
activity streamed live. Skips when `packets-cage:dev` is absent (local), hard-runs
under `PACKETS_REQUIRE_CAGE` (CI). Ran for REAL locally (PASS, 0.34s — cached base);
full suite 20/20 + vet green.

THE AGENT-CONTAINER PATH IS NOW PROVEN: profile (5a-i) → writable-HOME routing (5a-ii)
→ runner (5a-iii) → a real `docker run` settling a revision (5a-iv). The GOAL's
north-star ("a real harness instance does the work IN A CONTAINER, you review its
changeset") is demonstrated end-to-end (modulo the real `claude` + API key, which 5b's
seam-wiring + real image cover). Remaining: 5b wire the runHarness seam to select
RunContainer + the real agent Dockerfile; 5c+ egress allowlist / multi-tenant /
push-to-durable-storage.

## Build record — slice 5b-i SHIPPED (per-session runner selection)

`LiveConfig.UseContainer bool` + a `runHarnessContainer = harness.RunContainer` seam;
`runLiveOrder` selects `runner := runHarness; if cfg.UseContainer { runner =
runHarnessContainer }` — the ONLY change (timeout/lastMintedSHA/settleCatch/activity/
status all unchanged). Default false preserves the host-subprocess path (every
existing live test unaffected). tdd-rygba: two routing tests (UseContainer→container
runner, not subprocess; default→subprocess, not container — real discriminators); Blue
+ Audit clean; -race + full suite 20/20. The firewall is unchanged (both runners just
produce a revision the host settles). Remaining for 5b: the CLI `-container` flag
(sets UseContainer on the primary session) + the REAL agent Dockerfile (node/go/python
+ claude) + a CI build step — slice 5b-ii (wiring/infra; a real live run is
API-key-gated).

## Build record — slice 5b-ii SHIPPED (the container path is user-invocable + the real image)

`cmd/packets`: a `-container` bool flag sets `LiveConfig.UseContainer` on the primary
session (CLI wiring — build/vet/-h verified, no unit test). `internal/harness/Dockerfile`:
the REAL agent image (FROM golang:1.26 + apt node/python3 + `npm i -g
@anthropic-ai/claude-code` + git safe.directory; no ENTRYPOINT — RunContainer supplies
the argv; read-only rootfs with HOME/cache routed to tmpfs at run time), mirroring the
cage Dockerfile's proven patterns. README documents `docker build -f
internal/harness/Dockerfile -t packets-agent .` + `packets -container ... -live '...'`
with ANTHROPIC_API_KEY. The image is a production artifact (a real live run builds it +
needs the key); not CI-gated (no test depends on packets-agent — 5a-iv's integration
test uses the cage base). THE CONTAINER THREAD (5a-i→5b-ii) IS COMPLETE: a user can now,
from the shipped binary, dispatch a live work order that a real Claude Code harness
fills IN A HARDENED CONTAINER, reviewed as a changeset — the GOAL's north-star, end to
end (a real run is just `docker build` + an API key away).

## Thread close-out — CONTAINER THREAD COMPLETE; 5c+ gated (2026-06-11)

CI green on 5b-ii: the container thread (5a-i profile → 5a-ii writable-HOME routing →
5a-iii runner → 5a-iv real-docker-run proof → 5b-i per-session selection → 5b-ii CLI
flag + real image) is COMPLETE + verified. The maintainer's steer ("container
orchestration for the harness instances") is delivered end-to-end + user-invocable.

The deferred 5c+ items were assessed against the skeptic gate and are NOT
autonomous-safe non-marginal slices:
- egress allowlist — R34's allowlist-breaks-builds applies (the agent fetches deps
  live); needs the package mirror first + a maintainer policy call on which hosts.
- internal package mirror — only needed IF egress is restricted; the agent box is
  egress-allowed, so it's not needed standalone (interdependent with the allowlist).
- multi-tenant isolation — untrusted repos; a security thread needing a maintainer
  product decision.
- push-to-durable-storage — only matters with a remote/hibernation model that doesn't
  exist in the local prototype (in-process sessions; no remote to push to).

So the loop returns to MAINTENANCE MODE: the autonomous-safe high-value space across
the whole GOAL is built out; remaining work is gated on maintainer judgment or unbuilt
prerequisites. Hold at long cadence, never ping, await a steer.

## New clashes opened / resolved

Resolved: the agent container is a trust-isolation boundary (separate hardened
profile, egress+writable), distinct from the cage's containment; the firewall is
unchanged (host re-derives verdicts); the RunProcess→RunContainer seam leaves
Supervisor + runLiveOrder untouched. The maintenance-mode hold (R74) is superseded.
</content>
