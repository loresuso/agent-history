# agent-history

A Claude Code plugin that captures every Bash command a coding agent runs
and, at session end, flushes those commands into your native shell history
file. So your normal `Ctrl+R` finds them, like commands you typed yourself.

Built for Claude Code today; designed so other coding agents can be added
later.

## Why

Coding agents run Bash commands in a subprocess, so they never land in your
shell history. You can't `Ctrl+R` to recall that one useful `gh` or `gcloud`
invocation the agent figured out, and there's no easy way to review what the
agent actually did during a session.

This plugin captures every Bash tool call to an audit log, then batches the
session's commands into `~/.zsh_history` (or your shell's equivalent) when
the session ends.

## Install

### 1. Required zsh setup

If you use zsh, **add this line to your `~/.zshrc`**:

```sh
setopt INC_APPEND_HISTORY
```

Why: zsh's default save-on-exit can rewrite `~/.zsh_history` from a stale
in-memory snapshot, clobbering anything other processes (like this plugin)
wrote since that shell started. `INC_APPEND_HISTORY` forces every save to be
an incremental append. Pair it with `share_history` (already set by
oh-my-zsh) for cross-terminal history that doesn't lose external writes.

After editing `.zshrc`, **close every existing terminal window** and open a
fresh one — running shells still operate under whatever options they had at
startup.

### 2. Install the plugin

Requires Go on `$PATH` (used by the SessionStart bootstrap script).

```
/plugin marketplace add git@github.com:loresuso/agent-history
/plugin install agent-history@loresuso-plugins
/reload-plugins
```

The next time a Claude Code session starts, the binary builds; commands
captured during the session are flushed to your shell history when the
session ends.

## How it works

The plugin registers three hooks:

- **`SessionStart`** runs `scripts/bootstrap.sh`, which builds the bundled Go
  binary into `bin/agent-history` if it's missing or older than the sources.
- **`PostToolUse`** on the `Bash` tool runs `scripts/run.sh`, which forwards
  the hook JSON (on stdin) to `agent-history run`. That writes a structured
  JSONL record to `$XDG_DATA_HOME/agent-history/log/<session-id>.jsonl`.
- **`SessionEnd`** runs `scripts/flush.sh`, which calls `agent-history flush`.
  That reads the session's JSONL log and appends every captured command to
  your shell history file in a single batched write.

Per-session JSONL files mean **multiple concurrent Claude sessions don't
collide on writes** — each writes to its own file.

## Paths

| What        | Location                                                |
|-------------|---------------------------------------------------------|
| Config      | `$XDG_CONFIG_HOME/agent-history/config.json` (default `~/.config/agent-history/`) |
| Audit logs  | `$XDG_DATA_HOME/agent-history/log/<session-id>.jsonl` (default `~/.local/share/agent-history/log/`) |
| Shell hist  | Detected from `$SHELL` (zsh / bash / fish); overridable in config |

Edit the config file by hand to change the shell, history path, or set a
filter regex (commands matching the regex are skipped on capture).

## Audit a session

```
bin/agent-history tail -s <session-id> | jq
```

Session ids match the `<id>.jsonl` filenames in the log directory.

## Recover a session that never flushed

If Claude Code crashed or was force-killed, `SessionEnd` may not have fired
and the session's commands won't be in your shell history. The audit log is
still complete, so:

```
bin/agent-history flush -s <session-id>
```

will append everything captured in that session to your shell history file.

## Diagnostics

`scripts/flush.sh` writes one line per invocation to
`/tmp/agent-history-flush.log` (timestamp, before/after file size, exit
code, payload length). If a session's commands don't appear in your shell
history after `/exit`, that log tells you whether SessionEnd actually fired
and how many bytes the flush wrote — separately from whether those bytes
survived subsequent zsh saves.

## Security

The plugin captures Bash commands **verbatim**. Your shell history and the
audit log will contain whatever the agent ran — including secrets that
appeared inline (`export TOKEN=...`, `Authorization: Bearer ...`, etc.).
Same trust model as your normal `~/.zsh_history`.

Files are written mode `0600`, directories `0700`. Don't paste them into
chat, screenshots, or pastebins. Don't sync them to shared cloud storage.

## Caveats

- **Shell history shows up at SessionEnd, not in real time.** Commands land
  in `~/.zsh_history` when the Claude session exits, not while it's still
  running. Use `agent-history tail -s <id>` against the audit log for
  in-flight inspection.
- **`INC_APPEND_HISTORY` must be on in every interactive zsh.** A shell
  started before you added the setopt still operates under the old options
  and may overwrite the file on exit. Restart all zsh windows after
  editing `.zshrc`.
- **`fc -W` from any plugin/precmd overrides all setopts.** It does a full
  file rewrite. None of the zsh history setopts gate it. If you discover
  flushed entries vanishing, check your zsh plugins/themes for explicit
  `fc -W` calls.
- **Hard crash = no flush.** If Claude Code is killed (SIGKILL, OS reboot,
  power loss), `SessionEnd` doesn't fire. Recover with
  `agent-history flush -s <id>` against the still-complete audit log.
- **PostToolUse on Bash only.** Other tool calls are ignored. Long-running
  or aborted Bash calls that never complete are not captured.
- **Claude Code only for now.** The `agent` field in log records is
  hardcoded `claude-code`. Adapters for other agents would slot into the
  hook payload parser in `hook.go`.

## Layout

```
.claude-plugin/plugin.json   plugin manifest
.claude-plugin/marketplace.json   self-referential single-repo marketplace
hooks/hooks.json             SessionStart + PostToolUse + SessionEnd hooks
scripts/bootstrap.sh         build the Go binary if stale
scripts/run.sh               PostToolUse → agent-history run
scripts/flush.sh             SessionEnd  → agent-history flush
main.go                      subcommand dispatch
hook.go                      `run` handler: capture to JSONL audit log
flush.go                     `flush` handler: batch-append captured commands
                             to shell history at SessionEnd
history.go                   shell-format helpers + batched append
config.go                    XDG config load + first-run bootstrap
paths.go                     XDG_CONFIG_HOME / XDG_DATA_HOME resolution
tail.go                      `tail` subcommand
*_test.go                    table tests
```

## Status

v0.0.1. No promises of compatibility.
