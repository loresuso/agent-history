package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureSession writes a session JSONL log with the given commands so
// flush has something to read.
func captureSession(t *testing.T, sessionID string, commands ...string) {
	t.Helper()
	if err := os.MkdirAll(logDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(logDir(), sessionID+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, c := range commands {
		rec := logRecord{
			TS:      "2026-05-19T11:00:00Z",
			Agent:   "claude-code",
			Session: sessionID,
			Command: c,
			Event:   "PostToolUse",
		}
		raw, _ := json.Marshal(rec)
		f.Write(append(raw, '\n'))
	}
}

func TestCmdFlush_WritesAllSessionCommandsToHistory(t *testing.T) {
	home := withSandbox(t)
	histPath := filepath.Join(home, ".bash_history")
	writeConfig(t, "bash", histPath, "")
	captureSession(t, "sess-flush-1", "ls -la", "git status", "echo done")

	if err := cmdFlush([]string{"-s", "sess-flush-1"}); err != nil {
		t.Fatalf("cmdFlush: %v", err)
	}

	raw, err := os.ReadFile(histPath)
	if err != nil {
		t.Fatalf("history not written: %v", err)
	}
	got := string(raw)
	for _, want := range []string{"ls -la", "git status", "echo done"} {
		if !strings.Contains(got, want) {
			t.Errorf("history missing %q; got:\n%s", want, got)
		}
	}
}

func TestCmdFlush_ReadsSessionIDFromStdin(t *testing.T) {
	home := withSandbox(t)
	histPath := filepath.Join(home, ".bash_history")
	writeConfig(t, "bash", histPath, "")
	captureSession(t, "sess-from-stdin", "echo via-stdin")

	withStdin(t, `{"session_id":"sess-from-stdin","hook_event_name":"SessionEnd"}`)

	if err := cmdFlush(nil); err != nil {
		t.Fatalf("cmdFlush: %v", err)
	}

	raw, _ := os.ReadFile(histPath)
	if !strings.Contains(string(raw), "echo via-stdin") {
		t.Errorf("expected stdin-driven flush to write entry; got: %q", string(raw))
	}
}

func TestCmdFlush_MissingSessionLogIsNotAnError(t *testing.T) {
	home := withSandbox(t)
	writeConfig(t, "bash", filepath.Join(home, ".bash_history"), "")

	if err := cmdFlush([]string{"-s", "nonexistent-session"}); err != nil {
		t.Errorf("expected nil error for missing session log, got: %v", err)
	}
}

func TestCmdFlush_NoSessionIDIsAnError(t *testing.T) {
	withSandbox(t)
	withStdin(t, `{"hook_event_name":"SessionEnd"}`)
	if err := cmdFlush(nil); err == nil {
		t.Error("expected error when session id is missing from both -s and stdin")
	}
}

func TestCmdFlush_UsesOriginalCommandTimestamp(t *testing.T) {
	home := withSandbox(t)
	histPath := filepath.Join(home, ".zsh_history")
	writeConfig(t, "zsh", histPath, "")
	captureSession(t, "sess-ts", "echo ts-probe")

	if err := cmdFlush([]string{"-s", "sess-ts"}); err != nil {
		t.Fatalf("cmdFlush: %v", err)
	}

	raw, _ := os.ReadFile(histPath)
	// 2026-05-19T11:00:00Z == unix 1779188400
	if !strings.Contains(string(raw), ": 1779188400:0;echo ts-probe") {
		t.Errorf("expected original ts in zsh format; got: %q", string(raw))
	}
}
