package harness

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/joaomdsg/packets/internal/translate"
)

// ClaudeArgs builds the argv (after the "claude" binary) that launches the
// harness headless and streaming: -p carries the task; --output-format
// stream-json + --verbose are both required for the CLI to emit the per-event
// stream the Supervisor reduces; bypassPermissions lets the headless agent edit
// without an interactive prompt (the run is contained by the sandbox, a later
// slice — not by in-process permission checks).
func ClaudeArgs(prompt string) []string {
	return []string{
		"-p", prompt,
		"--output-format", "stream-json",
		"--verbose",
		"--permission-mode", "bypassPermissions",
	}
}

// RunProcess spawns a real Claude Code harness on prompt in repoDir, reduces its
// live stream into settled revisions (diffed from repoDir's current HEAD), and
// returns once the process exits. The harness mints nothing itself — every
// revision comes from the host-side settle step inside Run (the economy
// firewall). A spawn or non-zero exit surfaces as an error.
//
// This is process/IO wiring: it is verified by build/vet and a manual run, not
// unit-tested (a live run needs the claude binary and an API key). The reducer
// it drives (Supervisor.Run) and the arg builder (ClaudeArgs) are tested.
func RunProcess(ctx context.Context, repoDir, prompt string, onActivity func([]translate.UIEvent)) ([]Turn, error) {
	head, err := headRev(ctx, repoDir)
	if err != nil {
		return nil, err
	}

	var opts []Option
	if onActivity != nil {
		opts = append(opts, WithActivity(onActivity))
	}
	cmd := exec.CommandContext(ctx, "claude", ClaudeArgs(prompt)...)
	cmd.Dir = repoDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("harness: stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("harness: start claude: %v", err)
	}

	turns, runErr := New(repoDir, head, opts...).Run(ctx, stdout)
	if runErr != nil {
		// Run aborted mid-stream (malformed line, settle failure) with stdout
		// only partially read. Claude may still be writing; an unread pipe fills
		// its OS buffer and blocks the child, which would deadlock cmd.Wait.
		// Kill the process so Wait can reap it, then drain the pipe so Wait's
		// internal close doesn't race the still-buffered writer. runErr wins —
		// the kill-induced Wait error is just noise, so it is discarded.
		_ = cmd.Process.Kill()
		_, _ = io.Copy(io.Discard, stdout)
		_ = cmd.Wait()
		return turns, runErr
	}
	if waitErr := cmd.Wait(); waitErr != nil {
		return turns, fmt.Errorf("harness: claude exited: %v", waitErr)
	}
	return turns, nil
}

func headRev(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("harness: resolve HEAD: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}
