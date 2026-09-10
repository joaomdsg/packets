#!/bin/sh
# PostToolUse hook for Edit|Write: the in-container half of the smell
# checks (§13.4). The harness re-runs the same checks on the host after
# the container exits — this is defence in depth, not the real gate.
set -eu

input="$(cat)"

max_files=${MAX_FILES:-10}
base_branch=${BASE_BRANCH:-origin/main}
signals_dir=${SIGNALS_DIR:-/signals}
work_dir=${WORK_DIR:-/work}

halt() {
  mkdir -p "$signals_dir"
  {
    echo "$1"
    echo
    echo "$2"
  } > "$signals_dir/halt.md"
  exit 2
}

cd "$work_dir"

tracked=$(git diff --stat "$base_branch"..HEAD 2>/dev/null | grep -c '|' || true)
untracked=$(git ls-files --others --exclude-standard 2>/dev/null | wc -l)
touched=$((tracked + untracked))
if [ "$touched" -gt "$max_files" ]; then
  halt "smell:max_files" "files touched: $touched (limit $max_files)"
fi

file=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')
[ -z "$file" ] && exit 0

case "$file" in
  */test/*|*/tests/*|*/spec/*|*/__tests__/*|*/e2e/*| \
  test/*|tests/*|spec/*|__tests__/*|e2e/*)
    if git cat-file -e "$base_branch:$file" 2>/dev/null; then
      halt "smell:test_modified" "existing test file modified: $file"
    fi
    ;;
esac

case "$(basename "$file")" in
  go.mod|package.json|package-lock.json|requirements.txt|pyproject.toml)
    halt "smell:dependency" "dependency file changed: $file"
    ;;
esac

exit 0
