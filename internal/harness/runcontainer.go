package harness

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/joaomdsg/packets/internal/sandbox"
	"github.com/joaomdsg/packets/internal/translate"
)

// agentImage is the container image the live agent runs in — a base with the
// node/go/python toolchains + the claude CLI baked in. A var so an integration
// test can point it at a fake-claude image.
var agentImage = "packets-agent"

// agentSpec is the standard hardened agent profile RunContainer runs: the API key
// by name (secret never in argv), HOME + tool caches routed onto the read-only
// rootfs's writable /tmp (or the tools EROFS), and the run's identity threaded
// through. Pure: it composes the ContainerSpec; ContainerArgs renders the argv.
func agentSpec(repoDir, prompt, seccompPath, user, resumeID string) ContainerSpec {
	return ContainerSpec{
		Image:          agentImage,
		RepoDir:        repoDir,
		Prompt:         prompt,
		SeccompPath:    seccompPath,
		User:           user,
		ResumeID:       resumeID,
		EnvPassthrough: []string{"ANTHROPIC_API_KEY"},
		RouteEnv: []EnvVar{
			{Name: "HOME", Value: "/tmp"},
			{Name: "XDG_CACHE_HOME", Value: "/tmp/.cache"},
			{Name: "GOCACHE", Value: "/tmp/go"},
			{Name: "npm_config_cache", Value: "/tmp/npm"},
		},
		PidsLimit: 512,
		Memory:    "4g",
	}
}

// RunContainer runs the live Claude Code harness in the hardened agent container
// and reduces its stream-json into settled revisions — the containerized twin of
// RunProcess, with the SAME signature so the runHarness seam can swap to it. The
// repo is bind-mounted writable, so the agent's commits land on the host; the
// host still re-derives every catch (the economy firewall is unchanged).
//
// IO/exec wiring: verified by build/vet + a fake-claude-image integration test, not
// unit-tested (a live run needs the image + an API key). It reuses the cage's
// seccomp deny-list and its kill-by-name teardown.
func RunContainer(ctx context.Context, repoDir, prompt string, onActivity func([]translate.UIEvent)) ([]Turn, error) {
	head, err := headRev(ctx, repoDir)
	if err != nil {
		return nil, err
	}
	seccompPath, cleanup, err := sandbox.MaterializeSeccompProfile()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	user := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	argv := ContainerArgs(agentSpec(repoDir, prompt, seccompPath, user, resumeFrom(ctx)))

	name, err := agentContainerName()
	if err != nil {
		return nil, err
	}
	// argv is ["docker","run",...]; inject "--name <name>" right after "run" so a
	// cancel can remove the container by name (mirroring sandbox.DockerRunner).
	dockerArgs := append([]string{"run", "--name", name}, argv[2:]...)

	var opts []Option
	if onActivity != nil {
		opts = append(opts, WithActivity(onActivity))
	}
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	cmd.Dir = repoDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("harness: container stdout pipe: %v", err)
	}
	// On cancel, force-remove the named container (not just the docker client) so
	// the attached --rm box can't linger; mirrors DockerRunner.
	cmd.Cancel = func() error {
		_ = exec.Command("docker", "rm", "-f", name).Run()
		return cmd.Process.Kill()
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("harness: start container: %v", err)
	}

	turns, runErr := New(repoDir, head, opts...).Run(ctx, stdout)
	if runErr != nil {
		// Run aborted mid-stream; the container may still be writing. Kill+drain so a
		// full pipe can't deadlock Wait, then reap. runErr wins (the kill noise is
		// discarded) — mirrors RunProcess's deadlock-safe teardown.
		_ = exec.Command("docker", "rm", "-f", name).Run()
		_, _ = io.Copy(io.Discard, stdout)
		_ = cmd.Wait()
		return turns, runErr
	}
	if waitErr := cmd.Wait(); waitErr != nil {
		return turns, fmt.Errorf("harness: container exited: %v", waitErr)
	}
	return turns, nil
}

func agentContainerName() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("harness: container name: %v", err)
	}
	return "packets-agent-" + hex.EncodeToString(b[:]), nil
}
