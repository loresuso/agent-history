package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cmdFlush appends every command from a session's JSONL audit log to the
// native shell history file in a single open-write-close cycle. Wired into
// the SessionEnd hook so one batch per session reaches your shell history.
//
// Reliability depends on every interactive zsh having INC_APPEND_HISTORY (or
// at least share_history) set, so none of them ever rewrites ~/.zsh_history
// from a stale snapshot and clobbers our entries. See the README.
func cmdFlush(args []string) error {
	fs := flag.NewFlagSet("flush", flag.ExitOnError)
	sessionFlag := fs.String("s", "", "session id to flush (default: read from stdin hook payload)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	session := *sessionFlag
	if session == "" {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		var in hookInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return fmt.Errorf("parse hook input: %w", err)
		}
		session = in.SessionID
	}
	if session == "" {
		return fmt.Errorf("no session id (pass -s <id> or pipe SessionEnd JSON on stdin)")
	}

	logPath := filepath.Join(logDir(), session+".jsonl")
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open audit log: %w", err)
	}
	defer f.Close()

	cfg, err := loadOrBootstrapConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var blob strings.Builder
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var rec logRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.Command == "" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, rec.TS)
		if err != nil {
			ts = time.Now()
		}
		line, err := formatHistoryLine(cfg.Shell, rec.Command, ts)
		if err != nil {
			return err
		}
		blob.WriteString(line)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan audit log: %w", err)
	}

	return appendShellHistoryBatch(cfg.HistoryPath, blob.String())
}
