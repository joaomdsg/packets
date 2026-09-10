#!/bin/sh
# Stop hook: runs the packet's predicates and decides whether Claude Code
# may actually stop (§13.5).
#
# §13.5 as written omits the loop guard: stdin carries stop_hook_active,
# and returning "block" while it is true would loop until the 8-consecutive
# -block cap trips. Exit 0 immediately in that case.
set -eu

input="$(cat)"
stop_hook_active=$(printf '%s' "$input" | jq -r '.stop_hook_active // false')
[ "$stop_hook_active" = "true" ] && exit 0

signals_dir=${SIGNALS_DIR:-/signals}
predicates_file="$signals_dir/predicates.yaml"
results_file="$signals_dir/predicates.json"
retries_file="$signals_dir/retries_left"
first_run_marker="$signals_dir/first_run_done"
halt_file="$signals_dir/halt.md"

[ -f "$predicates_file" ] || exit 0

results="[]"
any_failed=0
new_passed=""
last_cmd=""; last_exit=0; last_out=""

flush() {
  [ -z "$cur_cmd" ] && return
  out=$(timeout 600 sh -c "$cur_cmd" 2>&1) && ec=0 || ec=$?
  results=$(printf '%s' "$results" | jq --arg cmd "$cur_cmd" --arg kind "$cur_kind" --argjson exit "$ec" \
    '. + [{"cmd":$cmd,"exit":$exit,"kind":$kind}]')
  if [ "$ec" -ne 0 ]; then
    any_failed=1
    last_cmd="$cur_cmd"; last_exit="$ec"; last_out="$out"
  fi
  if [ "$cur_is_new" = "true" ] && [ ! -f "$first_run_marker" ] && [ "$ec" -eq 0 ]; then
    new_passed="$cur_cmd"
  fi
}

cur_cmd=""; cur_kind=""; cur_is_new="false"
while IFS= read -r line; do
  case "$line" in
    *cmd:*)
      flush
      cur_cmd=$(printf '%s' "$line" | sed -E 's/^[[:space:]]*-?[[:space:]]*cmd:[[:space:]]*//; s/^"(.*)"$/\1/')
      cur_kind=""; cur_is_new="false"
      ;;
    *kind:*)
      cur_kind=$(printf '%s' "$line" | sed -E 's/^[[:space:]]*kind:[[:space:]]*//; s/^"(.*)"$/\1/')
      ;;
    *is_new:*)
      cur_is_new=$(printf '%s' "$line" | sed -E 's/^[[:space:]]*is_new:[[:space:]]*//; s/^"(.*)"$/\1/')
      ;;
  esac
done < "$predicates_file"
flush

printf '%s' "$results" > "$results_file"

if [ -n "$new_passed" ]; then
  mkdir -p "$signals_dir"
  {
    echo "smell:new_predicate_passed_before_implementation"
    echo
    echo "predicate marked is_new passed before implementation: $new_passed"
  } > "$halt_file"
  touch "$first_run_marker"
  exit 2
fi
touch "$first_run_marker"

[ "$any_failed" -eq 0 ] && exit 0

retries_left=$(cat "$retries_file" 2>/dev/null || echo 0)
if [ "$retries_left" -gt 0 ]; then
  retries_left=$((retries_left - 1))
  echo "$retries_left" > "$retries_file"
  tail_out=$(printf '%s' "$last_out" | tail -n 100)
  reason=$(printf 'PREDICATES FAILED. Fix and try again.\n%s: exit %s\n%s' "$last_cmd" "$last_exit" "$tail_out")
  jq -n --arg reason "$reason" '{"decision":"block","reason":$reason}'
  exit 0
fi

mkdir -p "$signals_dir"
{
  echo "budget:local_retries"
  echo
  echo "no local retries remain"
} > "$halt_file"
exit 2
