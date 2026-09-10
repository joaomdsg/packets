// Package fabric loads and saves fabric.yaml, the per-repo configuration
// under ~/.local/share/packets/<fabric-slug>/fabric.yaml.
package fabric

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// BudgetDefaults seeds a new packet's budget when none is specified.
type BudgetDefaults struct {
	Retries int `yaml:"retries"`
	Minutes int `yaml:"minutes"`
}

// SmellTriggers configures when the harness halts a build for smell.
type SmellTriggers struct {
	MaxFilesTouched      int  `yaml:"max_files_touched"`
	SameErrorRepeats     int  `yaml:"same_error_repeats"`
	DenyTestModification bool `yaml:"deny_test_modification"`
	DenyDependencyAdd    bool `yaml:"deny_dependency_add"`
	DenyTerminalEdit     bool `yaml:"deny_terminal_edit"`
}

// CI configures how the harness polls the fabric's CI workflow.
type CI struct {
	PollSeconds    int    `yaml:"poll_seconds"`
	TimeoutMinutes int    `yaml:"timeout_minutes"`
	WorkflowName   string `yaml:"workflow_name"`
}

// Merge configures automatic merge behavior on green CI.
type Merge struct {
	Auto     bool   `yaml:"auto"`
	Strategy string `yaml:"strategy"`
}

// Config is the fabric.yaml document.
type Config struct {
	Slug          string `yaml:"slug"`
	Remote        string `yaml:"remote"`
	RepoPath      string `yaml:"repo_path"`
	DefaultBranch string `yaml:"default_branch"`
	Image         string `yaml:"image"`
	// CredentialsPath pins the containerized agent's Claude Code OAuth
	// credentials file; empty means resolve via CLAUDE_CONFIG_DIR/HOME or
	// ANTHROPIC_API_KEY (internal/claudeauth).
	CredentialsPath string         `yaml:"credentials_path,omitempty"`
	BudgetDefaults  BudgetDefaults `yaml:"budget_defaults"`
	SmellTriggers   SmellTriggers  `yaml:"smell_triggers"`
	CI              CI             `yaml:"ci"`
	Merge           Merge          `yaml:"merge"`
}

// Load reads and parses fabric.yaml at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fabric: read %s: %s", path, err)
	}
	if len(data) == 0 {
		// yaml.Unmarshal treats an empty document as a no-op, not an
		// error, which would otherwise silently yield a zero-value Config.
		return nil, fmt.Errorf("fabric: %s is empty", path)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("fabric: parse %s: %s", path, err)
	}
	return &cfg, nil
}

// Save writes cfg as YAML to path with file mode 0600.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("fabric: marshal: %s", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("fabric: write %s: %s", path, err)
	}
	return nil
}
