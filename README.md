# agent-history

A Claude Code plugin that captures every Bash command a coding agent runs
into a per-session JSONL audit log, then lets you search across all sessions
with a single command. Built for Claude Code today; designed so other coding
agents can be added later.

## Why

Coding agents run Bash commands in a subprocess, so they never land in your
shell history. You can't `Ctrl+R` to recall that one useful `gh` or `gcloud`
invocation the agent figured out, and there's no easy way to review what the
agent actually did during a session.

This plugin solves both with a single owned data store. It does **not** write
to `~/.zsh_history` — that file is shared with multiple interactive shells
that overwrite it on save, making external appends fundamentally racy.
Instead it maintains its own per-session JSONL log and ships an
`agent-history search` command that's reliable, multi-Claude-safe, and easy
to wire to a hotkey.

## How it works

The plugin registers two hooks:

- **`SessionStart`** runs `scripts/bootstrap.sh`, which builds the bundled Go
  binary into `bin/agent-history` if it's missing or older than the sources.
  Requires `go` on `$PATH`.
- **`PostToolUse`** on the `Bash` tool runs `scripts/run.sh`, which forwards
  the hook JSON (on stdin) to `agent-history run`. That writes a structured
  JSONL record under `$XDG_DATA_HOME/agent-history/log/<session-id>.jsonl`.

You retrieve with `agent-history search [QUERY...]` — it scans every session
log, filters and sorts most-recent-first, and prints tab-separated
`<ts>\t<command>` lines that compose with `fzf`, `cut`, `grep`, etc.

### Multi-Claude is safe

Each Claude session writes to its **own** JSONL file (named by `session_id`),
in append-only mode. Two Claudes running concurrently → two different files
→ zero contention. `search` only reads files, so concurrent reads are always
fine.

## Install

Requires Go on `$PATH` (used by the SessionStart bootstrap script).

```
/plugin marketplace add git@github.com:loresuso/agent-history
/plugin install agent-history@loresuso-plugins
/reload-plugins
```

Restart Claude Code to fire the bootstrap. On the next Bash tool call the
plugin starts capturing.

## Usage

### Search

```sh
agent-history search                       # everything, most recent first
agent-history search gcloud                # substring match against the command
agent-history search -n 20                 # cap output
agent-history search -s <id-substring>     # filter to one session
agent-history search --with-cwd            # include the cwd column
agent-history search --jsonl               # raw JSONL for piping to jq
```

Output is `<timestamp>\t<command>` (tab-separated). Multiline commands are
flattened with a ` ⏎ ` marker so each record is one line.

> **Flag ordering matters.** Go's flag package stops at the first
> positional argument. `agent-history search -n 5 gcloud` works;
> `agent-history search gcloud -n 5` treats `-n 5` as part of the query.

### fzf widget bound to a hotkey

Add this to your `~/.zshrc` for `Alt-R`-style picking over Claude's history:

```sh
agent-history-widget() {
  local picked
  picked=$(agent-history search | fzf --delimiter='\t' --with-nth=2 \
                                       --prompt='agent$ ' \
                                       --preview='echo {} | cut -f2-' \
                                       --preview-window=down:3:wrap)
  if [[ -n $picked ]]; then
    LBUFFER+=$(echo "$picked" | cut -f2-)
  fi
  zle reset-prompt
}
zle -N agent-history-widget
bindkey '^[r' agent-history-widget  # Alt-R
```

The widget inserts the chosen command at the cursor; your real `Ctrl+R` for
zsh-typed history stays unchanged.

### Tail a single session

```
agent-history tail -s <session-id>
```

Prints the raw JSONL for the session. Session ids match the `<id>.jsonl`
filenames in the log directory.

## Paths

| What        | Location                                                |
|-------------|---------------------------------------------------------|
| Config      | `$XDG_CONFIG_HOME/agent-history/config.json` (default `~/.config/agent-history/`) |
| Audit logs  | `$XDG_DATA_HOME/agent-history/log/<session-id>.jsonl` (default `~/.local/share/agent-history/log/`) |

The config has one optional field, `filter` (regex). Commands matching it
are skipped on capture — useful for excluding noisy reads (`ls`, `cat`,
`pwd`).

## Security

The plugin captures Bash commands **verbatim**. The audit log will contain
whatever the agent ran — including secrets that appeared inline
(`export TOKEN=...`, `Authorization: Bearer ...`, etc.). Same trust model as
your normal shell history.

Files are written mode `0600`, directories `0700`. Don't paste them into
chat, screenshots, or pastebins. Don't sync them to shared cloud storage.

## Caveats

- **`Ctrl+R` against `~/.zsh_history` won't find Claude's commands.** This
  is deliberate. Use the `agent-history-widget` snippet above to get an
  fzf-bound key for Claude's history.
- **PostToolUse on Bash only.** Other tool calls (Read, Edit, etc.) are
  ignored. Long-running or aborted Bash calls that never complete are not
  captured either.
- **Claude Code only for now.** The `agent` field in log records is
  hardcoded `claude-code`. Adapters for other agents would slot into the
  hook payload parser in `hook.go`.

## Layout

```
.claude-plugin/plugin.json   plugin manifest
.claude-plugin/marketplace.json   self-referential single-repo marketplace
hooks/hooks.json             SessionStart + PostToolUse hooks
scripts/bootstrap.sh         build the Go binary if stale
scripts/run.sh               PostToolUse → agent-history run
main.go                      subcommand dispatch
hook.go                      `run` handler: capture to JSONL audit log
search.go                    `search` handler: scan + filter + sort + print
config.go                    XDG config load + first-run bootstrap
paths.go                     XDG_CONFIG_HOME / XDG_DATA_HOME resolution
tail.go                      `tail` subcommand
*_test.go                    table tests
```

## Status

v0.0.1. No promises of compatibility.
