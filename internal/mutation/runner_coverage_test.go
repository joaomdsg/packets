package mutation

import (
	"context"
	"testing"
)

// The oracle's silence is dangerous if it can't be told apart: an empty
// finding list means "all mutants killed (line is tested)" ONLY when there
// were mutants to begin with. A line with no mutable operator also yields an
// empty list — but proves nothing. Run's result must distinguish the two by
// reporting how many mutable sites it actually considered, so a checks panel
// never renders "no oracle signal" as "verified".
func TestZeroFindingsDistinguishesNoSitesFromAllKilled(t *testing.T) {
	t.Parallel()
	// All-killed: adult_strong's `>=` IS a mutable site, and the strong test
	// kills its mutant — so 0 findings but 1 site considered = genuinely tested.
	strong, err := Run(context.Background(), Options{
		Dir:     "testdata/adult_strong",
		File:    "adult.go",
		Lines:   []LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run(adult_strong) error: %v", err)
	}
	if len(strong.Findings) != 0 {
		t.Fatalf("strong suite kills the mutant, want 0 findings, got %+v", strong.Findings)
	}
	if strong.MutantsConsidered != 1 {
		t.Fatalf("adult_strong line 4 has exactly one mutable `>=` site; MutantsConsidered = %d, want 1 (so 0 findings reads as genuinely tested)", strong.MutantsConsidered)
	}

	// Weak: the SAME single `>=` site, but the weak suite lets the mutant
	// survive. This pins the meaning of MutantsConsidered as TOTAL sites
	// considered (1) — not killed (0 here) and not survivors-only — since
	// here it must equal 1 while there is also 1 finding.
	weak, err := Run(context.Background(), Options{
		Dir:     "testdata/adult_weak",
		File:    "adult.go",
		Lines:   []LineRange{{Start: 4, End: 4}},
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run(adult_weak) error: %v", err)
	}
	if len(weak.Findings) != 1 {
		t.Fatalf("weak suite leaves 1 surviving mutant, got %+v", weak.Findings)
	}
	if weak.MutantsConsidered != 1 {
		t.Fatalf("MutantsConsidered counts total sites (1), independent of how many survived; got %d", weak.MutantsConsidered)
	}

	// No-sites: the target line uses only `<<`, a bit operator the oracle never
	// mutates, so there is nothing to test — 0 findings AND 0 sites considered = no signal.
	nosites, err := Run(context.Background(), Options{
		Dir:     "testdata/no_mutable_ops",
		File:    "calc.go",
		Lines:   []LineRange{{Start: 7, End: 7}},
		TestCmd: goTestCmd,
	})
	if err != nil {
		t.Fatalf("Run(no_mutable_ops) error: %v", err)
	}
	if len(nosites.Findings) != 0 {
		t.Fatalf("no mutable operators means no findings, got %+v", nosites.Findings)
	}
	if nosites.MutantsConsidered != 0 {
		t.Fatalf("a line with no mutable operator must report 0 sites considered (no oracle signal), got %d", nosites.MutantsConsidered)
	}
}
