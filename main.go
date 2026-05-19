package main

import (
	"fmt"
	"os"
)

const version = "0.0.1"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		if err := cmdRun(); err != nil {
			fail(err)
		}
	case "flush":
		if err := cmdFlush(os.Args[2:]); err != nil {
			fail(err)
		}
	case "tail":
		if err := cmdTail(os.Args[2:]); err != nil {
			fail(err)
		}
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `agent-history - capture coding-agent Bash commands and flush them to your shell history at SessionEnd

Usage:
  agent-history run             PostToolUse handler. Reads hook JSON from stdin.
  agent-history flush [-s ID]   SessionEnd handler. Batch-appends a session's
                                captured commands to your shell history file.
                                Without -s, reads SessionEnd JSON from stdin.
  agent-history tail -s <id>    Print the raw JSONL audit log for a session.
  agent-history version

Normally invoked via the agent-history Claude Code plugin. Config lives at
$XDG_CONFIG_HOME/agent-history/config.json (default ~/.config/agent-history/).
Audit logs live at $XDG_DATA_HOME/agent-history/log/ (default
~/.local/share/agent-history/log/).`)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
