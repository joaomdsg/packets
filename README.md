# Packets

> **The agentic-coding experience as a management game you actually want to play.**
> You don't write the code. You route it.

[![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Status](https://img.shields.io/badge/status-research%20prototype-orange)](#project-status)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

---

Every other agentic tool keeps you in the coder's chair with an AI helper.
**Packets moves you up a level — to the gateway.** Your job becomes the
*actual* job of a senior engineer running a team: decide what gets built,
inspect what comes back — sharply and fast — keep many flows moving, and spend
a finite budget of attention and bandwidth wisely.

That is already a management sim. So we stop pretending it's an editor and build
the best version of one: a calm, instrumented gateway where the wire is your
codebase and the traffic is a fleet of Claude Code agents. Each unit of work is
a **packet** — a header you can read at a glance, a payload you inspect when it
matters. You **ACCEPT** it, you **DROP** it with a reason, or you send it back
to be retransmitted.

```text
┌─ THE FABRIC ─────────────────────────────────────── ◷ 14:32 ─┐
│  ⚙ auth-refactor      EMITTING    rev3   ░░░░▓ tests…         │
│  ✦ rate-limiter       QUEUED      rev2   ● inspect (2)        │
│  ⚙ docs-pass          PLANNING    —      plan ready ●         │
│  ⛔ migrate-db         BLOCKED     rev1   permission: drop tbl │
│  ✓ flaky-test-fix     FORWARDED   —      PR #412 merged       │
│                                                               │
│  Bandwidth ▓▓▓▓▓░░░ 312k / 500k tokens  ·  Queue: 2 to inspect│
└───────────────────────────────────────────────────────────────┘
```

---

## Why this is different

The pitch is not gamification-as-lipstick. The work *already has* every
property of a good tycoon game, so the game **is** the work:

| Tycoon-game property        | The real agentic-coding dynamic                       |
|-----------------------------|-------------------------------------------------------|
| A scarce resource you ration| **Your inspection attention** — the true bottleneck   |
| An economy                  | **Tokens / compute** = bandwidth; spend to move packets|
| Parallel workers            | **Agent fleet** — N packets in flight                 |
| A throughput goal           | **Forwarded, inspected packets per session**          |
| Triage under pressure       | Whose packet unblocks the most flows right now?       |
| Mastery curve               | Better tasks + conventions → fewer retransmits → flow |

The deep skill it trains is **calibrated delegation** — knowing exactly which
agents' flows you can **cut-through** (forward on a glance at the header, without
inspecting the payload) versus which still need **store-and-forward** (full deep
inspection) — *and being right.* Other tools level up the AI; Packets levels up
the human's judgment about the AI.

See [VISION.md](VISION.md) for the full design philosophy and
[DESIGN.md](DESIGN.md) / [council/](council/README.md) for the
architecture and its adversarial hardening.

---

## The hard problem this prototype solves

If a "good inspection" just means *"a test flipped red→green,"* the score is
**farmable** — the agent writes the very test that flips. RED→GREEN proves
*sequence, not constraint.* A pristine arc can sit on top of a tautological
test.

Packets' answer is an **independent oracle the agent didn't author:
diff-scoped mutation testing.** It's deep packet inspection of the *payload* —
not a checksum the sender handed you.

> A reviewer's `blocking:` comment counts as a **confirmed catch** — a real rule
> match — only if a mutation on the relevant line **survived before the fix and
> is killed after.** The fix didn't just turn a test green — it provably
> *constrained a line that used to be under-tested.*

That single, non-gameable definition is the spine of the whole trust economy.
This repo builds and proves that spine end-to-end.

### The loop, concretely

```text
two revisions (base → fix) + an anchored line
        │
        ▼
  mutation oracle ── was line N under-constrained at base, constrained at fix?
        │
        ▼
  confirmed catch ──▶ append-only ledger ──▶ fabric hit-rate
        │
        ▼
  live SSE packet inspector  (open a browser, watch one verdict resolve)
```

The included **golden replay** fixture demonstrates a real run: two adjacent
under-tested `>=` boundaries whose strengthened test kills both boundary
mutants — yielding **one catch at the anchor, two compounding catches on the
lines below, and one honest miss** on the operator-free closing brace. It
replays deterministically against the real oracle.

---

## Quick start

Requires **Go 1.24+** and **git**.

```bash
git clone https://github.com/joaomdsg/packets
cd packets
go build ./cmd/packets
```

Point it at any two revisions of a repo and the line you want adjudicated, then
open the live inspector:

```bash
./packets -repo . -base <weakSHA> -fix <fixSHA> -file adult.go -line 4
# → serving the packet inspector on :3000 — open it and watch adult.go:4 resolve
open http://localhost:3000
```

You'll watch a single inspection cycle go in-flight → resolved over SSE, with
any confirmed catch appended to `catches.jsonl`.

Route **several packets** through one gateway — each its own isolated flow:

```bash
./packets -repo . -base <sha> -fix <sha> -file a.go -line 4 \
  -session 'key=rate-limiter,base=<sha>,fix=<sha>,file=rl.go,line=12' \
  -session 'key=docs-pass,base=<sha>,fix=<sha>,file=doc.go,line=30'
# default flow at /  ·  keyed flows at /?key=rate-limiter and /?key=docs-pass
```

> **Tip:** if `go` errors with a `GOROOT` version mismatch, prefix commands
> with `env -u GOROOT`.

---

## Architecture

A thin `cmd/` wiring shell over focused `internal/` packages, each owning one
concern of the pipe.

| Package                 | Responsibility |
|-------------------------|----------------|
| `internal/mutation`     | Diff-scoped mutation oracle — deep payload inspection; mutates binary operators on changed lines |
| `internal/catch`        | The confirmed-catch oracle: the pure base→fix differential over mutation (the rule match) |
| `internal/diff`         | Structured git diff (changed files, hunks, line ranges) — the packet payload |
| `internal/reanchor`     | Maps a comment's line anchor across revisions; tells *moved* from *changed* |
| `internal/review`       | The inspection surface: anchored comment threads, surviving-mutant `question:` threads |
| `internal/settle`       | Turns a harness turn into a git revision (no-edit guard, secret-block) |
| `internal/orchestrator` | Host-side coordinator: composes settle + diff into a minted revision |
| `internal/ledger`       | Append-only event log of confirmed catches (the flow-table / Trust Ledger substrate) |
| `internal/surface`      | Projects oracle verdicts into the inspector / fabric view models |
| `internal/translate`    | Pure stream-json → review-event translation (Claude Code harness events) |
| `internal/pipe`         | End-to-end cycle composition |
| `internal/app`          | The live Via + Datastar SSE server, the fabric, flow registry |
| `cmd/packets`           | CLI entrypoint and flag/session wiring |

Built on [Via](https://github.com/go-via/via) + Datastar for a server-driven,
SSE-live UI — the Fabric *breathes* without a SPA.

---

## Project status

This is a **research prototype** that proves the hardest part of the vision —
the non-gameable confirmed-catch pipe — end-to-end against a real oracle. The
fabric, the live inspector, the ledger, and multi-flow isolation are real and
tested. The broader trust economy (the **flow allowlist** that earns an agent
cut-through forwarding), earned concurrency, merge-queue delivery, and the full
management-sim UX are **designed** (see VISION/DESIGN) but not yet built.

What's proven today:

- ✅ Diff-scoped mutation oracle (deep payload inspection) with honest verdicts (catch, miss, rename-lost, anchor-edited)
- ✅ Confirmed-catch cycle (the rule match), deterministic golden replay against the real oracle
- ✅ Append-only catch ledger with identity-dedup (no double-minting)
- ✅ Live SSE packet inspector + multi-flow fabric with honest hit-rate
- ✅ Uncapped fan-out (multicast) *and* its measured cost — see benchmarks below

Known risks and open design tensions are tracked candidly in
[RISKS.md](RISKS.md).

---

## Testing & benchmarks

```bash
env -u GOROOT go test ./...                    # full suite
```

Heavy concurrent-load benchmarks quantify the per-connect fan-out's saturation
knee (each cycle fires several full suite-execs):

```bash
env -u GOROOT go test ./internal/app -run='^$' -benchtime=2x \
  -bench='BenchmarkHeavyConcurrentCycle|BenchmarkSaturationSweep'
```

They show throughput plateauing once concurrency exceeds the host's core count
— past that point, more agents buy little throughput while multiplying memory
and scheduler pressure. That knee is the empirical bound the Fabric's
concurrency cap is meant to sit at; degradation past it is clean and linear
(per-cycle latency flat, allocations flat per cycle — no leak, no lock
pathology). Run them on your own hardware for the actual numbers.

---

## Documents

- **[VISION.md](VISION.md)** — the *what* and *why*: the management-sim reframe, the economy, the trust ledger.
- **[DESIGN.md](DESIGN.md)** — the *how*: the pipe, event spine, re-anchoring, sandboxing.
- **[council/](council/README.md)** — the adversarial design council's hardening rounds (one file per round).
- **[CONVENTIONS.md](CONVENTIONS.md)** — coding conventions fed to the agents.
- **[RISKS.md](RISKS.md)** — the honest risk register.

---

## License

[MIT](LICENSE) © João Gonçalves
