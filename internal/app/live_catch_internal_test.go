package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/orchestrator"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

func mintedCatchWithProducer(t *testing.T, f *fabric.Fabric, session, producer string) bool {
	t.Helper()
	p, err := ledger.ReplayProjection(context.Background(), f, session, "i")
	require.NoError(t, err)
	for _, r := range p.Records() {
		if r.Producer == producer && r.Outcome == catch.Catch {
			return true
		}
	}
	return false
}

// A live order's catch must be checked against the order's PRE-SPECIFIED anchor
// (Target.Path/Line) and the agent-PRODUCED revision — never an anchor derived
// from the agent's own diff (that would let the agent name its own denominator
// and farm confirmed-catches, V§13.5). When the produced revision yields a
// catch, the live order mints it exactly like a pre-funded order.
func TestRunLiveOrder_mintsACatchFromTheProducedRevisionAgainstThePreSpecifiedAnchor(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	base := gitOrder(t, repo, "rev-parse", "HEAD")

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	var liveHead string
	runHarness = func(_ context.Context, repoDir, _ string) ([]harness.Turn, error) {
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "feature.go"), []byte("package main\n"), 0o644))
		gitOrder(t, repoDir, "add", "-A")
		gitOrder(t, repoDir, "commit", "-qm", "live fix")
		liveHead = gitOrder(t, repoDir, "rev-parse", "HEAD")
		return []harness.Turn{{Outcome: orchestrator.TurnOutcome{Minted: true, SHA: liveHead}}}, nil
	}

	restoreCycle := resolveCycle
	t.Cleanup(func() { resolveCycle = restoreCycle })
	var gotBase, gotFix string
	var gotAnchor reanchor.Anchor
	resolveCycle = func(_ context.Context, _, base, fix, _ string, anchor reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		gotBase, gotFix, gotAnchor = base, fix, anchor
		return Resolution{Verdict: "caught", Record: &ledger.CatchRecord{Outcome: catch.Catch, Path: "feature.go", Line: 1, ReasonTag: "catch"}}, nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "live", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "seed.go", Line: 1, ReasonTag: "catch"})) // balance 1

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: base, Path: "target.go", Line: 42, Prompt: "fix the bug at target.go:42"}
	require.NoError(t, log.AppendDispatch("d1", live, own)) // spends the 1 → balance 0
	registerSession("live", LiveConfig{RepoDir: repo, BaseRev: base, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	drainQueuedOrders("live")

	assert.Equal(t, base, gotBase, "the catch cycle runs from the order's pre-specified base")
	assert.Equal(t, liveHead, gotFix, "the catch cycle runs on the agent-PRODUCED revision")
	assert.Equal(t, "target.go", gotAnchor.Path, "FIREWALL: the anchor is the order's pre-specified path, not the agent's diff")
	assert.Equal(t, 42, gotAnchor.Start, "FIREWALL: the anchor is the order's pre-specified line, not the agent's diff")

	assert.Equal(t, "done", statusOfOrder(t, log, 1), "the live order runs to done")
	bal, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "the live order's catch minted (the seed was spent, this is the new catch)")
	assert.True(t, mintedCatchWithProducer(t, f, "live", "wo:1"), "the catch is attributed to the work order (Producer wo:1)")
}

// A live order whose produced revision yields NO catch must mint nothing — the
// firewall holds when the agent's work doesn't kill the anchored survivor.
func TestRunLiveOrder_mintsNothingWhenTheLiveRevisionYieldsNoCatch(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	base := gitOrder(t, repo, "rev-parse", "HEAD")

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	runHarness = func(_ context.Context, repoDir, _ string) ([]harness.Turn, error) {
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "feature.go"), []byte("package main\n"), 0o644))
		gitOrder(t, repoDir, "add", "-A")
		gitOrder(t, repoDir, "commit", "-qm", "live fix")
		sha := gitOrder(t, repoDir, "rev-parse", "HEAD")
		return []harness.Turn{{Outcome: orchestrator.TurnOutcome{Minted: true, SHA: sha}}}, nil
	}

	restoreCycle := resolveCycle
	t.Cleanup(func() { resolveCycle = restoreCycle })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Verdict: "no catch"}, nil // no Record → nothing to mint
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "livenocatch", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "seed.go", Line: 1, ReasonTag: "catch"}))

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: base, Path: "target.go", Line: 42, Prompt: "attempt the fix"}
	require.NoError(t, log.AppendDispatch("d1", live, own))
	registerSession("livenocatch", LiveConfig{RepoDir: repo, BaseRev: base, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	drainQueuedOrders("livenocatch")

	assert.Equal(t, "done", statusOfOrder(t, log, 1), "the live order runs to done even with no catch")
	bal, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "a live revision that yields no catch mints nothing (balance unchanged)")
}

// A live run that produces NO revision (the agent committed nothing) must skip
// the catch cycle entirely — there is nothing to check the oracle against — and
// complete cleanly, minting nothing.
func TestRunLiveOrder_skipsTheCatchCycleWhenNoRevisionWasProduced(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	base := gitOrder(t, repo, "rev-parse", "HEAD")

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	runHarness = func(_ context.Context, _, _ string) ([]harness.Turn, error) {
		return []harness.Turn{{Outcome: orchestrator.TurnOutcome{Minted: false}}}, nil // the agent changed nothing
	}

	restoreCycle := resolveCycle
	t.Cleanup(func() { resolveCycle = restoreCycle })
	cycleCalled := false
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		cycleCalled = true
		return Resolution{}, nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "livenorev", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "seed.go", Line: 1, ReasonTag: "catch"}))

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: base, Path: "target.go", Line: 42, Prompt: "do nothing useful"}
	require.NoError(t, log.AppendDispatch("d1", live, own))
	registerSession("livenorev", LiveConfig{RepoDir: repo, BaseRev: base, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	drainQueuedOrders("livenorev")

	assert.False(t, cycleCalled, "no produced revision means the oracle cycle must not run")
	assert.Equal(t, "done", statusOfOrder(t, log, 1), "the order still completes")
	bal, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "no revision means no catch minted")
}

// The live fix revision is the SHA of the agent's last minted turn — a turn that
// changed nothing must not be mistaken for the produced revision.
func TestLastMintedSHA_returnsTheLastMintedTurnsRevision(t *testing.T) {
	tests := []struct {
		name  string
		turns []harness.Turn
		want  string
		ok    bool
	}{
		{"no turns", nil, "", false},
		{"none minted", []harness.Turn{
			{Outcome: orchestrator.TurnOutcome{Minted: false}},
			{Outcome: orchestrator.TurnOutcome{Minted: false}},
		}, "", false},
		{"last minted wins", []harness.Turn{
			{Outcome: orchestrator.TurnOutcome{Minted: true, SHA: "aaa"}},
			{Outcome: orchestrator.TurnOutcome{Minted: false}}, // a no-edit turn after the edit
			{Outcome: orchestrator.TurnOutcome{Minted: true, SHA: "ccc"}},
		}, "ccc", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sha, ok := lastMintedSHA(tt.turns)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, sha)
		})
	}
}
