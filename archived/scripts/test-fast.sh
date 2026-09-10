#!/usr/bin/env bash
set -euo pipefail

# Reproduces CI's test-fast + test-cage split locally, but runs both
# invocations as background jobs so wall time is max(fast, cage) instead of
# the sum. See .github/workflows/ci.yml for the CI-side split and the
# rationale for -p 1 on internal/cage + internal/sandbox.

cd "$(dirname "${BASH_SOURCE[0]}")/.."

# TMPDIR off tmpfs: cage tests fill /tmp. Env-overridable,
# default matches the existing gate.
export TMPDIR="${TMPDIR_OVERRIDE:-/home/jgonc/tmp}"
mkdir -p "$TMPDIR"

go build ./...
go vet ./...

# Single source of truth for the cage/sandbox exclusion pattern, shared with
# CI (.github/workflows/ci.yml).
CAGE_SANDBOX_EXCLUDE_PATTERN='/internal/(cage|sandbox)$'
FAST_PACKAGES=$(go list ./... | grep -Ev "$CAGE_SANDBOX_EXCLUDE_PATTERN")

echo "==> [fast] launching: $FAST_PACKAGES"
go test -race -timeout 5m $FAST_PACKAGES &
FAST_PID=$!

echo "==> [cage] launching: -p 1 ./internal/cage/ ./internal/sandbox/"
PACKETS_REQUIRE_CAGE=1 go test -race -timeout 10m -p 1 ./internal/cage/ ./internal/sandbox/ &
CAGE_PID=$!

FAST_STATUS=0
CAGE_STATUS=0
wait "$FAST_PID" || FAST_STATUS=$?
wait "$CAGE_PID" || CAGE_STATUS=$?

if [[ "$FAST_STATUS" -ne 0 ]]; then
  echo "==> [fast] FAILED (exit $FAST_STATUS): $FAST_PACKAGES" >&2
fi
if [[ "$CAGE_STATUS" -ne 0 ]]; then
  echo "==> [cage] FAILED (exit $CAGE_STATUS): ./internal/cage/ ./internal/sandbox/" >&2
fi

if [[ "$FAST_STATUS" -ne 0 || "$CAGE_STATUS" -ne 0 ]]; then
  exit 1
fi

echo "==> all packages passed"
