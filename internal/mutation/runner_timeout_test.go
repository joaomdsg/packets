package mutation

import (
	"context"
	"strings"
	"testing"
	"time"
)

// The oracle's whole value is that "no finding" means "the tests constrain
// this line." A mutant whose test run TIMES OUT proves nothing — the suite
// never finished — so it must never be silently treated as killed (which
// would falsely certify the line as covered). It must surface as a distinct
// UNDETERMINED finding so the reviewer knows coverage there is unknown.
func TestNonTerminatingMutantIsReportedUndeterminedNotSilentlyKilled(t *testing.T) {
	t.Parallel()
	// Generous budget so first-compile latency of the fixture module cannot
	// expire the context before the hanging mutant is actually reached.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := Run(ctx, Options{
		Dir:     "testdata/loop_hang",
		File:    "loop.go",
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	findings := result.Findings

	// The `<`->`<=` mutant on the loop guard terminates and is genuinely
	// KILLED (Grow returns 6, the test wants 5), so it must be omitted —
	// never surfaced as undetermined. This guards against a lazy fix that
	// blanket-tags every non-killed result undetermined.
	for _, f := range findings {
		if f.Original == "<" && f.Mutated == "<=" {
			t.Errorf("the killed `<`->`<=` mutant must be omitted, not reported as %q: %+v", f.Outcome, f)
		}
	}

	var undetermined []Finding
	for _, f := range findings {
		if f.Outcome == Undetermined {
			undetermined = append(undetermined, f)
		}
	}
	if len(undetermined) == 0 {
		t.Fatalf("the non-terminating `+`->`-` mutant must surface as undetermined, not be silently killed; got findings=%+v", findings)
	}

	var hang *Finding
	for i := range undetermined {
		if undetermined[i].Original == "+" && undetermined[i].Mutated == "-" {
			hang = &undetermined[i]
		}
	}
	if hang == nil {
		t.Fatalf("expected the +->- accumulator mutant among undetermined findings, got %+v", undetermined)
	}
	if hang.Message == "" {
		t.Errorf("undetermined finding must carry a message explaining the run did not complete")
	}
}

// A genuinely surviving mutant must be tagged Survived (not left with a
// zero/blank outcome), so callers can tell a real coverage gap apart from an
// undetermined (timed-out) one.
func TestSurvivingMutantIsTaggedSurvived(t *testing.T) {
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
	if findings[0].Outcome != Survived {
		t.Errorf("a surviving mutant must be tagged Survived, got %q", findings[0].Outcome)
	}
	if !strings.Contains(findings[0].Message, "survived") {
		t.Errorf("survived finding message should say so, got %q", findings[0].Message)
	}
}
