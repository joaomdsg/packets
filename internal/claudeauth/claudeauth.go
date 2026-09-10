// Package claudeauth resolves how the containerized agent authenticates
// with Claude Code, shared by `packets init`'s smoke test and the build
// node's container run so the two never drift out of sync
// (docs/claude-code-facts.md).
package claudeauth

import (
	"fmt"
	"path/filepath"
)

// ConfigDir is Claude Code's config dir inside the container, seeded at
// container start by the image's entrypoint from a read-only baked-in
// copy (docs/claude-code-facts.md).
const ConfigDir = "/home/agent/.claude-config"

// TmpfsMount is the `--tmpfs` flag value for ConfigDir. uid=1000,gid=1000
// matters because the entrypoint reseeds this directory as the agent
// user (uid 1000); docker's default root-owned tmpfs makes that mkdir/cp
// fail.
const TmpfsMount = ConfigDir + ":uid=1000,gid=1000"

const credentialsFileName = ".credentials.json"

// Resolve picks how the containerized agent authenticates and returns the
// extra `docker run` flags implementing it, in this precedence:
//
//  1. explicitPath, if non-empty (a fabric.yaml-pinned credentials path);
//  2. $CLAUDE_CONFIG_DIR/.credentials.json, else ~/.claude/.credentials.json,
//     if it exists;
//  3. ANTHROPIC_API_KEY, if set;
//  4. otherwise an error naming both options.
//
// getenv and exists are injected so callers, and their tests, never
// depend on the real environment or filesystem.
func Resolve(explicitPath string, getenv func(string) string, exists func(string) bool) ([]string, error) {
	path := explicitPath
	if path != "" && !exists(path) {
		return nil, fmt.Errorf("claudeauth: configured credentials_path %s does not exist", path)
	}
	if path == "" {
		dir := getenv("CLAUDE_CONFIG_DIR")
		if dir == "" {
			dir = filepath.Join(getenv("HOME"), ".claude")
		}
		candidate := filepath.Join(dir, credentialsFileName)
		if exists(candidate) {
			path = candidate
		}
	}
	if path != "" {
		return []string{
			"--tmpfs", TmpfsMount,
			"-v", path + ":" + ConfigDir + "/" + credentialsFileName + ":ro",
		}, nil
	}
	if getenv("ANTHROPIC_API_KEY") != "" {
		return []string{"--tmpfs", TmpfsMount, "-e", "ANTHROPIC_API_KEY"}, nil
	}
	return nil, fmt.Errorf("claudeauth: no credentials found; set credentials_path in " +
		"fabric.yaml or place OAuth credentials at $CLAUDE_CONFIG_DIR/.credentials.json " +
		"(default ~/.claude/.credentials.json), or set ANTHROPIC_API_KEY")
}
