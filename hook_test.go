package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withSandbox redirects HOME and XDG_* into a temp directory so the test
// touches no real state. Returns the temp home for path assertions.
func withSandbox(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, ".local", "share"))
	return dir
}

// writeConfig writes config.json under the sandboxed XDG_CONFIG_HOME.
func writeConfig(t *testing.T, shell, historyPath, filter string) {
	t.Helper()
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(configFile{Shell: shell, HistoryPath: historyPath, Filter: filter})
	if err := os.WriteFile(configPath(), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// runHook injects an agent hook payload on stdin and runs cmdRun.
func runHook(t *testing.T, payload string) {
	t.Helper()
	withStdin(t, payload)
	if err := cmdRun(); err != nil {
		t.Fatalf("cmdRun: %v", err)
	}
}

// withStdin replaces os.Stdin with a pipe containing payload, restored on cleanup.
func withStdin(t *testing.T, payload string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	if _, err := w.WriteString(payload); err != nil {
		t.Fatal(err)
	}
	w.Close()
}

func TestCmdRun_WritesAuditLog(t *testing.T) {
	home := withSandbox(t)
	writeConfig(t, "bash", filepath.Join(home, ".bash_history"), "")

	payload := `{
		"session_id": "sess-abc",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"cwd": "/tmp/work",
		"tool_input": {"command": "ls -la"}
	}`
	runHook(t, payload)

	logPath := filepath.Join(logDir(), "sess-abc.jsonl")
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("session log not written: %v", err)
	}
	var rec logRecord
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &rec); err != nil {
		t.Fatalf("session log not valid JSON: %v\nraw: %s", err, raw)
	}
	if rec.Command != "ls -la" || rec.Session != "sess-abc" || rec.CWD != "/tmp/work" {
		t.Errorf("unexpected record: %+v", rec)
	}
	if rec.Agent != "claude-code" {
		t.Errorf("expected agent=claude-code, got %q", rec.Agent)
	}
}

func TestCmdRun_IgnoresNonBashTools(t *testing.T) {
	home := withSandbox(t)
	writeConfig(t, "bash", filepath.Join(home, ".bash_history"), "")

	payload := `{
		"session_id": "sess-x",
		"hook_event_name": "PostToolUse",
		"tool_name": "Read",
		"cwd": "/tmp",
		"tool_input": {"command": "ignored"}
	}`
	runHook(t, payload)

	logPath := filepath.Join(logDir(), "sess-x.jsonl")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("audit log should not exist for non-Bash tool, got err=%v", err)
	}
}

func TestCmdRun_RespectsFilter(t *testing.T) {
	home := withSandbox(t)
	writeConfig(t, "bash", filepath.Join(home, ".bash_history"), `^ls\b`)

	payload := `{
		"session_id": "sess-f",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"cwd": "/tmp",
		"tool_input": {"command": "ls -la"}
	}`
	runHook(t, payload)

	logPath := filepath.Join(logDir(), "sess-f.jsonl")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("filtered command should not be captured, got err=%v", err)
	}
}

func TestCmdRun_BootstrapsConfigOnFirstRun(t *testing.T) {
	home := withSandbox(t)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("HISTFILE", filepath.Join(home, ".zsh_history"))

	payload := `{
		"session_id": "sess-bs",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"cwd": "/tmp",
		"tool_input": {"command": "echo hi"}
	}`
	runHook(t, payload)

	if _, err := os.Stat(configPath()); err != nil {
		t.Errorf("expected bootstrapped config at %s: %v", configPath(), err)
	}
}
