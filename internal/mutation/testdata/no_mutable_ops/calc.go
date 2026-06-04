package nomut

// Shift uses the bit-shift operator `<<`, which the oracle does NOT mutate
// (bitwise/shift operators are an accepted residual blind spot), so this line
// has zero mutable sites. The oracle must report that as "no signal", not as
// "all mutants killed" (which would falsely imply the line is tested).
func Shift(x uint) uint {
	return x << 2
}
