#!/usr/bin/env bash
# Builds the agent-history binary on session start if it's missing or stale.
# Runs once per session at the cost of a few seconds; subsequent PostToolUse
# hooks just exec the prebuilt binary.

set -euo pipefail

ROOT="${CLAUDE_PLUGIN_ROOT:?CLAUDE_PLUGIN_ROOT is not set}"
BIN="$ROOT/bin/agent-history"

needs_build=0
if [ ! -x "$BIN" ]; then
  needs_build=1
elif [ "$ROOT/go.mod" -nt "$BIN" ]; then
  needs_build=1
elif find "$ROOT" -maxdepth 2 -name '*.go' -newer "$BIN" -print -quit | grep -q .; then
  needs_build=1
fi

if [ "$needs_build" -eq 1 ]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "agent-history: 'go' not on PATH; install Go to enable command capture" >&2
    exit 0
  fi
  echo "agent-history: building..." >&2
  (cd "$ROOT" && go build -o "$BIN" .)
fi
