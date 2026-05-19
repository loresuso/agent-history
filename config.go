package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type config struct {
	Filter *regexp.Regexp
}

type configFile struct {
	Filter string `json:"filter,omitempty"`
}

// loadOrBootstrapConfig reads the on-disk config, or, on first run, writes an
// empty default and returns it. The config is optional — it currently only
// holds an opt-in regex of commands to skip on capture.
func loadOrBootstrapConfig() (*config, error) {
	path := configPath()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := bootstrapConfig(); err != nil {
			return nil, err
		}
		return &config{}, nil
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
	cfg := &config{}
	if cf.Filter != "" {
		re, err := regexp.Compile(cf.Filter)
		if err != nil {
			return nil, fmt.Errorf("invalid filter regex %q: %w", cf.Filter, err)
		}
		cfg.Filter = re
	}
	return cfg, nil
}

func bootstrapConfig() error {
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(configFile{}, "", "  ")
	if err := os.WriteFile(configPath(), append(raw, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "agent-history: wrote default config to %s\n", configPath())
	return nil
}
