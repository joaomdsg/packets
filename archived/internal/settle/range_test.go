package settle_test

import (
	"context"
	"testing"

	"github.com/joaomdsg/packets/internal/settle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The land flow squashes base→HEAD into one detached commit built blindly from HEAD's
// tree and pushes it — so the bytes leaving the machine are scanned by NOTHING unless
// this gate runs. A secret living in the pushed rev's tree (regardless of how it got
// there) must be caught BEFORE the push, or it leaks to the remote.
func TestScanCommitRange_catchesASecretInThePushedRev(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	writeCommit(t, dir, "leak.txt", "aws_key = AKIAIOSFODNN7EXAMPLE\n", "work that smuggled a key")
	// The exact commit the land flow would push: one squashed commit on base carrying
	// HEAD's whole tree.
	rev := runGit(t, dir, "commit-tree", "HEAD^{tree}", "-p", base, "-m", "squash")

	hits, err := settle.ScanCommitRange(context.Background(), dir, base, rev)
	require.NoError(t, err)
	assert.True(t, hasHit(hits, "leak.txt", "aws-access-key-id"),
		"a secret in the pushed rev must be caught before the push; got %v", hits)
}

// A squashed rev with no secrets must scan clean — the gate must not refuse honest work.
func TestScanCommitRange_isCleanWhenThePushedRevHasNoSecrets(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")
	writeCommit(t, dir, "feature.go", "package p\n\nfunc Add(a, b int) int { return a + b }\n", "honest work")
	rev := runGit(t, dir, "commit-tree", "HEAD^{tree}", "-p", base, "-m", "squash")

	hits, err := settle.ScanCommitRange(context.Background(), dir, base, rev)
	require.NoError(t, err)
	assert.Empty(t, hits, "a clean pushed rev must not be refused")
}

// The gate must scan only what the pushed range ADDS, not the rev's whole tree: a
// secret that already lived in base (e.g. a trunk test fixture) is not introduced by
// this work, so it must NOT block the push — otherwise every land against a repo with a
// pre-existing secret would be refused. This pins base..rev added-line semantics and
// defeats a "scan the whole tree" implementation.
func TestScanCommitRange_ignoresAPreexistingSecretAlreadyInBase(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	// The secret is already present at base — not added by the range under review.
	writeCommit(t, dir, "fixture.txt", "aws_key = AKIAIOSFODNN7EXAMPLE\n", "pre-existing fixture")
	base := runGit(t, dir, "rev-parse", "HEAD")

	writeCommit(t, dir, "feature.go", "package p\n", "new work on top")
	rev := runGit(t, dir, "commit-tree", "HEAD^{tree}", "-p", base, "-m", "squash")

	hits, err := settle.ScanCommitRange(context.Background(), dir, base, rev)
	require.NoError(t, err)
	assert.Empty(t, hits, "a secret already in base is not added by the range and must not block; got %v", hits)
}

// The gate must scan WHAT IS PUSHED, not discarded history: a secret that entered and
// was removed in abandoned intermediate commits is absent from the squashed rev's tree,
// so it must NOT block the push — even though ScanHistory (the --all-history scan) still
// sees it. This is the deliberate scope difference between the two scans.
func TestScanCommitRange_ignoresSecretsInAbandonedHistoryNotInThePushedRev(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	writeCommit(t, dir, "leak.txt", "aws_key = AKIAIOSFODNN7EXAMPLE\n", "oops, a key")
	runGit(t, dir, "rm", "-q", "leak.txt")
	runGit(t, dir, "commit", "-qm", "scrub the key")
	writeCommit(t, dir, "feature.go", "package p\n", "real work")
	// The squashed rev carries only HEAD's (clean) tree — the leak is gone from it.
	rev := runGit(t, dir, "commit-tree", "HEAD^{tree}", "-p", base, "-m", "squash")

	hits, err := settle.ScanCommitRange(context.Background(), dir, base, rev)
	require.NoError(t, err)
	assert.Empty(t, hits, "a secret only in abandoned history must not block the clean pushed rev; got %v", hits)

	// Contrast: the secret IS still reachable in --all history, so ScanHistory flags it —
	// proving the empty ScanCommitRange result is real scoping, not a scan that sees nothing.
	all, err := settle.ScanHistory(context.Background(), dir)
	require.NoError(t, err)
	assert.True(t, hasHit(all, "leak.txt", "aws-access-key-id"),
		"the secret must still live in --all history (else the scope contrast proves nothing)")
}
