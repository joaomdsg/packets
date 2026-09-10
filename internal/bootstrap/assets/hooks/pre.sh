#!/bin/sh
# PreToolUse hook for Bash: stdin carries {tool_name, tool_input}. Exit 2
# with the reason on stderr blocks the call; exit 0 means "no decision",
# not approval — Claude Code's normal permission flow still runs
# (docs/claude-code-facts.md).
set -eu

input="$(cat)"
command=$(printf '%s' "$input" | jq -r '.tool_input.command // empty')

[ -z "$command" ] && exit 0

while IFS= read -r pattern; do
  [ -z "$pattern" ] && continue
  if printf '%s' "$command" | grep -Eq "$pattern"; then
    echo "packets: denied — command matches blocked pattern: $pattern" >&2
    exit 2
  fi
done <<'PATTERNS'
^[[:space:]]*git[[:space:]]+(commit|push|pull|rebase|merge|checkout|reset|stash|remote|tag)\b
^[[:space:]]*(npm|pnpm|yarn)[[:space:]]+(install|add|i)\b
^[[:space:]]*pip3?[[:space:]]+install\b
^[[:space:]]*go[[:space:]]+get\b
^[[:space:]]*(curl|wget)\b
^[[:space:]]*(sudo|su)\b
(^|[[:space:]])(cd|ls|cat|rm|cp|mv)[[:space:]]+(/|~|\.\.)
PATTERNS

exit 0
