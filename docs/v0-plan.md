# Packets v0 — Implementation Plan

Audience: an implementing agent with limited judgment. Follow literally. Do not improvise. Do not add features. Do not skip acceptance checks. If a step is ambiguous, STOP and ask the human; do not guess.

---

## 0. Ground rules for the implementer

1. Language: **Go 1.22+**. Single module `packets`. Single binary `packets`.
2. Dependencies allowed: Go stdlib, `gopkg.in/yaml.v3`, `github.com/spf13/cobra` (CLI). Nothing else without asking.
3. External binaries required at runtime: `git`, `gh`, `docker`. Check presence at startup; fail with clear message if missing.
4. **Never** write files inside the user's product repo except: (a) the branch commits the harness itself makes, (b) the single CI workflow file created by `packets init`.
5. Every state mutation is written to disk **before** the action that depends on it (write-before-act).
6. Every function that shells out logs the exact command and exit code to the packet log.
7. Do not touch anything listed in section 14 (Out of scope).
8. Complete phases **in order**. Each phase ends with acceptance checks. Do not start the next phase until all checks pass.

---

## 1. Vocabulary (use these exact terms in code, comments, CLI output)

| Term | Meaning |
|---|---|
| **Fabric** | One product repo. Identified by its git remote URL. Has a slug. |
| **Packet** | Human-authored unit of intent. Has a slug, versions, state. |
| **Predicate** | A shell command. Exit code 0 = true, anything else = false. |
| **Constraint** | Predicate that must be true at CI. Lives in `packet.yaml`. |
| **Terminal** | Predicate(s) whose truth terminates the packet. Lives in `packet.yaml`. |
| **Node** | A processing step. v0 has exactly three: `gate`, `build`, `ci`. |
| **Halt** | Packet paused; needs human action. |
| **Amend** | Human edits packet → new version → restarts at `build`. |
| **Accretion** | A new permanent test/check added to the repo. |
| **Seam** | A CLI command a human uses to interact with a packet. |

---

## 2. Directory layout (host)

```
~/.config/packets/
  fabrics.yaml                        # index: remote URL → fabric slug + local repo path

~/.local/share/packets/<fabric-slug>/
  fabric.yaml                         # fabric config (see §4)
  registry.jsonl                      # accretion provenance (see §9)
  packets/<packet-slug>/
    packet.yaml                       # CURRENT version
    versions/v1.yaml, v2.yaml, ...    # immutable snapshots
    state.json                        # see §5
    log.jsonl                         # see §10
    failure.md                        # latest CI failure context, may not exist
    halt.md                           # latest halt reason, may not exist
    proposals/<n>.md                  # agent proposals, may not exist
    runs/<attempt-n>/
      claude-output.json              # raw claude -p output
      container.log                   # docker stdout/stderr
      CLAUDE.md                       # exact file injected into container
```

Create directories with `0700`. Files with `0600`.

Resolve `~/.config` via `$XDG_CONFIG_HOME` fallback `~/.config`. Resolve `~/.local/share` via `$XDG_DATA_HOME` fallback `~/.local/share`.

---

## 3. `packet.yaml` schema

```yaml
goal: string                # required, non-empty, single line
context: string             # optional, multiline
constraints:                # optional, list of shell commands
  - string
terminal:                   # required, at least one shell command
  - string
budget:
  retries: int              # required, >= 1
  minutes: int              # required, >= 1
caused_by: string | null    # optional, another packet slug
```

Validation rules (mechanical, implemented in Go, no LLM):
- `goal` non-empty.
- `terminal` length >= 1.
- Every entry in `constraints` and `terminal` is non-empty and parses with `sh -n -c "<cmd>"` (exit 0).
- `budget.retries >= 1`, `budget.minutes >= 1`.
- If `caused_by` set, that packet dir must exist.

Reject with a list of every failing rule. Do not stop at the first.

