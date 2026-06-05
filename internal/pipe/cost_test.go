package pipe_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/pipe"
)

// countingTestCmd returns a testCmd that records one byte to counterPath per
// suite execution, then runs the real Go suite. Because the runner copies the
// repo into per-worker dirs and integrateOnTip into a worktree, an ABSOLUTE
// counter path (outside those copies) is the one place every suite-exec is
// observed — so len(counter) is the exact number of full-suite runs a cycle
// fired. The `&&` preserves the real `go test` exit code as the verdict.
func countingTestCmd(counterPath string) []string {
	return []string{"sh", "-c", "printf x >> '" + counterPath + "' && exec env -u GOROOT go test ./..."}
}

func suiteExecCount(t *testing.T, counterPath string) int {
	t.Helper()
	b, err := os.ReadFile(counterPath)
	if os.IsNotExist(err) {
		return 0
	}
	require.NoError(t, err)
	return len(b)
}

func TestRunCatchCycle_firesExactlyOneSuitePerMutantPlusOneIntegration(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultpipe\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "fix: strengthen the test")

	counter := t.TempDir() + "/suite-execs"
	res, err := pipe.RunCatchCycle(context.Background(), dir, base, fix, fix, adultAnchor(), countingTestCmd(counter))
	require.NoError(t, err)

	// The anchored `>=` line has exactly one mutable operator, so the base oracle
	// and the fix oracle each fire 1 suite, and integrate-on-tip fires 1 more:
	// the cost of a cycle is M_base + M_fix + 1, pinned here as a regression guard
	// against the unquantified 3N→8N multiplier the live wire fans out per connect.
	assert.Equal(t, 1, beatsContaining(res.Trace, "oracle ran base: 1 considered"), "the base line has one mutable operator")
	assert.Equal(t, 3, suiteExecCount(t, counter),
		"a cycle fires exactly M_base(1) + M_fix(1) + integrate(1) = 3 full-suite executions")
}
