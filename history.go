package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// formatHistoryLine renders one command as a single on-disk record in the
// given shell's native history format. Returned string ends with a newline
// (or two, for fish's two-line YAML-ish entries).
func formatHistoryLine(shell, cmd string, ts time.Time) (string, error) {
	switch shell {
	case "zsh":
		// Extended history format: ": <epoch>:0;<command>\n".
		// Embedded newlines must be escaped with a trailing backslash.
		safe := strings.ReplaceAll(cmd, "\n", "\\\n")
		return fmt.Sprintf(": %d:0;%s\n", ts.Unix(), safe), nil
	case "bash":
		return fmt.Sprintf("#%d\n%s\n", ts.Unix(), cmd), nil
	case "fish":
		safe := strings.ReplaceAll(cmd, "\\", "\\\\")
		safe = strings.ReplaceAll(safe, "\n", "\\n")
		return fmt.Sprintf("- cmd: %s\n  when: %d\n", safe, ts.Unix()), nil
	}
	return "", fmt.Errorf("unsupported shell: %s", shell)
}

// appendShellHistoryBatch appends a single pre-formatted blob to the history
// file in one open-write-close cycle. Survival across other shells depends on
// every interactive zsh having INC_APPEND_HISTORY (or equivalent) set so that
// none of them ever rewrites the file from a stale in-memory snapshot.
func appendShellHistoryBatch(path, blob string) error {
	if blob == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(blob)
	return err
}