The literal string `human_approval` is allowed as a terminal entry. Harness substitutes it with `packets check-approved <slug>` at execution time. This command exits 0 iff `state.json.approved == true`.

---

## 4. `fabric.yaml` schema

```yaml
slug: string                  # derived from remote URL, see §6
remote: string                # git remote URL
repo_path: string             # absolute local path of the repo clone the harness uses
default_branch: string        # e.g. main
image: string                 # docker image tag, e.g. packets/<slug>:v0
budget_defaults:
  retries: 5
  minutes: 60
smell_triggers:
  max_files_touched: 10
  same_error_repeats: 3
  deny_test_modification: true
  deny_dependency_add: true
  deny_terminal_edit: true
ci:
  poll_seconds: 30
  timeout_minutes: 30
  workflow_name: packets
merge:
  auto: true                  # merge when green and no human_approval in terminal
  strategy: squash
```

---

## 5. `state.json` schema and state machine

```json
{
  "slug": "fix-login-timeout",
  "version": 2,
  "state": "building",
  "node": "build",
  "attempt": 3,
  "retries_used": 2,
  "minutes_used": 14.5,
  "tokens_used": 41200,
  "branch": "packets/fix-login-timeout-v2",
  "pr_number": 118,
  "approved": false,
  "halt_reason": null,
  "created_at": "2026-09-09T10:00:00Z",
  "updated_at": "2026-09-09T10:14:30Z"
}
```

Allowed `state` values and transitions. Anything not listed is a bug; return an error.

```
draft          --emit ok-->         emitted
emitted        --run-->             building
building       --local green-->     ci
building       --smell/budget/agent halt--> halted
ci             --red, retries left--> building
ci             --red, no retries--> halted (reason=budget)
ci             --rebase conflict--> halted (reason=conflict)
ci             --green, terminal has human_approval--> awaiting_approval
ci             --green, otherwise--> terminated (after merge)
awaiting_approval --approve-->      terminated (after merge)
halted         --resume-->          building
halted         --extend-->          building
halted         --amend-->           emitted (version+1)
awaiting_approval --amend-->        emitted (version+1)
any non-terminal --kill-->          killed
```

`terminated` and `killed` are final. No transitions out.

Write `state.json` atomically: write to `state.json.tmp`, fsync, rename.

---

## 6. Slug rules

**Fabric slug**: from remote URL. Take the last path segment, strip `.git`, lowercase, replace non `[a-z0-9-]` with `-`. Example `git@github.com:acme/Auth-Service.git` → `auth-service`.

**Packet slug**: from `goal`. Take first 4 words, lowercase, replace non `[a-z0-9]` with `-`, collapse repeated `-`, trim `-`, cap 40 chars. Example `Fix the login timeout on slow networks` → `fix-the-login-timeout`.

**Collision**: if `packets/<slug>/` already exists, append `-MMDD` (current month+day). If still exists, append `-MMDD-2`, `-3`, ... Log event `slug_collision`.

---

## 7. CLI seams (exact commands)

All commands take `--fabric <slug>` optionally; default = fabric whose `repo_path` contains the current working directory. Error if none.

