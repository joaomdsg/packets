package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// The cage receives its inputs as Spec.Mounts; the runner renders them into the
// single enforced argv as --mount= specs, so what the cage sees is exactly what
// the conform gate validated. The writable scratch carries no readonly suffix;
// read-only inputs carry it — and the whole rendered launch still conforms.
func TestHardenedArgs_rendersSpecMountsAsConformantArgs(t *testing.T) {
	t.Parallel()
	s := Spec{
		Image: "busybox:latest",
		Cmd:   []string{"true"},
		Mounts: []Mount{
			{Source: "/tmp/packets-cage-abc", Target: "/work"},
			{Source: "/tmp/pkts-modcache", Target: "/go/pkg/mod", Readonly: true},
		},
	}
	args := hardenedArgs(s, testProfile)

	assert.Contains(t, args, "--mount=type=bind,source=/tmp/packets-cage-abc,target=/work",
		"the writable scratch mount renders without a readonly suffix")
	assert.Contains(t, args, "--mount=type=bind,source=/tmp/pkts-modcache,target=/go/pkg/mod,readonly",
		"a read-only input mount renders with the readonly suffix")
	require.NoError(t, conform(args),
		"a launch with the cage's writable scratch + a read-only input must conform by construction")

	// The mounts precede the image, so they are docker run options, not args to
	// the containerized command.
	assert.Less(t, indexOf(args, "--mount=type=bind,source=/tmp/packets-cage-abc,target=/work"), indexOf(args, "busybox:latest"),
		"mounts must be rendered before the image")
}

// A Spec with no mounts must render exactly the bare hardened launch — adding
// the Mounts field must not inject a spurious or empty --mount= into the argv,
// or every existing mount-free caller would carry a phantom mount.
func TestHardenedArgs_noMountsRendersNoMountArg(t *testing.T) {
	t.Parallel()
	args := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)
	for _, a := range args {
		assert.NotContains(t, a, "--mount=", "a Spec with no mounts must not render any --mount= arg")
	}
	require.NoError(t, conform(args))
}

// A Spec carrying a forbidden mount produces an argv the gate refuses — no new
// validation lives on the Spec; the one conform gate is the single chokepoint,
// so the cage cannot be handed a dangerous mount by routing it through Spec.
func TestHardenedArgs_aForbiddenSpecMountFailsTheGate(t *testing.T) {
	t.Parallel()
	bad := []struct {
		name  string
		mount Mount
	}{
		{"writable to a non-workdir target", Mount{Source: "/tmp/x", Target: "/elsewhere"}},
		{"read-only sensitive source", Mount{Source: "/etc", Target: "/host-etc", Readonly: true}},
		{"writable sensitive source even at workdir", Mount{Source: "/etc", Target: "/work"}},
	}
	for _, b := range bad {
		t.Run(b.name, func(t *testing.T) {
			t.Parallel()
			args := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}, Mounts: []Mount{b.mount}}, testProfile)
			assert.Error(t, conform(args), "a Spec mount that is %s must make the launch fail conform", b.name)
		})
	}
}

