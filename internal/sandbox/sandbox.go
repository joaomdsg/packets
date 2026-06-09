// Package sandbox runs untrusted work in a one-shot, locked-down container. It
// is the isolation SEAM for roadmap #6c: Runner is the interface a gVisor or
// microVM backend can satisfy later; DockerRunner is impl #1 (hardened plain
// Linux container). The hardening is NOT caller-tunable — every launch goes
// through one enforced argv and a fail-closed conformance gate, so a launch that
// is missing a lock or carrying a forbidden flag never reaches the runtime.
//
// This package builds and gates the launch; the rigorous break-out proofs (a
// forbidden syscall is denied, egress is blocked, a fork-bomb is capped) live in
// real-container integration tests, never config-string assertions.
package sandbox

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// seccompProfile is a default-ALLOW profile that additionally DENIES a curated
// set of dangerous, workload-unneeded syscalls (namespace creation, mount,
// kernel-module and kexec, keyring, ptrace, bpf, reboot, swap). It strictly
// hardens beyond the daemon default for these syscalls. It is NOT yet a full
// default-DENY allowlist — that hardening is tuned to the verification workload's
// real syscall set at #6c (the verification flow); a hand-rolled allowlist before
// that workload exists would risk breaking it.
//
// Workload note for #6c-6: the curated deny set is verified safe for go build /
// go test / go test -race (the race detector uses ThreadSanitizer instrumentation,
// not ptrace). ptrace is denied deliberately — it does NOT affect those workloads,
// but a debugger-style verification step (e.g. delve, strace) would need it. Revisit
// the ptrace entry only if #6c-6 adds such a step.
//
//go:embed seccomp.json
var seccompProfile []byte

// Mount is one bind mount the cage needs: a host Source exposed at the
// in-container Target, read-only when Readonly. It is rendered into the enforced
// argv and validated by conform like any other launch flag — so a Mount that is
// not one of the allowed shapes (read-only non-sensitive input, or writable only
// at the cage workdir) makes the launch fail-closed, never reaching the runtime.
type Mount struct {
	Source   string
	Target   string
	Readonly bool
}

// Spec is what to run in a one-shot hardened container: the image, the command,
// and the bind mounts the cage's inputs need. The isolation is applied
// unconditionally by the runner, not chosen by the caller; the Mounts are the
// one caller-supplied surface, and they are gated by conform.
type Spec struct {
	Image  string
	Cmd    []string
	Mounts []Mount
}

// Result is the observable outcome of a finished run. ExitCode and Output are
// raw bytes from an untrusted process — NOT a trusted security verdict; a
// security property is proven by attempting the forbidden thing and observing
// the runtime deny it, never by reading these.
type Result struct {
	ExitCode int
	Output   string
}

// Runner runs a Spec in an isolated one-shot sandbox. DockerRunner is impl #1;
// a stronger-isolation backend (gVisor, microVM) can satisfy the same seam.
type Runner interface {
	Run(ctx context.Context, s Spec) (Result, error)
}

// DockerRunner runs the Spec in a hardened, ephemeral Docker container.
type DockerRunner struct{}

// Run launches the Spec through the single enforced hardened argv, refusing
// fail-closed if that argv is not conformant (so a non-conformant launch never
// execs), then runs it. A non-zero CONTAINER exit is a Result, not an error
// (the container ran); only a failure to invoke the runtime is an error.
func (DockerRunner) Run(ctx context.Context, s Spec) (Result, error) {
	profPath, cleanup, err := materializeSeccompProfile()
	if err != nil {
		return Result{}, err
	}
	defer cleanup()
	args := hardenedArgs(s, profPath)
	if err := conform(args); err != nil {
		return Result{}, err
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	res := Result{Output: out.String()}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			res.ExitCode = exit.ExitCode()
			return res, nil
		}
		return Result{}, fmt.Errorf("sandbox: docker run: %v: %s", err, strings.TrimSpace(out.String()))
	}
	return res, nil
}

// hardenedArgs is the SINGLE enforced launch path: a one-shot container with no
// network, no capabilities, no privilege escalation, a read-only rootfs (only a
// noexec tmpfs scratch is writable), bounded pids/memory/cpu, a non-root user,
// and a cleanup label. The agent gets none of the host.
func hardenedArgs(s Spec, seccompProfilePath string) []string {
	args := []string{
		"run", "--rm",
		"--network=none",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--security-opt=seccomp=" + seccompProfilePath,
		"--read-only",
		"--tmpfs=/tmp:rw,nosuid,nodev,noexec,size=64m",
		"--pids-limit=128",
		"--memory=256m",
		"--cpus=1",
		"--user=65534:65534",
		"--label=io.packets.sandbox=1",
	}
	for _, m := range s.Mounts {
		spec := "--mount=type=bind,source=" + m.Source + ",target=" + m.Target
		if m.Readonly {
			spec += ",readonly"
		}
		args = append(args, spec)
	}
	args = append(args, s.Image)
	args = append(args, s.Cmd...)
	return args
}

// materializeSeccompProfile writes the embedded profile to a temp file (the
// docker CLI reads the seccomp profile from a path) and returns the path plus a
// cleanup. The caller defers cleanup.
func materializeSeccompProfile() (string, func(), error) {
	f, err := os.CreateTemp("", "packets-seccomp-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("sandbox: seccomp tempfile: %v", err)
	}
	name := f.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := f.Write(seccompProfile); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, fmt.Errorf("sandbox: write seccomp profile: %v", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("sandbox: close seccomp profile: %v", err)
	}
	return name, cleanup, nil
}