| Command | Effect |
|---|---|
| `packets init` | See §11. |
| `packets new` | Writes template `packet.yaml` to `./packet.yaml` in cwd. Prints path. Does nothing else. |
| `packets emit <file>` | Runs gate (§8). On pass: create packet dir, `versions/v1.yaml`, `state=emitted`, print slug. On fail: print all reasons, exit 1. |
| `packets run <slug>` | Runs the loop (§12) until state is `halted`, `awaiting_approval`, `terminated`, or `killed`. Blocking. Resumable: reads `state.json` and continues from `node`. |
| `packets status <slug>` | Prints `state.json` fields human-readably. |
| `packets list` | One line per packet: slug, version, state, updated_at. |
| `packets resume <slug>` | Only if `halted`. Sets `state=building`, clears `halt_reason`. Does NOT run; user then runs `packets run`. |
| `packets extend <slug> --retries N` | Only if `halted` with `halt_reason=budget`. Adds N to `budget.retries` in `packet.yaml` (no new version). Then same as resume. |
| `packets amend <slug>` | Opens `packet.yaml` in `$EDITOR`. On save: run gate. On pass: `version+1`, snapshot to `versions/vN.yaml`, `state=emitted`, `retries_used=0`, write one line to `log.jsonl` with event `amend` and `summary` = first line of the diff. |
| `packets kill <slug>` | Any non-final state → `killed`. Closes PR if open (`gh pr close`). Does not delete branch. |
| `packets approve <slug>` | Only if `awaiting_approval`. Sets `approved=true`, then merges (§12 step 9), then `terminated`. |
| `packets check-approved <slug>` | Exit 0 iff `approved==true`. Used inside predicates. |
| `packets proposals <slug>` | Lists `proposals/*.md`. |
| `packets approve-proposal <slug> <n>` | Creates new draft `packet.yaml` at `./packet.yaml` with `goal` = proposal title, `context` = proposal body, `caused_by: <slug>`. Prints path. Human then edits + emits manually. |

Every seam command appends one `log.jsonl` line with `actor: "human"`.

---

## 8. Node 1 — Emit gate

Two stages. Stage A must pass before Stage B runs.

**Stage A — mechanical** (Go only): rules in §3. Any failure → reject.

**Stage B — LLM warnings**. Call `claude -p` on the **host** (not in container) with `--output-format json`. Prompt file `gate-prompt.txt` embedded in binary:

```
You are reviewing a work packet before an autonomous coding agent executes it.
Answer ONLY with JSON matching this schema, no prose, no markdown fences:
{
  "warnings": [ { "code": "AMBIGUOUS_GOAL" | "TERMINAL_DOES_NOT_MEASURE_GOAL" | "LOOSE_CONSTRAINTS" | "UNMEASURABLE", "detail": "one sentence" } ],
  "suggested_terminal": [ "shell command that would verify the goal, if one is missing" ]
}
Rules:
- AMBIGUOUS_GOAL: goal has more than one plausible outcome, or uses words like "improve", "clean up", "better" without a measure.
- TERMINAL_DOES_NOT_MEASURE_GOAL: terminal predicates would pass even if the goal were not achieved.
- LOOSE_CONSTRAINTS: no constraint runs the existing test suite or linter.
- UNMEASURABLE: goal cannot be verified by any shell command.
- suggested_terminal: at most 3 commands. Only suggest if terminal is missing something. Use the repo's existing test runner conventions if visible in REPO_FILES.
Return empty arrays if nothing applies.

PACKET:
<packet.yaml contents>

REPO_FILES (top-level listing):
<output of ls -1 in repo>
```

Parse JSON. If parse fails, retry once. If fails again, log `gate_llm_error` and proceed with zero warnings (do not block on LLM failure).

Interactive behavior of `packets emit`:
1. Print each warning.
2. For each `suggested_terminal`, prompt `Add to terminal? [y/N]`. `y` → append to `terminal` and mark `gate_suggested: true` in a sidecar list stored in `state.json` under `gate_suggested_terminals`.
3. If any warnings remain, prompt `Emit anyway? [y/N]`. `y` → log `emit_override` with the warning codes. `N` → exit 1, packet not created.

---

## 9. Accretion registry

`registry.jsonl`, one line per accretion:

```json
{"ts":"...","fabric":"auth-service","packet":"fix-login-timeout","version":2,"path":"test/login.timeout.spec.js","cause":"gate_suggested"|"ci_failure"|"proposal","commit":"abc123"}
```

Written by harness when:
- `gate_suggested`: after the packet's PR merges, for each `gate_suggested_terminals` entry, record the file(s) the terminal command references (best effort: any token in the command that is an existing path in the repo).
- `ci_failure`: after merge, for each new file under a test directory added in this branch that did not exist on `default_branch` and was not gate-suggested. Test directories: any path segment matching `test|tests|spec|__tests__|e2e`.
- `proposal`: when `approve-proposal` is run.

