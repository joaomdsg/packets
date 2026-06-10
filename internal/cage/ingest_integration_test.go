package cage_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/cage"
	"github.com/joaomdsg/packets/internal/ingest"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/sandbox"
)

// producerTwoCommitBundle builds a real producer repo with two commits (base then
// fix) on refs/heads/main and returns the base+fix SHAs plus a `git bundle --all`.
// Offline, no network — the shape a cross-process producer would upload.
func producerTwoCommitBundle(t *testing.T) (base, fix string, bundle []byte) {
	t.Helper()
	repo := t.TempDir()
	pgit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
		return strings.TrimSpace(string(out))
	}
	pgit("init", "-q", "-b", "main")
	pgit("config", "user.email", "p@p")
	pgit("config", "user.name", "p")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "adult.go"),
		[]byte("package adult\nfunc Adult(age int) bool { return age >= 18 }\n"), 0o644))
	pgit("add", "-A")
	pgit("commit", "-qm", "base")
	base = pgit("rev-parse", "HEAD")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "adult.go"),
		[]byte("package adult\nfunc Adult(age int) bool { return age >= 21 }\n"), 0o644))
	pgit("add", "-A")
	pgit("commit", "-qm", "fix")
	fix = pgit("rev-parse", "HEAD")
	bundlePath := filepath.Join(t.TempDir(), "p.bundle")
	pgit("bundle", "create", bundlePath, "--all")
	b, err := os.ReadFile(bundlePath)
	require.NoError(t, err)
	return base, fix, b
}

// THE SLICE-A PAYOFF: a cross-process producer's commits, delivered ONLY as an
// ingested bundle (never present in the host repo otherwise), are verifiable by
// the cage. ingest confines the producer's refs to refs/producers/<id>/* in a
// host store; cage.Materialize then resolves the claim's SHAs against THAT store
// (by content-address — the objects are present even though no branch in the
// disposable clone points at them) and the oracle can check them out. This proves
// the bundle-over-channel transport (council R38) actually delivers verifiable
// claims, end to end minus the wire.
func TestIngestThenMaterialize_aProducersBundledCommitsAreVerifiableInTheCage(t *testing.T) {
	t.Parallel()
	base, fix, bundle := producerTwoCommitBundle(t)

	// The host store starts WITHOUT the producer's commits; ingest is the only way
	// they arrive.
	store := t.TempDir()
	runGit(t, store, "init", "-q")
	require.False(t, hasObject(t, store, base), "the host must not already hold the producer's commit")

	require.NoError(t, ingest.IngestProducerObjects(context.Background(), store, "alice", bundle, 1<<20))
	require.True(t, hasObject(t, store, base), "after ingest the host store holds the producer's base commit")
	require.True(t, hasObject(t, store, fix), "and its fix commit")

	target := ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "adult.go", Line: 2}

	// Materialize resolves the ingested revisions and produces a disposable workdir
	// whose clone actually CONTAINS them (checkout-able by SHA) — the precondition
	// for the cage oracle to run.
	wd, cleanup, err := cage.Materialize(context.Background(), store, target)
	require.NoError(t, err, "Materialize must resolve a claim against the producer's ingested objects")
	t.Cleanup(cleanup)
	for _, rev := range []string{base, fix} {
		out, err := exec.Command("git", "-C", wd.Repo, "worktree", "add", "--detach",
			filepath.Join(t.TempDir(), "wt-"+rev[:8]), rev).CombinedOutput()
		require.NoErrorf(t, err, "the cage clone must check out the ingested rev %s: %s", rev, out)
	}

	// And the FULL host verifier path composes: with the cage faked at the runner
	// seam (returning a catch transcript on the claim's anchor), the ingested claim
	// mints a confirmed catch.
	fake := &fakeRunner{result: sandbox.Result{Output: catchTranscriptJSON(t, "adult.go", 2)}}
	rec, err := cage.CageVerifier(fake, store, "img", 30*time.Second)(ledger.ClaimRecord{Target: target})
	require.NoError(t, err)
	require.NotNil(t, rec, "the ingested producer claim verifies into a confirmed catch")
	assert.Equal(t, base, rec.BeforeRev)
	assert.Equal(t, fix, rec.AfterRev)
}

// hasObject reports whether the store holds the commit object by SHA, regardless
// of which ref (if any) points at it.
func hasObject(t *testing.T, store, sha string) bool {
	t.Helper()
	return exec.Command("git", "-C", store, "rev-parse", "--verify", "--quiet", sha+"^{commit}").Run() == nil
}
