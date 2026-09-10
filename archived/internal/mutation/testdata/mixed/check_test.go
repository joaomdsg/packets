package mixed

import "testing"

// One input that pins the boundary for b, d, g (so those mutants are KILLED)
// but stays far from the boundary for a, c, e (so those mutants SURVIVE).
// b=10 kills `>=`->`>`; d=100 kills `<=`->`<`; g=20 kills `>=`->`>`.
// a=5,c=1,e=9 leave `>`,`<`,`>` unconstrained.
func TestCheck(t *testing.T) {
	if Check(5, 10, 1, 100, 9, 20) != 6 {
		t.Fatal("want 6")
	}
}
