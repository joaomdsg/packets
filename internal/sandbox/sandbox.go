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
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Spec is what to run in a one-shot hardened container. It carries only the
// image and command: the isolation is applied unconditionally by the runner, not
// chosen by the caller.
type Spec struct {
	Image string
	Cmd   []string
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
	args := hardenedArgs(s)
	if err := conform(args); err != nil {
		return Result{}, err
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
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
func hardenedArgs(s Spec) []string {
	args := []string{
		"run", "--rm",
		"--network=none",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--read-only",
		"--tmpfs=/tmp:rw,nosuid,nodev,noexec,size=64m",
		"--pids-limit=128",
		"--memory=256m",
		"--cpus=1",
		"--user=65534:65534",
		"--label=io.packets.sandbox=1",
	}
	args = append(args, s.Image)
	args = append(args, s.Cmd...)
	return args
}

// requiredFlags must each appear verbatim; requiredPrefixes must each appear as
// the prefix of some arg (the flags that carry a value).
var (
	requiredFlags    = []string{"--rm", "--network=none", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--read-only"}
	requiredPrefixes = []string{"--pids-limit=", "--memory=", "--user="}
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
		case a == "--network=host", a == "--net=host", a == "--pid=host":
			return fmt.Errorf("sandbox: forbidden host-namespace flag %q", a)
		case a == "-v", a == "--volume", strings.HasPrefix(a, "-v="), strings.HasPrefix(a, "--volume="), strings.HasPrefix(a, "--mount="):
			return fmt.Errorf("sandbox: forbidden host mount %q", a)
		case strings.Contains(a, "docker.sock"):
			return fmt.Errorf("sandbox: forbidden docker-socket reference %q", a)
		}
	}
	return nil
}
