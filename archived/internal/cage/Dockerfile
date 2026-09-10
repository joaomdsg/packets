# The cage image: a pinned Go toolchain with the trusted `packets` binary baked
# in. verify-catch runs `git` and `go test` inside the cage, so the image keeps
# the full golang toolchain (git is already present in the golang image) — this
# is NOT a slim runtime image.
#
# The binary is built FROM the same pinned toolchain that runs it, so the build
# that produced the verdict tool matches the toolchain the verdict runs under.
# The tag tracks go.mod's `go` directive (1.26) — the cage must be able to build
# and test modules at the version the codebase requires, and matching the host
# toolchain is what the later in-proc-vs-cage equivalence lock needs.
# Build context is the repo root:
#
#   docker build -f internal/cage/Dockerfile -t packets-cage:dev .
#
# Image-BUILD has network (to fetch modules); the cage RUN is launched with
# --network=none by the sandbox, so nothing here depends on egress at run time.
FROM golang:1.26

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO off: packets shells out to git/go and needs no cgo, so a static binary
# runs regardless of the base libc.
RUN CGO_ENABLED=0 go build -o /usr/local/bin/packets ./cmd/packets

# The cage runs as a non-root user (65534) over a repo bind-mounted from the
# host and owned by a different uid, which trips git's "dubious ownership" guard
# (exit 128). Disable it SYSTEM-wide (/etc/gitconfig, read regardless of HOME) so
# the launch needs no runtime `git config`. Safe here: this single-purpose image
# only ever runs git over the verification repo mounted into it.
RUN git config --system --add safe.directory '*'

# No ENTRYPOINT: the sandbox supplies the full argv (packets verify-catch ...),
# and an empty default keeps the launch explicit.
WORKDIR /work
