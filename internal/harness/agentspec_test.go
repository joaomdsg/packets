package harness

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The agent spec is the standard hardened profile RunContainer runs: it must pass
// the API key by name (secret), route the agent's tools' writable HOME/caches onto
// the read-only rootfs's tmpfs (or they EROFS), and thread the run's identity
// (repo, prompt, seccomp, host uid:gid) through — so a live containerized run is
// correctly hardened AND actually works.
func TestAgentSpec_buildsTheHardenedWritableHomeProfile(t *testing.T) {
	t.Parallel()
	s := agentSpec("/home/lead/proj", "fix the bug", "/tmp/seccomp.json", "1000:1000")

	assert.Equal(t, "/home/lead/proj", s.RepoDir)
	assert.Equal(t, "fix the bug", s.Prompt)
	assert.Equal(t, "/tmp/seccomp.json", s.SeccompPath)
	assert.Equal(t, "1000:1000", s.User)

	assert.Contains(t, s.EnvPassthrough, "ANTHROPIC_API_KEY", "the API key passes by name (secret)")

	home, ok := routeValue(s.RouteEnv, "HOME")
	require.True(t, ok, "HOME must be routed so the agent's tools have a writable home on the read-only rootfs")
	assert.True(t, strings.HasPrefix(home, "/tmp"), "HOME must route to the writable /tmp tmpfs, not a read-only path (got %q)", home)
	cache, ok := routeValue(s.RouteEnv, "GOCACHE")
	require.True(t, ok, "the Go build cache must be routed")
	assert.True(t, strings.HasPrefix(cache, "/tmp"), "GOCACHE must route to the writable /tmp tmpfs (got %q)", cache)

	assert.Greater(t, s.PidsLimit, 0, "a pids cap is set")
	assert.NotEmpty(t, s.Memory, "a memory cap is set")
	assert.NotEmpty(t, s.Image, "the agent image is set")
}

func routeValue(env []EnvVar, name string) (string, bool) {
	for _, e := range env {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}
