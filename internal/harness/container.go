package harness

import "strconv"

// ContainerSpec is what to run the live agent harness in a one-shot container: the
// image, the repo to work in, the task prompt, and the host-supplied isolation
// inputs (seccomp profile path, the uid:gid to run as, resource caps, the env var
// NAMES to pass through). The values are host-set, never agent-supplied.
type ContainerSpec struct {
	Image       string
	RepoDir     string
	Prompt      string
	SeccompPath string
	// User is the "uid:gid" the container runs as — the HOST user's, so the agent's
	// writes to the bind-mounted repo are host-owned (non-root, but NOT a nobody uid
	// that couldn't write host files).
	User string
	// EnvPassthrough names env vars passed through BY NAME (e.g. ANTHROPIC_API_KEY) —
	// the value comes from the host env at run time, so a secret never enters argv.
	EnvPassthrough []string
	// RouteEnv is host-set NON-secret routing (HOME, GOCACHE, npm cache, …) pointing
	// the agent's tools at a writable path. Rendered as -e NAME=VALUE — safe because
	// these are not secrets. Needed because the rootfs is --read-only: without a
	// writable HOME/cache the tools (claude/git/go/node) fail with EROFS.
	RouteEnv  []EnvVar
	PidsLimit int
	Memory    string
}

// EnvVar is one host-set NAME=VALUE routing variable for the agent container.
type EnvVar struct {
	Name  string
	Value string
}

// ContainerArgs builds the `docker run` argv for the AGENT container: hardened like
// the verification cage (cap-drop, no-new-privileges, seccomp, read-only rootfs,
// pids/memory caps, non-root) — BUT egress-allowed and with the repo bind-mounted
// WRITABLE, because a live agent must reach the model API and edit the code. It is
// a TRUST/ISOLATION boundary (a trusted harness in a box), distinct from the cage's
// CONTAINMENT of untrusted oracle code, so it does NOT carry --network=none.
//
// The repo is the ONLY writable surface (read-only rootfs + a writable /tmp tmpfs +
// the one repo bind), the docker socket is never mounted, and secrets pass by name —
// so the blast radius of a runaway agent is the repo workdir, never the host.
func ContainerArgs(s ContainerSpec) []string {
	args := []string{
		"docker", "run", "--rm",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--security-opt=seccomp=" + s.SeccompPath,
		"--read-only",
		"--tmpfs=/tmp",
		"--pids-limit=" + strconv.Itoa(s.PidsLimit),
		"--memory=" + s.Memory,
		"--user=" + s.User,
		"-v", s.RepoDir + ":/work",
		"-w", "/work",
	}
	for _, name := range s.EnvPassthrough {
		args = append(args, "-e", name) // by NAME only — the secret value stays out of argv
	}
	for _, re := range s.RouteEnv {
		args = append(args, "-e", re.Name+"="+re.Value) // non-secret routing for the read-only rootfs
	}
	args = append(args, s.Image, "claude")
	return append(args, ClaudeArgs(s.Prompt)...)
}
