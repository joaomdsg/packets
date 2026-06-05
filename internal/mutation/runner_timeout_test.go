package mutation_test

import (
	"context"
	"testing"
	"time"

	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_reportsNonTerminatingMutantUndeterminedNotKilled(t *testing.T) {
	t.Parallel()
	// Generous budget so first-compile latency of the fixture module cannot
	// expire the context before the hanging mutant is actually reached.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := mutation.Run(ctx, mutation.Options{
		Dir:     "testdata/loop_hang",
		File:    "loop.go",
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	findings := result.Findings

	// The `<`->`<=` mutant on the loop guard terminates and is genuinely
	// KILLED, so it must be omitted — never surfaced as undetermined. This
	// guards against a lazy fix that blanket-tags every non-killed result
	// undetermined.
	for _, f := range findings {
		if f.Original == "<" && f.Mutated == "<=" {
			assert.Fail(t, "the killed `<`->`<=` mutant must be omitted, not reported", "%+v", f)
		}
	}

	var undetermined []mutation.Finding
	for _, f := range findings {
		if f.Outcome == mutation.Undetermined {
			undetermined = append(undetermined, f)
		}
	}
	require.NotEmpty(t, undetermined, "the non-terminating `+`->`-` mutant must surface as undetermined; got findings=%+v", findings)

	var hang *mutation.Finding
	for i := range undetermined {
		if undetermined[i].Original == "+" && undetermined[i].Mutated == "-" {
			hang = &undetermined[i]
		}
	}
	require.NotNil(t, hang, "expected the +->- accumulator mutant among undetermined findings, got %+v", undetermined)
	assert.NotEmpty(t, hang.Message)
}

func TestRun_tagsSurvivingMutantSurvived(t *testing.T) {
	t.Parallel()
	result, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []mutation.LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	findings := result.Findings
	require.Len(t, findings, 1)
	assert.Equal(t, mutation.Survived, findings[0].Outcome)
	assert.Contains(t, findings[0].Message, "survived")
}
