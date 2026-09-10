package bootstrap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/bootstrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_writesImageAssetsWithExpectedContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module x\n"), 0o600))

	deps := stubDeps(t)
	_, err := bootstrap.Init(deps, bootstrap.InitOptions{
		RepoDir:   repo,
		ConfigDir: filepath.Join(dir, "config"),
		DataDir:   filepath.Join(dir, "data"),
	})
	require.NoError(t, err)

	fabricDir := filepath.Join(dir, "data", "example")

	dockerfile, err := os.ReadFile(filepath.Join(fabricDir, "Dockerfile"))
	require.NoError(t, err)
	assert.Contains(t, string(dockerfile), "FROM golang:1.22")
	assert.Contains(t, string(dockerfile), "CLAUDE_CONFIG_DIR")
	assert.NotContains(t, string(dockerfile), "{{BASE}}")

	settings, err := os.ReadFile(filepath.Join(fabricDir, "settings.json"))
	require.NoError(t, err)
	assert.Contains(t, string(settings), "/home/agent/.claude-config/hooks/pre.sh")
	assert.Contains(t, string(settings), "/home/agent/.claude-config/hooks/post.sh")
	assert.Contains(t, string(settings), "/home/agent/.claude-config/hooks/stop.sh")

	for _, hook := range []string{"pre.sh", "post.sh", "stop.sh"} {
		info, err := os.Stat(filepath.Join(fabricDir, "hooks", hook))
		require.NoError(t, err)
		assert.NotZero(t, info.Mode()&0o100, "hook %s must be executable", hook)
	}

	entrypoint, err := os.ReadFile(filepath.Join(fabricDir, "entrypoint.sh"))
	require.NoError(t, err)
	assert.Contains(t, string(entrypoint), "CLAUDE_CONFIG_DIR")
}
