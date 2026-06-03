package mutation

import "testing"

// A mutation site inside the changed region must surface as exactly one
// mutant carrying its complementary operator — this is the raw material
// the oracle later runs against the test suite.
func TestComparisonInChangedLineYieldsItsPairedMutant(t *testing.T) {
	src := []byte("package p\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n")
	// `>=` sits on line 4.
	mutants, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
	if err != nil {
		t.Fatalf("GenerateMutants returned error: %v", err)
	}
	if len(mutants) != 1 {
		t.Fatalf("want exactly 1 mutant, got %d: %+v", len(mutants), mutants)
	}
	m := mutants[0]
	if m.Original != ">=" || m.Mutated != ">" {
		t.Errorf("want >= -> >, got %q -> %q", m.Original, m.Mutated)
	}
	if m.Line != 4 {
		t.Errorf("want mutant anchored to line 4, got line %d", m.Line)
	}
}

// Diff-scoping is the whole point: code the agent did not touch must not
// generate review noise, so sites outside the changed lines are skipped.
func TestSitesOutsideChangedLinesAreNotMutated(t *testing.T) {
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\tx := a > b\n\ty := a < b\n\treturn x && y\n}\n")
	// `>` line 4, `<` line 5, `&&` line 6 — only line 5 changed.
	mutants, err := GenerateMutants(src, []LineRange{{Start: 5, End: 5}})
	if err != nil {
		t.Fatalf("GenerateMutants returned error: %v", err)
	}
	if len(mutants) != 1 {
		t.Fatalf("want exactly 1 mutant (only the changed line), got %d: %+v", len(mutants), mutants)
	}
	m := mutants[0]
	if m.Line != 5 || m.Original != "<" || m.Mutated != "<=" {
		t.Errorf("want line 5 < -> <=, got line %d %q -> %q", m.Line, m.Original, m.Mutated)
	}
}

// The mutated source the runner will compile must differ from the
// original in exactly the targeted operator and nothing else, or the
// experiment measures the wrong change.
func TestMutatedSourceReplacesOnlyTheTargetedOperator(t *testing.T) {
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\treturn a >= b\n}\n")
	mutants, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
	if err != nil {
		t.Fatalf("GenerateMutants returned error: %v", err)
	}
	if len(mutants) != 1 {
		t.Fatalf("want exactly 1 mutant, got %d", len(mutants))
	}
	want := "package p\n\nfunc f(a, b int) bool {\n\treturn a > b\n}\n"
	if string(mutants[0].Source) != want {
		t.Errorf("mutated source mismatch:\n got: %q\nwant: %q", mutants[0].Source, want)
	}
}

// Every operator the oracle claims to cover must flip to its documented
// complement; a gap here is a class of weak test the oracle silently
// cannot detect.
func TestSupportedOperatorsMapToTheirComplement(t *testing.T) {
	cases := []struct {
		expr     string
		original string
		mutated  string
	}{
		{"a > b", ">", ">="},
		{"a >= b", ">=", ">"},
		{"a < b", "<", "<="},
		{"a <= b", "<=", "<"},
		{"a == b", "==", "!="},
		{"a != b", "!=", "=="},
		{"a + b", "+", "-"},
		{"a - b", "-", "+"},
		{"a && b", "&&", "||"},
		{"a || b", "||", "&&"},
	}
	for _, c := range cases {
		t.Run(c.expr, func(t *testing.T) {
			src := []byte("package p\n\nfunc f(a, b int) int {\n\t_ = " + c.expr + "\n\treturn 0\n}\n")
			mutants, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
			if err != nil {
				t.Fatalf("GenerateMutants returned error: %v", err)
			}
			if len(mutants) != 1 {
				t.Fatalf("want exactly 1 mutant for %q, got %d", c.expr, len(mutants))
			}
			if mutants[0].Original != c.original || mutants[0].Mutated != c.mutated {
				t.Errorf("for %q want %q -> %q, got %q -> %q",
					c.expr, c.original, c.mutated, mutants[0].Original, mutants[0].Mutated)
			}
		})
	}
}

