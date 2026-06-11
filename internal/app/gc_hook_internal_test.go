package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ingest"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
)

// When a producer's claim RESOLVES (here: mints), the post-verdict GC hook
// reclaims that producer's ingested objects immediately — no periodic sweep, no
// manual PruneIdleProducers call. This proves the production wiring
// (StartCageClaimConsumers sets Admission.OnResolved) actually fires on
// resolution. NOT parallel (shared liveReg/liveFabric).
func TestClaimResolution_reclaimsTheProducersObjectsViaThePostVerdictHook(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix, tip := inlineRepoWithTwoRevs(t)

	// Ingest a producer bundle so the session's namespace holds objects to reclaim.
	bundle, _ := producerCommitBundle(t)
	require.NoError(t, ingest.IngestProducerObjects(context.Background(), repo, defaultSessionKey, bundle, 1<<20))
	require.True(t, resolvesRef(t, repo, "refs/producers/"+defaultSessionKey+"/heads/main"),
		"precondition: the producer's objects are ingested before the claim resolves")

	output, err := json.Marshal(pipe.Transcript{
		Outcome: catch.Catch, Reason: pipe.ReasonNone, Path: "adult.go", Line: 2, Land: pipe.LandClean,
		Before: catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}},
		After:  catch.LineState{Inventory: []string{">="}, Survivors: nil},
	})
	require.NoError(t, err)

	_, log, err := NewServer(LiveConfig{
		RepoDir: repo, BaseRev: base, FixRev: fix, TipRev: tip, Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(t.TempDir(), "default.jsonl"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	var invoked atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	StartCageClaimConsumers(ctx, "img", blessingRunner{output: string(output), invoked: &invoked})

	publishClaim(t, defaultSessionKey, ledger.Target{BaseRev: base, FixRev: fix, TipRev: tip, Path: "adult.go", Line: 2})

	require.Eventually(t, func() bool {
		b, err := log.Balance()
		return err == nil && b == 1
	}, 5*time.Second, 25*time.Millisecond, "the claim must mint (resolve) first")

	// The post-verdict hook must have reclaimed the now-idle producer's objects —
	// no claim is in flight after the mint, so its namespace is dead weight.
	require.Eventually(t, func() bool {
		return !resolvesRef(t, repo, "refs/producers/"+defaultSessionKey+"/heads/main")
	}, 5*time.Second, 25*time.Millisecond,
		"a resolved claim must trigger the post-verdict prune of its producer's objects")
}
