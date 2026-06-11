package harness_test

import (
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
	require.NotEmpty(t, envs, "an -e env passthrough must be present")
	assert.Contains(t, envs, "ANTHROPIC_API_KEY", "the API key is passed through by name")
	for _, e := range envs {
		assert.NotContains(t, e, "=", "EVERY -e value is a bare NAME, never NAME=VALUE — no secret reaches argv")
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
