# Verified Claude Code facts for §13

Checked against current docs. Where these differ from `v0-plan.md` §13,
these win — §13 flags itself as unverified.

## Install (replaces §13.1 `curl | sh`)

The documented Debian/Ubuntu route is an apt repository, not a shell
installer:

```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends \
      curl gnupg ca-certificates git \
 && install -d -m 0755 /etc/apt/keyrings \
 && curl -fsSL https://downloads.claude.ai/keys/claude-code.asc \
      -o /etc/apt/keyrings/claude-code.asc \
 && echo "deb [signed-by=/etc/apt/keyrings/claude-code.asc] \
https://downloads.claude.ai/claude-code/apt/stable stable main" \
      > /etc/apt/sources.list.d/claude-code.list \
 && apt-get update && apt-get install -y claude-code \
 && rm -rf /var/lib/apt/lists/*
```

There is no official prebuilt image; the only first-party container
artifact is a dev container feature, which is not usable via `docker run`.

## Hooks schema (§13.2 confirmed)

The nesting in §13.2 is correct as written: `hooks` → event name → list of
`{matcher, hooks: [{type: "command", command}]}`. `Stop` entries take no
`matcher`.

## PreToolUse blocking (§13.3)

Hook stdin carries `tool_name` and `tool_input`; for Bash the command is at
`.tool_input.command`. Exit 2 blocks the call and the reason must go to
**stderr**, not stdout. Exit 0 means "no decision" — the normal permission
flow continues, it is not an approval.

## Stop hook (§13.5) — plan gap

`{"decision":"block","reason":"..."}` on stdout is current and does force
continuation. But §13.5 omits the loop guard: stdin carries
`stop_hook_active`, and a hook that ignores it will block repeatedly until
the 8-consecutive-block cap trips. `stop.sh` must exit 0 immediately when
`stop_hook_active` is true.

## Token accounting (§12 step 3)

`--output-format json` returns `result`, `session_id`, `total_cost_usd`,
and `usage` with `input_tokens`, `output_tokens`,
`cache_creation_input_tokens`, `cache_read_input_tokens`. Sum input and
output for `state.tokens_used`; `total_cost_usd` is worth recording too.

`--max-turns N` and `--permission-mode` are spelled as §13.6 assumes.
`"Bash(*)"` and bare `"Bash"` are both valid and both mean all Bash.

## Read-only config dir — conflicts with §13.1 and §13.6

§13.1 does `chmod -R a-w /home/agent/.claude` and §13.6 runs
`--read-only` with only `/tmp` as tmpfs. Claude Code needs its config dir
writable (auth/session state) plus `~/.claude.json`, so that combination
will not start. Undocumented behaviour, so it needs a live test rather
than a guess.

Resolution taken for v0: point `CLAUDE_CONFIG_DIR` at a writable tmpfs and
seed it at container start from a root-owned, read-only copy baked into the
image. The agent can then tamper with its own hooks, which is accepted
because hooks are defence in depth only — the harness re-runs every smell
check on the host after the container exits (§12 step 6) and CI is the
real gate.

## Additions to the §2 layout

`emit_reject` (§10) has nowhere to live under §2's layout: Stage A
rejection happens before a packet dir exists, so there is no
`log.jsonl` yet to append to. Resolved by adding one file §2 doesn't
mention:

```
~/.local/share/packets/<fabric-slug>/
  log.jsonl                           # fabric-level log; see below
```

This lives alongside `registry.jsonl`, same line shape as a packet's
`log.jsonl` (§10), `slug`/`version` left blank when there is no packet yet.
It carries `emit_reject` (Stage A validation failure, before a packet dir
exists) and `list` (§7's `packets list` has no single packet to log
against, since it spans every packet in the fabric). `packets report`
(§16) reads this file in addition to every per-packet `log.jsonl`.

## Credential mount supersedes §13.6's `-e ANTHROPIC_API_KEY`-only approach

That account has no credits, so §13.6's `-e ANTHROPIC_API_KEY` alone
cannot be verified end to end. Mounting Claude Code's own OAuth
credentials works instead and was verified live against the built image:

```
docker run --rm \
  --tmpfs /home/agent/.claude-config:uid=1000,gid=1000 \
  -v /home/user/.claude/.credentials.json:/home/agent/.claude-config/.credentials.json:ro \
  packets/smoke:v0 claude -p "reply with the single word OK" --output-format json
```

returning `"result":"OK"`, `"is_error":false`. `~/.claude.json` is not
needed; the credentials file alone suffices.

Resolution order, implemented once in `internal/claudeauth` and shared by
`packets init`'s smoke test and the build node's container run:

1. `fabric.yaml`'s `credentials_path`, if set (error if it names a file
   that does not exist — a pinned path that vanished is a
   misconfiguration, not a silent fallback).
2. `$CLAUDE_CONFIG_DIR/.credentials.json`, else
   `~/.claude/.credentials.json`, if that file exists.
3. `ANTHROPIC_API_KEY`, if set.
4. Otherwise fail, naming both options.

When a credentials file is used it is bind-mounted read-only into the
container's `$CLAUDE_CONFIG_DIR` as `.credentials.json`; when none is
found, `ANTHROPIC_API_KEY` is passed through with `-e` as §13.6 describes.

### `uid=1000,gid=1000` requirement

The config dir's tmpfs must be mounted `uid=1000,gid=1000`. The
entrypoint reseeds `$CLAUDE_CONFIG_DIR` from a root-owned image copy
before exec, running as the `agent` user (uid 1000, per the Dockerfile);
docker's default tmpfs is root-owned, so that reseed's `mkdir`/`cp` fails
without the explicit ownership. This applies wherever the harness spawns
the container, credential mount or not.

### Security note

The agent runs with `Bash(*)` and can read whatever is mounted into its
filesystem, including a mounted credentials file. `:ro` stops it from
modifying that file, not from exfiltrating its contents — nothing stops
the agent from reading it and shipping it out over the network in a
request. `pre.sh`'s denial of `curl`/`wget` is a deny-list against a
Bash-command string, not a sandbox: it can be evaded by any tool or
technique that doesn't literally contain those substrings, and it does
nothing to constrain a non-Bash egress path. This is exactly why §13.6's
deferred egress proxy is a real control rather than a nicety — without
it, credential exfiltration risk is bounded only by the agent's own
compliance with hooks it can also see and, per the read-only config dir
tradeoff above, tamper with.
