package mutation

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// Options configures a mutation-testing run over a single file.
type Options struct {
	Dir     string      // working directory the test command runs in
	File    string      // target file to mutate, relative to Dir
	Lines   []LineRange // changed lines to scope mutation to (empty = all)
	TestCmd []string    // argv of the suite to run; exit 0 = pass
}

// Outcome classifies what the test suite did to a mutant. Only non-killed
// mutants are reported as Findings; the Outcome says why each one matters.
type Outcome string

const (
	// Survived means the suite passed with the mutant in place — the line is
	// not constrained by any test (green is decorative there).
	Survived Outcome = "survived"
	// Undetermined means the suite did not finish (e.g. the mutant made it
	// hang and the run was timed out), so it proves nothing. It must NOT be
	// read as evidence the line is covered.
	Undetermined Outcome = "undetermined"
)

// Finding reports a mutant the suite failed to KILL: either it survived (the
// tests passed anyway) or its run was undetermined (the suite never finished).
// Either way it marks a line whose coverage the suite does not establish.
type Finding struct {
	File     string
	Line     int
	Original string
	Mutated  string
	Outcome  Outcome
	Message  string
}

// Result is the outcome of a Run: the non-killed Findings plus how many
// mutable sites the oracle actually considered. MutantsConsidered disambiguates
// an empty Findings list — MutantsConsidered > 0 means "every mutant was killed"
// (the line is genuinely constrained), whereas MutantsConsidered == 0 means
// "no mutable operators here, so the oracle has no signal". The latter must
// never be read as "verified".
type Result struct {
	Findings          []Finding
	MutantsConsidered int
}

// mutantVerdict is the internal tri-state of one mutant's test run.
type mutantVerdict int

const (
	mutantKilled       mutantVerdict = iota // suite ran and failed → killed
	mutantSurvived                          // suite ran and passed → survived
	mutantUndetermined                      // run did not complete (timed out)
)

// maxWorkers caps how many isolated working copies (and concurrent test
// processes) a single Run will use, so a large diff cannot oversubscribe the
// machine with test suites all at once.
const maxWorkers = 8

// Run mutates the target file one operator at a time, runs TestCmd against each
// mutant CONCURRENTLY in isolated working copies, and returns a Result: a
// Finding for every mutant the suite did not KILL (survivors — suite still
// passed; and undetermined mutants — the run timed out, so the verdict can't be
// trusted), plus MutantsConsidered, the number of mutable sites generated.
// Killed mutants are omitted from Findings. Findings are ordered by line.
//
// The caller's opts.Dir is NEVER modified: each worker mutates its own copy.
func Run(ctx context.Context, opts Options) (Result, error) {
	path := filepath.Join(opts.Dir, opts.File)
	original, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read target file: %w", err)
	}

	mutants, err := GenerateMutants(original, opts.Lines)
	if err != nil {
		return Result{}, err
	}
	if len(mutants) == 0 {
		return Result{MutantsConsidered: 0}, nil
	}

	// Isolated working copies — never touch opts.Dir. One copy per worker,
	// reused across the mutants that worker processes.
	parent, err := os.MkdirTemp("", "agntpr-mutation-*")
	if err != nil {
		return Result{}, fmt.Errorf("create temp work dir: %w", err)
	}
	defer os.RemoveAll(parent)

	numWorkers := len(mutants)
	if n := runtime.NumCPU(); n < numWorkers {
		numWorkers = n
	}
	if numWorkers > maxWorkers {
		numWorkers = maxWorkers
	}

	workerDirs := make([]string, numWorkers)
	for i := range workerDirs {
		wd := filepath.Join(parent, fmt.Sprintf("w%d", i))
		if copyErr := copyDir(opts.Dir, wd); copyErr != nil {
			return Result{}, fmt.Errorf("copy working dir: %w", copyErr)
		}
		workerDirs[i] = wd
	}

	// Cancel siblings on a genuine infra error (bad/unlaunchable TestCmd, write
	// failure). A parent-ctx timeout is NOT cancelled here: remaining mutants
	// then run against a done ctx and classify as Undetermined (never killed).
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	verdicts := make([]mutantVerdict, len(mutants))
	jobs := make(chan int)
	var (
		mu       sync.Mutex
		firstErr error
		wg       sync.WaitGroup
	)
	recordErr := func(e error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = e
		}
		mu.Unlock()
		cancel()
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()
			target := filepath.Join(dir, opts.File)
			for idx := range jobs {
				if writeErr := os.WriteFile(target, mutants[idx].Source, 0o644); writeErr != nil {
					recordErr(fmt.Errorf("write mutant: %w", writeErr))
					continue
				}
				v, runErr := runTests(runCtx, dir, opts.TestCmd)
				if runErr != nil {
					recordErr(runErr)
					continue
				}
				verdicts[idx] = v
			}
		}(workerDirs[w])
	}

	// Feed every mutant; workers always drain the channel, so this never
	// deadlocks and no mutant is left unclassified on the success path.
	for i := range mutants {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return Result{}, firstErr
	}

	var findings []Finding
	for idx, m := range mutants {
		switch verdicts[idx] {
		case mutantSurvived:
			findings = append(findings, Finding{
				File:     opts.File,
				Line:     m.Line,
				Original: m.Original,
				Mutated:  m.Mutated,
				Outcome:  Survived,
				Message: fmt.Sprintf(
					"Mutation survived: changed `%s` to `%s` on line %d and all tests still passed — is line %d actually constrained by a test?",
					m.Original, m.Mutated, m.Line, m.Line),
			})
		case mutantUndetermined:
			findings = append(findings, Finding{
				File:     opts.File,
				Line:     m.Line,
				Original: m.Original,
				Mutated:  m.Mutated,
				Outcome:  Undetermined,
				Message: fmt.Sprintf(
					"Mutation undetermined: changed `%s` to `%s` on line %d but the test run did not complete (timed out) — coverage of line %d is unknown, not confirmed killed.",
					m.Original, m.Mutated, m.Line, m.Line),
			})
		}
	}
	return Result{Findings: findings, MutantsConsidered: len(mutants)}, nil
}

// copyDir recursively copies the tree at src into dst. Copied files are made
// owner-writable (|0o200) so a read-only original (e.g. a checked-out source
// tree) can still be mutated in the copy.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm()|0o200)
	})
}

// runTests executes the suite in dir and classifies the mutant. A non-nil
// error is returned ONLY when the command could not be run to completion for
// reasons unrelated to the suite's own verdict (bad config, can't launch) —
// which must never be mistaken for a killed mutant.
//
// Crucially, a process killed because ctx fired (a timeout/cancel) is
// reported as mutantUndetermined, NOT mutantKilled: the suite never produced
// a trustworthy verdict, so treating it as "killed" would falsely certify the
// line as covered.
func runTests(ctx context.Context, dir string, argv []string) (mutantVerdict, error) {
	if len(argv) == 0 {
		return mutantKilled, errors.New("empty test command")
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir

	err := cmd.Run()
	if err == nil {
		return mutantSurvived, nil // exit 0 → mutant survived
	}
	// If the context fired, WE killed the process; the suite's exit is not a
	// real verdict. Check this before ExitError, since a ctx-killed process
	// also surfaces as an *exec.ExitError (signal: killed).
	if ctx.Err() != nil {
		return mutantUndetermined, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return mutantKilled, nil // suite ran and failed → mutant killed
	}
	return mutantKilled, fmt.Errorf("run test command %v: %w", argv, err)
}
