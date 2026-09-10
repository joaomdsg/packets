package app

import "testing"

// On a re-land the leased push updates the already-open PR, but `gh pr create` then fails
// with an "already exists" error. isPRAlreadyExists distinguishes that benign re-land
// signal from a real create failure (auth, network) — so the Lead sees the updated PR's
// URL, not a spurious "PR failed". It must match both gh 2.92 message shapes yet not
// over-match an unrelated error.
func TestIsPRAlreadyExists_recognizesTheReLandSignalWithoutOverMatching(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want bool
	}{
		{"the branch-form create error is a re-land",
			`a pull request for branch "packets/sess" into branch "main" already exists:` + "\nhttps://github.com/o/r/pull/7", true},
		{"the GraphQL-form create error is a re-land",
			"GraphQL: A pull request already exists for OWNER:packets/sess (createPullRequest)", true},
		{"mixed-case still recognizes the re-land signal",
			"A Pull Request Already Exists for the branch", true},
		{"a generic create failure is NOT a re-land",
			"failed to create pull request: HTTP 403: Resource not accessible", false},
		{"an empty output is not a re-land", "", false},
		{"a pull-request message without 'already exists' is not a re-land",
			"failed to create pull request: validation failed", false},
		{"an 'already exists' for something unrelated is not a re-land",
			"error: a label with this name already exists", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPRAlreadyExists(tc.out); got != tc.want {
				t.Errorf("isPRAlreadyExists(%q) = %v, want %v", tc.out, got, tc.want)
			}
		})
	}
}
