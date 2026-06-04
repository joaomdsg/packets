package manymutants

import "testing"

// Weak: every value is far from the boundary, so flipping any `>` to `>=`
// leaves the count unchanged — all twelve mutants survive.
func TestCount(t *testing.T) {
	if Count([]int{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}) != 12 {
		t.Fatal("want 12")
	}
}
