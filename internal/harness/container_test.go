package harness_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/harness"
)

func sampleContainerSpec() harness.ContainerSpec {
	return harness.ContainerSpec{
		Image:          "packets-agent:latest",
		RepoDir:        "/home/lead/project",
		Prompt:         "fix the bug at auth.go:42",
		SeccompPath:    "/tmp/seccomp.json",
		User:           "1000:1000",
		EnvPassthrough: []string{"ANTHROPIC_API_KEY"},
		PidsLimit:      256,
		Memory:         "2g",
	}
}

func argAfter(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func valuesAfter(args []string, flag string) []string {
	var vs []string
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			vs = append(vs, args[i+1])
		}
	}
	return vs
}

// The agent container must be HARDENED even though it is trusted: a compromised
// or runaway harness still runs with dropped caps, no privilege escalation, a
// seccomp filter, a read-only rootfs, and bounded pids/memory — so the blast
// radius is the repo workdir, never the host.
func TestContainerArgs_appliesTheHardenedAgentProfile(t *testing.T) {
	t.Parallel()
	args := harness.ContainerArgs(sampleContainerSpec())

	assert.Equal(t, []string{"docker", "run", "--rm"}, args[:3], "a one-shot docker run")
	assert.Contains(t, args, "--cap-drop=ALL")
	assert.Contains(t, args, "--security-opt=no-new-privileges")
	assert.Contains(t, args, "--security-opt=seccomp=/tmp/seccomp.json")
	assert.Contains(t, args, "--read-only", "rootfs is read-only")
	assert.Contains(t, args, "--tmpfs=/tmp", "a writable /tmp since the rootfs is read-only")
	assert.Contains(t, args, "--pids-limit=256")
	assert.Contains(t, args, "--memory=2g")
	assert.Contains(t, args, "--user=1000:1000", "runs as the host uid:gid so writes to the bind-mounted repo are host-owned")
}

// The defining difference from the verification cage: the agent NEEDS egress to
// the Anthropic API, so the container must NOT be network-isolated. If this
// regresses to --network=none, every live run silently fails to reach the model.
func TestContainerArgs_allowsEgressUnlikeTheVerificationCage(t *testing.T) {
	t.Parallel()
	args := harness.ContainerArgs(sampleContainerSpec())
	assert.NotContains(t, args, "--network=none", "the agent needs the API — egress must NOT be disabled")
	for _, a := range args {
		assert.NotContains(t, a, "network=none", "no network isolation flag in any form")
	}
}

// The repo is bind-mounted WRITABLE at /work (no :ro) so the agent's edits and
// commits land directly on the host repo — no copy, no loss on teardown.
func TestContainerArgs_bindMountsTheRepoWritableAtWork(t *testing.T) {
	t.Parallel()
	args := harness.ContainerArgs(sampleContainerSpec())
	binds := valuesAfter(args, "-v")
	require.Len(t, binds, 1, "the repo is the ONLY bind mount — no extra writable host paths leak in")
	assert.Equal(t, "/home/lead/project:/work", binds[0], "the repo is mounted writable (no :ro suffix)")
	assert.NotContains(t, binds[0], ":ro", "the repo mount must be writable")
	w, ok := argAfter(args, "-w")
	require.True(t, ok)
	assert.Equal(t, "/work", w, "the working dir is the mounted repo")
}

// The API key is passed by NAME only (-e ANTHROPIC_API_KEY), never as a value, so
// it can't leak into `ps`/argv. The builder never even takes the secret value.
func TestContainerArgs_passesSecretsByNameNeverByValue(t *testing.T) {
	t.Parallel()
	args := harness.ContainerArgs(sampleContainerSpec())
	envs := valuesAfter(args, "-e")
	require.Contains(t, envs, "ANTHROPIC_API_KEY", "the API key is passed through by bare NAME")
	for _, e := range envs {
		assert.False(t, strings.HasPrefix(e, "ANTHROPIC_API_KEY="),
			"the secret must NEVER appear as NAME=VALUE (its value would leak into argv/ps)")
	}
	// Structural guarantee: the builder takes EnvPassthrough as NAMES ([]string), so
	// it never even holds a secret value to leak. (RouteEnv carries NON-secret
	// NAME=VALUE routing — covered by the read-only-rootfs test.)
}

// The rootfs is read-only, so the agent's tools (claude/git/go/node) would EROFS
// writing to $HOME/caches unless those route to the writable /tmp. RouteEnv carries
// that NON-secret routing as -e NAME=VALUE, kept distinct from the by-name secret
// passthrough (which stays bare).
func TestContainerArgs_routesWritableHomeAndCachesForTheReadOnlyRootfs(t *testing.T) {
	t.Parallel()
	s := sampleContainerSpec()
	s.RouteEnv = []harness.EnvVar{{Name: "HOME", Value: "/tmp"}, {Name: "GOCACHE", Value: "/tmp/go"}}
	args := harness.ContainerArgs(s)

	envs := valuesAfter(args, "-e")
	assert.Contains(t, envs, "HOME=/tmp", "HOME routes to the writable /tmp despite the read-only rootfs")
	assert.Contains(t, envs, "GOCACHE=/tmp/go", "the Go cache routes to a writable path")
	assert.Contains(t, envs, "ANTHROPIC_API_KEY", "the secret stays a bare by-name passthrough, distinct from the routing")
	for _, e := range envs {
		assert.False(t, strings.HasPrefix(e, "ANTHROPIC_API_KEY="), "the secret is never NAME=VALUE even alongside RouteEnv")
	}
}

// The container must never get the host's docker socket — that would be a trivial
// container escape (the agent could launch unconstrained sibling containers).
func TestContainerArgs_neverMountsTheDockerSocket(t *testing.T) {
	t.Parallel()
	args := harness.ContainerArgs(sampleContainerSpec())
	for _, a := range args {
		assert.NotContains(t, a, "docker.sock", "the docker socket must never be exposed to the agent")
	}
}

// The container runs the real harness command: `claude` with the headless
// streaming flags (slice 4c-i's ClaudeArgs), so the supervisor reduces the same
// stream-json whether the agent runs host-side or in a box.
func TestContainerArgs_runsClaudeWithTheHeadlessStreamingFlags(t *testing.T) {
	t.Parallel()
	args := harness.ContainerArgs(sampleContainerSpec())

	i := -1
	for j, a := range args {
		if a == "packets-agent:latest" {
			i = j
			break
		}
	}
	require.GreaterOrEqual(t, i, 0, "the image must appear in the argv")
	tail := args[i+1:]
	require.NotEmpty(t, tail, "the image is followed by the command")
	assert.Equal(t, "claude", tail[0], "the container runs claude")
	assert.Equal(t, harness.ClaudeArgs("fix the bug at auth.go:42"), tail[1:], "with the exact headless-streaming ClaudeArgs")
	assert.Contains(t, tail, "stream-json")
	assert.Contains(t, tail, "--verbose")
}
