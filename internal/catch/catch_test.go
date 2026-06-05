package catch_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/mutation"
)

func TestDetect_mintsCatchWhenStableLineSurvivorsCleared(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	assert.Equal(t, catch.Catch, catch.Detect(before, after))
}

func TestDetect_refusesCatchWhenFixChangesOperatorInventory(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">"}, Survivors: nil}
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after))
}

func TestDetect_refusesCatchWhenLineAlreadyConstrained(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after))
}

func TestDetect_refusesCatchForNoOpChurn(t *testing.T) {
	t.Parallel()
	state := catch.LineState{Inventory: []string{">=", "&&"}, Survivors: []string{">="}}
	assert.Equal(t, catch.NoCatch, catch.Detect(state, state))
}

func TestDetect_reportsNoOracleSignalForOperatorFreeLine(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: nil, Survivors: nil}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	assert.Equal(t, catch.NoOracleSignal, catch.Detect(before, after))
}

func TestDetect_reportsPartialCatchWhenSurvivorsShrinkButRemain(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">=", "<="}, Survivors: []string{">=", "<="}}
	after := catch.LineState{Inventory: []string{">=", "<="}, Survivors: []string{">="}}
	assert.Equal(t, catch.PartialCatch, catch.Detect(before, after))
}

func TestDetect_refusesCatchWhenNewSurvivorAppears(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">=", "&&"}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">=", "&&"}, Survivors: []string{"&&"}}
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after))
}

func TestDetect_refusesCatchWhenInventoryShrinks(t *testing.T) {
	t.Parallel()
	before := catch.LineState{Inventory: []string{">=", "&&"}, Survivors: []string{">="}}
	after := catch.LineState{Inventory: []string{">="}, Survivors: nil}
	assert.Equal(t, catch.NoCatch, catch.Detect(before, after))
}

func TestDetect_reportsNoOracleSignalWhenBothRevisionsOperatorFree(t *testing.T) {
	t.Parallel()
	empty := catch.LineState{}
	assert.Equal(t, catch.NoOracleSignal, catch.Detect(empty, empty))
}

const twoOpSrc = "package p\n\nfunc f(a, b, c, d int) bool {\n\treturn a >= b && c == d\n}\n"

func TestLineStateAt_deduplicatesRepeatedOperatorsIntoASet(t *testing.T) {
	t.Parallel()
	src := "package p\n\nfunc f(a, b, c, d int) bool {\n\treturn a >= b && c >= d\n}\n"
	res := mutation.Result{Findings: []mutation.Finding{
		{File: "p.go", Line: 4, Original: ">=", Outcome: mutation.Survived},
		{File: "p.go", Line: 4, Original: ">=", Outcome: mutation.Survived},
	}}
	ls, err := catch.LineStateAt([]byte(src), 4, res)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{">=", "&&"}, ls.Inventory)
	assert.ElementsMatch(t, []string{">="}, ls.Survivors)
}

func TestLineStateAt_derivesInventoryAndSurvivorsForTheAnchoredLine(t *testing.T) {
	t.Parallel()
	res := mutation.Result{
		Findings: []mutation.Finding{
			{File: "p.go", Line: 4, Original: ">=", Outcome: mutation.Survived},
			{File: "p.go", Line: 99, Original: "<", Outcome: mutation.Survived}, // other line
		},
		MutantsConsidered: 3,
	}
	ls, err := catch.LineStateAt([]byte(twoOpSrc), 4, res)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{">=", "&&", "=="}, ls.Inventory)
	assert.ElementsMatch(t, []string{">="}, ls.Survivors)
}

func TestLineStateAt_excludesUndeterminedFindingsFromSurvivors(t *testing.T) {
	t.Parallel()
	res := mutation.Result{
		Findings: []mutation.Finding{
			{File: "p.go", Line: 4, Original: "==", Outcome: mutation.Undetermined},
		},
		MutantsConsidered: 3,
	}
	ls, err := catch.LineStateAt([]byte(twoOpSrc), 4, res)
	require.NoError(t, err)
	assert.Empty(t, ls.Survivors)
}

func TestLineStateAt_reportsEmptyInventoryForOperatorFreeLine(t *testing.T) {
	t.Parallel()
	src := "package p\n\nfunc f() bool {\n\treturn true\n}\n"
	ls, err := catch.LineStateAt([]byte(src), 4, mutation.Result{})
	require.NoError(t, err)
	assert.Empty(t, ls.Inventory)
}

func TestLineStateAt_returnsErrorOnUnparseableSource(t *testing.T) {
	t.Parallel()
	_, err := catch.LineStateAt([]byte("package p\n\nfunc f( {"), 3, mutation.Result{})
	require.Error(t, err)
}

var goTestCmd = []string{"env", "-u", "GOROOT", "go", "test", "./..."}

const adultSrc = "package adult\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n"

func TestDetect_mintsCatchAcrossRealOracleRevisionsWhenTestStrengthened(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module adultcatch\n\ngo 1.23\n")
	writeFile(t, dir, "adult.go", adultSrc)
	writeFile(t, dir, "adult_test.go",
		"package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif !IsAdult(25) {\n\t\tt.Fatal(\"25\")\n\t}\n}\n")

	opts := mutation.Options{Dir: dir, File: "adult.go", Lines: []mutation.LineRange{{Start: 4, End: 4}}, TestCmd: goTestCmd}
	baseRes, err := mutation.Run(context.Background(), opts)
	require.NoError(t, err)
	before, err := catch.LineStateAt([]byte(adultSrc), 4, baseRes)
	require.NoError(t, err)
	require.NotEmpty(t, before.Survivors, "weak suite must leave a surviving mutant on line 4")

	// Strengthen the test; the anchored source line is unchanged.
	writeFile(t, dir, "adult_test.go",
		"package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif IsAdult(17) {\n\t\tt.Fatal(\"17 is not an adult\")\n\t}\n\tif !IsAdult(18) {\n\t\tt.Fatal(\"18 is an adult\")\n\t}\n}\n")
	fixRes, err := mutation.Run(context.Background(), opts)
	require.NoError(t, err)
	after, err := catch.LineStateAt([]byte(adultSrc), 4, fixRes)
	require.NoError(t, err)

	assert.Equal(t, catch.Catch, catch.Detect(before, after))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}
