package packet

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

// truncateTail's rune-safety cannot be reliably observed through the public
// RunBuildVetGate API (the compiler controls the exact byte offsets of a
// build/vet failure's output), so this drops to an internal test as the
// documented last resort (CONVENTIONS.md Test Scope).
func TestTruncateTail_neverSplitsAMultiByteRuneAtTheCutPoint(t *testing.T) {
	t.Parallel()

	// "日" is a 3-byte rune (0xE6 0x97 0xA5). A naive byte-slice at
	// len(s)-max==2 would start mid-rune, on its second byte.
	s := "x日y"
	got := truncateTail(s, 3)

	assert.True(t, utf8.ValidString(got), "a rune-splitting cut must never produce invalid UTF-8")
}

func TestTruncateTail_leavesShortInputUntouched(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "short", truncateTail("  short  ", 200))
}
