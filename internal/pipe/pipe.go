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

// Reason is the orthogonal cause behind a quiet (NoOracleSignal) verdict — a
// dimension distinct from the catch.Outcome token, so three different "the
// oracle is silent" truths never collapse into one and the surface can never
// state a false reason. It is empty (ReasonNone) for any outcome that carries
// its own meaning (Catch / NoCatch / PartialCatch).
type Reason string

const (
	// ReasonNone: the outcome speaks for itself; there is no quiet cause to explain.
	ReasonNone Reason = ""
	// ReasonNoMutableOperator: the anchor survived but the line has no mutable
	// operator, so the oracle genuinely has nothing to say about it.
	ReasonNoMutableOperator Reason = "no_mutable_operator"
	// ReasonAnchorEdited: the anchored line was edited between the revisions, so
	// the oracle can no longer speak to the original line.
	ReasonAnchorEdited Reason = "anchor_edited"
	// ReasonFileRenamed: the anchored file was renamed and the line was lost, so
	// the oracle cannot follow it.
	ReasonFileRenamed Reason = "file_renamed"
)

// CatchAcross is the only sanctioned way to turn an anchored line plus two
// revisions into a catch outcome. It re-anchors the line itself (the
// safety-critical step that cannot be skipped) and only lets the after-state
// count when the anchor genuinely survived: if re-anchoring reports the line
// lost via rename or outdated, it returns NoOracleSignal and IGNORES `after`,
// so a stale or mis-derived after-state can never mint a phantom catch. The
// before/after LineStates are supplied by the caller (which runs the mutation
// oracle on each revision); CatchAcross decides whether the after-state is even
// allowed to count. The returned Reason carries WHY a quiet verdict is quiet so
// the surface can state a true cause rather than a single overloaded token.
func CatchAcross(ctx context.Context, repoDir string, anchor reanchor.Anchor, beforeRev, afterRev string, before, after catch.LineState) (catch.Outcome, Reason, error) {
	ra, err := reanchor.Reanchor(ctx, repoDir, anchor, beforeRev, afterRev)
	if err != nil {
		return "", ReasonNone, err
	}
	// Fail closed: the after-state is consulted ONLY when the anchor provably
	// survived (Same or Moved — both re-point to content that still hash-matches
	// the original line). Every other state suppresses to NoOracleSignal, so a
	// new reanchor state can never silently open a phantom-catch path.
	if ra.State == reanchor.Same || ra.State == reanchor.Moved {
		o := catch.Detect(before, after)
		if o == catch.NoOracleSignal {
			return o, ReasonNoMutableOperator, nil
		}
		return o, ReasonNone, nil
	}
	if ra.State == reanchor.LostViaRename {
		return catch.NoOracleSignal, ReasonFileRenamed, nil
	}
	// Outdated: the anchored line drifted/was edited. (A future "lost"-like
	// reanchor state would land here too and read as edited until given its own
	// reason — fail-closed safety is preserved regardless.)
	return catch.NoOracleSignal, ReasonAnchorEdited, nil
}
