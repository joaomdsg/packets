// Package slug derives fabric and packet slugs and resolves collisions.
package slug

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9-]+`)
var nonAlnumChars = regexp.MustCompile(`[^a-z0-9]+`)
var repeatedDashes = regexp.MustCompile(`-+`)

// Fabric derives a fabric slug from a git remote URL: last path segment,
// ".git" suffix stripped, lowercased, non [a-z0-9-] chars replaced with "-".
func Fabric(remote string) string {
	last := remote
	if idx := strings.LastIndexAny(remote, "/:"); idx != -1 {
		last = remote[idx+1:]
	}
	last = strings.TrimSuffix(last, ".git")
	last = strings.ToLower(last)
	last = nonSlugChars.ReplaceAllString(last, "-")
	return strings.Trim(last, "-")
}

const packetSlugMaxLen = 40

// Packet derives a packet slug from a goal: first 4 words, lowercased,
// non [a-z0-9] chars replaced with "-", repeated dashes collapsed, trimmed,
// capped at 40 chars.
func Packet(goal string) string {
	words := strings.Fields(goal)
	if len(words) > 4 {
		words = words[:4]
	}
	joined := strings.ToLower(strings.Join(words, " "))
	replaced := nonAlnumChars.ReplaceAllString(joined, "-")
	collapsed := repeatedDashes.ReplaceAllString(replaced, "-")
	trimmed := strings.Trim(collapsed, "-")
	if len(trimmed) > packetSlugMaxLen {
		trimmed = trimmed[:packetSlugMaxLen]
	}
	return trimmed
}

// Deconflict appends a date suffix (then a counter) to base until exists
// reports false, matching the -MMDD, -MMDD-2, ... collision scheme.
// It reports whether a collision occurred (i.e. base itself was taken).
func Deconflict(base string, now time.Time, exists func(string) bool) (string, bool) {
	if !exists(base) {
		return base, false
	}
	dated := fmt.Sprintf("%s-%s", base, now.Format("0102"))
	if !exists(dated) {
		return dated, true
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", dated, n)
		if !exists(candidate) {
			return candidate, true
		}
	}
}
