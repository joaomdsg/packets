package mutation

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Findings must reach the reviewer top-to-bottom in file order, the way they
// scan a diff. Running mutants concurrently must NOT scramble that order, and
// each finding must carry the operator of ITS OWN line — a collector that
// appends in completion order, or maps a verdict to the wrong mutant index,
// would betray itself here. The fixture has six mutable sites; three are KILLED
// (interleaved) and three SURVIVE at non-contiguous lines 5, 11, 17. The result
// must be exactly those three survivors, in order, each with the right operator,
// and the killed ones omitted — while MutantsConsidered still counts all six.
func TestConcurrentRunKeepsSurvivorsOrderedAndCorrectlyAttributed(t *testing.T) {
	t.Parallel()
	result, err := Run(context.Background(), Options{
		Dir:     "testdata/mixed",
		File:    "check.go",
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.MutantsConsidered != 6 {
		t.Errorf("six mutable sites considered, got %d", result.MutantsConsidered)
	}
	if len(result.Findings) != 3 {
		t.Fatalf("three mutants survive (b/d/g killed), got %d: %+v", len(result.Findings), result.Findings)
	}
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
		if got.Line != w.line || got.Original != w.original || got.Mutated != w.mutated {
			t.Errorf("finding %d = line %d %q->%q, want line %d %q->%q (full: %+v)",
				i, got.Line, got.Original, got.Mutated, w.line, w.original, w.mutated, result.Findings)
		}
	}
}

// More mutable sites than the worker cap (12 > maxWorkers) forces several
// mutants through each worker and runs many test processes at once — the
// regime most likely to expose an order- or index-scrambling collector. Every
// site survives, so all twelve must come back, counted and strictly ascending
// by line, no matter how their concurrent runs interleave.
func TestManyMutantsExceedingWorkerCapStayOrdered(t *testing.T) {
	t.Parallel()
	result, err := Run(context.Background(), Options{
		Dir:     "testdata/many_mutants",
		File:    "count.go",
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.MutantsConsidered != 12 {
		t.Errorf("twelve mutable sites considered, got %d", result.MutantsConsidered)
	}
	if len(result.Findings) != 12 {
		t.Fatalf("all twelve mutants survive, got %d: %+v", len(result.Findings), result.Findings)
	}
	lines := make([]int, len(result.Findings))
	for i, f := range result.Findings {
		lines[i] = f.Line
	}
	if !sort.IntsAreSorted(lines) {
		t.Errorf("findings must be strictly ascending by line under concurrency, got %v", lines)
	}
}

// The oracle must never mutate the user's actual tree — it works on isolated
// copies. A read-only original (e.g. a checked-out source the reviewer hasn't
// made writable) must therefore run fine, not fail trying to write a mutant in
// place, and must be left byte-for-byte unchanged. This is the stronger
// successor to the old "restore failed" path: corruption is now impossible by
// construction, so a read-only tree is simply supported.
func TestReadOnlyOriginalRunsFineAndIsNeverModified(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module ro\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const src = "package ro\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n"
	target := filepath.Join(dir, "adult.go")
	if err := os.WriteFile(target, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	// Weak test: far-from-boundary value, so the `>=`->`>` mutant survives.
	if err := os.WriteFile(filepath.Join(dir, "adult_test.go"),
		[]byte("package ro\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif !IsAdult(25) {\n\t\tt.Fatal(\"25\")\n\t}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make the target file and its directory read-only: writing a mutant in
	// place would now fail, so a copy-based oracle is the only thing that works.
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o755)
		_ = os.Chmod(target, 0o644)
	})

	result, err := Run(context.Background(), Options{
		Dir:     dir,
		File:    "adult.go",
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run on a read-only original must succeed (it copies, never writes the original); got error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("the weak suite leaves the `>=` mutant surviving; want 1 finding, got %+v", result.Findings)
	}

	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read original after Run: %v", err)
	}
	if string(after) != src {
		t.Errorf("original file must be byte-for-byte unchanged; it differs after Run")
	}
}
