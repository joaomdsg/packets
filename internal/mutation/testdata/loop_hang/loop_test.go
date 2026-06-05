package loophang

import "testing"

func TestGrow(t *testing.T) {
	if Grow(5) != 5 {
		t.Fatal("want 5")
	}
}
