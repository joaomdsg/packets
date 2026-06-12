package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

func TestFundableBacklog_aSplitReplacesItsParentTargetWithItsSubTargets(t *testing.T) {
	t.Parallel()
	log := scratchLog(t)
	parent := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}
	other := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "auth.go", Line: 12}
	subA := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 90}
	subB := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 120}
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 1, Target: parent, Refine: "split", Splits: []ledger.Target{subA, subB},
	}))

	cfg := LiveConfig{BaseRev: "own-b", FixRev: "own-f", TipRev: "own-f", Anchor: anchorForCap(),
		DispatchBacklog: []ledger.Target{parent, other}}

	// A split sharpening folds on read: the broad parent is no longer fundable as
	// one unit — it is replaced IN PLACE by its sub-targets, so the Lead funds the
	// sharpened work, not the vague original.
	out := fundableBacklog(cfg, log)
	assert.Equal(t, []ledger.Target{subA, subB, other}, out,
		"the split's sub-targets replace the parent in order; the rest of the backlog is untouched")
	assert.NotContains(t, out, parent, "the broad parent target is no longer fundable once split")
}

func TestFundableBacklog_criteriaAndConventionRefinementsLeaveTheTargetListUnchanged(t *testing.T) {
	t.Parallel()
	log := scratchLog(t)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 1, Target: tgt, Refine: "criteria", Criteria: []string{"rejects a negative amount"},
	}))
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 2, Target: tgt, Refine: "convention", Note: "wrap errors with an origin prefix",
	}))

	cfg := LiveConfig{BaseRev: "own-b", FixRev: "own-f", TipRev: "own-f", Anchor: anchorForCap(),
		DispatchBacklog: []ledger.Target{tgt}}

	// Criteria/convention annotate the card body — they do NOT split or remove the
	// target — so the fundable set is exactly the original work.
	out := fundableBacklog(cfg, log)
	assert.Equal(t, []ledger.Target{tgt}, out,
		"a criteria/convention refinement annotates, never changes what is fundable")
}

func TestFundableBacklog_splitSubTargetsStillPassTheConsumedAndOwnFilters(t *testing.T) {
	t.Parallel()
	log := scratchLog(t)
	cfg := LiveConfig{BaseRev: "own-b", FixRev: "own-f", TipRev: "own-f", Anchor: anchorForCap()}
	own := ownTargetOf(cfg)
	parent := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}
	fresh := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 90}

	// A split must not smuggle a banned target past the filters: one sub-target IS
	// the card's own caught cycle (which AppendDispatch refuses), so it must be
	// dropped exactly as a top-level own target would be.
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 1, Target: parent, Refine: "split", Splits: []ledger.Target{own, fresh},
	}))
	cfg.DispatchBacklog = []ledger.Target{parent}

	out := fundableBacklog(cfg, log)
	assert.Equal(t, []ledger.Target{fresh}, out,
		"the split's sub-targets pass the same own/consumed filters — the own-cycle sub-target is dropped")
}

func TestFundableBacklog_aSecondSplitOfTheSameTargetSupersedesTheFirst(t *testing.T) {
	t.Parallel()
	log := scratchLog(t)
	parent := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}
	first := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 90}
	second := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 120}
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{RefineID: 1, Target: parent, Refine: "split", Splits: []ledger.Target{first}}))
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{RefineID: 2, Target: parent, Refine: "split", Splits: []ledger.Target{second}}))

	cfg := LiveConfig{BaseRev: "own-b", FixRev: "own-f", TipRev: "own-f", Anchor: anchorForCap(),
		DispatchBacklog: []ledger.Target{parent}}

	// Re-splitting the same target is the Lead correcting an earlier sharpening; the
	// latest split is authoritative (last-writer-wins, like an order's status), not
	// an accumulation of both attempts.
	out := fundableBacklog(cfg, log)
	assert.Equal(t, []ledger.Target{second}, out, "the latest split supersedes the earlier one")
}

func TestFundableBacklog_anEmptySplitLeavesTheParentFundableRatherThanErasingIt(t *testing.T) {
	t.Parallel()
	log := scratchLog(t)
	parent := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{RefineID: 1, Target: parent, Refine: "split", Splits: nil}))

	cfg := LiveConfig{BaseRev: "own-b", FixRev: "own-f", TipRev: "own-f", Anchor: anchorForCap(),
		DispatchBacklog: []ledger.Target{parent}}

	// A split into nothing is malformed (the proposed-then-accept UI never emits it);
	// it must NOT silently erase the parent from the fundable set — losing real work
	// is worse than ignoring a degenerate refinement.
	out := fundableBacklog(cfg, log)
	assert.Equal(t, []ledger.Target{parent}, out, "an empty split is ignored; the parent stays fundable")
}
