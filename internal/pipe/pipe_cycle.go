package pipe

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// LandState is the integration verdict for a settled revision: the result of
// rebasing the fix onto the current trunk tip and running the checks on the
// integrated tree. It is computed on the tree that actually integrates, so the
// catch is never priced against a stale pre-integration base — "Landed" is not
// "Merged" until this verdict is LandClean.
type LandState string

const (
	// LandClean: the fix rebases onto trunk tip with no conflict AND the
	// integrated tree's checks pass — the fix genuinely integrates.
	LandClean LandState = "clean"
	// LandConflict: the fix conflicts textually with trunk tip and cannot
	// integrate without a manual rebase. Checks are not run (nothing to check).
	LandConflict LandState = "conflict"
	// LandChecksRed: the fix rebases cleanly but the integrated tree's checks
	// fail — a fix green in isolation is a regression once integrated.
	LandChecksRed LandState = "checks_red"
)

// TraceEvent is one beat of the cycle: a typed, timestamped event stamped at the
// real transition it reports (after the work, not before). Kind is the stable
// vocabulary the surface streams and paces against (settle-base, oracle-base,
// settle-fix, oracle-fix, catch, land); Msg is the human-readable line; T lets
// the surface tune cadence to the real wall-clock of oracle + rebase work.
type TraceEvent struct {
	T    time.Time
	Kind string
	Msg  string
}

// CycleResult is the outcome of running one confirmed-catch cycle over two
// revisions: the catch verdict, the re-anchored anchor at the fix revision
// (Path/Line), the honest integration state, and an ordered, replayable Trace
// of the beats the cycle emitted (the catch appears as exactly one beat).
type CycleResult struct {
	Outcome catch.Outcome
	// Reason is the orthogonal cause behind a quiet Outcome (NoOracleSignal): why
	// the oracle is silent — no mutable operator vs the anchor edited vs the file
	// renamed — so the surface states a true cause instead of one overloaded
	// token. It is ReasonNone for an Outcome that carries its own meaning.
	Reason  Reason
	Path    string
	Line    int
	Land    LandState
	Trace   []TraceEvent
	// Before and After are the anchored line's operator-inventory state at each
	// revision, exposed so the surface presenter can tell the verified-strong
	// "Tested" screen from blind no-signal, and the ledger can record the
	// survivor-set transition. After is the zero LineState when the anchor did
	// not survive (Outdated/LostViaRename).
	Before catch.LineState
	After  catch.LineState
	// Findings are the FIX revision's oracle findings — the per-line mutation
	// results (incl. the surviving mutants the reviewer should see as "question:"
	// threads), carried up so the review surface can render them instead of dying
	// inside the cycle. Stamped at the reanchored (fix) coordinates. Nil when the
	// anchor did not survive to the fix (Outdated/LostViaRename) — no fix oracle ran,
	// so there are no honest fix-revision findings to show.
	Findings []mutation.Finding
}

// RunCatchCycle mints the catch's first real transaction from two real
// revisions: it runs the mutation oracle on baseRev and (when the anchor
// survives) on fixRev, each in a throwaway git worktree, builds the before/after
// LineStates, and routes them through CatchAcross — the authoritative,
// fail-closed gate. The verdict logic lives in CatchAcross/catch.Detect; this
// driver is the git+oracle orchestration around it.
func RunCatchCycle(ctx context.Context, repoDir, baseRev, fixRev, tipRev string, anchor reanchor.Anchor, testCmd []string) (CycleResult, error) {
	return RunCatchCycleStreaming(ctx, repoDir, baseRev, fixRev, tipRev, anchor, testCmd, nil)
}

