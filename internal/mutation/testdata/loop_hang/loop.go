package loophang

// Grow counts x up to limit. The `+` on the accumulator line is the trap:
// mutating it to `-` makes x decrease forever while `x < limit` stays true,
// so that mutant never terminates — the oracle must not silently score it.
func Grow(limit int) int {
	x := 0
	for x < limit {
		x = x + 1
	}
	return x
}
