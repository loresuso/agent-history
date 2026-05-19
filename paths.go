package main

import (
	"os"
	"path/filepath"
)

// XDG-honoring paths. Falls back to ~/.config and ~/.local/share on Linux/macOS.
// On macOS we intentionally use the same XDG defaults rather than
// ~/Library/Application Support, to keep one mental model across platforms.

func configDir() string {
	if p := os.Getenv("XDG_CONFIG_HOME"); p != "" {
		return filepath.Join(p, "agent-history")
	}
	return filepath.Join(homeDir(), ".config", "agent-history")
}

func dataDir() string {
	if p := os.Getenv("XDG_DATA_HOME"); p != "" {
		return filepath.Join(p, "agent-history")
	}
	return filepath.Join(homeDir(), ".local", "share", "agent-history")
}

func configPath() string { return filepath.Join(configDir(), "config.json") }
func logDir() string     { return filepath.Join(dataDir(), "log") }

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}