If detection fails, log `registry_detect_failed` and continue. Never block on registry.

---

## 10. Log format

`log.jsonl`, one JSON object per line, fields always present (null if n/a):

```json
{"ts":"RFC3339","slug":"...","version":1,"node":"gate|build|ci|null","event":"...","reason":"string|null","tokens":0,"actor":"human|harness|agent","detail":{}}
```

Event names (exact strings):
`emit_reject`, `emit_warn`, `emit_override`, `emit`, `slug_collision`, `enter_node`, `container_start`, `container_exit`, `local_pred`, `smell_halt`, `agent_halt`, `budget_halt`, `conflict_halt`, `push`, `pr_open`, `ci_poll`, `ci_pred`, `rebase`, `merge`, `resume`, `extend`, `amend`, `kill`, `approve`, `proposal`, `accrete`, `terminate`, `gate_llm_error`, `registry_detect_failed`, `error`.

`detail` for `local_pred` and `ci_pred`: `{"cmd":"...","exit":0,"kind":"constraint|terminal"}`.

---

## 11. `packets init` — bootstrap

Run inside a git repo clone. Steps, in order; stop on first failure with a message:

1. `git remote get-url origin` → remote. Compute fabric slug (§6).
2. If `~/.config/packets/fabrics.yaml` has this remote → print "already initialized" and exit 0.
3. Create `~/.local/share/packets/<slug>/` and write `fabric.yaml` with defaults (§4). `repo_path` = `git rev-parse --show-toplevel`. `default_branch` = `gh repo view --json defaultBranchRef -q .defaultBranchRef.name`.
4. Detect toolchain, pick exactly one base, in this priority: `go.mod` → `golang:1.22`; `package.json` → `node:20`; `pyproject.toml` or `requirements.txt` → `python:3.12`; else `ubuntu:24.04`. If more than one present, ask the human which.
5. Write Dockerfile to `~/.local/share/packets/<slug>/Dockerfile` from the embedded template (§13). Build: `docker build -t packets/<slug>:v0 .`.
6. Smoke test: `docker run --rm -e ANTHROPIC_API_KEY packets/<slug>:v0 claude -p "reply with the single word OK" --output-format json`. Assert output contains `OK`.
7. Write CI workflow to `<repo>/.github/workflows/packets.yml` (§13). Commit on a branch `packets/init`, push, open PR, print the PR URL. Tell the human to merge it. This is the only file the harness ever adds to the repo.
8. Append to `fabrics.yaml`.

---

## 12. `packets run <slug>` — the loop

Pseudocode. Implement exactly.

