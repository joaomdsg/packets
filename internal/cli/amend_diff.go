package cli

import "strings"

// firstDiffLine returns a one-line unified-diff-style summary of the
// first place before and after disagree: the removed line if one exists
// there, otherwise the added line. Good enough for a log summary; a full
// diff algorithm is out of scope for what's just a one-line breadcrumb.
func firstDiffLine(before, after string) string {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	for i := 0; i < len(beforeLines) || i < len(afterLines); i++ {
		var b, a string
		if i < len(beforeLines) {
			b = beforeLines[i]
		}
		if i < len(afterLines) {
			a = afterLines[i]
		}
		if b == a {
			continue
		}
		if i >= len(beforeLines) {
			return "+" + a
		}
		return "-" + b
	}
	return ""
}
