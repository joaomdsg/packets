package app

import (
	"slices"
	"testing"
)

// pushRefspec builds the land push's git args so the force is LEASED to an explicit
// expectation, never the bare --force-with-lease that degrades to an unguarded clobber on
// a branch with no remote-tracking ref. First push leases against "must not exist" (empty
// expected); a re-land leases against the SHA we last pushed — so a legitimate re-push is
// allowed but one over a branch someone else moved is refused.
func TestPushRefspec_leasesTheForceToAnExplicitExpectation(t *testing.T) {
	const branch, sha, old = "packets/sess-1", "newsha111", "oldsha000"

	t.Run("first push leases against must-not-exist, never a bare force", func(t *testing.T) {
		args := pushRefspec(branch, sha, "")
		assertContains(t, args, "--force-with-lease=refs/heads/packets/sess-1:")
		assertContains(t, args, "origin")
		assertContains(t, args, "newsha111:refs/heads/packets/sess-1")
		if slices.Contains(args, "--force-with-lease") {
			t.Errorf("the bare unqualified --force-with-lease must never appear (it is the vacuous lease): %v", args)
		}
	})

	t.Run("a re-land leases against the previously-pushed sha", func(t *testing.T) {
		args := pushRefspec(branch, sha, old)
		assertContains(t, args, "--force-with-lease=refs/heads/packets/sess-1:oldsha000")
		assertContains(t, args, "newsha111:refs/heads/packets/sess-1")
		if slices.Contains(args, "--force-with-lease") {
			t.Errorf("the bare unqualified --force-with-lease must never appear: %v", args)
		}
	})
}

func assertContains(t *testing.T, args []string, want string) {
	t.Helper()
	if !slices.Contains(args, want) {
		t.Errorf("push args %v\n  missing expected arg %q", args, want)
	}
}