```
load fabric.yaml, packet.yaml, state.json
if state in [terminated, killed, halted, awaiting_approval]: print state, exit 0
loop:
  switch state.node:

  case "build":
    // 1. prepare branch
    if state.branch is empty:
      state.branch = "packets/<slug>-v<version>"
      git -C repo fetch origin
      git -C repo checkout -B <branch> origin/<default_branch>
    else:
      git -C repo checkout <branch>
    // 2. write CLAUDE.md (see §13 template) into runs/<attempt>/CLAUDE.md
    //    include failure.md contents if it exists
    //    include halt.md contents if it exists
    //    substitute human_approval with `packets check-approved <slug>` in shown predicates
    // 3. spawn container (see §13 docker run command); timeout = budget.minutes - minutes_used
    state.attempt += 1; save; log container_start
    exit_code, output = run container
    log container_exit
    parse output json → tokens; state.tokens_used += tokens; save
    // 4. inspect results written by hooks into the mounted /signals dir
    if /signals/halt.md exists:
      copy to packet halt.md; state.state="halted"; state.halt_reason = first line; save; log agent_halt or smell_halt; exit
    if exit_code == timeout:
      state.state="halted"; halt_reason="budget"; save; log budget_halt; exit
    // 5. local predicate results are in /signals/predicates.json ({cmd, exit, kind}[]); log each as local_pred
    // 6. smell re-check on host (belt and braces): git diff --stat origin/<default_branch>..HEAD
    if files_touched > max_files_touched: halt reason=smell:max_files
    if deny_test_modification and any modified (not added) file is in a test dir: halt reason=smell:test_modified
    if deny_dependency_add and any of go.mod/package.json/requirements.txt/pyproject.toml modified: halt reason=smell:dependency
    // 7. commit
    git add -A; git commit -m "packets: <slug> v<version> attempt <attempt>"  (skip if nothing to commit → halt reason=no_changes)
    // 8. rebase
    git fetch origin; git rebase origin/<default_branch>
    if conflict: git rebase --abort; state.state=halted; halt_reason=conflict; save; log conflict_halt; exit
    // 9. push + pr
    git push --force-with-lease origin <branch>; log push
    if state.pr_number == 0:
      gh pr create --title "<goal>" --body "<packet.yaml in a yaml fence>" --head <branch> --base <default_branch>
      state.pr_number = parsed number; save; log pr_open
    state.node="ci"; state.state="ci"; save; log enter_node

  case "ci":
    // poll
    deadline = now + ci.timeout_minutes
    loop every ci.poll_seconds until deadline:
      res = gh pr checks <pr> --json name,state,link
      log ci_poll
      find check named fabric.ci.workflow_name
      if state == "SUCCESS": break green
      if state in ["FAILURE","ERROR","CANCELLED"]: break red
    if deadline hit: halt reason=ci_timeout; exit
    if red:
      logs = gh run view <run-id> --log-failed   (run-id from the check link)
      write failure.md (format §13); log ci_pred exit=1
      state.retries_used += 1
      if retries_used >= budget.retries: halted reason=budget; save; log budget_halt; exit
      state.node="build"; state.state="building"; save; continue loop
    if green:
      log ci_pred exit=0
      if "human_approval" in terminal and not state.approved:
        state.state="awaiting_approval"; save; print PR url; exit
      // merge
      if fabric.merge.auto:
        gh pr merge <pr> --<strategy> --delete-branch=false
        log merge
        write registry entries (§9)
        state.state="terminated"; state.node=null; save; log terminate; exit
      else:
        state.state="awaiting_approval"; save; exit
```

`packets approve` performs the merge block above and sets `terminated`.

---

## 13. Templates (embed in binary; do not edit at runtime)

### 13.1 Dockerfile

```dockerfile
FROM {{BASE}}
RUN apt-get update && apt-get install -y --no-install-recommends git curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL https://claude.ai/install.sh | sh   # or the documented install; verify current method in Claude Code docs before finalizing
RUN useradd -m -u 1000 agent
COPY settings.json /home/agent/.claude/settings.json
COPY hooks/ /home/agent/.claude/hooks/
RUN chmod +x /home/agent/.claude/hooks/*.sh && chown -R agent:agent /home/agent/.claude && chmod -R a-w /home/agent/.claude
USER agent
WORKDIR /work
```

Verify the exact Claude Code install command against current docs before building. Do not guess.

### 13.2 `settings.json` (in image, read-only)

```json
{
  "permissions": {
    "allow": ["Read", "Edit", "Write", "Glob", "Grep", "Bash(*)"],
    "deny": []
  },
  "hooks": {
    "PreToolUse": [{ "matcher": "Bash", "hooks": [{ "type": "command", "command": "/home/agent/.claude/hooks/pre.sh" }] }],
    "PostToolUse": [{ "matcher": "Edit|Write", "hooks": [{ "type": "command", "command": "/home/agent/.claude/hooks/post.sh" }] }],
    "Stop": [{ "hooks": [{ "type": "command", "command": "/home/agent/.claude/hooks/stop.sh" }] }]
  }
}
```

