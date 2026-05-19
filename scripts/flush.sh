#!/usr/bin/env bash
# Fired by the SessionEnd hook. Reads the SessionEnd JSON payload on stdin
# and tells agent-history to flush the session's captured commands to the
# user's shell history file in one batched write.
#
# Also writes a one-line invocation record to /tmp/agent-history-flush.log
# so it's possible to verify after the fact whether SessionEnd fired and
# how many bytes the write delivered, independent of whether the file's
# content survives subsequent zsh saves.

set -u

DBG=/tmp/agent-history-flush.log

BIN="${CLAUDE_PLUGIN_ROOT:-}/bin/agent-history"
if [ ! -x "$BIN" ]; then
  {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] flush.sh: BIN not executable: $BIN"
  } >>"$DBG" 2>&1
  cat >/dev/null
  exit 0
fi

PAYLOAD=$(cat)

HIST_PATH=$(awk -F'"' '/"history_path"/{print $4}' "${XDG_CONFIG_HOME:-$HOME/.config}/agent-history/config.json" 2>/dev/null)
size_of() { stat -f '%z' "$1" 2>/dev/null || echo 0; }

BEFORE=$(size_of "$HIST_PATH")
printf '%s' "$PAYLOAD" | "$BIN" flush 2>>"$DBG"
RC=$?
AFTER=$(size_of "$HIST_PATH")

{
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] flush.sh: PLUGIN_ROOT=${CLAUDE_PLUGIN_ROOT:-<unset>} hist=$HIST_PATH before=$BEFORE after=$AFTER delta=$((AFTER-BEFORE)) rc=$RC payload_len=${#PAYLOAD}"
} >>"$DBG" 2>&1

exit $RC
