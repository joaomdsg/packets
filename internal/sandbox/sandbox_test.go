package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProfile is a stand-in seccomp profile path for the pure conform/hardenedArgs
// unit tests (conform checks the flag, never reads the file).
const testProfile = "/tmp/packets-test-seccomp.json"

func TestConform_acceptsTheCanonicalHardenedLaunch(t *testing.T) {
	t.Parallel()
	args := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)
	require.NoError(t, conform(args), "the single enforced launch path must be conformant by construction")
}

// The launch carries --rm and a cleanup label, so the reaped-after assertion in
// the integration test is non-vacuous (an empty label filter genuinely means the
// one-shot container was reaped, not that no labeled container ever existed).
func TestHardenedArgs_tagsTheOneShotContainerForReapVerification(t *testing.T) {
	t.Parallel()
	args := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)
	assert.Contains(t, args, "--rm")
	assert.Contains(t, args, "--label=io.packets.sandbox=1")
}

// The launch is FAIL-CLOSED: any launch missing a non-negotiable lock or
// carrying a forbidden flag must be refused before it ever reaches Docker. A
// misconfigured hardened container is a plain container.
func TestConform_refusesATamperedLaunch(t *testing.T) {
	t.Parallel()
	base := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)
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
		{"missing seccomp profile", without(base, "--security-opt=seccomp="+testProfile)},
		{"seccomp unconfined", append(clone(base), "--security-opt=seccomp=unconfined")},
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

// The cage must mount its verification inputs (the checked-out repo, the module
// cache) READ-ONLY. conform admits exactly that — a read-only bind mount of a
// non-sensitive source — and nothing more: a writable mount, a docker.sock
// mount, a sensitive host source, or the -v short form are all refused.
func TestConform_admitsReadOnlyInputMountsButRefusesDangerousOnes(t *testing.T) {
	t.Parallel()
	base := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)

	require.NoError(t,
		conform(append(clone(base), "--mount=type=bind,source=/tmp/pkts-work,target=/work,readonly")),
		"a read-only bind mount of a non-sensitive source is the cage's verification input")

	bad := []struct{ name, mount string }{
		{"readonly docker.sock", "--mount=type=bind,source=/var/run/docker.sock,target=/sock,readonly"},
		{"readonly sensitive /etc", "--mount=type=bind,source=/etc,target=/host-etc,readonly"},
		{"readonly sensitive subpath", "--mount=type=bind,source=/var/lib/x,target=/x,readonly"},
		{"sensitive /etc via src= alias", "--mount=type=bind,src=/etc,target=/x,readonly"},
		{"sensitive /etc trailing slash", "--mount=type=bind,source=/etc/,target=/x,readonly"},
		{"non-bind mount type=volume", "--mount=type=volume,source=somevol,target=/x,readonly"},
		{"readonly=true value form not the bare token (so treated as writable, non-workdir)", "--mount=type=bind,source=/tmp/pkts-work,target=/elsewhere,readonly=true"},
		{"short -v form even with ro", "-v=/tmp/pkts-work:/work:ro"},
		// Docker parses --mount keys case-insensitively: `type=bind,Source=/etc`
		// mounts the host /etc, but a case-sensitive gate would see no `source=`
		// field (empty source) and wave it through. The gate must match keys the
		// same way Docker does.
		{"sensitive /etc via uppercase Source key", "--mount=type=bind,Source=/etc,target=/x,readonly"},
		{"sensitive /etc via uppercase Src key", "--mount=type=bind,Src=/etc,target=/x,readonly"},
		{"sensitive /etc via uppercase Type+Source keys", "--mount=Type=bind,Source=/etc,target=/x,readonly"},
		{"docker.sock via uppercase Source key", "--mount=type=bind,Source=/var/run/docker.sock,target=/s,readonly"},
		// An empty source (no source field at all) must never be admitted: it
		// would clean to "." and slip past the sensitive-path check.
		{"bind mount with no source field", "--mount=type=bind,target=/x,readonly"},
		// Mixed-case type value: Docker accepts type=BIND, so the gate must treat
		// it as a bind mount and still enforce the sensitive-source rule.
		{"sensitive /etc via uppercase type value", "--mount=type=BIND,Source=/etc,target=/x,readonly"},
	}
	for _, b := range bad {
		t.Run(b.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, conform(append(clone(base), b.mount)), "%s must be refused", b.name)
		})
	}
}