// RunCatchCycleStreaming is RunCatchCycle with a live beats channel: it emits
// each TraceEvent on `beats` AT its real transition (in addition to the returned
// CycleResult.Trace), so a consumer can stream the felt loop beat-by-beat instead
// of seeing one terminal snap. `beats` must be buffered to at least the beat count
// (≈6) or drained concurrently — sends are synchronous; a nil channel is the
// non-streaming path (RunCatchCycle). The verdict/Land logic is identical.
func RunCatchCycleStreaming(ctx context.Context, repoDir, baseRev, fixRev, tipRev string, anchor reanchor.Anchor, testCmd []string, beats chan<- TraceEvent) (CycleResult, error) {
	var trace []TraceEvent
	emit := func(evs ...TraceEvent) {
		for _, ev := range evs {
			trace = append(trace, ev)
			if beats != nil {
				beats <- ev
			}
		}
	}

	baseRes, srcBase, err := runOracleAt(ctx, repoDir, baseRev, anchor.Path, anchor.Start, testCmd)
	if err != nil {
		return CycleResult{}, err
	}
	emit(
		TraceEvent{T: time.Now(), Kind: "settle-base", Msg: fmt.Sprintf("settled base %s", short(baseRev))},
		TraceEvent{T: time.Now(), Kind: "oracle-base", Msg: fmt.Sprintf("oracle ran base: %d considered", baseRes.MutantsConsidered)})
	beforeLS, err := catch.LineStateAt(srcBase, anchor.Start, baseRes)
	if err != nil {
		return CycleResult{}, err
	}

	ra, err := reanchor.Reanchor(ctx, repoDir, anchor, baseRev, fixRev)
	if err != nil {
		return CycleResult{}, err
	}

	var afterLS catch.LineState
	var findings []mutation.Finding
	outPath, outLine := anchor.Path, anchor.Start
	if ra.State == reanchor.Same || ra.State == reanchor.Moved {
		outPath, outLine = ra.Path, ra.Start
		fixRes, srcFix, runErr := runOracleAt(ctx, repoDir, fixRev, ra.Path, ra.Start, testCmd)
		if runErr != nil {
			return CycleResult{}, runErr
		}
		findings = fixRes.Findings // the reviewed revision's findings — carried up for the review surface
		emit(
			TraceEvent{T: time.Now(), Kind: "settle-fix", Msg: fmt.Sprintf("settled fix %s", short(fixRev))},
			TraceEvent{T: time.Now(), Kind: "oracle-fix", Msg: fmt.Sprintf("oracle ran fix: %d considered", fixRes.MutantsConsidered)})
		afterLS, err = catch.LineStateAt(srcFix, ra.Start, fixRes)
		if err != nil {
			return CycleResult{}, err
		}
	} else {
		// Anchor lost (Outdated/LostViaRename): the fix settled but no oracle-fix
		// beat fires — the stream reports the real path taken, not a fixed animation.
		emit(TraceEvent{T: time.Now(), Kind: "settle-fix", Msg: fmt.Sprintf("settled fix %s (anchor %s)", short(fixRev), ra.State)})
	}

	outcome, reason, err := CatchAcross(ctx, repoDir, anchor, baseRev, fixRev, beforeLS, afterLS)
	if err != nil {
		return CycleResult{}, err
	}
	emit(TraceEvent{T: time.Now(), Kind: "catch", Msg: fmt.Sprintf("catch: %s", outcome)})

	land, err := integrateOnTip(ctx, repoDir, fixRev, tipRev, testCmd)
	if err != nil {
		return CycleResult{}, err
	}
	emit(TraceEvent{T: time.Now(), Kind: "land", Msg: fmt.Sprintf("land: %s", land)})

	return CycleResult{
		Outcome: outcome, Reason: reason, Path: outPath, Line: outLine, Land: land, Trace: trace,
		Before: beforeLS, After: afterLS, Findings: findings,
	}, nil
}

// integrateOnTip computes the integration verdict for the fix against trunk tip:
// it rebases fixRev onto tipRev in a throwaway detached worktree and runs the
// checks on the INTEGRATED tree. A textual conflict short-circuits to
// LandConflict (no checks — there is nothing coherent to check); a clean rebase
// runs testCmd, green → LandClean, red → LandChecksRed (a fix green in isolation
// regressing once integrated). The catch is minted on the base revisions; this
// verdict is the orthogonal answer to "does it integrate", never folded into it.
//
// This is ONE serialized integration lane per call. A multi-card server MUST
// route Land through a single queue and never fan out N concurrent rebases onto
// a shared tip (the O(N^2)/8N contention regime) — the verdict is meaningless if
// the tip it integrates onto is itself racing.
func integrateOnTip(ctx context.Context, repoDir, fixRev, tipRev string, testCmd []string) (LandState, error) {
	// Fail closed like mutation.Run's runTests, not with a panic: an
	// operator-free anchored line lets the oracle return 0 mutants without ever
	// tripping its own empty-testCmd guard, so this is the first place an empty
	// testCmd is observed — and the clean-rebase path below would index
	// testCmd[0]/testCmd[1:].
	if len(testCmd) == 0 {
		return "", fmt.Errorf("pipe: empty test command")
	}
	// Fail closed on an empty tipRev too. `git rebase ""` exits non-zero
	// ("invalid upstream"), and the clean-rebase path below maps ANY rebase
	// failure to LandConflict — so an omitted tip (a LiveConfig zero-value) would
	// silently render "Trunk moved — rebase needed", a dishonest verdict for what
	// is a caller/config error. An empty tip has no integration answer; surface it.
	if tipRev == "" {
		return "", fmt.Errorf("pipe: empty trunk tip revision")
	}
	parent, err := os.MkdirTemp("", "packets-land-*")
	if err != nil {
		return "", fmt.Errorf("pipe: temp worktree dir: %w", err)
	}
	defer os.RemoveAll(parent)
	wt := filepath.Join(parent, "wt")
	if _, err := git(ctx, repoDir, "worktree", "add", "--detach", wt, fixRev); err != nil {
		return "", err
	}
	// Same cleanup discipline as runOracleAt: clean up against context.Background()
	// so a cancelled parent ctx can't strand a .git/worktrees entry in the prod repo.
	defer func() {
		git(context.Background(), repoDir, "worktree", "remove", "--force", wt) //nolint:errcheck // best-effort cleanup
		git(context.Background(), repoDir, "worktree", "prune")                 //nolint:errcheck // reaps a remove that couldn't run
	}()

	// Replay the fix's commits (merge-base(tipRev,fixRev)..fixRev) onto tipRev. A
	// non-zero exit is a textual conflict; abort it (so the worktree is clean for
	// removal) and report the conflict without running checks.
	if _, err := git(ctx, wt, "rebase", tipRev); err != nil {
		git(context.Background(), wt, "rebase", "--abort") //nolint:errcheck // clears the in-progress rebase before removal
		return LandConflict, nil
	}

	// Clean rebase: run the checks on the integrated tree via controlled exec and
	// read the real exit code (not an agent's shell), so a masked failure can't
	// read as green.
	check := exec.CommandContext(ctx, testCmd[0], testCmd[1:]...)
	check.Dir = wt
	if err := check.Run(); err != nil {
		return LandChecksRed, nil
	}
	return LandClean, nil
}