func TestDockerRunner_specWritableWorkMountPersistsToTheHost(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o777)) // the non-root cage user must be able to write

	res, err := DockerRunner{}.Run(context.Background(), Spec{
		Image:  "busybox:latest",
		Cmd:    []string{"sh", "-c", "echo cage-wrote > /work/out.txt"},
		Mounts: []Mount{{Source: dir, Target: "/work"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode)

	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	require.NoError(t, err, "the cage's write to the Spec's /work mount must land on the host bind source")
	assert.Contains(t, string(got), "cage-wrote")
}

// A forbidden mount is refused by the gate BEFORE exec: the run returns the
// conform error and the command never runs — proven non-vacuously by also giving
// the cmd a writable /work sentinel write that would appear on the host if it ran.
func TestDockerRunner_refusesToRunASpecWithAForbiddenMount(t *testing.T) {
	requireDocker(t)
	work := t.TempDir()
	require.NoError(t, os.Chmod(work, 0o777))

	res, err := DockerRunner{}.Run(context.Background(), Spec{
		Image: "busybox:latest",
		Cmd:   []string{"sh", "-c", "echo ran > /work/sentinel.txt"},
		Mounts: []Mount{
			{Source: work, Target: "/work"},
			{Source: "/etc", Target: "/host-etc", Readonly: true}, // forbidden: sensitive source
		},
	})
	require.Error(t, err, "a Spec with a forbidden mount must be refused")
	assert.Equal(t, Result{}, res, "a refused launch returns the zero Result — nothing ran")

	_, statErr := os.Stat(filepath.Join(work, "sentinel.txt"))
	assert.True(t, os.IsNotExist(statErr), "the command must never have executed — no sentinel on the host")
}

// A Mount.Source or Mount.Target carrying an injected comma/key renders extra
// comma-separated fields into the single --mount= arg. The gate must not parse
// those fields differently from Docker: Docker resolves duplicate keys by
// LAST-positional-wins across the source/src, target/destination/dst and type
// alias families, and treats any whitespace, quote, empty value or trailing
// comma as a hard parse error (so the launch fails closed). checkMount uses the
// same last-wins loop, so a crafted Mount can neither (a) forge a safe field the
// gate reads while Docker reads a dangerous one, nor (b) pass the gate while
// Docker mounts something else. This pins that agreement: every injected case
// is REFUSED by the gate (because the smuggled last-wins field is itself
// caught), so no comma-injection slips a sensitive source or off-workdir
// writable target past conform.
func TestCheckMount_commaInjectionMatchesDockerLastWinsAndStaysFailClosed(t *testing.T) {
	t.Parallel()
	base := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)

	// Each mount is what hardenedArgs would render for a crafted Mount whose
	// Source or Target smuggles extra fields via an embedded comma. All must be
	// refused: the gate reads the same last-wins field Docker would.
	refused := []struct{ name, mount string }{
		// Target smuggles a later source= override to a sensitive path; both the
		// gate and Docker take the last source= (=/etc), so the gate rejects it.
		{"target smuggles source=/etc override", "--mount=type=bind,source=/tmp/ok,target=/work,source=/etc,readonly"},
		// Source smuggles a later sensitive source=; last-wins => /etc.
		{"source smuggles trailing sensitive source", "--mount=type=bind,source=/tmp/ok,target=/work,source=/var/lib,readonly"},
		// Source smuggles a docker.sock source override.
		{"source smuggles docker.sock override", "--mount=type=bind,source=/tmp/ok,target=/work,source=/var/run/docker.sock,readonly"},
		// type downgraded to a non-bind by a trailing duplicate (Docker last-wins
		// => volume; the gate must also see non-bind and refuse).
		{"type smuggled to volume by trailing dup", "--mount=type=bind,source=/tmp/ok,target=/work,type=volume"},
		// readonly=<value> is NOT the bare token the gate accepts, so the mount is
		// treated as writable; its target last-wins to a non-workdir => refused.
		{"writable off-workdir via duplicate target", "--mount=type=bind,source=/tmp/ok,target=/work,target=/elsewhere"},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, conform(append(clone(base), c.mount)),
				"%s: the gate must read the same last-wins field Docker does and refuse", c.name)
		})
	}

	// The benign mirror: a duplicate target= whose LAST value is the workdir is
	// admitted (Docker mounts at /work too) — proving the refusals above are the
	// smuggled field being caught, not a blanket reject of any duplicate key.
	assert.NoError(t,
		conform(append(clone(base), "--mount=type=bind,source=/tmp/ok,target=/elsewhere,target=/work")),
		"a duplicate target= whose last value is the workdir mounts at /work in Docker too — admitted")
}

