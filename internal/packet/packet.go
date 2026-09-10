// Package packet loads, saves, and validates packet.yaml.
package packet

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Budget bounds how much retry and time a packet may consume.
type Budget struct {
	Retries int `yaml:"retries"`
	Minutes int `yaml:"minutes"`
}

// Packet is the packet.yaml document.
type Packet struct {
	Goal        string   `yaml:"goal"`
	Context     string   `yaml:"context,omitempty"`
	Constraints []string `yaml:"constraints,omitempty"`
	Terminal    []string `yaml:"terminal"`
	Budget      Budget   `yaml:"budget"`
	CausedBy    *string  `yaml:"caused_by,omitempty"`
}

// Load reads and parses packet.yaml at path.
func Load(path string) (*Packet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("packet: read %s: %s", path, err)
	}
	var p Packet
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("packet: parse %s: %s", path, err)
	}
	return &p, nil
}

// Save writes p as YAML to path with file mode 0600.
func Save(path string, p *Packet) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("packet: marshal: %s", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("packet: write %s: %s", path, err)
	}
	return nil
}

// Template is the starter packet.yaml written by `packets new`.
const Template = `goal: ""                # required, non-empty, single line
context: ""              # optional, multiline
constraints: []          # optional, list of shell commands
terminal:                # required, at least one shell command
  - ""
budget:
  retries: 5              # required, >= 1
  minutes: 60             # required, >= 1
caused_by: null           # optional, another packet slug
`

// humanApproval is the literal terminal entry the harness substitutes with
// `packets check-approved <slug>` at execution time.
const humanApproval = "human_approval"

// Validate checks p against the packet.yaml validation rules and returns
// every failing rule as a human-readable message. packetsDir is the
// directory containing existing packet dirs, used to check caused_by; pass
// "" to skip that check.
func Validate(p *Packet, packetsDir string) []string {
	var failures []string

	if strings.TrimSpace(p.Goal) == "" {
		failures = append(failures, "goal: must not be empty")
	}

	if len(p.Terminal) == 0 {
		failures = append(failures, "terminal: must contain at least one entry")
	}

	failures = append(failures, validateCommands("constraints", p.Constraints)...)
	failures = append(failures, validateCommands("terminal", p.Terminal)...)

	if p.Budget.Retries < 1 {
		failures = append(failures, "budget.retries: must be >= 1")
	}
	if p.Budget.Minutes < 1 {
		failures = append(failures, "budget.minutes: must be >= 1")
	}

	if p.CausedBy != nil && *p.CausedBy != "" && packetsDir != "" {
		if _, err := os.Stat(filepath.Join(packetsDir, *p.CausedBy)); err != nil {
			failures = append(failures, fmt.Sprintf(
				"caused_by: packet %q not found in %s", *p.CausedBy, packetsDir))
		}
	}

	return failures
}

func validateCommands(field string, cmds []string) []string {
	var failures []string
	for i, cmd := range cmds {
		if strings.TrimSpace(cmd) == "" {
			failures = append(failures, fmt.Sprintf("%s[%d]: must not be empty", field, i))
			continue
		}
		if cmd == humanApproval {
			continue
		}
		if err := exec.Command("sh", "-n", "-c", cmd).Run(); err != nil {
			failures = append(failures, fmt.Sprintf(
				"%s[%d]: does not parse as a shell command: %q", field, i, cmd))
		}
	}
	return failures
}
