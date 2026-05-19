package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// hookInput is the JSON the agent sends on stdin for a tool-use hook.
// Field names match the Claude Code hook payload; other agents would need
// adapters before this is truly multi-agent.
type hookInput struct {
	SessionID     string    `json:"session_id"`
	HookEventName string    `json:"hook_event_name"`
	ToolName      string    `json:"tool_name"`
	CWD           string    `json:"cwd"`
	ToolInput     toolInput `json:"tool_input"`
}

type toolInput struct {
	Command string `json:"command"`
}

type logRecord struct {
	TS      string `json:"ts"`
	Agent   string `json:"agent"`
	Session string `json:"session_id"`
	CWD     string `json:"cwd"`
	Command string `json:"command"`
	Event   string `json:"hook_event"`
}

// agentName is the identifier written to each log record. Hardcoded for now;
// becomes a config option once a second agent integration ships.
const agentName = "claude-code"

func cmdRun() error {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	var in hookInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return fmt.Errorf("parse hook input: %w", err)
	}

	if in.ToolName != "Bash" {
		return nil
	}

	cmd := in.ToolInput.Command
	if cmd == "" {
		return nil
	}

	cfg, err := loadOrBootstrapConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.Filter != nil && cfg.Filter.MatchString(cmd) {
		return nil
	}

	// Shell history is written by cmdFlush at SessionEnd, not here.
	// Writing per-command races with other interactive shells that overwrite
	// the file from their in-memory snapshot.

	session := in.SessionID
	if session == "" {
		session = "no-session-" + time.Now().UTC().Format("2006-01-02")
	}
	rec := logRecord{
		TS:      time.Now().UTC().Format(time.RFC3339),
		Agent:   agentName,
		Session: in.SessionID,
		CWD:     in.CWD,
		Command: cmd,
		Event:   in.HookEventName,
	}
	if err := appendSessionLog(session, rec); err != nil {
		return fmt.Errorf("write session log: %w", err)
	}

	return nil
}

func appendSessionLog(session string, rec logRecord) error {
	dir := logDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, session+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}
