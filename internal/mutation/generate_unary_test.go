package mutation

import (
	"strings"
	"testing"
)

// Negated guards (`if !ok`, `if !valid`) are a common coverage blind spot: a
// test that doesn't pin the condition's polarity passes whether or not the `!`
// is there. The oracle must mutate unary `!` by REMOVING the negation so that
// gap surfaces.
func TestUnaryNotIsMutatedByRemovingTheNegation(t *testing.T) {
	src := []byte("package p\n\nfunc f(ok bool) bool {\n\treturn !ok\n}\n")
	muts, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
	if err != nil {
		t.Fatalf("GenerateMutants: %v", err)
	}
	if len(muts) != 1 {
		t.Fatalf("want exactly 1 mutant for `!ok`, got %d: %+v", len(muts), muts)
	}
	if muts[0].Original != "!" || muts[0].Mutated != "" {
		t.Errorf("want `!` -> `` (removed), got %q -> %q", muts[0].Original, muts[0].Mutated)
	}
	if !strings.Contains(string(muts[0].Source), "return ok") {
		t.Errorf("mutated source must drop the `!` (yielding `return ok`), got:\n%s", muts[0].Source)
	}
	if strings.Contains(string(muts[0].Source), "!ok") {
		t.Errorf("mutated source must not still contain `!ok`, got:\n%s", muts[0].Source)
	}
}

// `!=` is a single NEQ token (a binary operator), NOT a unary `!` followed by
// `=`. It must produce only the binary NEQ->EQL mutant, never a spurious
// unary-removal mutant.
func TestNotEqualsIsBinaryNeqNotUnaryNot(t *testing.T) {
	src := []byte("package p\n\nfunc f(a, b int) bool {\n\treturn a != b\n}\n")
	muts, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
	if err != nil {
		t.Fatalf("GenerateMutants: %v", err)
	}
	if len(muts) != 1 {
		t.Fatalf("`a != b` is one binary NEQ mutation, got %d: %+v", len(muts), muts)
	}
	if muts[0].Original != "!=" || muts[0].Mutated != "==" {
		t.Errorf("want `!=` -> `==`, got %q -> %q", muts[0].Original, muts[0].Mutated)
	}
}

// The new unary path must COEXIST with the binary path, not shadow it:
// `!(a && b)` has two independent sites — the `!` (removed) and the inner `&&`
// (→`||`) — and both must be produced. (Also confirms removing the `!` from a
// parenthesized expr yields valid code: `(a && b)`.)
func TestUnaryNotCoexistsWithInnerBinaryMutation(t *testing.T) {
	src := []byte("package p\n\nfunc f(a, b bool) bool {\n\treturn !(a && b)\n}\n")
	muts, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
	if err != nil {
		t.Fatalf("GenerateMutants: %v", err)
	}
	if len(muts) != 2 {
		t.Fatalf("want 2 mutants (the `!` removal AND the inner `&&`->`||`), got %d: %+v", len(muts), muts)
	}
	var sawNot, sawAnd bool
	for _, m := range muts {
		if m.Original == "!" && m.Mutated == "" {
			sawNot = true
		}
		if m.Original == "&&" && m.Mutated == "||" {
			sawAnd = true
		}
	}
	if !sawNot {
		t.Errorf("the `!` removal mutant is missing: %+v", muts)
	}
	if !sawAnd {
		t.Errorf("the inner `&&`->`||` mutant must still be produced inside a `!`: %+v", muts)
	}
}

// The changed-line scope must apply to unary `!` too: a `!` outside the given
// line ranges must not be mutated (otherwise the oracle would mutate code the
// turn didn't touch).
func TestUnaryNotOutsideChangedLinesIsNotMutated(t *testing.T) {
	// `!ok` is on line 4; scope only line 8 (`return x`, which has no operator).
	src := []byte("package p\n\nfunc f(ok bool) bool {\n\treturn !ok\n}\n\nfunc g(x int) int {\n\treturn x\n}\n")
	muts, err := GenerateMutants(src, []LineRange{{Start: 8, End: 8}})
	if err != nil {
		t.Fatalf("GenerateMutants: %v", err)
	}
	if len(muts) != 0 {
		t.Errorf("the `!` on line 4 is outside the changed lines [8] and must not mutate, got %+v", muts)
	}
}

// Each `!` in a double negation is its own site, so both must be mutated.
func TestDoubleNegationYieldsOneMutantPerNot(t *testing.T) {
	src := []byte("package p\n\nfunc f(ok bool) bool {\n\treturn !!ok\n}\n")
	muts, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
	if err != nil {
		t.Fatalf("GenerateMutants: %v", err)
	}
	if len(muts) != 2 {
		t.Fatalf("`!!ok` has two negations, want 2 mutants, got %d: %+v", len(muts), muts)
	}
}
