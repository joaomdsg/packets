# packets

![status: research preview](https://img.shields.io/badge/status-research_preview-e6b23e)
[![CI](https://github.com/joaomdsg/packets/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/joaomdsg/packets/actions/workflows/ci.yml)

A control plane for agent-written code changes. Agents compose, verify, and
forward changes ("packets") at line rate against machine-checkable contracts
("handshakes"); the few with real blast radius are held for a human.
Networking automated forwarding but never automated away inspection — packets
does the same for code.

Nothing self-reports. A lane is measured, not declared; a catch is a
differential a mutation survived, not a claim; a delivery is an ACK, not an
agent saying "done". Where a number can't be earned honestly, the surface
says so — `no history yet` — rather than inventing one.

## Concepts

| term | what it is |
|---|---|
| **addr** | the repo a session tracks, in `owner/repo` form |
| **packet** | one agent-written change — named, revisioned, addressed, with a replayable timeline; composed from a prose **intent** |
| **handshake** | a runnable contract authored before and independently of the agent's code; protected paths its turn cannot touch |
| **lane** | best-effort → standard → strict → irreversible, from a MEASURED blast radius (never self-reported); a stronger lane runs more gates |
| **the gauntlet** | six gates per packet: intent fidelity, handshake conformance, handshake tightness, build·vet·lint, test sensitivity, independent cage check |
| **forward / hold** | forwards on its own unless inspection must hold it — advisory (amber) or blocking (red); holds debit the interrupt budget |
| **inspection** | four modes: pull (open any packet), push (the needs-you queue), standing (gauntlet + watches), adversarial (`packets probe`) |
| **attention** | a real interrupt budget that counts down as holds interrupt; calibration draws sample auto-forwarded packets. an empty queue is success |
| **learning** | a repo new to packets is *learning* until it has enough settled history, then *converged* — never a fabricated result |
| **delivery** | delivered means ACK'd healthy (`packets deployed` / `regressed`), never an agent's self-report |

The fuller model, voice, and visual system live in
[design/](design/) — start at
[design/guidelines/concepts.md](design/guidelines/concepts.md).

## Quick start

Requires **Go 1.26+** and **git**.

```bash
git clone https://github.com/joaomdsg/packets.git && cd packets && go build ./cmd/packets
```

**See it work** — the scripted walkthrough builds a throwaway repo and drives
compose → forward → hold → inspect → deliver end to end:

```bash
./scripts/demo.sh
```

**Or review a real revision pair** and open the console at `localhost:3000`:

```bash
./packets -repo . -base <baseSHA> -fix <fixSHA> -file pkg/auth/session.go -line 42
```

Routes: `/` console · `/review` inspector · `/board` fleet · `/settings` key.

More flags: `-live 'file=…,line=…,base=…,prompt=…'` seeds a live order a real
Claude Code harness runs (needs `claude` on `PATH` + `ANTHROPIC_API_KEY`);
`-container` runs that harness in a hardened image
(`docker build -f internal/harness/Dockerfile -t packets-agent .`); `-session
'key=…,base=…,fix=…,file=…,line=…'` tracks another addr on its own console. A
`GOROOT` version mismatch? Prefix with `env -u GOROOT`.

| command | does |
|---|---|
| `packets` | starts the server (flags above) |
| `packets verify-catch -repo -base -fix -file -line [-tip]` | runs the mutation/catch oracle over a revision pair; prints the outcome as JSON |
| `packets deployed -ledger -session -wo [-check]` | ACKs a packet delivered (healthy); an optional `-check` exit code must agree |
| `packets regressed -ledger -session -wo [-check]` | ACKs a packet regressed |
| `packets probe` | seeds a known-bad revision in a throwaway repo, runs it through the real gates, reports caught / escaped |

## Architecture

Server-rendered Go — [go-via/via](https://github.com/go-via/via) `h.*`
compositions over Datastar SSE, one hand-rolled CSS system, Monaco as JS
islands. No Tailwind, React, shadcn, or WebSockets. The packages trace the
signal path:

- **spine** — `fabric` (embedded JetStream, the source of truth), `ledger` (append-only log folded from it), `bridge`
- **read-models** (folded from the ledger) — `packet` (lane · handshake · gauntlet · hold), `review`, `reanchor`
- **the gauntlet** — `diff`, `mutation`, `catch`, `cage` (the no-egress G6 sandbox), `sandbox`
- **producing a packet** — `harness`, `translate`, `settle`, `orchestrator`, `pipe`, `ingest`, `assist`
- **surfaces** — `surface` (view models), `app` (the console/inspector server + fleet board), `tokenstore`, `cli` (flag parsing + subcommand dispatch), `cmd/packets` (the thin entrypoint shell)

## Testing

```bash
./scripts/test-fast.sh
```

Two concurrent groups: everything except `internal/cage` and
`internal/sandbox` at full parallelism, and that pair serialized (`-p 1`)
because both assert on a shared docker container label. Conventions live in
[CONVENTIONS.md](CONVENTIONS.md).

## License

[MIT](LICENSE) © João Gonçalves
