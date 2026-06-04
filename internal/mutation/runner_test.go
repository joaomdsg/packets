package mutation

import (
	"context"
	"os"
	"strings"
	"testing"
)

// goTestCmd runs the fixture module's own suite. `env -u GOROOT` works
// around this box's stale GOROOT; the runner itself stays env-agnostic.
var goTestCmd = []string{"env", "-u", "GOROOT", "go", "test", "./..."}

// The whole thesis: a green-but-weak test suite must be exposed. A test
// that can't distinguish `>=` from `>` lets the mutant survive, and the
// oracle must surface that exact line as a finding.
func TestWeakTestSuiteSurfacesSurvivingMutantAsFinding(t *testing.T) {
	t.Parallel()
	result, err := Run(context.Background(), Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	findings := result.Findings
	if len(findings) != 1 {
		t.Fatalf("weak suite must surface exactly 1 surviving mutant, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.File != "adult.go" {
		t.Errorf("finding file = %q, want adult.go", f.File)
	}
	if f.Line != 4 {
		t.Errorf("finding anchored to line %d, want 4", f.Line)
	}
	if f.Original != ">=" || f.Mutated != ">" {
		t.Errorf("want a >= -> > finding, got %q -> %q", f.Original, f.Mutated)
	}
	if !strings.Contains(f.Message, ">=") || !strings.Contains(f.Message, "line 4") {
		t.Errorf("message should explain the surviving mutation, got %q", f.Message)
	}
}

// The counter-case proves the oracle does not cry wolf: a suite that
// pins the boundary kills the mutant, so there is nothing to report.
func TestStrongTestSuiteLeavesNoFindings(t *testing.T) {
	t.Parallel()
	result, err := Run(context.Background(), Options{
		Dir:     "testdata/adult_strong",
		File:    "adult.go",
		Lines:   []LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("strong suite kills the mutant, want 0 findings, got %d: %+v", len(result.Findings), result.Findings)
	}
}

// Mutating the working tree must never be observable afterwards, or the
// oracle would corrupt the very code it inspects.
func TestRunRestoresTheOriginalFileAfterMutating(t *testing.T) {
	t.Parallel()
	const path = "testdata/adult_weak/adult.go"
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if _, err := Run(context.Background(), Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture after Run: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("Run must restore the original file byte-for-byte; it differs after running")
	}
}

// A missing target file is an operator error, not "no weak tests" — it
// must surface as an error rather than a silent clean report.
func TestRunErrorsWhenTargetFileIsMissing(t *testing.T) {
	t.Parallel()
	if _, err := Run(context.Background(), Options{
		Dir:     "testdata/adult_weak",
		File:    "does_not_exist.go",
		TestCmd: goTestCmd,
	}); err == nil {
		t.Fatal("expected an error when the target file is missing, got nil")
	}
}

// An empty test command is a configuration error and must fail cleanly
// rather than panic on argv[0] or be read as a killed mutant.
func TestRunErrorsOnEmptyTestCommand(t *testing.T) {
	t.Parallel()
	if _, err := Run(context.Background(), Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []LineRange{{Start: 4, End: 4}},
		TestCmd: nil,
	}); err == nil {
		t.Fatal("expected an error for an empty test command, got nil")
	}
}

// (The former TestRunErrorsWhenOriginalCannotBeRestored was removed: the
// oracle no longer mutates the original in place, so there is no restore path
// to fail. The stronger invariant — a read-only original runs fine and is
// never modified — is covered by TestReadOnlyOriginalRunsFineAndIsNeverModified
// in runner_parallel_test.go.)

// A test command that cannot even be launched is an infrastructure
// failure, NOT a killed mutant. Misreading it as "killed" would silently
// declare every weak test strong — the worst possible false negative.
func TestRunErrorsWhenTestCommandCannotStart(t *testing.T) {
	t.Parallel()
	if _, err := Run(context.Background(), Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []LineRange{{Start: 4, End: 4}},
		TestCmd: []string{"agntpr_no_such_command_zzz"},
	}); err == nil {
		t.Fatal("expected an error when the test command cannot start, got nil")
	}
}
