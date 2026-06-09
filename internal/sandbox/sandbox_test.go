package sandbox

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConform_acceptsTheCanonicalHardenedLaunch(t *testing.T) {
	t.Parallel()
	args := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}})
	require.NoError(t, conform(args), "the single enforced launch path must be conformant by construction")
}

// The launch carries --rm and a cleanup label, so the reaped-after assertion in
// the integration test is non-vacuous (an empty label filter genuinely means the
// one-shot container was reaped, not that no labeled container ever existed).
func TestHardenedArgs_tagsTheOneShotContainerForReapVerification(t *testing.T) {
	t.Parallel()
	args := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}})
	assert.Contains(t, args, "--rm")
	assert.Contains(t, args, "--label=io.packets.sandbox=1")
}

// The launch is FAIL-CLOSED: any launch missing a non-negotiable lock or
// carrying a forbidden flag must be refused before it ever reaches Docker. A
// misconfigured hardened container is a plain container.
func TestConform_refusesATamperedLaunch(t *testing.T) {
	t.Parallel()
	base := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}})
	tests := []struct {
		name string
		args []string
	}{
		{"missing no-egress", without(base, "--network=none")},
		{"missing cap-drop", without(base, "--cap-drop=ALL")},
		{"missing no-new-privileges", without(base, "--security-opt=no-new-privileges")},
		{"missing read-only", without(base, "--read-only")},
		{"missing pids-limit", without(base, "--pids-limit=128")},
		{"missing memory cap", without(base, "--memory=256m")},
		{"missing non-root user", without(base, "--user=65534:65534")},
		{"privileged", append(clone(base), "--privileged")},
		{"host network --network=host", append(clone(base), "--network=host")},
		{"host network --net=host", append(clone(base), "--net=host")},
		{"pid host", append(clone(base), "--pid=host")},
		{"docker.sock via -v", append(clone(base), "-v", "/var/run/docker.sock:/var/run/docker.sock")},
		{"docker.sock via --mount=", append(clone(base), "--mount=type=bind,src=/var/run/docker.sock,dst=/sock")},
		{"host bind mount -v=", append(clone(base), "-v=/etc:/host-etc")},
		{"host bind mount --volume=", append(clone(base), "--volume=/etc:/host-etc")},
		{"host bind mount --volume exact", append(clone(base), "--volume", "/etc:/host-etc")},
		{"host bind mount --mount=", append(clone(base), "--mount=type=bind,src=/etc,dst=/host-etc")},
		{"docker.sock as bare reference", append(clone(base), "/var/run/docker.sock")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, conform(tt.args), "a %s launch must be refused", tt.name)
		})
	}
}

func TestDockerRunner_runsAndReapsAHardenedContainer(t *testing.T) {
	requireDocker(t)
	res, err := DockerRunner{}.Run(context.Background(), Spec{Image: "busybox:latest", Cmd: []string{"true"}})
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode, "a trivial command must run to a clean exit inside the hardened container")

	// Ephemeral: the one-shot container leaves nothing behind.
	left := dockerPS(t, "label=io.packets.sandbox=1")
	assert.Empty(t, left, "the one-shot container must be reaped, leaving no leftover: %q", left)
}

func TestDockerRunner_reportsANonZeroExitAsAResultNotAnError(t *testing.T) {
	requireDocker(t)
	// A container that ran and exited non-zero is a Result (the verdict layer
	// reads it), not a runtime-invocation error.
	res, err := DockerRunner{}.Run(context.Background(), Spec{Image: "busybox:latest", Cmd: []string{"false"}})
	require.NoError(t, err)
	assert.NotEqual(t, 0, res.ExitCode, "a non-zero container exit must surface as a non-zero ExitCode")
}

func clone(s []string) []string { return append([]string(nil), s...) }

func without(s []string, drop string) []string {
	out := make([]string, 0, len(s))
	for _, a := range s {
		if a != drop {
			out = append(out, a)
		}
	}
	return out
}

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker not available; skipping real-container enforcement test")
	}
}

func dockerPS(t *testing.T, filter string) string {
	t.Helper()
	out, err := exec.Command("docker", "ps", "-aq", "--filter", filter).Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}