// The conform gate's verdict must match where Docker actually mounts a
// comma-injected --mount, end to end in a real container. For a duplicate
// target= the gate admits (last value /work), Docker must bind the source at
// /work and nowhere else; for one it refuses (last value off-workdir), Docker
// would have bound off-workdir — so admitting it would be the bug. This proves
// the unit test's "same last-wins" claim against the real parser.
func TestDockerRunner_commaInjectedMountResolvesWhereConformExpects(t *testing.T) {
	requireDocker(t)
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "mark"), []byte("SRCMARK"), 0o644))
	require.NoError(t, os.Chmod(src, 0o777))

	// Gate ADMITS this (duplicate target=, last value /work). Docker must mount
	// the source at /work and leave /elsewhere empty.
	admitted := "--mount=type=bind,source=" + src + ",target=/elsewhere,target=/work"
	base := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)
	require.NoError(t, conform(append(clone(base), admitted)),
		"sanity: this comma-injected mount is admitted by the gate")
	probe := []string{"sh", "-c", "cat /work/mark 2>/dev/null; cat /elsewhere/mark 2>/dev/null && echo AT_ELSEWHERE || echo NOT_AT_ELSEWHERE"}
	args := []string{
		"run", "--rm", "--network=none", "--cap-drop=ALL",
		"--security-opt=no-new-privileges", "--read-only", "--user=65534:65534",
		admitted, "busybox:latest",
	}
	out := runRaw(t, append(args, probe...))
	assert.Contains(t, out, "SRCMARK", "Docker must bind the source at /work, exactly where the last-wins target resolves")
	assert.Contains(t, out, "NOT_AT_ELSEWHERE", "the shadowed first target= must NOT receive a mount — Docker is last-wins like the gate")
}

func indexOf(s []string, want string) int {
	for i, v := range s {
		if v == want {
			return i
		}
	}
	return -1
}

// The cage-exec launch must route HOME/GOCACHE/TMPDIR/etc into the writable
// workdir; Spec.Env carries those and the runner renders them as --env pairs.
// Order is preserved and deterministic (the equivalence lock pins byte-identical
// argv for the same Spec), and every env pair precedes the image so it is a
// docker run option, not an argument to the containerized command.
func TestHardenedArgs_rendersSpecEnvAsOrderedDeterministicArgs(t *testing.T) {
	t.Parallel()
	s := Spec{
		Image: "busybox:latest",
		Cmd:   []string{"true"},
		Env: []EnvVar{
			{Name: "HOME", Value: "/work"},
			{Name: "GOCACHE", Value: "/work/gocache"},
			{Name: "GOFLAGS", Value: "-mod=mod"}, // a value containing '=' must survive verbatim
		},
	}
	args := hardenedArgs(s, testProfile)

	homeAt := indexOf(args, "HOME=/work")
	cacheAt := indexOf(args, "GOCACHE=/work/gocache")
	flagsAt := indexOf(args, "GOFLAGS=-mod=mod")
	imageAt := indexOf(args, "busybox:latest")
	require.NotEqual(t, -1, homeAt, "HOME env value must be rendered")
	require.NotEqual(t, -1, cacheAt, "GOCACHE env value must be rendered")
	require.NotEqual(t, -1, flagsAt, "a value containing '=' must render verbatim (NAME=VALUE splits on the first '=')")

	// Each value is immediately preceded by a --env flag (the two-token form).
	assert.Equal(t, "--env", args[homeAt-1], "the HOME value must be introduced by --env")
	assert.Equal(t, "--env", args[cacheAt-1], "the GOCACHE value must be introduced by --env")

	assert.Less(t, homeAt, cacheAt, "env pairs render in Spec order")
	assert.Less(t, cacheAt, flagsAt, "env pairs render in Spec order")
	assert.Less(t, flagsAt, imageAt, "every env pair precedes the image")

	require.NoError(t, conform(args), "env pairs do not disturb conformance")

	// Deterministic: the same Spec renders byte-identical argv (the equivalence
	// lock compares the in-proc and cage launches), so no map iteration leaks in.
	assert.Equal(t, args, hardenedArgs(s, testProfile), "the same Spec must render an identical argv")
}

// A Spec with no Env renders no --env — adding the field must not perturb the
// bare hardened launch every existing env-free caller relies on.
func TestHardenedArgs_noEnvRendersNoEnvArg(t *testing.T) {
	t.Parallel()
	args := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)
	assert.NotContains(t, args, "--env", "a Spec with no Env must not render any --env flag")
	require.NoError(t, conform(args))
}