Verify hook JSON shape against current Claude Code docs. Field names may differ; the semantics must be: Bash pre-check, Edit/Write post-check, Stop check.

### 13.3 `hooks/pre.sh` — deny list

Reads tool input JSON on stdin. Extract the bash command. Deny (exit 2, print reason) if command matches any regex:

```
^\s*git\s+(commit|push|pull|rebase|merge|checkout|reset|stash|remote|tag)\b
^\s*(npm|pnpm|yarn)\s+(install|add|i)\b
^\s*pip3?\s+install\b
^\s*go\s+get\b
^\s*(curl|wget)\b
^\s*(sudo|su)\b
(^|\s)(cd|ls|cat|rm|cp|mv)\s+(/|~|\.\.)
```

Allowed: `git status`, `git diff`, `git log`. Everything else exit 0.

### 13.4 `hooks/post.sh` — smell triggers

After each Edit/Write:
1. `git diff --stat origin/main..HEAD` plus untracked → count files. If > `MAX_FILES` (env, default 10) → write `/signals/halt.md` with `smell:max_files` and exit 2.
2. If edited file path matches a test dir and file existed on `origin/main` (i.e. modified, not new) → `/signals/halt.md` `smell:test_modified`, exit 2.
3. If edited file is one of `go.mod package.json package-lock.json requirements.txt pyproject.toml` → `smell:dependency`, exit 2.
4. Else exit 0.

Exit 2 must cause Claude Code to stop. Verify against docs which exit code/JSON blocks; adjust to match.

### 13.5 `hooks/stop.sh` — predicate runner

Reads `/signals/predicates.yaml` (harness writes: list of `{cmd, kind, is_new}`).
1. For each, run `sh -c "$cmd"` with 10-minute timeout. Append `{cmd, exit, kind}` to `/signals/predicates.json`.
2. **Red-first rule**: if `/signals/first_run_done` does not exist: for each predicate with `is_new: true`, if exit == 0 → write `/signals/halt.md` `smell:new_predicate_passed_before_implementation`, exit 2. Then `touch /signals/first_run_done`.
3. If all exit 0 → exit 0 (agent stops, harness takes over).
4. If any failed: read `/signals/retries_left` (harness writes an integer). If > 0: decrement, print to stdout a block:
   ```
   PREDICATES FAILED. Fix and try again.
   <cmd>: exit <n>
   <last 100 lines of its output>
   ```
   and return the "continue" signal per Claude Code docs (JSON `{"decision":"block","reason":"<text above>"}` or equivalent). If 0: write `/signals/halt.md` `budget:local_retries`, exit 2.

Note: local retries inside one session count against `budget.retries` too. Harness writes `retries_left = budget.retries - retries_used`.

### 13.6 `docker run` command

```
docker run --rm \
  --name packets-<slug>-<attempt> \
  --network <fabric-egress-network> \
  -e ANTHROPIC_API_KEY \
  -e MAX_FILES=<max_files_touched> \
  -v <repo_path>:/work:rw \
  -v <runs/attempt>/CLAUDE.md:/home/agent/.claude/CLAUDE.md:ro \
  -v <runs/attempt>/signals:/signals:rw \
  --user 1000 \
  --read-only --tmpfs /tmp \
  packets/<slug>:v0 \
  timeout <seconds> claude -p "Read /home/agent/.claude/CLAUDE.md and complete the packet." --output-format json --max-turns 200
```

Egress: v0 acceptable approach = `--network host` **only if** the human confirms. Preferred: create a docker network with an egress proxy container allowing `api.anthropic.com` and the package registry for the toolchain. If proxy setup exceeds one day, fall back to `--network bridge` and log `egress_unrestricted` once per run. Ask the human before choosing.

`/signals` contents before start: `predicates.yaml`, `retries_left`. Harness clears the dir each attempt.

