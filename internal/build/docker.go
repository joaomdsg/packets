package build

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/joaomdsg/packets/internal/claudeauth"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/procexec"
)

// Docker is the docker external-process boundary the build node needs
// (§12 step 3, §13.6). Unlike bootstrap.Docker, Run reports the process's
// own exit code: the container's entrypoint is `timeout <n> claude ...`,
// so exit code 124 is how a budget timeout surfaces.
type Docker interface {
	Run(args []string) (output string, exitCode int, err error)
}

// ExecDocker is the real Docker, shelling out to the host's docker binary.
type ExecDocker struct{}

func (ExecDocker) Run(args []string) (string, int, error) {
	return procexec.RunCode("", "docker", args...)
}

// dockerTimeoutExitCode is what GNU coreutils `timeout` exits with when it
// kills the wrapped command.
const dockerTimeoutExitCode = 124

// claudePromptPath is where the CLAUDE.md bind mount lands inside the
// tmpfs-backed $CLAUDE_CONFIG_DIR (docs/claude-code-facts.md: the config
// dir must be writable, so it's reseeded from a read-only image copy at
// container start rather than the whole thing being read-only).
const claudePromptPath = claudeauth.ConfigDir + "/CLAUDE.md"

// containerArgs builds the `docker run` argv per §13.6, adjusted per
// docs/claude-code-facts.md: $CLAUDE_CONFIG_DIR is a second tmpfs (not
// just /tmp), seeded by the image's entrypoint, and CLAUDE.md plus
// whatever claudeauth.Resolve picks (a credentials mount or
// ANTHROPIC_API_KEY passthrough) land inside it. Egress uses the
// bridge-network fallback §13.6 allows when a proxy isn't built for v0;
// each attempt logs egress_unrestricted once (see build.go).
func containerArgs(fab *fabric.Config, getenv func(string) string, exists func(string) bool, slug string, attempt, timeoutSeconds int, runDir string) ([]string, error) {
	authArgs, err := claudeauth.Resolve(fab.CredentialsPath, getenv, exists)
	if err != nil {
		return nil, fmt.Errorf("build: %s", err)
	}

	name := fmt.Sprintf("packets-%s-%d", slug, attempt)
	args := []string{
		"run", "--rm",
		"--name", name,
		"--network", "bridge",
		"-e", fmt.Sprintf("MAX_FILES=%d", fab.SmellTriggers.MaxFilesTouched),
		"-v", fab.RepoPath + ":/work:rw",
		"-v", filepath.Join(runDir, "CLAUDE.md") + ":" + claudePromptPath + ":ro",
		"-v", filepath.Join(runDir, "signals") + ":/signals:rw",
		"--user", "1000",
		"--read-only",
		"--tmpfs", "/tmp",
	}
	args = append(args, authArgs...)
	args = append(args,
		fab.Image,
		"timeout", fmt.Sprintf("%d", timeoutSeconds),
		"claude", "-p", fmt.Sprintf("Read %s and complete the packet.", claudePromptPath),
		"--output-format", "json", "--max-turns", "200",
	)
	return args, nil
}

// claudeUsage is the subset of `claude -p --output-format json`'s result
// this harness records (docs/claude-code-facts.md overrides §12 step 3's
// assumption of a top-level "tokens" field).
//
// IsError is the only reliable failure signal: an API-level failure (e.g.
// credit exhaustion) can still report subtype "success" and exit code 0,
// so subtype must never be used to detect this.
type claudeUsage struct {
	IsError        bool    `json:"is_error"`
	APIErrorStatus int     `json:"api_error_status"`
	TerminalReason string  `json:"terminal_reason"`
	Result         string  `json:"result"`
	TotalCostUSD   float64 `json:"total_cost_usd"`
	Usage          struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// parseClaudeUsage extracts the trailing JSON object from raw container
// output and reads its token/cost accounting. A parse failure returns a
// zero claudeUsage and the error; callers treat it as "nothing to add",
// not fatal, since a timed-out or halted run may produce no JSON at all.
func parseClaudeUsage(raw string) (claudeUsage, string, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return claudeUsage{}, "", fmt.Errorf("build: no JSON object found in claude output")
	}
	blob := raw[start : end+1]
	var u claudeUsage
	if err := json.Unmarshal([]byte(blob), &u); err != nil {
		return claudeUsage{}, "", fmt.Errorf("build: parse claude output: %s", err)
	}
	return u, blob, nil
}