// The cage runs the oracle in a DISPOSABLE scratch repo, which the oracle must
// write (git worktree add). conform therefore admits exactly ONE writable mount:
// a bind mount at the designated workdir (/work) of a non-sensitive source. A
// writable mount anywhere else, or of a sensitive/docker.sock/empty source even
// at /work, stays refused — so a drifted launch can't turn the one needed
// writable surface into write access to host state.
func TestConform_admitsTheWritableCageWorkdirMountButRefusesOtherWritableMounts(t *testing.T) {
	t.Parallel()
	base := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)

	good := []struct{ name, mount string }{
		{"writable scratch at the workdir", "--mount=type=bind,source=/tmp/packets-cage-abc,target=/work"},
		{"writable scratch via destination= alias", "--mount=type=bind,source=/tmp/packets-cage-abc,destination=/work"},
		{"writable scratch via dst= alias", "--mount=type=bind,source=/tmp/packets-cage-abc,dst=/work"},
		{"writable scratch via uppercase Target key", "--mount=type=bind,source=/tmp/packets-cage-abc,Target=/work"},
		{"writable scratch at the workdir with a trailing slash", "--mount=type=bind,source=/tmp/packets-cage-abc,target=/work/"},
		{"read-only at the workdir is also allowed", "--mount=type=bind,source=/tmp/packets-cage-abc,target=/work,readonly"},
		{"read-only non-sensitive input still admitted", "--mount=type=bind,source=/tmp/pkts-cache,target=/go/pkg/mod,readonly"},
	}
	for _, g := range good {
		t.Run(g.name, func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, conform(append(clone(base), g.mount)), "%s must be admitted", g.name)
		})
	}

	bad := []struct{ name, mount string }{
		{"writable to a non-workdir target", "--mount=type=bind,source=/tmp/x,target=/elsewhere"},
		{"writable to a workdir subpath, not the exact workdir", "--mount=type=bind,source=/tmp/x,target=/work/sub"},
		{"writable sensitive source even at the workdir", "--mount=type=bind,source=/etc,target=/work"},
		{"writable docker.sock at the workdir", "--mount=type=bind,source=/var/run/docker.sock,target=/work"},
		{"writable empty source at the workdir", "--mount=type=bind,target=/work"},
		{"writable to non-workdir via uppercase Target key", "--mount=type=bind,source=/tmp/x,Target=/elsewhere"},
		{"writable bind mount with no target field", "--mount=type=bind,source=/tmp/x"},
		{"writable to a workdir prefix, not the exact workdir", "--mount=type=bind,source=/tmp/x,target=/workspace"},
	}
	for _, b := range bad {
		t.Run(b.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, conform(append(clone(base), b.mount)), "%s must be refused", b.name)
		})
	}
}

func TestDockerRunner_readOnlyInputMountIsReadableButNotWritable(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "in.txt"), []byte("input-bytes"), 0o644))
	// World-writable, so a denied write is the readonly mount flag, not the
	// non-root user (which could write a 777 dir).
	require.NoError(t, os.Chmod(dir, 0o777))
	probe := []string{"sh", "-c", "cat /work/in.txt; touch /work/x 2>/dev/null && echo WRITE_OK || echo WRITE_BLOCKED"}

	ro := runRaw(t, mountedHardenedArgs(dir, true, probe))
	assert.Contains(t, ro, "input-bytes", "the read-only input mount must be readable")
	assert.Contains(t, ro, "WRITE_BLOCKED", "the read-only input mount must not be writable")

	rw := runRaw(t, mountedHardenedArgs(dir, false, probe))
	assert.Contains(t, rw, "WRITE_OK",
		"the SAME mount writable must permit the write — proving the block was the readonly flag, not the user")
}

// The writable workdir mount must be a REAL host bind, not the container's
// tmpfs: the oracle's worktrees and outputs written at /work have to land on the
// host's disposable scratch repo so the host can read the verdict afterward. A
// write inside the cage must therefore be visible on the host bind source.
func TestDockerRunner_writableWorkdirMountPersistsToTheHostScratch(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o777)) // non-root cage user must be able to write
	probe := []string{"sh", "-c", "echo cage-wrote > /work/out.txt && echo WRITE_OK || echo WRITE_FAIL"}

	out := runRaw(t, mountedHardenedArgs(dir, false, probe))
	assert.Contains(t, out, "WRITE_OK", "the cage must be able to write the writable workdir mount")

	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	require.NoError(t, err, "the cage's write must appear on the host bind source — a tmpfs would leave nothing")
	assert.Contains(t, string(got), "cage-wrote", "the bytes the cage wrote at /work must be the bytes on the host")
}

