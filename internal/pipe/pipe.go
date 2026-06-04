// Package pipe wires the standalone bricks (settle, diff, mutation, reanchor,
// catch, review) into the end-to-end review loop (DESIGN §17). Its first piece
// is the reanchor→catch JOIN: the single sanctioned path from a raw anchor and
// two revisions to a catch outcome.
package pipe

import (
	"context"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

// CatchAcross is the only sanctioned way to turn an anchored line plus two
// revisions into a catch outcome. It re-anchors the line itself (the
// safety-critical step that cannot be skipped) and only lets the after-state
// count when the anchor genuinely survived: if re-anchoring reports the line
// lost via rename or outdated, it returns NoOracleSignal and IGNORES `after`,
// so a stale or mis-derived after-state can never mint a phantom catch. The
// before/after LineStates are supplied by the caller (which runs the mutation
// oracle on each revision); CatchAcross decides whether the after-state is even
// allowed to count.
func CatchAcross(ctx context.Context, repoDir string, anchor reanchor.Anchor, beforeRev, afterRev string, before, after catch.LineState) (catch.Outcome, error) {
	ra, err := reanchor.Reanchor(ctx, repoDir, anchor, beforeRev, afterRev)
	if err != nil {
		return "", err
	}
	// Fail closed: the after-state is consulted ONLY when the anchor provably
	// survived (Same or Moved — both re-point to content that still hash-matches
	// the original line). Every other state — Outdated, LostViaRename, or any
	// future "lost"-like state — suppresses to NoOracleSignal, so a new reanchor
	// state can never silently open a phantom-catch path.
	if ra.State == reanchor.Same || ra.State == reanchor.Moved {
		return catch.Detect(before, after), nil
	}
	return catch.NoOracleSignal, nil
}
