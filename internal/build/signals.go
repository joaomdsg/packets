package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"gopkg.in/yaml.v3"
)

// predicateSpec is one entry hooks/stop.sh reads from
// /signals/predicates.yaml (§13.5, §13.6).
type predicateSpec struct {
	Cmd   string `yaml:"cmd"`
	Kind  string `yaml:"kind"`
	IsNew bool   `yaml:"is_new"`
}

const (
	kindConstraint = "constraint"
	kindTerminal   = "terminal"
)

// seedSignals clears signalsDir and writes predicates.yaml and
// retries_left, the two files §13.6 requires present before the container
// starts.
func seedSignals(signalsDir string, pkt *packet.Packet, st *state.State) error {
	if err := os.RemoveAll(signalsDir); err != nil {
		return fmt.Errorf("build: clear %s: %s", signalsDir, err)
	}
	if err := os.MkdirAll(signalsDir, 0o700); err != nil {
		return fmt.Errorf("build: create %s: %s", signalsDir, err)
	}

	var specs []predicateSpec
	for _, c := range pkt.Constraints {
		specs = append(specs, predicateSpec{Cmd: substituteApproval(st.Slug, c), Kind: kindConstraint})
	}
	for _, t := range pkt.Terminal {
		specs = append(specs, predicateSpec{
			Cmd:   substituteApproval(st.Slug, t),
			Kind:  kindTerminal,
			IsNew: isGateSuggested(st, t),
		})
	}

	data, err := yaml.Marshal(specs)
	if err != nil {
		return fmt.Errorf("build: marshal predicates.yaml: %s", err)
	}
	if err := os.WriteFile(filepath.Join(signalsDir, "predicates.yaml"), data, 0o600); err != nil {
		return fmt.Errorf("build: write predicates.yaml: %s", err)
	}

	retriesLeft := pkt.Budget.Retries - st.RetriesUsed
	if err := os.WriteFile(filepath.Join(signalsDir, "retries_left"),
		[]byte(strconv.Itoa(retriesLeft)), 0o600); err != nil {
		return fmt.Errorf("build: write retries_left: %s", err)
	}
	return nil
}

// readHaltMD returns the contents of /signals/halt.md and whether it
// exists.
func readHaltMD(signalsDir string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(signalsDir, "halt.md"))
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("build: read halt.md: %s", err)
	}
	return string(data), true, nil
}

// haltReason is halt.md's first line, the value state.json.halt_reason
// takes.
func haltReason(haltMD string) string {
	line, _, _ := strings.Cut(haltMD, "\n")
	return strings.TrimSpace(line)
}

// predicateResult is one entry hooks/stop.sh appends to
// /signals/predicates.json (§13.5, §10 local_pred detail shape).
type predicateResult struct {
	Cmd  string `json:"cmd"`
	Exit int    `json:"exit"`
	Kind string `json:"kind"`
}

// readPredicateResults parses /signals/predicates.json. A missing file is
// not an error: the container may have halted or timed out before
// hooks/stop.sh ever ran.
func readPredicateResults(signalsDir string) ([]predicateResult, error) {
	data, err := os.ReadFile(filepath.Join(signalsDir, "predicates.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("build: read predicates.json: %s", err)
	}
	var results []predicateResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("build: parse predicates.json: %s", err)
	}
	return results, nil
}

// harvestProposals copies every /signals/proposal-*.md into
// packetDir/proposals, preserving each file's basename, and returns the
// basenames copied so the caller can log one journal entry per file
// (§13.7).
func harvestProposals(signalsDir, packetDir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(signalsDir, "proposal-*.md"))
	if err != nil {
		return nil, fmt.Errorf("build: glob proposals: %s", err)
	}
	if len(matches) == 0 {
		return nil, nil
	}

	proposalsDir := filepath.Join(packetDir, "proposals")
	if err := os.MkdirAll(proposalsDir, 0o700); err != nil {
		return nil, fmt.Errorf("build: create %s: %s", proposalsDir, err)
	}

	var copied []string
	for _, src := range matches {
		data, err := os.ReadFile(src)
		if err != nil {
			return nil, fmt.Errorf("build: read %s: %s", src, err)
		}
		name := filepath.Base(src)
		dest := filepath.Join(proposalsDir, name)
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return nil, fmt.Errorf("build: write %s: %s", dest, err)
		}
		copied = append(copied, name)
	}
	return copied, nil
}
