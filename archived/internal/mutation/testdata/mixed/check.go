package mixed

func Check(a, b, c, d, e, g int) int {
	n := 0
	if a > 0 {
		n++
	}
	if b >= 10 {
		n++
	}
	if c < 5 {
		n++
	}
	if d <= 100 {
		n++
	}
	if e > 3 {
		n++
	}
	if g >= 20 {
		n++
	}
	return n
}
