package packet

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// buildVetTimeout bounds the COMBINED go build + go vet run — the one exec
// seam this slice adds is cheap (no docker, no network), but a wedged
// toolchain must never hang the caller.
const buildVetTimeout = 60 * time.Second

// detailTailLimit caps how much of a failing command's combined output a
// Gate's Detail may carry — a one-clause honest note, never a log dump.
const detailTailLimit = 200

// notRunNoRevision is the Detail for the two "genuinely nothing to build"
// cases: no fix revision minted yet, or repoDir isn't even a git repo. A
// scratch-dir preparation failure is a DIFFERENT honest cause (see below)
// and gets its own message, never this one.
const notRunNoRevision = "no revision to build yet"

// RunBuildVetGate is G4 (build/vet/lint): it materializes
// fixRev into a THROWAWAY git worktree (never repoDir's own working tree,
// so a concurrent caller or a dirty checkout can never leak into the
// result) and runs `go build ./...` then `go vet ./...` inside it, bounded
// by buildVetTimeout combined. Passed when both exit 0; Failed names the
// first failing command plus a truncated tail of its combined output — an
// honest one-clause note, never a full log dump. fixRev=="" (no revision
// minted yet) or repoDir not being a git repo is answered as GateNotRun
// rather than an error, since there is genuinely nothing to build yet, not
// a failure of the attempt. A revision that IS a git repo path but doesn't
// resolve (bad/unknown SHA) is likewise GateNotRun — an unresolvable
// revision is a real absence, never fabricated into a pass or fail. This
// function never returns an error to the caller: every failure mode maps to
// one of the three Gate outcomes.
//
// Concurrency: unlike Lane's Measure (a read-only `go list`), this MUTATES
// repoDir's git metadata (`git worktree add`/`remove`). Two concurrent calls
// against the SAME repoDir serialize safely at the git level (each gets its
// own uniquely-named scratch worktree, and git's own locking around
// .git/worktrees prevents corruption) but do contend — a caller driving many
// packets against one repo concurrently should expect queuing here, not a
// race. No explicit lock is taken in this slice; revisit if that contention
// becomes a real bottleneck.
func RunBuildVetGate(ctx context.Context, repoDir, fixRev string) Gate {
	if fixRev == "" || !isGitRepo(ctx, repoDir) {
		return Gate{Status: GateNotRun, Detail: notRunNoRevision}
	}

	ctx, cancel := context.WithTimeout(ctx, buildVetTimeout)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "packets-gauntlet-build-*")
	if err != nil {
		return Gate{Status: GateNotRun, Detail: "could not prepare a scratch worktree"}
	}
	defer os.RemoveAll(tmpDir)

	worktree := filepath.Join(tmpDir, "wt")
	if _, err := runIn(ctx, repoDir, "git", "worktree", "add", "--detach", "--quiet", worktree, fixRev); err != nil {
		if ctx.Err() != nil {
			return Gate{Status: GateNotRun, Detail: "timed out checking out fix revision"}
		}
		return Gate{Status: GateNotRun, Detail: "could not check out fix revision"}
	}
	defer func() {
		// Best-effort: the tmpDir removal above still reclaims the disk even if
		// git's own bookkeeping entry lingers, so a failure here is never surfaced.
		_, _ = runIn(context.Background(), repoDir, "git", "worktree", "remove", "--force", worktree)
	}()

	if out, err := runIn(ctx, worktree, "go", "build", "./..."); err != nil {
		return GateForExecError(ctx, "go build", out, err)
	}
	if out, err := runIn(ctx, worktree, "go", "vet", "./..."); err != nil {
		return GateForExecError(ctx, "go vet", out, err)
	}
	return Gate{Status: GatePassed, Detail: "go build and go vet both clean"}
}

// GateForExecError decides a Gate from a failed exec call: if ctx is already
// done, the process was KILLED by the deadline, not proven to have failed on
// its own merits — reporting Failed here would fabricate a specific failure
// claim about a command that may well have succeeded given more time, which
// violates this package's "never fabricate a metric" invariant. Only a real
// (non-timeout) exit reports Failed, naming cmdName plus a truncated tail of
// its combined output. Shared by every exec-based gate (RunBuildVetGate,
// RunHandshakeGate) so the timeout-honesty rule lives in exactly one place.
func GateForExecError(ctx context.Context, cmdName, out string, err error) Gate {
	if ctx.Err() != nil {
		return Gate{Status: GateNotRun, Detail: cmdName + ": timed out before finishing"}
	}
	return Gate{Status: GateFailed, Detail: cmdName + ": " + truncateTail(out, detailTailLimit)}
}

// isGitRepo reports whether dir is inside a git working tree — the cheapest
// real check (never a heuristic like "does .git exist", which a worktree or
// a submodule can legitimately lack as a directory).
func isGitRepo(ctx context.Context, dir string) bool {
	_, err := runIn(ctx, dir, "git", "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// runIn runs name+args in dir and returns its combined stdout+stderr; err is
// non-nil on a non-zero exit or a failure to start.
func runIn(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// truncateTail trims s and, when it exceeds max BYTES, keeps only the LAST
// max bytes — the tail of a build/vet failure is where the actual error
// usually lands, ahead of any preamble. The cut point is advanced forward to
// the next rune boundary when it would otherwise land mid-rune, so the
// result is always valid UTF-8 (never a mangled multi-byte character at the
// front of a truncated Detail).
func truncateTail(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	cut := len(s) - max
	for cut < len(s) && !utf8.RuneStart(s[cut]) {
		cut++
	}
	return s[cut:]
}