// With no changed-line filter the oracle must consider the whole file,
// so a caller can mutation-test an entire module when no diff is given.
func TestEmptyChangedLinesConsidersAllSites(t *testing.T) {
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\treturn a > b\n}\n\nfunc g(a, b int) bool {\n\treturn a < b\n}\n")
	mutants, err := GenerateMutants(src, nil)
	if err != nil {
		t.Fatalf("GenerateMutants returned error: %v", err)
	}
	if len(mutants) != 2 {
		t.Fatalf("want 2 mutants (whole file), got %d: %+v", len(mutants), mutants)
	}
}

// Unparseable input must fail loudly rather than silently producing no
// findings, which would read as "all tests are strong".
func TestUnparseableSourceReturnsError(t *testing.T) {
	if _, err := GenerateMutants([]byte("this is not valid go {{{"), nil); err == nil {
		t.Fatal("expected an error for unparseable source, got nil")
	}
}

// Operators the oracle has no defined complement for must be left alone;
// mutating them would emit garbage findings the reviewer can't act on.
func TestUnsupportedBinaryOperatorsAreNotMutated(t *testing.T) {
	for _, op := range []string{"*", "/", "%", "<<", ">>", "&", "|", "^", "&^"} {
		t.Run(op, func(t *testing.T) {
			src := []byte("package p\n\nfunc f(a, b int) int {\n\treturn a " + op + " b\n}\n")
			mutants, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
			if err != nil {
				t.Fatalf("GenerateMutants returned error: %v", err)
			}
			if len(mutants) != 0 {
				t.Errorf("operator %q is unsupported and must not mutate, got %d mutants", op, len(mutants))
			}
		})
	}
}

// Unary +/- are not binary arithmetic; treating them as mutation sites
// would both miscount and produce nonsensical mutants.
func TestUnaryPlusAndMinusAreNotMutated(t *testing.T) {
	src := []byte("package p\n\nfunc f(y int) int {\n\tx := -y\n\treturn +x\n}\n")
	mutants, err := GenerateMutants(src, nil)
	if err != nil {
		t.Fatalf("GenerateMutants returned error: %v", err)
	}
	if len(mutants) != 0 {
		t.Errorf("unary +/- must not be mutated, got %d mutants: %+v", len(mutants), mutants)
	}
}

// Several mutation sites on the same changed line must each become their
// own mutant; collapsing them would hide weak tests.
func TestMultipleSitesOnOneLineEachProduceAMutant(t *testing.T) {
	src := []byte("package p\n\nfunc f(a, b, c, d int) bool {\n\treturn a > b && c < d\n}\n")
	mutants, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
	if err != nil {
		t.Fatalf("GenerateMutants returned error: %v", err)
	}
	if len(mutants) != 3 { // `>`, `&&`, `<`
		t.Fatalf("want 3 mutants on the one changed line, got %d: %+v", len(mutants), mutants)
	}
	for _, m := range mutants {
		if m.Line != 4 {
			t.Errorf("mutant %q->%q anchored to line %d, want 4", m.Original, m.Mutated, m.Line)
		}
	}
}

// A multi-line changed range must include every site between Start and
// End inclusive, and exclude sites outside it.
func TestMultiLineRangeIncludesEverySiteWithin(t *testing.T) {
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\tp := a > b\n\tq := a < b\n\tr := a == b\n\treturn p && q || r\n}\n")
	// sites: line4 `>`, line5 `<`, line6 `==`, line7 `&&` and `||`.
	mutants, err := GenerateMutants(src, []LineRange{{Start: 4, End: 6}})
	if err != nil {
		t.Fatalf("GenerateMutants returned error: %v", err)
	}
	if len(mutants) != 3 {
		t.Fatalf("want 3 mutants within lines 4-6, got %d: %+v", len(mutants), mutants)
	}
	for _, m := range mutants {
		if m.Line < 4 || m.Line > 6 {
			t.Errorf("mutant on line %d is outside the changed range 4-6", m.Line)
		}
	}
}
