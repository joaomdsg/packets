# The agent image: the live Claude Code harness's container. Unlike the
# verification cage (internal/cage/Dockerfile — a locked-down, --network=none box
# for UNTRUSTED oracle code), this runs the TRUSTED harness and needs the model
# API + a writable repo, so harness.ContainerArgs launches it egress-allowed with
# the repo bind-mounted writable (but still hardened: cap-drop, seccomp, non-root,
# read-only rootfs, pids/mem caps — see internal/harness/container.go).
#
# The image bakes in the language toolchains the agent does real work with
# (node + go + python) plus git and the `claude` CLI itself. Build context is the
# repo root:
#
#   docker build -f internal/harness/Dockerfile -t packets-agent .
#
# A live run is then `packets -container -repo . -base <sha> ... -live '...'` with
# an ANTHROPIC_API_KEY in the environment (passed through by name — never baked
# into the image or the argv).
FROM golang:1.26

# node (the claude CLI is an npm package) + python3 — so a task in any of the three
# baked languages can build/test inside the box. git is already in the golang image.
RUN apt-get update \
  && apt-get install -y --no-install-recommends nodejs npm python3 \
  && rm -rf /var/lib/apt/lists/*

# The Claude Code CLI — the harness the supervisor drives via stream-json.
RUN npm install -g @anthropic-ai/claude-code

# The container runs as the HOST uid (harness.ContainerArgs --user) over a repo
# bind-mounted from the host; matching uids avoids git's "dubious ownership" guard,
# but set it system-wide too so a differing uid never trips exit 128. Read
# regardless of HOME (which RouteEnv points at the writable /tmp).
RUN git config --system --add safe.directory '*'

# No ENTRYPOINT: harness.RunContainer supplies the full argv
# (claude -p <task> --output-format stream-json --verbose --permission-mode
# bypassPermissions). The rootfs is launched read-only; HOME and the tool caches
# are routed onto a writable tmpfs at run time (harness RouteEnv), so no writable
# state is baked here.
WORKDIR /work
