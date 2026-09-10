package mutation_test

import (
	"context"
	"testing"

	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_distinguishesNoSitesFromAllKilled(t *testing.T) {
	t.Parallel()
	// adult_strong's `>=` IS a mutable site, and the strong test kills its
	// mutant — so 0 findings but 1 site considered = genuinely tested.
	strong, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/adult_strong",
		File:    "adult.go",
		Lines:   []mutation.LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	assert.Empty(t, strong.Findings)
	assert.Equal(t, 1, strong.MutantsConsidered)

	// The SAME single `>=` site, but the weak suite lets the mutant survive:
	// MutantsConsidered is TOTAL sites considered (1), not killed and not
	// survivors-only, so here it equals 1 while there is also 1 finding.
	weak, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []mutation.LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	require.Len(t, weak.Findings, 1)
	assert.Equal(t, 1, weak.MutantsConsidered)

	// The target line (the `x &^= 2` body, line 10) is a COMPOUND ASSIGNMENT —
	// a single token.AND_NOT_ASSIGN in an *ast.AssignStmt, not an
	// *ast.BinaryExpr — so the oracle finds nothing to test: 0 findings AND 0
	// sites = no signal.
	nosites, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/no_mutable_ops",
		File:    "calc.go",
		Lines:   []mutation.LineRange{{Start: 10, End: 10}},
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	assert.Empty(t, nosites.Findings)
	assert.Equal(t, 0, nosites.MutantsConsidered)
}
