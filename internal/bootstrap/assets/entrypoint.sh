#!/bin/sh
# $CLAUDE_CONFIG_DIR must be writable for Claude Code's own auth/session
# state; the tmpfs mounted there starts empty on every run, so reseed it
# from the root-owned read-only copy baked into the image before exec.
set -eu

mkdir -p "$CLAUDE_CONFIG_DIR"
cp -r /opt/packets/claude-config/. "$CLAUDE_CONFIG_DIR/"

exec "$@"