// mountedHardenedArgs is a hardened launch carrying a single bind mount of dir at
// /work (readonly when ro), for the RO-enforcement differential.
func mountedHardenedArgs(dir string, ro bool, cmd []string) []string {
	m := "--mount=type=bind,source=" + dir + ",target=/work"
	if ro {
		m += ",readonly"
	}
	args := []string{
		"run", "--rm", "--network=none", "--cap-drop=ALL",
		"--security-opt=no-new-privileges", "--read-only", "--user=65534:65534",
		m, "busybox:latest",
	}
	return append(args, cmd...)
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

// The egress lock is proven by attempting to observe the network from inside a
// real container, never by reading the launch flags. Differential + non-vacuous:
// the hardened (--network=none) container must have NO default route, while a
// networked control container MUST — so a broken probe (which would report the
// same for both) cannot pass, and a regression that drops --network=none flips
// the hardened result to HASNET and fails the test.
func TestDockerRunner_deniesEgressTheHardenedContainerHasNoNetwork(t *testing.T) {
	requireDocker(t)
	probe := []string{"sh", "-c", "ip route 2>/dev/null | grep -q default && echo HASNET || echo NONET"}

	res, err := DockerRunner{}.Run(context.Background(), Spec{Image: "busybox:latest", Cmd: probe})
	require.NoError(t, err)
	assert.Contains(t, res.Output, "NONET", "the hardened container must have no route to the network")
	assert.NotContains(t, res.Output, "HASNET")

	control := runWithNetwork(t, probe)
	assert.Contains(t, control, "HASNET",
		"a networked control must see a route — else the probe is broken and the hardened NONET would be vacuous")
}

// The pids cap is proven by trying to outrun it in a real container: a probe
// spawns far more processes than --pids-limit allows; busybox sh emits a fork
// failure when the cgroup denies the next process. Differential + non-vacuous:
// the ONLY difference from the control is --pids-limit, and the control (which
// keeps every other lock) must reach DONE with no fork failure — so the failure
// is attributable to the cap, not memory or a broken probe. Mutation-verified:
// dropping --pids-limit makes the hardened run reach DONE without a fork failure.
func TestDockerRunner_capsProcessesAtThePidsLimit(t *testing.T) {
	requireDocker(t)
	probe := []string{"sh", "-c", "i=0; while [ $i -lt 200 ]; do sleep 30 & i=$((i+1)); done; echo DONE"}

	res, err := DockerRunner{}.Run(context.Background(), Spec{Image: "busybox:latest", Cmd: probe})
	require.NoError(t, err)
	assert.Contains(t, res.Output, "can't fork", "the pids cap must deny processes beyond the limit")

	control := runRaw(t, without(hardenedArgs(Spec{Image: "busybox:latest", Cmd: probe}, "unconfined"), "--pids-limit=128"))
	assert.Contains(t, control, "DONE", "the control must run the probe to completion (else the signal is vacuous)")
	assert.NotContains(t, control, "can't fork", "without the cap the same probe spawns freely — the cap is the cause")
}

// The seccomp profile is proven by attempting a dangerous, cap-free syscall in a
// real container: creating a user namespace (`unshare -U`). cap-drop alone does
// NOT block it (it's an unprivileged operation), so a control with the SAME
// hardened launch minus seccomp (seccomp=unconfined — the only difference)
// permits it (UNSHARE_OK), while our profile denies it (UNSHARE_BLOCKED). The
// difference is attributable to seccomp, not the dropped caps; mutation-verified
// by flipping the profile to unconfined.
func TestDockerRunner_blocksNamespaceCreationViaSeccomp(t *testing.T) {
	requireDocker(t)
	probe := []string{"sh", "-c", "unshare -U true 2>/dev/null && echo UNSHARE_OK || echo UNSHARE_BLOCKED"}

	res, err := DockerRunner{}.Run(context.Background(), Spec{Image: "busybox:latest", Cmd: probe})
	require.NoError(t, err)
	assert.Contains(t, res.Output, "UNSHARE_BLOCKED", "the seccomp profile must deny namespace creation (unshare)")

	control := runRaw(t, hardenedArgs(Spec{Image: "busybox:latest", Cmd: probe}, "unconfined"))
	assert.Contains(t, control, "UNSHARE_OK",
		"without seccomp the same launch permits unshare — proving seccomp, not cap-drop, is the cause")
}

// runRaw execs a raw `docker` argv (a control launch isolating one lock) and
// returns its combined output; the security signal is in the output, not the
// exit code.
func runRaw(t *testing.T, args []string) string {
	t.Helper()
	out, _ := exec.Command("docker", args...).CombinedOutput()
	return string(out)
}

// runWithNetwork runs the probe in a control container WITH default networking
// (no --network=none) — the differential baseline that proves the probe works.
func runWithNetwork(t *testing.T, cmd []string) string {
	t.Helper()
	args := append([]string{"run", "--rm", "busybox:latest"}, cmd...)
	out, err := exec.Command("docker", args...).CombinedOutput()
	require.NoError(t, err, "control run failed: %s", out)
	return string(out)
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
