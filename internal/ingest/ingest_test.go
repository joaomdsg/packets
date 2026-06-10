package ingest_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ingest"
)

// runGit runs git in dir and fails the test on error (offline, no network).
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return string(out)
}

// hostStore is a fresh, empty host object store (a real git repo) the producer's
// objects are ingested INTO.
func hostStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	return dir
}

// producerBundle builds a real one-commit producer repo with its commit on
// refs/heads/main and returns a `git bundle --all` of it plus the commit SHA.
func producerBundle(t *testing.T) (bundle []byte, sha string) {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-q", "-b", "main")
	runGit(t, repo, "config", "user.email", "p@p")
	runGit(t, repo, "config", "user.name", "p")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "work.go"), []byte("package work\n"), 0o644))
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-qm", "producer work")
	sha = strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	bundlePath := filepath.Join(t.TempDir(), "p.bundle")
	runGit(t, repo, "bundle", "create", bundlePath, "--all")
	b, err := os.ReadFile(bundlePath)
	require.NoError(t, err)
	return b, sha
}

// resolves reports whether a ref/rev resolves in the store.
func resolves(t *testing.T, store, rev string) bool {
	t.Helper()
	return exec.Command("git", "-C", store, "rev-parse", "--verify", "--quiet", rev+"^{commit}").Run() == nil
}

// allRefs lists every ref in the store — a fresh `git init` store has none, so an
// empty result after a rejected ingest proves NOTHING was written anywhere (not
// just that the target ref is absent, which an empty store gives vacuously).
func allRefs(t *testing.T, store string) string {
	t.Helper()
	return strings.TrimSpace(runGit(t, store, "for-each-ref", "--format=%(refname)"))
}

// A producer must NOT be able to smuggle a ref into the host's own namespace:
// every ingested ref is forced under refs/producers/<id>/*, so a bundle whose
// commit sits on refs/heads/main lands at refs/producers/alice/heads/main and
// NEVER at the host's refs/heads/main. This namespacing is what stops one
// producer (or the host) having its refs moved by another's upload ("move the judge").
func TestIngestProducerObjects_rejectsObjectsOutsideTheProducerNamespace(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	bundle, sha := producerBundle(t)

	err := ingest.IngestProducerObjects(context.Background(), store, "alice", bundle, 1<<20)
	require.NoError(t, err)

	assert.True(t, resolves(t, store, "refs/producers/alice/heads/main"),
		"the producer's commit must land inside its own namespace")
	assert.Equal(t, sha, strings.TrimSpace(runGit(t, store, "rev-parse", "refs/producers/alice/heads/main")),
		"the namespaced ref points at the producer's actual commit")
	assert.False(t, resolves(t, store, "refs/heads/main"),
		"the producer must NOT be able to write the host's own refs/heads/main")
}

// A well-formed bundle ingests cleanly and its commit becomes resolvable under
// the producer namespace — the host now holds the objects a later claim's SHAs
// reference.
func TestIngestProducerObjects_acceptsAValidBundleIntoTheNamespace(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	bundle, sha := producerBundle(t)

	require.NoError(t, ingest.IngestProducerObjects(context.Background(), store, "bob", bundle, 1<<20))
	assert.True(t, resolves(t, store, sha), "the producer's commit SHA is now held by the host store")
}

// Valid producer ids span the safe ref-segment alphabet (letters incl.
// uppercase, digits, dot, dash, underscore) — these are ACCEPTED, proving the
// id check rejects only genuinely-unsafe segments rather than everything.
func TestIngestProducerObjects_acceptsSafeProducerIDsAcrossTheAlphabet(t *testing.T) {
	t.Parallel()
	for _, good := range []string{"alice", "Bob", "team_42", "svc.prod", "my-app", "A1.b-c_d"} {
		store := hostStore(t)
		bundle, _ := producerBundle(t)
		require.NoErrorf(t, ingest.IngestProducerObjects(context.Background(), store, good, bundle, 1<<20),
			"producer id %q is a safe ref segment and must be accepted", good)
		assert.Truef(t, resolves(t, store, "refs/producers/"+good+"/heads/main"),
			"%q's commit lands in its own namespace", good)
	}
}

// A bundle past the byte cap is rejected BEFORE any git work — a pack/bundle bomb
// can't make the host spend unbundling effort, and nothing lands in the store.
func TestIngestProducerObjects_rejectsAnOversizedBundle(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	bundle, _ := producerBundle(t)

	err := ingest.IngestProducerObjects(context.Background(), store, "alice", bundle, 10)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ingest.ErrCapExceeded), "an oversized bundle is refused by the cap")
	assert.Empty(t, allRefs(t, store), "the cap rejected before anything was written to the store")
}

// A malformed (non-bundle) payload under the cap is refused as invalid, and
// nothing is written — garbage can't poison the store.
func TestIngestProducerObjects_rejectsAMalformedBundle(t *testing.T) {
	t.Parallel()
	store := hostStore(t)

	err := ingest.IngestProducerObjects(context.Background(), store, "alice", []byte("not a git bundle"), 1<<20)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ingest.ErrBundleInvalid), "a non-bundle payload is invalid")
	assert.Empty(t, allRefs(t, store), "an invalid bundle writes nothing to the store")
}

// An unsafe producer id (path traversal / extra ref segments) is refused before
// any git runs — it could otherwise escape the per-producer namespace via the
// refspec and let a producer write outside its own subtree.
func TestIngestProducerObjects_rejectsAnUnsafeProducerID(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	bundle, _ := producerBundle(t)

	for _, bad := range []string{"../evil", "a/b", "..", ".", "", "has space", "star*", "a..b", "x.."} {
		err := ingest.IngestProducerObjects(context.Background(), store, bad, bundle, 1<<20)
		require.Errorf(t, err, "producer id %q must be refused", bad)
		assert.Truef(t, errors.Is(err, ingest.ErrBadProducerID), "%q is an unsafe ref segment", bad)
	}
	// Nothing was written anywhere in the store.
	assert.Empty(t, allRefs(t, store), "an unsafe id writes no refs at all")
}

