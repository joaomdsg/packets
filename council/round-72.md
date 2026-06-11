# Round 72 — next direction: make the live pipe CLI-invocable (`-live`) — 2026-06-11

Trigger: the live-harness pipe (R67–R71) is complete at the data+render level, but a
user can't actually invoke it — live orders (Target.Prompt set) were test-seeded
only. A 3-lens council (CI/CD, Systems, TDD) chose the next autonomous-safe slice and
settled the gating boundary.

## The fork + the gating question

Remaining steps to an actual live run: (A) a CLI flag to dispatch a prompt-bearing
live order; (B) the agent CONTAINER (run claude isolated — needs egress + a writable
repo); (C) other. KEY: the standing directive hard-gates "the live NETWORK boundary
on explicit maintainer sign-off" — does that gate a HOST-SUBPROCESS claude run on the
user's OWN trusted repo?

## Convergence (3/3)

- CI/CD: building (A) is AUTONOMOUS-SAFE — a host-subprocess claude run on a trusted
  local repo (the user's own API key, no untrusted producer, no cross-process
  boundary) is NOT the gated #6 boundary (R69: "host-subprocess-first is fine for a
  trusted local repo"). The plumbing + tests stay in CI (no live run). The agent
  CONTAINER (B) IS the gated round (egress + writable repo, the opposite of the
  --network=none cage) — defer for maintainer sign-off. (A) is the smallest path to an
  invocable pipe.
- Systems: the FIREWALL holds — the Lead specifying the anchor (file/line) is SAFE.
  R70's anti-farming rule is against the untrusted AGENT deriving its own denominator
  from its diff, NOT against the trusted Lead choosing the target. A -live order is
  "fix the known weak spot at X" — same economy as a -backlog order, no new mint path,
  no new degenerate strategy.
- TDD: clean, mirrors -backlog exactly — a PURE parseLiveSpec(spec)→Target
  unit-tested data→data + CLI wiring (flag.Var, lineHashAt, DispatchBacklog append)
  verified by build/vet/-h, NOT fake-tested. No API key in CI. The live `claude`
  spawn stays wiring (RunProcess, R69 slice 2).

## Build record — slice A SHIPPED

`cmd/packets/live.go`: `liveFlag` (repeatable, mirrors backlogFlag) + pure
`parseLiveSpec`. Grammar: `file=F,line=N,base=SHA[,tip=SHA],prompt=<task>` — prompt=
is the trailing free-text (a `(^|,)\s*prompt\s*=` regexp delimiter, whitespace-
tolerant, key-anchored) so a task may contain commas/`=`; prompt MUST be last (a key
after it is swallowed → its missing anchor fail-closes). No FixRev (the agent
produces it; tip defaults to base). `main.go`: a `-live` loop after `-backlog` —
parse → compute LineHash vs base (same anchor identity, so fundableBacklog's
full-Target-equality dedup holds) → append to dispatchBacklog. A live Target funds as
a live order (Prompt!="" routes to runLiveOrder). tdd-rygba: Red → Yellow (added a
prompt-must-be-last contract test; switched a literal "prompt=" to a whitespace-
tolerant regexp) → Green → Blue (100% branch coverage; routing confirmed) → Audit
(clean; regexp leftmost-match + embedded-"prompt=" traced safe). Full suite 20/20,
vet clean, `-live` shows in `-h`. README updated.

## Status

The live-harness pipe is now CLI-invocable end-to-end: `-live` seeds a prompt-bearing
order → Spend dispatches it → a real `claude` harness (host subprocess) produces the
fix → its activity streams on the card → the catch cycle mints on the pre-specified
anchor. What remains for production: the agent CONTAINER (gated round B, maintainer
sign-off) and a real ANTHROPIC_API_KEY at run time.

## New clashes opened / resolved

Resolved: the gating boundary — a host-subprocess live run on a trusted local repo is
autonomous-safe; the isolated agent container is the gated round. No doc contradiction.
</content>
