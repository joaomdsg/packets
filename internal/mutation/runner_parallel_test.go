package mutation_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/joaomdsg/agntpr/internal/mutation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_keepsConcurrentSurvivorsOrderedAndAttributed(t *testing.T) {
	t.Parallel()
	result, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/mixed",
		File:    "check.go",
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	assert.Equal(t, 6, result.MutantsConsidered)
	require.Len(t, result.Findings, 3)
	want := []struct {
		line     int
		original string
		mutated  string
	}{
		{5, ">", ">="},
		{11, "<", "<="},
		{17, ">", ">="},
	}
	for i, w := range want {
		got := result.Findings[i]
		assert.Equal(t, w.line, got.Line)
		assert.Equal(t, w.original, got.Original)
		assert.Equal(t, w.mutated, got.Mutated)
	}
}

func TestRun_keepsManyMutantsExceedingWorkerCapOrdered(t *testing.T) {
	t.Parallel()
	result, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     "testdata/many_mutants",
		File:    "count.go",
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	assert.Equal(t, 12, result.MutantsConsidered)
	require.Len(t, result.Findings, 12)
	lines := make([]int, len(result.Findings))
	for i, f := range result.Findings {
		lines[i] = f.Line
	}
	assert.True(t, sort.IntsAreSorted(lines), "findings must be ascending by line under concurrency, got %v", lines)
}

func TestRun_leavesReadOnlyOriginalUnmodified(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module ro\n\ngo 1.23\n"), 0o644))
	const src = "package ro\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n"
	target := filepath.Join(dir, "adult.go")
	require.NoError(t, os.WriteFile(target, []byte(src), 0o644))
	// Weak test: far-from-boundary value, so the `>=`->`>` mutant survives.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "adult_test.go"),
		[]byte("package ro\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif !IsAdult(25) {\n\t\tt.Fatal(\"25\")\n\t}\n}\n"), 0o644))

	// Make the target file and its directory read-only: writing a mutant in
	// place would now fail, so a copy-based oracle is the only thing that works.
	require.NoError(t, os.Chmod(target, 0o444))
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o755)
		_ = os.Chmod(target, 0o644)
	})

	result, err := mutation.Run(context.Background(), mutation.Options{
		Dir:     dir,
		File:    "adult.go",
		TestCmd: goTestCmd,
	})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)

	after, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, src, string(after))
}
