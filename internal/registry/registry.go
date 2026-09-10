// Package registry appends and detects entries in registry.jsonl, the
// fabric-level record of accretions: permanent tests/checks a packet
// added to the repo.
package registry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Cause values, exact strings per the accretion registry spec.
const (
	CauseGateSuggested = "gate_suggested"
	CauseCIFailure     = "ci_failure"
	CauseProposal      = "proposal"
)

// Entry is one registry.jsonl line.
type Entry struct {
	Timestamp string `json:"ts"`
	Fabric    string `json:"fabric"`
	Packet    string `json:"packet"`
	Version   int    `json:"version"`
	Path      string `json:"path"`
	Cause     string `json:"cause"`
	Commit    string `json:"commit"`
}

// Append writes e as one JSON line to the registry.jsonl at path, creating
// the file if it does not exist. Used for all three causes; approve-
// proposal (cause=proposal) is Phase 6 and does not call this yet.
func Append(path string, e Entry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("registry: marshal entry: %s", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("registry: open %s: %s", path, err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("registry: write %s: %s", path, err)
	}
	return nil
}

// ReadAll parses every line of the registry.jsonl at path. A missing file
// reads as no entries, the normal state before any accretion has merged.
func ReadAll(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("registry: open %s: %s", path, err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("registry: parse %s: %s", path, err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("registry: read %s: %s", path, err)
	}
	return entries, nil
}

// ReadAllTolerant is ReadAll but skips lines that fail to parse instead of
// failing outright, returning how many were skipped. A torn last line is
// exactly what a crash mid-append leaves behind; callers that need to
// report metrics from a fabric's full history (packets report) cannot let
// one such line make every line unreadable.
func ReadAllTolerant(path string) ([]Entry, int, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("registry: open %s: %s", path, err)
	}
	defer f.Close()

	var entries []Entry
	var skipped int
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			skipped++
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, skipped, fmt.Errorf("registry: read %s: %s", path, err)
	}
	return entries, skipped, nil
}
