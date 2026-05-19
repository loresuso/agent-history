#!/usr/bin/env bash
# Forwards the hook payload (JSON on stdin) to the agent-history binary.
# If the binary doesn't exist yet (e.g. SessionStart bootstrap hasn't fired
# or failed silently), drop the event rather than blocking the agent.

set -eu

BIN="${CLAUDE_PLUGIN_ROOT:?CLAUDE_PLUGIN_ROOT is not set}/bin/agent-history"
if [ ! -x "$BIN" ]; then
  cat >/dev/null
  exit 0
fi
exec "$BIN" run