### 13.7 `CLAUDE.md` template

```
# Packet: {{slug}} v{{version}} (attempt {{attempt}})

## Goal
{{goal}}

## Context from the human
{{context or "(none)"}}

## Constraints (must pass)
{{for c in constraints}}- `{{c}}`
{{end}}

## Terminal (defines done)
{{for t in terminal}}- `{{t}}`{{if is_new}}  ← NEW: this check does not exist yet. You must create it.{{end}}
{{end}}

## Rules
1. Work only inside /work. Do not use git write commands. Do not install dependencies. Do not fetch from the network.
2. If any Terminal predicate is marked NEW: create it FIRST, run it, confirm it FAILS, then implement the goal. Do not write a test that passes before the implementation exists.
3. Do not modify existing test files. Adding new test files is fine.
4. Keep changes minimal. Touch as few files as possible.
5. When you believe you are done, simply stop. A hook will run the predicates. If they fail you will be told and may continue.
6. If you are unsure the goal is achievable, or the packet seems wrong, or you need something you are not allowed to do: write a file /signals/halt.md with a one-line reason on the first line and details below, then stop.
7. If you notice a missing check, lint, or convention that should exist but is not required by this packet: write /signals/proposal-<n>.md with a title on the first line and rationale below. Do not act on it. Continue with the goal.

{{if failure_md}}
## Previous CI failure
{{failure_md}}
{{end}}

{{if halt_md}}
## Previous halt (human has resumed you)
{{halt_md}}
{{end}}
```

Harness copies any `/signals/proposal-*.md` into `proposals/` and logs `proposal`.

### 13.8 `failure.md` format

```
attempt: {{attempt}}
failed: {{kind}} `{{cmd}}`
exit: {{code}}
--- last 200 lines ---
{{tail}}
--- diff since previous attempt ---
{{git diff --stat <prev-commit>..HEAD}}
```

### 13.9 CI workflow `.github/workflows/packets.yml`

```yaml
name: packets
on:
  pull_request:
    branches: ['**']
jobs:
  packets:
    if: startsWith(github.head_ref, 'packets/')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Extract predicates from PR body
        id: pred
        run: |
          echo "${{ github.event.pull_request.body }}" | awk '/^```yaml/{f=1;next}/^```/{f=0}f' > packet.yaml
      - name: Setup toolchain
        run: |
          # {{TOOLCHAIN_SETUP}}  — filled by init: e.g. actions/setup-node + npm ci
      - name: Run constraints
        run: |
          yq '.constraints[]' packet.yaml | while read -r c; do echo "+ $c"; sh -c "$c" || exit 1; done
      - name: Run terminal
        run: |
          yq '.terminal[]' packet.yaml | sed 's/^human_approval$/true/' | while read -r t; do echo "+ $t"; sh -c "$t" || exit 1; done
```

`human_approval` is replaced by `true` in CI; approval is checked by harness, not CI. Install `yq` in the workflow if not present on the runner.

---

## 14. Out of scope for v0 (do not build)

Web UI. Multiple fabrics per repo. Multi-repo packets. Agent-resolved merge conflicts. Concurrency control between packets. Secrets injection into CI. Charter, conventions as entities. Any node beyond gate/build/ci. Packet splitting/children. Custom fabric tools. Live authoring assist. Slug collision UX beyond date suffix. Automatic accretion decisions beyond what §9 detects.

---

## 15. Phases with acceptance checks

### Phase 1 — Skeleton (est. 1 day)
Build: module, cobra CLI with all commands stubbed, config paths, `fabric.yaml` + `packet.yaml` + `state.json` load/save, state machine with transition validation, `log.jsonl` writer, slug functions.
Accept:
- [ ] `go build` clean, `go vet` clean.
- [ ] Unit tests: every legal transition passes, every illegal transition errors. Slug examples in §6 produce exact outputs.
- [ ] `packets new` writes template. `packets emit` with invalid yaml prints ALL rule failures.
- [ ] `state.json` written atomically (test: kill during write leaves old file intact).

