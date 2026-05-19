package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatHistoryLine_Zsh(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	got, err := formatHistoryLine("zsh", "ls -la", ts)
	if err != nil {
		t.Fatal(err)
	}
	if got != ": 1700000000:0;ls -la\n" {
		t.Errorf("unexpected zsh format: %q", got)
	}
}

func TestFormatHistoryLine_ZshMultiline(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	got, err := formatHistoryLine("zsh", "echo a\necho b", ts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "echo a\\\necho b") {
		t.Errorf("expected escaped newlines, got: %q", got)
	}
}

func TestFormatHistoryLine_Bash(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	got, err := formatHistoryLine("bash", "ls -la", ts)
	if err != nil {
		t.Fatal(err)
	}
	if got != "#1700000000\nls -la\n" {
		t.Errorf("unexpected bash format: %q", got)
	}
}

func TestFormatHistoryLine_Fish(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	got, err := formatHistoryLine("fish", "echo hi", ts)
	if err != nil {
		t.Fatal(err)
	}
	if got != "- cmd: echo hi\n  when: 1700000000\n" {
		t.Errorf("unexpected fish format: %q", got)
	}
}

func TestFormatHistoryLine_Unsupported(t *testing.T) {
	if _, err := formatHistoryLine("powershell", "x", time.Now()); err == nil {
		t.Error("expected error for unsupported shell")
	}
}

func TestAppendShellHistoryBatch_AppendsAndDoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	if err := appendShellHistoryBatch(path, "first\n"); err != nil {
		t.Fatal(err)
	}
	if err := appendShellHistoryBatch(path, "second\n"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "first\nsecond\n" {
		t.Errorf("unexpected content: %q", string(raw))
	}
}

func TestAppendShellHistoryBatch_EmptyBlobIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	if err := appendShellHistoryBatch(path, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected no file to be created, got err=%v", err)
	}
}
