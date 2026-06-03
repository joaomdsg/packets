package adult

import "testing"

// Weak: only checks a value far from the boundary, so it cannot tell
// `>=` from `>`.
func TestIsAdult(t *testing.T) {
	if !IsAdult(25) {
		t.Fatal("25 should be an adult")
	}
}
