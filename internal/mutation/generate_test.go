package mutation_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMutants_pairsComparisonOperatorInChangedLine(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n")
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, mutants, 1)
	m := mutants[0]
	assert.Equal(t, ">=", m.Original)
	assert.Equal(t, ">", m.Mutated)
	assert.Equal(t, 4, m.Line)
}

func TestGenerateMutants_skipsSitesOutsideChangedLines(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\tx := a > b\n\ty := a < b\n\treturn x && y\n}\n")
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 5, End: 5}})
	require.NoError(t, err)
	require.Len(t, mutants, 1)
	m := mutants[0]
	assert.Equal(t, 5, m.Line)
	assert.Equal(t, "<", m.Original)
	assert.Equal(t, "<=", m.Mutated)
}

func TestGenerateMutants_replacesOnlyTheTargetedOperator(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\treturn a >= b\n}\n")
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, mutants, 1)
	want := "package p\n\nfunc f(a, b int) bool {\n\treturn a > b\n}\n"
	assert.Equal(t, want, string(mutants[0].Source))
}

func TestGenerateMutants_splicesOperatorsOfDifferentLengths(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b uint) uint {\n\treturn a ^ b\n}\n")
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, mutants, 1)
	want := "package p\n\nfunc f(a, b uint) uint {\n\treturn a &^ b\n}\n"
	assert.Equal(t, want, string(mutants[0].Source))

	shrinkSrc := []byte("package p\n\nfunc f(a, b uint) uint {\n\treturn a &^ b\n}\n")
	shrinkMuts, err := mutation.GenerateMutants(shrinkSrc, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, shrinkMuts, 1)
	shrinkWant := "package p\n\nfunc f(a, b uint) uint {\n\treturn a ^ b\n}\n"
	assert.Equal(t, shrinkWant, string(shrinkMuts[0].Source))
}

func TestGenerateMutants_considersAllSitesWhenChangedLinesEmpty(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\treturn a > b\n}\n\nfunc g(a, b int) bool {\n\treturn a < b\n}\n")
	mutants, err := mutation.GenerateMutants(src, nil)
	require.NoError(t, err)
	assert.Len(t, mutants, 2)
}

func TestGenerateMutants_returnsErrorOnUnparseableSource(t *testing.T) {
	t.Parallel()
	_, err := mutation.GenerateMutants([]byte("this is not valid go {{{"), nil)
	assert.Error(t, err)
}

func TestGenerateMutants_leavesCompoundAssignmentOperatorsUnmutated(t *testing.T) {
	t.Parallel()
	for _, op := range []string{"+=", "&=", "<<=", "&^="} {
		t.Run(op, func(t *testing.T) {
			t.Parallel()
			src := []byte("package p\n\nfunc f(a, b int) {\n\ta " + op + " b\n}\n")
			mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
			require.NoError(t, err)
			assert.Empty(t, mutants)
		})
	}
}

func TestGenerateMutants_treatsAndNotAsOneTokenNotSplit(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b uint) uint {\n\treturn a &^ b\n}\n")
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, mutants, 1)
	assert.Equal(t, "&^", mutants[0].Original)
	assert.Equal(t, "^", mutants[0].Mutated)
}

func TestGenerateMutants_leavesUnaryXorComplementUnmutated(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(x int) int {\n\treturn ^x\n}\n")
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	assert.Empty(t, mutants)
}

func TestGenerateMutants_leavesUnaryPlusAndMinusUnmutated(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(y int) int {\n\tx := -y\n\treturn +x\n}\n")
	mutants, err := mutation.GenerateMutants(src, nil)
	require.NoError(t, err)
	assert.Empty(t, mutants)
}

func TestGenerateMutants_producesOneMutantPerSiteOnOneLine(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b, c, d int) bool {\n\treturn a > b && c < d\n}\n")
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 4}})
	require.NoError(t, err)
	require.Len(t, mutants, 3)
	for _, m := range mutants {
		assert.Equal(t, 4, m.Line)
	}
}

func TestGenerateMutants_includesEverySiteWithinMultiLineRange(t *testing.T) {
	t.Parallel()
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\tp := a > b\n\tq := a < b\n\tr := a == b\n\treturn p && q || r\n}\n")
	mutants, err := mutation.GenerateMutants(src, []mutation.LineRange{{Start: 4, End: 6}})
	require.NoError(t, err)
	require.Len(t, mutants, 3)
	for _, m := range mutants {
		assert.GreaterOrEqual(t, m.Line, 4)
		assert.LessOrEqual(t, m.Line, 6)
	}
}