### Phase 2 — Gate (est. 1 day)
Build: §8 fully.
Accept:
- [ ] Packet with `goal: improve things` gets `AMBIGUOUS_GOAL` warning.
- [ ] Accepting a suggestion appends to terminal and records in `gate_suggested_terminals`.
- [ ] LLM failure (simulate bad JSON) → proceeds with `gate_llm_error` logged, no crash.
- [ ] Override logged with codes.

### Phase 3 — Init + image (est. 2 days)
Build: §11, §13.1–13.5.
Accept:
- [ ] `packets init` on a node repo produces working image; smoke test passes.
- [ ] Inside container as user `agent`: `git commit` via Bash → denied with reason. `npm install` → denied. `ls /` → denied. `npm test` → allowed.
- [ ] Editing an existing test file → `/signals/halt.md` contains `smell:test_modified`.
- [ ] `/home/agent/.claude` is not writable by agent.
- [ ] Workflow PR opened.

### Phase 4 — Build node (est. 2 days)
Build: §12 `build` case, §13.6–13.8.
Accept (use a throwaway repo with a trivial failing test as terminal):
- [ ] `packets run` spawns container, agent makes test pass, harness commits, pushes, opens PR, state → `ci`.
- [ ] With `is_new` terminal that agent makes pass before implementing → halt `smell:new_predicate_passed_before_implementation`.
- [ ] Container timeout → halt `budget`.
- [ ] Agent writes `/signals/halt.md` → state halted, `halt.md` copied.
- [ ] Proposal file → copied to `proposals/`, logged.
- [ ] Kill container mid-run, rerun `packets run` → resumes without corrupting state.

### Phase 5 — CI node + merge (est. 1 day)
Build: §12 `ci` case, §9, `approve`.
Accept:
- [ ] Red CI → `failure.md` written with correct format, state → building, `retries_used` +1.
- [ ] Retries exhausted → halt `budget`.
- [ ] Green + auto merge → PR merged, state terminated, registry lines written.
- [ ] Green + `human_approval` → awaiting_approval; `packets approve` merges.
- [ ] Rebase conflict (create one deliberately) → halt `conflict`, rebase aborted, branch intact.

### Phase 6 — Seams + robustness (est. 1 day)
Build: remaining CLI commands, `extend`, `amend`, `kill`, `approve-proposal`.
Accept:
- [ ] `amend` bumps version, snapshots, resets retries, logs summary, restarts at build with previous branch checked out.
- [ ] `kill` closes PR, state killed, further commands refuse.
- [ ] `extend` only works from `halted/budget`.
- [ ] All seams produce a log line with `actor: human`.
- [ ] Missing `gh`/`docker`/`git` → clear error at startup.

### Phase 7 — Dogfood (est. 2 days, ongoing)
Run 5 real packets on a real repo. For each, human reviews `log.jsonl`. Fix bugs only; do not add features. Then hand over for the 20–30 packet experiment.

---

## 16. Metrics to compute from logs (script, `packets report`)

- gate: count `emit_reject`, `emit_warn`, `emit_override`; distribution of warning codes.
- termination: % of `terminate` events where terminal contained `human_approval` vs not.
- accretion: count registry lines by cause; for each `ci_failure` accretion, count later packets failing the same test path (repeat-failure rate).
- amend: mean amends per packet; tokens + minutes between `amend` and next `terminate`.
- halts: count by `halt_reason`.
- budget: count `budget_halt`.

Output as plain text table. No charts.

---

## 17. When to stop and ask the human

- Claude Code hook/settings schema differs from §13.2–13.5 assumptions.
- Egress proxy takes more than one day.
- Toolchain detection finds more than one ecosystem.
- Any acceptance check cannot be made to pass within a day.
- Anything not covered by this document.
