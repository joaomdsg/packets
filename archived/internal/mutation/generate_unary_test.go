package mutation_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMutants_mutatesUnaryNotByRemovingNegation(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(ok bool) bool {\n\treturn !ok\n}\n")
	muts, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, muts, 1)
	assert.Equal(t, "!", muts[0].Original)
	assert.Empty(t, muts[0].Mutated)
	assert.Contains(t, string(muts[0].Source), "return ok")
	assert.NotContains(t, string(muts[0].Source), "!ok")
}

func TestGenerateMutants_treatsNotEqualsAsBinaryNeqNotUnaryNot(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\treturn a != b\n}\n")
	muts, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, muts, 1)
	assert.Equal(t, "!=", muts[0].Original)
	assert.Equal(t, "==", muts[0].Mutated)
}

func TestGenerateMutants_mutatesUnaryNotAlongsideInnerBinary(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b bool) bool {\n\treturn !(a && b)\n}\n")
	muts, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, muts, 2)
	var sawNot, sawAnd bool
	for _, m := range muts {
		if m.Original == "!" && m.Mutated == "" {
			sawNot = true
		}
		if m.Original == "&&" && m.Mutated == "||" {
			sawAnd = true
		}
	}
	assert.True(t, sawNot, "the `!` removal mutant is missing: %+v", muts)
	assert.True(t, sawAnd, "the inner `&&`->`||` mutant must still be produced inside a `!`: %+v", muts)
}

func TestGenerateMutants_leavesUnaryNotOutsideChangedLinesUnmutated(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(ok bool) bool {\n\treturn !ok\n}\n\nfunc g(x int) int {\n\treturn x\n}\n")
	muts, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 8, End: 8}})
	require.NoError(t, err)
	assert.Empty(t, muts)
}

func TestGenerateMutants_producesOneMutantPerNotInDoubleNegation(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(ok bool) bool {\n\treturn !!ok\n}\n")
	muts, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	assert.Len(t, muts, 2)
}
