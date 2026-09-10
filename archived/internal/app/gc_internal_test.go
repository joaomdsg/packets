package app

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ingest"
	"github.com/joaomdsg/packets/internal/ledger"
)

// The peer-GC sweep must reclaim an IDLE peer's ingested objects while
// NEVER touching a peer that still has a claim in flight — pruning a pending
// claim's objects would orphan the very revisions the cage needs to verify it.
// One pass over the registry enforces that economy-safe rule per session. NOT
// parallel (shared liveReg/liveFabric).
func TestPruneIdlePeers_keepsASessionWithClaimsInFlightButReclaimsAnIdleOne(t *testing.T) {
	ctx := context.Background()

	// Session "idle": a fresh repo with an ingested bundle, no claims.
	idleRepo := freshGitRepo(t)
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: idleRepo, BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(t.TempDir(), "default.jsonl"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })
	idleBundle, _ := peerCommitBundle(t)
	require.NoError(t, ingest.IngestPeerObjects(ctx, idleRepo, defaultSessionKey, idleBundle, 1<<20))
	require.True(t, resolvesRef(t, idleRepo, "refs/producers/"+defaultSessionKey+"/heads/main"))

	// Session "busy": a fresh repo with an ingested bundle AND a published (still
	// in-flight, unverified) claim.
	busyRepo := freshGitRepo(t)
	busyLog, err := AddSession("busy", LiveConfig{
		RepoDir: busyRepo, BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = busyLog.Close() })
	busyBundle, _ := peerCommitBundle(t)
	require.NoError(t, ingest.IngestPeerObjects(ctx, busyRepo, "busy", busyBundle, 1<<20))
	require.True(t, resolvesRef(t, busyRepo, "refs/producers/busy/heads/main"))
	_, err = ledger.PublishClaim(ctx, liveFabric, "busy", LedgerInstance,
		ledger.ClaimRecord{Target: ledger.Target{BaseRev: "x", FixRev: "y", TipRev: "y", Path: "a.go", Line: 1}})
	require.NoError(t, err)
	inflight, err := busyLog.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 1, inflight, "the busy session has a claim in flight")

	PruneIdlePeers(ctx)

	require.False(t, resolvesRef(t, idleRepo, "refs/producers/"+defaultSessionKey+"/heads/main"),
		"an idle peer's ingested objects are reclaimed")
	require.True(t, resolvesRef(t, busyRepo, "refs/producers/busy/heads/main"),
		"a peer with a claim in flight is NEVER pruned — its objects back a pending verify")
}

// A session configured with no RepoDir has no store to prune: the sweep skips it
// without error or panic (it must never fall back to the process cwd).
func TestPruneIdlePeers_skipsASessionWithNoRepoDir(t *testing.T) {
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: "", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(t.TempDir(), "default.jsonl"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	require.NotPanics(t, func() { PruneIdlePeers(context.Background()) },
		"a session with no RepoDir is skipped cleanly, never pruning the process cwd")
}

// resolvesRef reports whether a ref resolves to a commit in the store.
func resolvesRef(t *testing.T, store, ref string) bool {
	t.Helper()
	return exec.Command("git", "-C", store, "rev-parse", "--verify", "--quiet", ref+"^{commit}").Run() == nil
}
