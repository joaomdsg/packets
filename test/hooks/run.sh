#!/bin/bash
# Exercises the packets hook scripts (§13.3-13.5) as real shell against a
# throwaway git repo, standing in for the §15 Phase 3 acceptance checks
# that need a booted container.
set -euo pipefail

HOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../internal/bootstrap/assets/hooks" && pwd)"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

pass=0
fail=0

expect_exit() {
  name="$1"; want="$2"; got="$3"
  if [ "$got" = "$want" ]; then
    echo "ok   - $name"
    pass=$((pass + 1))
  else
    echo "FAIL - $name (want exit $want, got $got)"
    fail=$((fail + 1))
  fi
}

expect_contains() {
  name="$1"; haystack="$2"; needle="$3"
  if printf '%s' "$haystack" | grep -qF "$needle"; then
    echo "ok   - $name"
    pass=$((pass + 1))
  else
    echo "FAIL - $name (expected to find: $needle)"
    fail=$((fail + 1))
  fi
}

# --- pre.sh ---

run_pre() {
  printf '%s' "$1" | "$HOOKS_DIR/pre.sh"
}

set +e
out=$(run_pre '{"tool_name":"Bash","tool_input":{"command":"git commit -m x"}}' 2>/tmp/pre_stderr)
ec=$?
set -e
expect_exit "git commit denied" 2 "$ec"
expect_contains "git commit denial reason on stderr" "$(cat /tmp/pre_stderr)" "denied"

set +e
run_pre '{"tool_name":"Bash","tool_input":{"command":"npm install"}}' >/dev/null 2>/tmp/pre_stderr
ec=$?
set -e
expect_exit "npm install denied" 2 "$ec"

set +e
run_pre '{"tool_name":"Bash","tool_input":{"command":"ls /"}}' >/dev/null 2>/tmp/pre_stderr
ec=$?
set -e
expect_exit "ls / denied" 2 "$ec"

set +e
run_pre '{"tool_name":"Bash","tool_input":{"command":"npm test"}}' >/dev/null 2>/tmp/pre_stderr
ec=$?
set -e
expect_exit "npm test allowed" 0 "$ec"

set +e
run_pre '{"tool_name":"Bash","tool_input":{"command":"git status"}}' >/dev/null 2>/tmp/pre_stderr
ec=$?
set -e
expect_exit "git status allowed" 0 "$ec"

# --- post.sh: editing an existing test file ---

REPO="$WORKDIR/repo"
mkdir -p "$REPO/tests"
git -C "$REPO" init -q -b work
git -C "$REPO" config user.email test@example.com
git -C "$REPO" config user.name test
echo "original" > "$REPO/tests/example_test.go"
git -C "$REPO" add -A
git -C "$REPO" commit -q -m base
git -C "$REPO" branch origin/main work
echo "modified" > "$REPO/tests/example_test.go"

set +e
out=$(printf '%s' '{"tool_input":{"file_path":"tests/example_test.go"}}' \
  | WORK_DIR="$REPO" SIGNALS_DIR="$WORKDIR/signals" "$HOOKS_DIR/post.sh")
ec=$?
set -e
expect_exit "editing existing test file denied" 2 "$ec"
expect_contains "halt.md contains smell:test_modified" "$(cat "$WORKDIR/signals/halt.md" 2>/dev/null)" "smell:test_modified"

# --- stop.sh: stop_hook_active loop guard ---

set +e
printf '%s' '{"stop_hook_active":true}' | "$HOOKS_DIR/stop.sh"
ec=$?
set -e
expect_exit "stop_hook_active true exits 0 immediately" 0 "$ec"

echo
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
