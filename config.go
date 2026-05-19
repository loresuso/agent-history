package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type config struct {
	Shell       string
	HistoryPath string
	Filter      *regexp.Regexp
}

type configFile struct {
	Shell       string `json:"shell"`
	HistoryPath string `json:"history_path"`
	Filter      string `json:"filter,omitempty"`
}

// loadOrBootstrapConfig reads the on-disk config, or, on first run, writes a
// sensible default derived from $SHELL and returns it.
func loadOrBootstrapConfig() (*config, error) {
	path := configPath()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cf, werr := bootstrapConfig()
		if werr != nil {
			return nil, werr
		}
		return parseConfig(*cf)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var cf configFile
	if err := json.Unmarshal(raw, &cf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return parseConfig(cf)
}

func parseConfig(cf configFile) (*config, error) {
	if cf.Shell == "" {
		return nil, fmt.Errorf("config missing shell")
	}
	if cf.HistoryPath == "" {
		return nil, fmt.Errorf("config missing history_path")
	}
	cfg := &config{Shell: cf.Shell, HistoryPath: cf.HistoryPath}
	if cf.Filter != "" {
		re, err := regexp.Compile(cf.Filter)
		if err != nil {
			return nil, fmt.Errorf("invalid filter regex %q: %w", cf.Filter, err)
		}
		cfg.Filter = re
	}
	return cfg, nil
}

func bootstrapConfig() (*configFile, error) {
	shell := detectShell()
	if shell == "" {
		return nil, fmt.Errorf("could not detect shell from $SHELL=%q; set it manually in %s", os.Getenv("SHELL"), configPath())
	}
	cf := &configFile{
		Shell:       shell,
		HistoryPath: defaultHistoryPath(shell),
	}
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o700); err != nil {
		return nil, err
	}
	raw, _ := json.MarshalIndent(cf, "", "  ")
	if err := os.WriteFile(configPath(), append(raw, '\n'), 0o600); err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "agent-history: wrote default config to %s\n", configPath())
	return cf, nil
}

func detectShell() string {
	s := os.Getenv("SHELL")
	switch {
	case strings.HasSuffix(s, "/zsh"):
		return "zsh"
	case strings.HasSuffix(s, "/bash"):
		return "bash"
	case strings.HasSuffix(s, "/fish"):
		return "fish"
	}
	return ""
}

func defaultHistoryPath(shell string) string {
	h := homeDir()
	switch shell {
	case "zsh":
		if p := os.Getenv("HISTFILE"); p != "" {
			return p
		}
		return filepath.Join(h, ".zsh_history")
	case "bash":
		if p := os.Getenv("HISTFILE"); p != "" {
			return p
		}
		return filepath.Join(h, ".bash_history")
	case "fish":
		return filepath.Join(h, ".local/share/fish/fish_history")
	}
	return ""
}
