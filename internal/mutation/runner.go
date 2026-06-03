package mutation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Options configures a mutation-testing run over a single file.
type Options struct {
	Dir     string      // working directory the test command runs in
	File    string      // target file to mutate, relative to Dir
	Lines   []LineRange // changed lines to scope mutation to (empty = all)
	TestCmd []string    // argv of the suite to run; exit 0 = pass
}

// Finding reports a mutant that survived the test suite — a line whose
// behaviour the tests fail to constrain.
type Finding struct {
	File     string
	Line     int
	Original string
	Mutated  string
	Message  string
}

// Run mutates the target file one operator at a time, runs TestCmd
// against each mutant, and returns a Finding for every mutant that
// SURVIVES (the suite still passes). The original file is always
// restored before Run returns, even on error.
func Run(ctx context.Context, opts Options) (findings []Finding, err error) {
	path := filepath.Join(opts.Dir, opts.File)
	original, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read target file: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat target file: %w", err)
	}
	mode := info.Mode().Perm()

	mutants, err := GenerateMutants(original, opts.Lines)
	if err != nil {
		return nil, err
	}

	// Always restore the original, even on early return or panic. A
	// failed restore leaves the working tree mutated, which the spec
	// forbids, so surface it as an error — but never mask an error Run is
	// already returning.
	defer func() {
		if restoreErr := os.WriteFile(path, original, mode); restoreErr != nil && err == nil {
			err = fmt.Errorf("restore target file: %w", restoreErr)
			findings = nil
		}
	}()

	for _, m := range mutants {
		if writeErr := os.WriteFile(path, m.Source, mode); writeErr != nil {
			return nil, fmt.Errorf("write mutant: %w", writeErr)
		}
		passed, runErr := runTests(ctx, opts.Dir, opts.TestCmd)
		if runErr != nil {
			return nil, runErr
		}
		if passed {
			findings = append(findings, Finding{
				File:     opts.File,
				Line:     m.Line,
				Original: m.Original,
				Mutated:  m.Mutated,
				Message: fmt.Sprintf(
					"Mutation survived: changed `%s` to `%s` on line %d and all tests still passed — is line %d actually constrained by a test?",
					m.Original, m.Mutated, m.Line, m.Line),
			})
		}
	}
	return findings, nil
}

// runTests executes the suite in dir. It returns (true, nil) when the
// suite passes (exit 0 → mutant survived), (false, nil) when it fails
// (the mutant is killed), and a non-nil error only when the command
// could not be run to completion — which must never be mistaken for a
// killed mutant.
func runTests(ctx context.Context, dir string, argv []string) (bool, error) {
	if len(argv) == 0 {
		return false, errors.New("empty test command")
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir

	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil // suite ran and failed → mutant killed
	}
	return false, fmt.Errorf("run test command %v: %w", argv, err)
}