// conform must permit the per-run container name the runner injects (a docker
// run option, before the image) — else the gate would refuse every launch the
// moment the runner names the container for kill-on-cancel.
func TestConform_permitsTheInjectedContainerName(t *testing.T) {
	t.Parallel()
	base := hardenedArgs(Spec{Image: "busybox:latest", Cmd: []string{"true"}}, testProfile)
	withName := append([]string{base[0], "--name", "packets-cage-deadbeef"}, base[1:]...) // mirror Run's injection after "run"

	require.NoError(t, conform(withName), "the injected container name must not trip the gate")
	assert.Less(t, indexOf(withName, "packets-cage-deadbeef"), indexOf(withName, "busybox:latest"),
		"the name is a run option, so it must precede the image")
}

// A cancelled run must KILL the container, not orphan it. exec.CommandContext
// alone SIGKILLs only the docker client — the attached --rm container keeps
// running (verified empirically). So a cancelled Run must (a) surface as an
// error, not a clean Result, and (b) leave no container behind. Non-vacuous: the
// command is `sleep 600`, so a Run that merely waited would take 10 minutes; this
// returns in seconds, proving the cancel actually killed the container.
func TestDockerRunner_killsTheContainerWhenTheContextIsCancelled(t *testing.T) {
	requireDocker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	res, err := DockerRunner{}.Run(ctx, Spec{Image: "busybox:latest", Cmd: []string{"sleep", "600"}})
	elapsed := time.Since(start)

	require.Error(t, err, "a cancelled run must surface as an error, not a clean Result")
	assert.ErrorIs(t, err, context.DeadlineExceeded, "the error must be the cancellation itself, not an image/launch failure (rules out a vacuous pass where no container ran)")
	require.Equal(t, Result{}, res, "a cancelled run returns the zero Result")
	assert.Less(t, elapsed, 30*time.Second, "Run must return on cancel, not wait out the 600s sleep")

	require.Eventually(t, func() bool {
		return dockerPS(t, "label=io.packets.sandbox=1") == ""
	}, 15*time.Second, 250*time.Millisecond, "the container must be killed and reaped on ctx-cancel, not left orphaned")
}

// A cancel that lands in the create-but-not-yet-started window must STILL leave
// no container behind. `docker kill` only signals a RUNNING container, so a
// container the daemon registered but had not started (status Created) is immune
// to kill and, never having run, never hits --rm/AutoRemove — it orphans forever.
// Stressing a spread of sub-second cancel deadlines reliably lands in that window;
// the backstop must force-remove by name regardless of container state.
func TestDockerRunner_leavesNoOrphanWhenCancelLandsInTheCreateWindow(t *testing.T) {
	requireDocker(t)
	delays := []time.Duration{0, 1, 2, 3, 5, 8, 12, 20, 35, 60, 100, 150, 250, 400, 600, 900}
	for round := 0; round < 3; round++ {
		for _, d := range delays {
			ctx, cancel := context.WithTimeout(context.Background(), d*time.Millisecond)
			_, _ = DockerRunner{}.Run(ctx, Spec{Image: "busybox:latest", Cmd: []string{"sleep", "600"}})
			cancel()
		}
	}
	require.Eventually(t, func() bool {
		return dockerPS(t, "label=io.packets.sandbox=1") == ""
	}, 15*time.Second, 250*time.Millisecond,
		"a cancel in the create-but-not-started window must force-remove the container, not orphan it")
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
	// PACKETS_REQUIRE_CAGE (set in CI) turns a missing-docker SKIP into a hard FAIL,
	// so the real-container enforcement proofs can't silently skip green in the
	// pipeline. Locally (env unset) they skip gracefully when docker is absent.
	if err := exec.Command("docker", "info").Run(); err != nil {
		if os.Getenv("PACKETS_REQUIRE_CAGE") != "" {
			t.Fatalf("PACKETS_REQUIRE_CAGE set but docker is not available: %v", err)
		}
		t.Skip("docker not available; skipping real-container enforcement test")
	}
}

func dockerPS(t *testing.T, filter string) string {
	t.Helper()
	out, err := exec.Command("docker", "ps", "-aq", "--filter", filter).Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}
