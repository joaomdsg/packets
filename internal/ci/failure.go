package ci

import "strings"

const failureTailLines = 200

// tail returns the last n lines of s, or all of s if it has n or fewer.
// Blank lines are preserved; only the single empty element strings.Split
// produces after a trailing newline is dropped.
func tail(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
