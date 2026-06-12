package ledger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

func TestRefinement_replaysBackAsTheSharpeningSoTheBacklogCanFoldIt(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	// A split sharpens one broad target into two — the bench card the Lead
	// accepts. The backlog projection (a later brick) folds this on read, so the
	// fact must replay back carrying every field it needs to do so.
	into := []ledger.Target{distinctTarget(), {BaseRev: "b2", FixRev: "f2", TipRev: "f2", Path: "other.go", Line: 40}}
	require.NoError(t, l.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 1, Target: ownTarget(), Refine: "split", Splits: into,
	}))

	refs, err := l.Refinements()
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, 1, refs[0].RefineID)
	assert.Equal(t, ownTarget(), refs[0].Target, "the fold needs the parent target the split replaces")
	assert.Equal(t, "split", refs[0].Refine)
	assert.Equal(t, into, refs[0].Splits, "the fold needs the resulting sub-targets")
}

func TestRefinement_carriesCriteriaAndConventionTextBackForTheCardBody(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	require.NoError(t, l.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 1, Target: distinctTarget(), Refine: "criteria",
		Criteria: []string{"rejects a negative amount", "caps at the daily ceiling"},
	}))
	require.NoError(t, l.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 2, Target: distinctTarget(), Refine: "convention",
		Note: "errors wrap with a short origin prefix",
	}))

	refs, err := l.Refinements()
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, []string{"rejects a negative amount", "caps at the daily ceiling"}, refs[0].Criteria)
	assert.Equal(t, "errors wrap with a short origin prefix", refs[1].Note)
}

func TestRefinement_isNotEconomicSoSharpeningNeverMintsOrFundsWork(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	// Sharpening is an append-only fact beside the economy, NOT a catch, spend, or
	// work-order. It also exercises the replay path: the fold errors on an unknown
	// kind, so a refinement that did not thread through would break every read.
	require.NoError(t, l.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 1, Target: distinctTarget(), Refine: "criteria",
		Criteria: []string{"rejects a negative amount"},
	}))

	bal, err := l.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "a refinement mints no credit and debits nothing")
	recs, err := l.Records()
	require.NoError(t, err)
	assert.Empty(t, recs, "a refinement is not a catch")
	orders, err := l.WorkOrders()
	require.NoError(t, err)
	assert.Empty(t, orders, "sharpening a target funds no work-order on its own")
}

func TestRefinement_replaysFromThePersistedLogAlone(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)
	require.NoError(t, l.AppendRefine(ledger.RefinedOrderRecord{
		RefineID: 1, Target: ownTarget(), Refine: "split", Splits: []ledger.Target{distinctTarget()},
	}))
	require.NoError(t, l.Close())

	reopened := boundLog(t)
	refs, err := reopened.Refinements()
	require.NoError(t, err)
	require.Len(t, refs, 1, "a refinement is an appended line, so it survives a reopen with no in-memory state")
	assert.Equal(t, "split", refs[0].Refine)
}
