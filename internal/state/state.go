// Package state loads, saves, and transitions a packet's state.json.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Values of State.State.
const (
	Draft            = "draft"
	Emitted          = "emitted"
	Building         = "building"
	CI               = "ci"
	Halted           = "halted"
	AwaitingApproval = "awaiting_approval"
	Terminated       = "terminated"
	Killed           = "killed"
)

// Values of State.Node.
const (
	NodeBuild = "build"
	NodeCI    = "ci"
)

// State is the state.json document.
type State struct {
	Slug        string    `json:"slug"`
	Version     int       `json:"version"`
	State       string    `json:"state"`
	Node        *string   `json:"node"`
	Attempt     int       `json:"attempt"`
	RetriesUsed int       `json:"retries_used"`
	MinutesUsed float64   `json:"minutes_used"`
	TokensUsed  int       `json:"tokens_used"`
	Branch      string    `json:"branch"`
	PRNumber    int       `json:"pr_number"`
	Approved    bool      `json:"approved"`
	HaltReason  *string   `json:"halt_reason"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// GateSuggestedTerminals records terminal predicates the emit gate
	// suggested and the human accepted (§8 Stage B), so provenance survives
	// independent of the packet.yaml's terminal list.
	GateSuggestedTerminals []string `json:"gate_suggested_terminals"`
}

// Load reads and parses state.json at path.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("state: read %s: %s", path, err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("state: parse %s: %s", path, err)
	}
	return &s, nil
}

// LoadForSlug loads state.json at path and confirms its slug field matches
// the directory it was loaded from. A mismatch means the packet dir was
// renamed or its state.json copied from elsewhere; trusting the directory
// name over the file would silently operate on the wrong packet.
func LoadForSlug(path, slug string) (*State, error) {
	s, err := Load(path)
	if err != nil {
		return nil, err
	}
	if s.Slug != slug {
		return nil, fmt.Errorf(
			"state: %s: directory slug %q does not match state.json slug %q",
			path, slug, s.Slug)
	}
	return s, nil
}

// Save atomically writes s to path: it writes path+".tmp", fsyncs, then
// renames over path. A crash between the write and the rename leaves the
// prior contents of path untouched.
func Save(path string, s *State) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("state: marshal: %s", err)
	}

	tmpPath := path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("state: open %s: %s", tmpPath, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("state: write %s: %s", tmpPath, err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("state: sync %s: %s", tmpPath, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("state: close %s: %s", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("state: rename %s to %s: %s", tmpPath, path, err)
	}
	return nil
}
