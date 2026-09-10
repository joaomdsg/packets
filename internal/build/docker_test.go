package build_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/build"
	"github.com/joaomdsg/packets/internal/claudeauth"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_containerRunMountsCredentialsFileWhenPresentInsteadOfAPIKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "fix-the-login-timeout", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)

	credDir := t.TempDir()
	credPath := filepath.Join(credDir, ".credentials.json")
	require.NoError(t, os.WriteFile(credPath, []byte("placeholder"), 0o600))

	deps, _, _, docker := newStubDeps()
	deps.Getenv = func(key string) string {
		if key == "CLAUDE_CONFIG_DIR" {
			return credDir
		}
		return ""
	}
	deps.Exists = func(p string) bool { return p == credPath }

	_, err := build.Run(deps, fab, pkt, st, dir)
	require.NoError(t, err)

	require.Len(t, docker.calls, 1)
	assert.Contains(t, docker.calls[0], "--tmpfs")
	assert.Contains(t, docker.calls[0], claudeauth.TmpfsMount)
	assert.Contains(t, docker.calls[0], credPath+":"+claudeauth.ConfigDir+"/.credentials.json:ro")
	assert.NotContains(t, docker.calls[0], "ANTHROPIC_API_KEY")
}

func TestRun_containerRunPrefersFabricPinnedCredentialsPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	credPath := filepath.Join(t.TempDir(), "pinned.json")
	require.NoError(t, os.WriteFile(credPath, []byte("placeholder"), 0o600))
	fab.CredentialsPath = credPath
	pkt := testPacket()
	st := &state.State{Slug: "fix-the-login-timeout", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)

	deps, _, _, docker := newStubDeps()
	deps.Exists = func(p string) bool { return p == credPath }

	_, err := build.Run(deps, fab, pkt, st, dir)
	require.NoError(t, err)

	require.Len(t, docker.calls, 1)
	assert.Contains(t, docker.calls[0], credPath+":"+claudeauth.ConfigDir+"/.credentials.json:ro")
}

func TestRun_errorsClearlyWhenNeitherCredentialsFileNorAPIKeyAvailable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "fix-the-login-timeout", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)

	deps, _, _, docker := newStubDeps()
	deps.Getenv = func(string) string { return "" }
	deps.Exists = func(string) bool { return false }

	_, err := build.Run(deps, fab, pkt, st, dir)

	assert.ErrorContains(t, err, "credentials_path")
	assert.ErrorContains(t, err, "ANTHROPIC_API_KEY")
	assert.Empty(t, docker.calls, "must not invoke docker run when no credentials are resolvable")
}
