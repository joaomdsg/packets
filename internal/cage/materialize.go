// Package cage holds the trusted host-side preparation for running the catch
// oracle inside a sandbox: turning a claim's revisions into an isolated,
// disposable repo the cage can verify against without ever touching host state.
package cage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/joaomdsg/packets/internal/ledger"
)

// Materialize produces a DISPOSABLE, WRITABLE git repo containing the Target's
// base/fix/tip revisions, cloned from hostRepo with `git clone --local
// --no-hardlinks` so it shares no objects (no inodes) with the host. The host's
// real repo is never handed to the cage; this copy is what gets bind-mounted in,
// and being writable it lets the oracle's `git worktree add` succeed (a
// read-only mount of the host repo could not).
//
// It is fail-closed: a revision the host cannot resolve (absent or empty) is
// rejected before anything is materialized. On success it returns the scratch
// dir and a cleanup that removes it; on any error it returns an empty dir, a nil
// cleanup, and leaves nothing behind.
func Materialize(ctx context.Context, hostRepo string, t ledger.Target) (string, func(), error) {
	for _, rev := range []string{t.BaseRev, t.FixRev, t.TipRev} {
		if strings.TrimSpace(rev) == "" {
			return "", nil, fmt.Errorf("cage: empty target revision")
		}
		// `--end-of-options` stops a leading-dash rev (e.g. a forged claim's
		// "--git-dir=...") from being parsed as a git flag: everything after it is
		// a revision, never an option. Plain `--` does NOT serve this role for
		// `rev-parse --verify`.
		if err := git(ctx, hostRepo, "rev-parse", "--verify", "--quiet", "--end-of-options", rev+"^{commit}"); err != nil {
			return "", nil, fmt.Errorf("cage: host repo cannot resolve revision %q: %w", rev, err)
		}
	}

	scratch, err := os.MkdirTemp("", "packets-cage-*")
	if err != nil {
		return "", nil, fmt.Errorf("cage: scratch dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(scratch) }

	// `git clone --local` copies the whole object store, so every rev the host
	// resolved above is present in the clone — no second verification needed.
	// `--` separates options from the positional <repo> <dir> args so a
	// leading-dash hostRepo can never be parsed as a clone flag (e.g. `-c`).
	if err := git(ctx, "", "clone", "--local", "--no-hardlinks", "--quiet", "--", hostRepo, scratch); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("cage: clone into scratch: %w", err)
	}

	return scratch, cleanup, nil
}

// git runs a git command (in dir, or the process cwd when dir is empty) and
// returns an error carrying stderr on failure.
func git(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return nil
}
