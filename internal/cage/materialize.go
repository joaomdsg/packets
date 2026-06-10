// Package cage holds the trusted host-side preparation for running the catch
// oracle inside a sandbox: turning a claim's revisions into an isolated,
// disposable repo the cage can verify against without ever touching host state.
package cage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joaomdsg/packets/internal/ledger"
)

// ErrUnresolvableRevision marks a PERMANENT materialization failure: a target
// revision the host cannot resolve (absent, empty, or a producer's commit that
// never reached the host). No retry can succeed, so the caller maps it to the
// ledger's permanent-reject sentinel rather than treating it as a transient
// clone/IO failure. errors.Is against this distinguishes it from those.
var ErrUnresolvableRevision = errors.New("cage: host repo cannot resolve a target revision")

// The fixed layout of the cage's writable /work mount. The repo clone lives in a
// subdir; the go-tool scratch dirs are SIBLINGS so the mutation oracle's
// per-mutant copy of the repo tree does not recurse into them (and so TMPDIR
// lands here, off the small noexec /tmp tmpfs). The orchestrator derives the
// in-container paths (/work/repo, /work/gocache, …) from these same names.
const (
	subdirRepo    = "repo"
	subdirGoCache = "gocache"
	subdirGoTmp   = "gotmp"
	subdirGoPath  = "gopath"
	subdirTmp     = "tmp"
)

// Workdir is the cage's writable /work mount laid out for a verify-catch run.
// Root is the host dir to bind-mount at /work; Repo is the disposable validated
// clone (the -repo target); GoCache/GoTmp/GoPath/Tmp are empty writable scratch
// dirs the launch points HOME/GOCACHE/GOTMPDIR/GOPATH/TMPDIR at.
type Workdir struct {
	Root    string
	Repo    string
	GoCache string
	GoTmp   string
	GoPath  string
	Tmp     string
}

// Materialize builds a DISPOSABLE, WRITABLE cage workdir for the Target: a fresh
// Root holding a `git clone --local --no-hardlinks` of hostRepo in Repo (sharing
// no objects/inodes with the host) plus empty go-tool scratch siblings. The
// host's real repo is never handed to the cage; this Root is what gets
// bind-mounted, and the writable clone lets the oracle's `git worktree add`
// succeed (a read-only host-repo mount could not).
//
// It is fail-closed: a revision the host cannot resolve (absent or empty) is
// rejected before anything is created. On success it returns the Workdir and a
// cleanup that removes the whole Root; on any error it returns a nil Workdir, a
// nil cleanup, and leaves nothing behind.
func Materialize(ctx context.Context, hostRepo string, t ledger.Target) (*Workdir, func(), error) {
	for _, rev := range []string{t.BaseRev, t.FixRev, t.TipRev} {
		if strings.TrimSpace(rev) == "" {
			return nil, nil, fmt.Errorf("cage: empty target revision: %w", ErrUnresolvableRevision)
		}
		// `--end-of-options` stops a leading-dash rev (e.g. a forged claim's
		// "--git-dir=...") from being parsed as a git flag: everything after it is
		// a revision, never an option. Plain `--` does NOT serve this role for
		// `rev-parse --verify`.
		if err := git(ctx, hostRepo, "rev-parse", "--verify", "--quiet", "--end-of-options", rev+"^{commit}"); err != nil {
			// A context cancellation/deadline is TRANSIENT, not permanent: the
			// revision may be perfectly resolvable and the verify just ran out of
			// time (or the host is shutting down). Surface it unwrapped of the
			// permanent sentinel so the caller retries rather than durably rejecting
			// a valid claim. Only a genuine resolve failure wraps ErrUnresolvableRevision.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, nil, fmt.Errorf("cage: revision %q: %w", rev, ctxErr)
			}
			// Wrap the permanent sentinel (errors.Is target); fold the git stderr into
			// the message text since fmt.Errorf allows only one %w.
			return nil, nil, fmt.Errorf("cage: revision %q (%v): %w", rev, err, ErrUnresolvableRevision)
		}
	}

	root, err := os.MkdirTemp("", "packets-cage-*")
	if err != nil {
		return nil, nil, fmt.Errorf("cage: scratch dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(root) }

	wd := &Workdir{
		Root:    root,
		Repo:    filepath.Join(root, subdirRepo),
		GoCache: filepath.Join(root, subdirGoCache),
		GoTmp:   filepath.Join(root, subdirGoTmp),
		GoPath:  filepath.Join(root, subdirGoPath),
		Tmp:     filepath.Join(root, subdirTmp),
	}

	for _, dir := range []string{wd.GoCache, wd.GoTmp, wd.GoPath, wd.Tmp} {
		if err := os.Mkdir(dir, 0o777); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("cage: scratch subdir: %w", err)
		}
	}

	// `git clone --local` copies the whole object store, so every rev the host
	// resolved above is present in the clone — no second verification needed.
	// `--` separates options from the positional <repo> <dir> args so a
	// leading-dash hostRepo can never be parsed as a clone flag (e.g. `-c`).
	if err := git(ctx, "", "clone", "--local", "--no-hardlinks", "--quiet", "--", hostRepo, wd.Repo); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("cage: clone into scratch: %w", err)
	}

	return wd, cleanup, nil
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
