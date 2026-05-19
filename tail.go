package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func cmdTail(args []string) error {
	fs := flag.NewFlagSet("tail", flag.ExitOnError)
	session := fs.String("s", "", "session id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *session == "" {
		return fmt.Errorf("usage: agent-history tail -s <session-id>")
	}
	path := filepath.Join(logDir(), *session+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(os.Stdout, f)
	return err
}
