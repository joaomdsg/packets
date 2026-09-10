package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pushSquashFixture wires a work repo whose "origin" is a LOCAL bare repo (file path, no
// network), returning the work dir, the bare dir, and base SHA. squashed commits are
// built off base via commit-tree, mirroring the land flow.
func pushSquashFixture(t *testing.T) (work, bare, base string) {
	t.Helper()
	work = initGitRepoForPacket(t)
	bare = t.TempDir()
	gitPacket(t, bare, "init", "--bare", "-q")
	gitPacket(t, work, "remote", "add", "origin", bare)
	base = gitPacket(t, work, "rev-parse", "HEAD")
	return work, bare, base
}

// squashOf builds a fresh one-commit squash off base. Each call writes a DISTINCT file,
// so the tree (and thus the squash SHA) genuinely differs — commit-tree is deterministic
// in tree+parent+message, so distinct SHAs come from the tree change, not from timing.
func squashOf(t *testing.T, work, base, file, content string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(work, file), []byte(content), 0o644))
	gitPacket(t, work, "add", "-A")
	gitPacket(t, work, "commit", "-qm", "rev")
	return gitPacket(t, work, "commit-tree", "HEAD^{tree}", "-p", base, "-m", "squash")
}

// The land push must land a fresh PR branch on the remote — the first push of a new
// session's branch has to succeed and create the branch at exactly the squashed SHA.
func TestPushSquash_firstPushCreatesTheBranchOnTheRemote(t *testing.T) {
	work, bare, base := pushSquashFixture(t)
	sha := squashOf(t, work, base, "a.txt", "a\n")

	require.NoError(t, pushSquash(context.Background(), work, sha, "packets/fresh", ""),
		"the first push of a fresh branch must succeed")
	assert.Equal(t, sha, gitPacket(t, bare, "rev-parse", "refs/heads/packets/fresh"),
		"the remote branch must now point at the pushed squash")
}

// Re-landing after adjustments re-pushes the SAME branch: a push whose lease names the
// SHA we last pushed must SUCCEED and advance the branch — the lease permits our own
// legitimate update.
func TestPushSquash_relandWithCurrentLeaseAdvancesTheBranch(t *testing.T) {
	work, bare, base := pushSquashFixture(t)
	sha1 := squashOf(t, work, base, "a.txt", "a\n")
	require.NoError(t, pushSquash(context.Background(), work, sha1, "packets/reland", ""))

	sha2 := squashOf(t, work, base, "b.txt", "b\n")
	require.NoError(t, pushSquash(context.Background(), work, sha2, "packets/reland", sha1),
		"a re-land leasing against the last-pushed sha must succeed")
	assert.Equal(t, sha2, gitPacket(t, bare, "rev-parse", "refs/heads/packets/reland"),
		"the remote branch must advance to the new squash")
}

// The whole point of the lease: a push whose expectation is STALE (the remote branch has
// since moved past it) must be REFUSED and leave the remote untouched — never a silent
// clobber of work the expectation no longer matches.
func TestPushSquash_staleLeaseIsRefusedAndLeavesTheRemoteUntouched(t *testing.T) {
	work, bare, base := pushSquashFixture(t)
	sha1 := squashOf(t, work, base, "a.txt", "a\n")
	require.NoError(t, pushSquash(context.Background(), work, sha1, "packets/stale", ""))
	sha2 := squashOf(t, work, base, "b.txt", "b\n")
	require.NoError(t, pushSquash(context.Background(), work, sha2, "packets/stale", sha1)) // branch now at sha2

	sha3 := squashOf(t, work, base, "c.txt", "c\n")
	// Lease names sha1, but the remote is at sha2 — stale, must be refused.
	assert.Error(t, pushSquash(context.Background(), work, sha3, "packets/stale", sha1),
		"a stale lease must be refused, not force through")
	assert.Equal(t, sha2, gitPacket(t, bare, "rev-parse", "refs/heads/packets/stale"),
		"the remote branch must be unchanged after a refused push")
}

// First-push safety: an empty expectation means "must not exist", so pushing with
// expected=="" over a branch that ALREADY exists on the remote must be REFUSED — first
// push never silently clobbers a pre-existing branch. (This test verifies git's
// empty-expected semantics rather than assuming them.)
func TestPushSquash_emptyLeaseRefusesToClobberAnExistingBranch(t *testing.T) {
	work, bare, base := pushSquashFixture(t)
	sha1 := squashOf(t, work, base, "a.txt", "a\n")
	require.NoError(t, pushSquash(context.Background(), work, sha1, "packets/exists", ""))

	sha2 := squashOf(t, work, base, "b.txt", "b\n")
	assert.Error(t, pushSquash(context.Background(), work, sha2, "packets/exists", ""),
		"empty expected = must-not-exist: a first-push lease must refuse an existing branch")
	assert.Equal(t, sha1, gitPacket(t, bare, "rev-parse", "refs/heads/packets/exists"),
		"the existing branch must be unchanged")
}
