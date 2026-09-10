// Package orchestrator is the host-side coordinator. This brick is
// its core revision-minting step: composing settle (the turn-boundary commit
// guard, with its no-edit and secret-block protections) and diff (the
// structured base..head changeset) into the data behind a revision.created /
// diff.data event. The broader orchestrator — harness supervision, the
// WebSocket gateway, container lifecycle, and the stateful event reducer
// (threads/checks/permissions) — is deferred.
package orchestrator

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/settle"
)

// TurnOutcome is the result of settling one harness turn:
//   - Minted: a revision was committed; SHA, Added, Deleted, and Diff describe it.
//   - Secrets non-empty: the turn was BLOCKED by a detected secret — no revision.
//   - neither: a no-edit / net-revert turn — nothing to revise.
//
// Added/Deleted are the changeset totals across all files (the revision.created
// stats); the changed-file count is len(Diff.Files).
type TurnOutcome struct {
	Minted  bool
	SHA     string
	Added   int
	Deleted int
	Diff    diff.Diff
	Secrets []settle.SecretHit
}

// SettleTurn settles the working tree at repoDir into a revision when the turn
// changed something, and computes the baseRev..newRevision diff for it. A
// secret in the change blocks the revision (surfaced, not errored); a no-change
// turn mints nothing. Errors from the underlying git operations propagate.
func SettleTurn(ctx context.Context, repoDir, baseRev, message string) (TurnOutcome, error) {
	res, err := settle.Settle(ctx, repoDir, message)
	if err != nil {
		return TurnOutcome{}, err
	}
	if len(res.Secrets) > 0 {
		return TurnOutcome{Secrets: res.Secrets}, nil // blocked: no revision, no diff
	}
	if !res.Committed {
		return TurnOutcome{}, nil // no-edit / net-revert turn
	}

	d, err := diff.Compute(ctx, repoDir, baseRev, res.SHA)
	if err != nil {
		return TurnOutcome{}, err
	}
	added, deleted := 0, 0
	for _, f := range d.Files {
		added += f.Added
		deleted += f.Deleted
	}
	return TurnOutcome{Minted: true, SHA: res.SHA, Added: added, Deleted: deleted, Diff: d}, nil
}

// DiscardWorkingTree rolls the repo's working tree back to rev: it restores tracked files
// (git reset --hard) and removes untracked, non-ignored files/dirs a partial turn created
// (git clean -fd). It is the rollback for an errored/timed-out turn (DESIGN §1183 / the
// timeout row: "partial edits discarded"): the residue is cleared so settle's whole-tree
// `git add -A` cannot sweep it into a LATER turn's mint. It resets to the explicit rev
// (the caller's last revision), not HEAD, so the intended base is honored even if HEAD
// ever drifts.
func DiscardWorkingTree(ctx context.Context, repoDir, rev string) error {
	if err := gitDiscard(ctx, repoDir, "reset", "--hard", rev); err != nil {
		return err
	}
	return gitDiscard(ctx, repoDir, "clean", "-fd")
}

func gitDiscard(ctx context.Context, repoDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoDir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
