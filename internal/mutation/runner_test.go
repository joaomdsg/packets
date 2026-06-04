package mutation_test

import (
	"context"
	"os"
	"testing"

	"github.com/joaomdsg/agntpr/internal/mutation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goTestCmd runs the fixture module's own suite. `env -u GOROOT` works
// around this box's stale GOROOT; the runner itself stays env-agnostic.
var goTestCmd = []string{"env", "-u", "GOROOT", "go", "test", "./..."}

func TestRun_surfacesSurvivingMutantFromWeakSuite(t *testing.T) {
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
	f := findings[0]
	assert.Equal(t, "adult.go", f.File)
	assert.Equal(t, 4, f.Line)
	assert.Equal(t, ">=", f.Original)
	assert.Equal(t, ">", f.Mutated)
	assert.Contains(t, f.Message, ">=")
	assert.Contains(t, f.Message, "line 4")
}

func TestRun_leavesNoFindingsForStrongSuite(t *testing.T) {
	t.Parallel()
	result, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/adult_strong",
		File:    "adult.go",
		Lines:   []mutation.LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
}

func TestRun_restoresOriginalFileAfterMutating(t *testing.T) {
	t.Parallel()
	const path = "testdata/adult_weak/adult.go"
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	_, err = mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []mutation.LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestRun_returnsErrorWhenTargetFileMissing(t *testing.T) {
	t.Parallel()
	_, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/adult_weak",
		File:    "does_not_exist.go",
		TestCmd: goTestCmd,
	})
	assert.Error(t, err)
}

func TestRun_returnsErrorOnEmptyTestCommand(t *testing.T) {
	t.Parallel()
	_, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []mutation.LineRange{{Start: 4, End: 4}},
		TestCmd: nil,
	})
	assert.Error(t, err)
}

func TestRun_returnsErrorWhenTestCommandCannotStart(t *testing.T) {
	t.Parallel()
	_, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []mutation.LineRange{{Start: 4, End: 4}},
		TestCmd: []string{"agntpr_no_such_command_zzz"},
	})
	assert.Error(t, err)
}
