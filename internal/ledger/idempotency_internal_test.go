package ledger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// targetAlreadyMinted decides whether a claim's verify can be skipped, so it must
// match EXACTLY the catch identity (BeforeRev, AfterRev, Path, Line) — never on
// TipRev (not part of the identity), and never loosely (a genuinely new target
// must still be verified, or real catches would be silently dropped).
func TestTargetAlreadyMinted_matchesTheCatchIdentityExactly(t *testing.T) {
	t.Parallel()
	minted := []CatchRecord{{BeforeRev: "b", AfterRev: "f", Path: "a.go", Line: 4}}

	assert.True(t, targetAlreadyMinted(minted, Target{BaseRev: "b", FixRev: "f", TipRev: "differs", Path: "a.go", Line: 4}),
		"the same Before/After/Path/Line is already minted even when TipRev differs (tip is not part of the identity)")

	assert.False(t, targetAlreadyMinted(minted, Target{BaseRev: "b", FixRev: "f", Path: "a.go", Line: 9}),
		"a different line is a different target — must still be verified")
	assert.False(t, targetAlreadyMinted(minted, Target{BaseRev: "b", FixRev: "other", Path: "a.go", Line: 4}),
		"a different fix rev is a different target — must still be verified")
	assert.False(t, targetAlreadyMinted(minted, Target{BaseRev: "other", FixRev: "f", Path: "a.go", Line: 4}),
		"a different base rev is a different target — must still be verified")
	assert.False(t, targetAlreadyMinted(minted, Target{BaseRev: "b", FixRev: "f", Path: "other.go", Line: 4}),
		"a different path is a different target — must still be verified")
	assert.False(t, targetAlreadyMinted(nil, Target{BaseRev: "b", FixRev: "f", Path: "a.go", Line: 4}),
		"an empty economy has minted nothing")

	// A match anywhere in the economy counts, not just the first record.
	many := []CatchRecord{
		{BeforeRev: "x", AfterRev: "y", Path: "z.go", Line: 1},
		{BeforeRev: "b", AfterRev: "f", Path: "a.go", Line: 4},
	}
	assert.True(t, targetAlreadyMinted(many, Target{BaseRev: "b", FixRev: "f", Path: "a.go", Line: 4}),
		"an already-minted target is found even when it is not the first record")
}
