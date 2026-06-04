package pipe

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/mutation"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

// LandState is the honest integration state of a settled revision. Integrate-on-
// tip (the {clean|conflict|checks-red} verdict on a rebased tree) is a later
// brick; until it exists the pipe reports Unintegrated rather than a fake
// "merged" — "Landed" is not "Merged".
type LandState string

// Unintegrated means the revision has not been integrated onto trunk tip: no
// rebase, no integrated checks have run. It is never to be read as merged.
const Unintegrated LandState = "unintegrated"

// CycleResult is the outcome of running one confirmed-catch cycle over two
// revisions: the catch verdict, the re-anchored anchor at the fix revision
// (Path/Line), the honest integration state, and an ordered, replayable Trace
// of the beats the cycle emitted (the catch appears as exactly one beat).
type CycleResult struct {
	Outcome catch.Outcome
	Path    string
	Line    int
	Land    LandState
	Trace   []string
}

// RunCatchCycle mints the catch's first real transaction from two real
// revisions: it runs the mutation oracle on baseRev and (when the anchor
// survives) on fixRev, each in a throwaway git worktree, builds the before/after
// LineStates, and routes them through CatchAcross — the authoritative,
// fail-closed gate. The verdict logic lives in CatchAcross/catch.Detect; this
// driver is the git+oracle orchestration around it.
func RunCatchCycle(ctx context.Context, repoDir, baseRev, fixRev string, anchor reanchor.Anchor, testCmd []string) (CycleResult, error) {
	var trace []string

	baseRes, srcBase, err := runOracleAt(ctx, repoDir, baseRev, anchor.Path, anchor.Start, testCmd)
	if err != nil {
		return CycleResult{}, err
	}
	trace = append(trace,
		fmt.Sprintf("settled base %s", short(baseRev)),
		fmt.Sprintf("oracle ran base: %d considered", baseRes.MutantsConsidered))
	beforeLS, err := catch.LineStateAt(srcBase, anchor.Start, baseRes)
	if err != nil {
		return CycleResult{}, err
	}

	ra, err := reanchor.Reanchor(ctx, repoDir, anchor, baseRev, fixRev)
	if err != nil {
		return CycleResult{}, err
	}

	var afterLS catch.LineState
	outPath, outLine := anchor.Path, anchor.Start
	if ra.State == reanchor.Same || ra.State == reanchor.Moved {
		outPath, outLine = ra.Path, ra.Start
		fixRes, srcFix, runErr := runOracleAt(ctx, repoDir, fixRev, ra.Path, ra.Start, testCmd)
		if runErr != nil {
			return CycleResult{}, runErr
		}
		trace = append(trace,
			fmt.Sprintf("settled fix %s", short(fixRev)),
			fmt.Sprintf("oracle ran fix: %d considered", fixRes.MutantsConsidered))
		afterLS, err = catch.LineStateAt(srcFix, ra.Start, fixRes)
		if err != nil {
			return CycleResult{}, err
		}
	} else {
		trace = append(trace, fmt.Sprintf("settled fix %s (anchor %s)", short(fixRev), ra.State))
	}

	outcome, err := CatchAcross(ctx, repoDir, anchor, baseRev, fixRev, beforeLS, afterLS)
	if err != nil {
		return CycleResult{}, err
	}
	trace = append(trace, fmt.Sprintf("catch: %s", outcome))

	return CycleResult{Outcome: outcome, Path: outPath, Line: outLine, Land: Unintegrated, Trace: trace}, nil
}

// runOracleAt materializes rev in a throwaway detached worktree, runs the
// mutation oracle scoped to the given line, and returns the result plus the
// file's content at that revision. The worktree is always cleaned up.
func runOracleAt(ctx context.Context, repoDir, rev, file string, line int, testCmd []string) (mutation.Result, []byte, error) {
	parent, err := os.MkdirTemp("", "agntpr-pipe-*")
	if err != nil {
		return mutation.Result{}, nil, fmt.Errorf("pipe: temp worktree dir: %w", err)
	}
	defer os.RemoveAll(parent)
	wt := filepath.Join(parent, "wt")
	if _, err := git(ctx, repoDir, "worktree", "add", "--detach", wt, rev); err != nil {
		return mutation.Result{}, nil, err
	}
	// Clean up against context.Background(), not ctx: a cancelled/timed-out
	// parent ctx would otherwise kill the cleanup git itself, leaving the
	// working dir removed (by RemoveAll) but a stale .git/worktrees/<id> entry
	// in the PRODUCTION repo. prune reaps that entry if remove still couldn't
	// (its gitdir now points at the deleted dir), so leaked admin metadata
	// can't accumulate across cycles.
	defer func() {
		git(context.Background(), repoDir, "worktree", "remove", "--force", wt) //nolint:errcheck // best-effort cleanup
		git(context.Background(), repoDir, "worktree", "prune")                 //nolint:errcheck // reaps a remove that couldn't run
	}()

	res, err := mutation.Run(ctx, mutation.Options{
		Dir:     wt,
		File:    file,
		Lines:   []mutation.LineRange{{Start: line, End: line}},
		TestCmd: testCmd,
	})
	if err != nil {
		return mutation.Result{}, nil, err
	}
	src, err := os.ReadFile(filepath.Join(wt, file))
	if err != nil {
		return mutation.Result{}, nil, fmt.Errorf("pipe: read %s at %s: %w", file, short(rev), err)
	}
	return res, src, nil
}

func short(rev string) string {
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}

func git(ctx context.Context, repoDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}
