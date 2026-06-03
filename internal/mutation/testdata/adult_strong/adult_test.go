package adult

import "testing"

// Strong: pins the exact boundary, so flipping `>=` to `>` (18 no longer
// adult) makes this fail — the mutant is killed.
func TestIsAdult(t *testing.T) {
	if IsAdult(17) {
		t.Fatal("17 must not be an adult")
	}
	if !IsAdult(18) {
		t.Fatal("18 must be an adult")
	}
}