// requiredFlags must each appear verbatim; requiredPrefixes must each appear as
// the prefix of some arg (the flags that carry a value).
var (
	requiredFlags    = []string{"--rm", "--network=none", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--read-only"}
	requiredPrefixes = []string{"--pids-limit=", "--memory=", "--user=", "--security-opt=seccomp="}
)

// conform is the fail-closed gate: it returns an error if the argv is missing any
// non-negotiable lock or carries a forbidden flag (privilege escalation, host
// network/pid namespace, or a host/docker-socket mount). It is a denylist over a
// fixed launch path — it guards against drift/tampering of hardenedArgs, not
// against an arbitrary caller-supplied argv.
func conform(args []string) error {
	present := make(map[string]bool, len(args))
	for _, a := range args {
		present[a] = true
	}
	for _, r := range requiredFlags {
		if !present[r] {
			return fmt.Errorf("sandbox: launch missing required hardening %q", r)
		}
	}
	for _, p := range requiredPrefixes {
		found := false
		for _, a := range args {
			if strings.HasPrefix(a, p) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("sandbox: launch missing required hardening %q", p)
		}
	}
	for _, a := range args {
		switch {
		case a == "--privileged":
			return fmt.Errorf("sandbox: forbidden flag %q", a)
		case a == "--security-opt=seccomp=unconfined":
			return fmt.Errorf("sandbox: forbidden seccomp=unconfined (no syscall filter) %q", a)
		case a == "--network=host", a == "--net=host", a == "--pid=host":
			return fmt.Errorf("sandbox: forbidden host-namespace flag %q", a)
		case a == "-v", a == "--volume", strings.HasPrefix(a, "-v="), strings.HasPrefix(a, "--volume="):
			return fmt.Errorf("sandbox: forbidden host mount %q (use --mount=...,readonly)", a)
		case strings.HasPrefix(a, "--mount="):
			if err := checkMount(a); err != nil {
				return err
			}
		case strings.Contains(a, "docker.sock"):
			return fmt.Errorf("sandbox: forbidden docker-socket reference %q", a)
		}
	}
	return nil
}

// sensitiveSources are host paths a bind mount must never expose to the cage,
// even read-only: the root, system config/state/binaries, and the kernel
// pseudo-filesystems. A source is sensitive if it equals one of these or lies
// under it.
var sensitiveSources = []string{"/", "/etc", "/var", "/proc", "/sys", "/dev", "/root", "/boot", "/usr", "/bin", "/sbin", "/lib", "/run"}

// cageWorkdir is the ONE in-container path a writable bind mount may target: the
// cage's disposable scratch repo, which the oracle must write (git worktree add).
// A writable mount anywhere else is refused.
const cageWorkdir = "/work"

// checkMount admits a --mount= arg in one of two shapes, and refuses everything
// else fail-closed:
//   - a read-only (bare `readonly` token) bind mount of a non-empty,
//     non-sensitive, non-docker.sock source (read-only inputs: repo, module cache); or
//   - a WRITABLE bind mount whose target is EXACTLY the cage workdir and whose
//     source is non-empty, non-sensitive, non-docker.sock (the disposable scratch repo).
//
// Refused: a writable mount to any other target, any mount of a sensitive or
// docker.sock source, an empty source, a non-bind type, or a `readonly=true`
// value form rather than the bare token. Field keys (and the type value) are
// matched case-insensitively because Docker parses them that way: a capitalized
// `Source=/etc` mounts the host /etc just like `source=/etc`, so the gate must
// see it too rather than read an empty source and wave it through. The target
// PATH is compared case-sensitively (Linux mount semantics: /Work != /work) after
// filepath.Clean, so a trailing slash matches but a subpath or prefix does not.
func checkMount(arg string) error {
	val := strings.TrimPrefix(arg, "--mount=")
	fields := strings.Split(val, ",")
	var typ, source, target string
	readonly := false
	for _, f := range fields {
		// Match the KEY case-insensitively (Docker does), but preserve the VALUE
		// verbatim (paths are case-sensitive). Split the key off the value once.
		key, value, hasValue := strings.Cut(f, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		switch {
		case !hasValue && key == "readonly":
			readonly = true
		case key == "type":
			typ = strings.ToLower(value)
		case key == "source", key == "src":
			source = value
		case key == "target", key == "destination", key == "dst":
			target = value
		}
	}
	if typ != "bind" {
		return fmt.Errorf("sandbox: forbidden non-bind mount %q", arg)
	}
	if !readonly && filepath.Clean(target) != cageWorkdir {
		return fmt.Errorf("sandbox: forbidden writable mount %q (writable only at the cage workdir %q)", arg, cageWorkdir)
	}
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("sandbox: forbidden bind mount with no source %q", arg)
	}
	clean := filepath.Clean(source)
	if strings.Contains(clean, "docker.sock") {
		return fmt.Errorf("sandbox: forbidden docker-socket reference %q", arg)
	}
	for _, s := range sensitiveSources {
		if clean == s || (s != "/" && strings.HasPrefix(clean, s+"/")) {
			return fmt.Errorf("sandbox: forbidden sensitive mount source %q", arg)
		}
	}
	return nil
}
