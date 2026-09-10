package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// squashToCommit must collapse the whole base→HEAD range into ONE commit so the opened
// PR reads as a single clean change, not every intermediate session revision. The
// squashed commit must carry HEAD's full tree (the complete diff from base) yet sit
// directly on base as its sole parent. NOT parallel (shells out to git only).
func TestSquashToCommit_collapsesTheRangeIntoOneCommitOnBase(t *testing.T) {
	repo := initGitRepoForPacket(t)
	base := gitPacket(t, repo, "rev-parse", "HEAD")

	// Two more session revisions on top of base — the kind of multi-commit history a
	// PR should land as a single squashed commit.
	require.NoError(t, os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644))
	gitPacket(t, repo, "add", "-A")
	gitPacket(t, repo, "commit", "-qm", "rev one")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "b.txt"), []byte("b\n"), 0o644))
	gitPacket(t, repo, "add", "-A")
	gitPacket(t, repo, "commit", "-qm", "rev two")
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	sha, err := squashToCommit(context.Background(), repo, base, "Landed task\n\nbody here")
	require.NoError(t, err)
	require.NotEmpty(t, sha)

	// Exactly one commit ahead of base — the two session revisions collapsed to one.
	count := gitPacket(t, repo, "rev-list", "--count", base+".."+sha)
	assert.Equal(t, "1", count, "the squashed commit must be exactly one commit ahead of base")

	// Base is its sole parent.
	assert.Equal(t, base, gitPacket(t, repo, "rev-parse", sha+"^"), "base must be the squash's only parent")

	// Its tree equals HEAD's tree — the squash carries the full base→HEAD diff.
	assert.Equal(t, gitPacket(t, repo, "rev-parse", head+"^{tree}"),
		gitPacket(t, repo, "rev-parse", sha+"^{tree}"),
		"the squashed commit's tree must match HEAD (the full base→HEAD diff)")

	// Local refs and working tree are untouched — squashing builds a detached commit
	// object, it does not move HEAD.
	assert.Equal(t, head, gitPacket(t, repo, "rev-parse", "HEAD"), "squashing must not move HEAD")
}

// The squash commit's message must be the message passed in, so the PR's single commit
// carries the derived title/body rather than an intermediate revision's message.
func TestSquashToCommit_usesTheGivenMessage(t *testing.T) {
	repo := initGitRepoForPacket(t)
	base := gitPacket(t, repo, "rev-parse", "HEAD")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "c.txt"), []byte("c\n"), 0o644))
	gitPacket(t, repo, "add", "-A")
	gitPacket(t, repo, "commit", "-qm", "intermediate")

	sha, err := squashToCommit(context.Background(), repo, base, "My PR subject")
	require.NoError(t, err)
	assert.Equal(t, "My PR subject", gitPacket(t, repo, "log", "-1", "--format=%s", sha),
		"the squash must carry the supplied message, not an intermediate one")
}

// When HEAD already sits at base (no session revisions to collapse) squashing must
// still produce a valid commit one ahead of base carrying base's tree — a benign no-op
// change, never an error. The land guard refuses empty work upstream; the plumbing
// itself stays total.
func TestSquashToCommit_handlesAnEmptyRangeAsABenignOneAheadCommit(t *testing.T) {
	repo := initGitRepoForPacket(t)
	base := gitPacket(t, repo, "rev-parse", "HEAD")

	sha, err := squashToCommit(context.Background(), repo, base, "no-op land")
	require.NoError(t, err)
	assert.Equal(t, "1", gitPacket(t, repo, "rev-list", "--count", base+".."+sha),
		"even an empty range yields one commit ahead of base")
	assert.Equal(t, gitPacket(t, repo, "rev-parse", base+"^{tree}"),
		gitPacket(t, repo, "rev-parse", sha+"^{tree}"),
		"with no revisions the squash carries base's own tree")
}

// A bad base ref must surface an error rather than silently producing a bogus commit —
// the land flow needs to fail calmly, not push garbage.
func TestSquashToCommit_failsOnUnknownBase(t *testing.T) {
	repo := initGitRepoForPacket(t)
	_, err := squashToCommit(context.Background(), repo, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "msg")
	assert.Error(t, err, "an unknown base ref must be a calm error, not a bogus commit")
}
