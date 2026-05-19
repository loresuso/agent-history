package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// cmdSearch scans every per-session JSONL audit log under logDir() and
// prints matching commands. Default output is tab-separated:
//
//	<ts>\t<command>
//
// so it composes with fzf, cut, awk, etc. --jsonl emits the raw records.
//
// Most-recent-first by timestamp, mirroring the Ctrl+R mental model.
func cmdSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	sessionFlag := fs.String("s", "", "filter to records whose session id contains this substring")
	limit := fs.Int("n", 0, "max records to print (0 = unlimited)")
	asJSONL := fs.Bool("jsonl", false, "emit raw JSONL records instead of tab-separated output")
	withCWD := fs.Bool("with-cwd", false, "include the cwd column (tab-separated default only)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	query := strings.ToLower(strings.Join(fs.Args(), " "))

	dir := logDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", dir, err)
	}

	var hits []logRecord
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		if *sessionFlag != "" && !strings.Contains(name, *sessionFlag) {
			continue
		}
		recs, err := readSessionLog(filepath.Join(dir, name))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", name, err)
			continue
		}
		for _, r := range recs {
			if query != "" && !strings.Contains(strings.ToLower(r.Command), query) {
				continue
			}
			hits = append(hits, r)
		}
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].TS > hits[j].TS })
	if *limit > 0 && len(hits) > *limit {
		hits = hits[:*limit]
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	if *asJSONL {
		for _, r := range hits {
			b, err := json.Marshal(r)
			if err != nil {
				continue
			}
			w.Write(b)
			w.WriteByte('\n')
		}
		return nil
	}

	for _, r := range hits {
		cmd := strings.ReplaceAll(r.Command, "\n", " ⏎ ")
		if *withCWD {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.TS, r.CWD, cmd)
		} else {
			fmt.Fprintf(w, "%s\t%s\n", r.TS, cmd)
		}
	}
	return nil
}

func readSessionLog(path string) ([]logRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []logRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var r logRecord
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			continue
		}
		out = append(out, r)
	}
	if err := sc.Err(); err != nil && err != io.EOF {
		return out, err
	}
	return out, nil
}