// Pruning must NEVER run while a claim is in flight: the producer's ingested
// objects are exactly what the cage needs to verify that pending claim. So with
// an in-flight claim, PruneProducerObjects keeps everything.
func TestPruneProducerObjects_keepsObjectsWhileAClaimIsInFlight(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	bundle, _ := producerBundle(t)
	require.NoError(t, ingest.IngestProducerObjects(context.Background(), store, "alice", bundle, 1<<20))

	deleted, err := ingest.PruneProducerObjects(context.Background(), store, "alice", true)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted, "nothing is pruned while a claim is in flight")
	assert.True(t, resolves(t, store, "refs/producers/alice/heads/main"),
		"a pending claim's objects must survive — pruning them would orphan the verify")
}

// Once a producer has NO claims in flight (all resolved, or it only uploaded and
// never claimed), its ingested namespace is dead weight and is reclaimed: the
// refs are deleted so the objects become unreachable and collectable.
func TestPruneProducerObjects_reclaimsAnIdleProducersNamespaceWhenNothingInFlight(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	bundle, _ := producerBundle(t)
	require.NoError(t, ingest.IngestProducerObjects(context.Background(), store, "alice", bundle, 1<<20))
	require.True(t, resolves(t, store, "refs/producers/alice/heads/main"))

	deleted, err := ingest.PruneProducerObjects(context.Background(), store, "alice", false)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, 1, "an idle producer's refs are reclaimed")
	assert.False(t, resolves(t, store, "refs/producers/alice/heads/main"),
		"the reclaimed ref no longer keeps the producer's objects reachable")
}

// An unsafe producer id is refused before any ref-delete runs, so a traversal /
// extra-segment id can never drive deletions outside its own namespace (the
// host's own refs are untouched).
func TestPruneProducerObjects_refusesAnUnsafeProducerID(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	// Give the store a real host ref the prune must never touch.
	bundle, sha := producerBundle(t)
	runGit(t, store, "fetch", "--no-tags", "--end-of-options", bundleFile(t, bundle), "refs/heads/*:refs/heads/*")
	require.Equal(t, sha, strings.TrimSpace(runGit(t, store, "rev-parse", "refs/heads/main")))

	deleted, err := ingest.PruneProducerObjects(context.Background(), store, "../evil", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ingest.ErrBadProducerID), "an unsafe id is refused")
	assert.Equal(t, 0, deleted)
	assert.True(t, resolves(t, store, "refs/heads/main"), "the host's own refs are untouched by a refused prune")
}

// Pruning a producer that never ingested anything is a harmless no-op.
func TestPruneProducerObjects_isANoOpForAProducerWithNoIngestedObjects(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	deleted, err := ingest.PruneProducerObjects(context.Background(), store, "bob", false)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted, "nothing to prune for a producer with no ingested objects")
}

// A sibling producer whose id shares a prefix (alice vs alicelong) lives under a
// distinct namespace; the trailing slash on the for-each-ref pattern must keep
// pruning "alice" from spilling into "alicelong" — both valid safe ids.
func TestPruneProducerObjects_doesNotMatchAPrefixSiblingNamespace(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	a, _ := producerBundle(t)
	b, _ := producerBundle(t)
	require.NoError(t, ingest.IngestProducerObjects(context.Background(), store, "alice", a, 1<<20))
	require.NoError(t, ingest.IngestProducerObjects(context.Background(), store, "alicelong", b, 1<<20))
	require.True(t, resolves(t, store, "refs/producers/alice/heads/main"))
	require.True(t, resolves(t, store, "refs/producers/alicelong/heads/main"))

	deleted, err := ingest.PruneProducerObjects(context.Background(), store, "alice", false)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, 1)
	assert.False(t, resolves(t, store, "refs/producers/alice/heads/main"), "alice's namespace is reclaimed")
	assert.True(t, resolves(t, store, "refs/producers/alicelong/heads/main"),
		"the prefix-sibling alicelong is untouched — the trailing slash confines the glob")
}

// bundleFile writes a bundle to a temp file and returns its path.
func bundleFile(t *testing.T, bundle []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "b.bundle")
	require.NoError(t, os.WriteFile(p, bundle, 0o644))
	return p
}

// Pruning one producer must touch ONLY its own namespace — never another
// producer's refs (a prefix-glob bug that deleted refs/producers/* or matched
// refs/producers/alicebob would corrupt a bystander's pending verifies).
func TestPruneProducerObjects_leavesOtherProducersUntouched(t *testing.T) {
	t.Parallel()
	store := hostStore(t)
	a, _ := producerBundle(t)
	b, _ := producerBundle(t)
	require.NoError(t, ingest.IngestProducerObjects(context.Background(), store, "alice", a, 1<<20))
	require.NoError(t, ingest.IngestProducerObjects(context.Background(), store, "bob", b, 1<<20))
	require.True(t, resolves(t, store, "refs/producers/alice/heads/main"))
	require.True(t, resolves(t, store, "refs/producers/bob/heads/main"))

	deleted, err := ingest.PruneProducerObjects(context.Background(), store, "alice", false)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, 1)
	assert.False(t, resolves(t, store, "refs/producers/alice/heads/main"), "alice's namespace is reclaimed")
	assert.True(t, resolves(t, store, "refs/producers/bob/heads/main"), "bob's namespace is untouched — only alice was pruned")
}
