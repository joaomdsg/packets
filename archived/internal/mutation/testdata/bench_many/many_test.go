package many

import "testing"

// Loose on purpose: lets most mutants survive so the benchmark exercises
// a full test run per mutant.
func TestCount(t *testing.T) {
	if Count([]int{5, 10}) < 0 {
		t.Fatal("count is never negative")
	}
}