// buildWorktreeAt materializes rev in a throwaway detached git worktree and returns
// the worktree path plus a cleanup func the caller MUST defer. Extracted so both the
// catch-cycle oracle (runOracleAt) and the review re-run (RerunWithTestOverlay) share
// one worktree lifecycle instead of duplicating the create/cleanup discipline.
//
// cleanup removes the worktree and prunes its admin entry against
// context.Background(), NOT ctx: a cancelled/timed-out parent ctx would otherwise
// kill the cleanup git itself, leaving the working dir removed but a stale
// .git/worktrees/<id> entry in the PRODUCTION repo. prune reaps that entry if remove
// couldn't (its gitdir now points at the deleted dir), so leaked admin metadata
// can't accumulate across cycles.
func buildWorktreeAt(ctx context.Context, repoDir, rev string) (wt string, cleanup func(), err error) {
	parent, err := os.MkdirTemp("", "packets-pipe-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("pipe: temp worktree dir: %w", err)
	}
	wt = filepath.Join(parent, "wt")
	if _, err := git(ctx, repoDir, "worktree", "add", "--detach", wt, rev); err != nil {
		os.RemoveAll(parent)
		return "", func() {}, err
	}
	cleanup = func() {
		git(context.Background(), repoDir, "worktree", "remove", "--force", wt) //nolint:errcheck // best-effort cleanup
		git(context.Background(), repoDir, "worktree", "prune")                 //nolint:errcheck // reaps a remove that couldn't run
		os.RemoveAll(parent)
	}
	return wt, cleanup, nil
}

// RerunWithTestOverlay re-runs the mutation oracle at rev, scoped to file:line, with
// the reviewer's test files OVERLAID into a throwaway worktree — the diagnostic
// re-run behind the editable-answering review flow: the reviewer writes a test that
// should kill a surviving mutant, and this reports the fresh findings so the surface
// can tell whether the answer constrained the line (a killing test → the finding
// disappears; a weak one → it remains). Each overlay entry path→content is written
// into the worktree (parent dirs created), overwriting any committed file at that
// path, so `go test ./...` picks up the reviewer's test. It is READ-ONLY w.r.t. the
// economy: it returns findings and touches no ledger.
func RerunWithTestOverlay(ctx context.Context, repoDir, rev, file string, line int, testCmd []string, overlay map[string]string) ([]mutation.Finding, error) {
	wt, cleanup, err := buildWorktreeAt(ctx, repoDir, rev)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	for path, content := range overlay {
		// The overlay carries reviewer-influenced paths; reject any that isn't a plain
		// relative path inside the worktree (absolute, or "..\"-escaping) BEFORE any
		// write, so an answer can never clobber a file outside the throwaway worktree.
		if !filepath.IsLocal(path) {
			return nil, fmt.Errorf("pipe: overlay path %q is not local to the worktree", path)
		}
		dst := filepath.Join(wt, path)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, fmt.Errorf("pipe: overlay mkdir %s: %w", path, err)
		}
		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("pipe: overlay write %s: %w", path, err)
		}
	}

	res, err := mutation.Run(ctx, mutation.Options{
		Dir:     wt,
		File:    file,
		Lines:   []mutation.LineRange{{Start: line, End: line}},
		TestCmd: testCmd,
	})
	if err != nil {
		return nil, err
	}
	return res.Findings, nil
}

// runOracleAt materializes rev in a throwaway detached worktree, runs the
// mutation oracle scoped to the given line, and returns the result plus the
// file's content at that revision. The worktree is always cleaned up.
func runOracleAt(ctx context.Context, repoDir, rev, file string, line int, testCmd []string) (mutation.Result, []byte, error) {
	wt, cleanup, err := buildWorktreeAt(ctx, repoDir, rev)
	if err != nil {
		return mutation.Result{}, nil, err
	}
	defer cleanup()

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
