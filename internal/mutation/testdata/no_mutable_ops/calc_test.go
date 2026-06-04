package nomut

import "testing"

// Deliberately weak: asserts nothing meaningful. If the oracle had any
// mutable site here it would survive — but there are none to begin with.
func TestShift(t *testing.T) {
	_ = Shift(8)
}
