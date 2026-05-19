package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout swaps os.Stdout for a pipe and returns the captured output
// after the body runs.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	w.Close()
	return <-done
}

func writeRecord(t *testing.T, sessionID, ts, cmd string) {
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
	raw, _ := json.Marshal(logRecord{
		TS: ts, Agent: "claude-code", Session: sessionID, Command: cmd, Event: "PostToolUse",
	})
	f.Write(append(raw, '\n'))
}

func TestCmdSearch_NoLogsIsNoOp(t *testing.T) {
	withSandbox(t)
	out := captureStdout(t, func() {
		if err := cmdSearch(nil); err != nil {
			t.Fatalf("cmdSearch: %v", err)
		}
	})
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestCmdSearch_PrintsAcrossSessionsMostRecentFirst(t *testing.T) {
	withSandbox(t)
	writeRecord(t, "sess-A", "2026-05-19T10:00:00Z", "old command")
	writeRecord(t, "sess-A", "2026-05-19T10:01:00Z", "middle A")
	writeRecord(t, "sess-B", "2026-05-19T11:00:00Z", "newest in B")

	out := captureStdout(t, func() {
		if err := cmdSearch(nil); err != nil {
			t.Fatalf("cmdSearch: %v", err)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "newest in B") {
		t.Errorf("expected newest first, got: %q", lines[0])
	}
	if !strings.Contains(lines[2], "old command") {
		t.Errorf("expected oldest last, got: %q", lines[2])
	}
}

func TestCmdSearch_QueryFiltersByCommandSubstring(t *testing.T) {
	withSandbox(t)
	writeRecord(t, "s1", "2026-05-19T10:00:00Z", "git status")
	writeRecord(t, "s1", "2026-05-19T10:01:00Z", "ls -la")
	writeRecord(t, "s1", "2026-05-19T10:02:00Z", "git log")

	out := captureStdout(t, func() {
		if err := cmdSearch([]string{"git"}); err != nil {
			t.Fatalf("cmdSearch: %v", err)
		}
	})
	if !strings.Contains(out, "git status") || !strings.Contains(out, "git log") {
		t.Errorf("expected git matches: %q", out)
	}
	if strings.Contains(out, "ls -la") {
		t.Errorf("ls should be filtered out: %q", out)
	}
}

func TestCmdSearch_SessionFilter(t *testing.T) {
	withSandbox(t)
	writeRecord(t, "sess-keep", "2026-05-19T10:00:00Z", "keep me")
	writeRecord(t, "sess-drop", "2026-05-19T10:01:00Z", "drop me")

	out := captureStdout(t, func() {
		if err := cmdSearch([]string{"-s", "keep"}); err != nil {
			t.Fatalf("cmdSearch: %v", err)
		}
	})
	if !strings.Contains(out, "keep me") {
		t.Errorf("expected keep, got: %q", out)
	}
	if strings.Contains(out, "drop me") {
		t.Errorf("expected drop to be filtered out, got: %q", out)
	}
}

func TestCmdSearch_LimitN(t *testing.T) {
	withSandbox(t)
	for i, ts := range []string{"2026-05-19T10:00:00Z", "2026-05-19T10:01:00Z", "2026-05-19T10:02:00Z"} {
		writeRecord(t, "s", ts, "cmd-"+string(rune('a'+i)))
	}
	out := captureStdout(t, func() {
		if err := cmdSearch([]string{"-n", "2"}); err != nil {
			t.Fatalf("cmdSearch: %v", err)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestCmdSearch_JSONLPassthrough(t *testing.T) {
	withSandbox(t)
	writeRecord(t, "s", "2026-05-19T10:00:00Z", "echo hello")

	out := captureStdout(t, func() {
		if err := cmdSearch([]string{"--jsonl"}); err != nil {
			t.Fatalf("cmdSearch: %v", err)
		}
	})
	var rec logRecord
	if err := json.Unmarshal([]byte(strings.TrimRight(out, "\n")), &rec); err != nil {
		t.Fatalf("jsonl output not valid JSON: %v\nraw: %q", err, out)
	}
	if rec.Command != "echo hello" {
		t.Errorf("unexpected record: %+v", rec)
	}
}

func TestCmdSearch_NewlinesInCommandAreFlattened(t *testing.T) {
	withSandbox(t)
	writeRecord(t, "s", "2026-05-19T10:00:00Z", "line1\nline2")

	out := captureStdout(t, func() {
		if err := cmdSearch(nil); err != nil {
			t.Fatalf("cmdSearch: %v", err)
		}
	})
	if strings.Count(strings.TrimRight(out, "\n"), "\n") != 0 {
		t.Errorf("expected single output line, got: %q", out)
	}
	if !strings.Contains(out, "line1") || !strings.Contains(out, "line2") {
		t.Errorf("both lines should be present: %q", out)
	}
}
