package app

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

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/orchestrator"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

func gitOrder(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v\n%s", args, out)
	return strings.TrimSpace(string(out))
}

func initGitRepoForOrder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitOrder(t, dir, "init", "-q")
	gitOrder(t, dir, "config", "user.email", "t@t")
	gitOrder(t, dir, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644))
	gitOrder(t, dir, "add", "-A")
	gitOrder(t, dir, "commit", "-qm", "base")
	return dir
}

func statusOfOrder(t *testing.T, log *ledger.Log, id int) string {
	t.Helper()
	views, err := log.RecentDispatches(0)
	require.NoError(t, err)
	for _, v := range views {
		if v.ID == id {
			return v.Status
		}
	}
	return ""
}

// A work order that carries a TASK PROMPT must be filled by a real live harness
// (which produces a revision in the repo), NOT by the pre-funded catch cycle —
// this is what makes a funded order do REAL work instead of replaying a baked
// base→fix diff.
func TestDrainQueuedOrders_routesAPromptOrderToTheLiveHarnessNotThePrefundedCycle(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	headBefore := gitOrder(t, repo, "rev-parse", "HEAD")

	restoreCycle := resolveCycle
	t.Cleanup(func() { resolveCycle = restoreCycle })
	cycleCalled := false
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		cycleCalled = true
		return Resolution{}, nil
	}

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	var gotRepoDir, gotPrompt string
	runHarness = func(_ context.Context, repoDir, prompt string) ([]harness.Turn, error) {
		gotRepoDir, gotPrompt = repoDir, prompt
		// A faithful stub of the agent: it edits the working tree and commits,
		// producing a real revision (moving HEAD) just as a live run would.
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "feature.go"), []byte("package main\n"), 0o644))
		gitOrder(t, repoDir, "add", "-A")
		gitOrder(t, repoDir, "commit", "-qm", "live turn")
		sha := gitOrder(t, repoDir, "rev-parse", "HEAD")
		return []harness.Turn{{Outcome: orchestrator.TurnOutcome{Minted: true, SHA: sha}}}, nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "live", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"})) // balance 1 to fund the dispatch

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: headBefore, Prompt: "add a feature.go file"}
	require.NoError(t, log.AppendDispatch("d1", live, own)) // spends the 1 → balance 0
	registerSession("live", LiveConfig{RepoDir: repo, BaseRev: headBefore, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	balBefore, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, 0, balBefore, "the seed catch was spent funding the dispatch")

	drainQueuedOrders("live")

	assert.Equal(t, repo, gotRepoDir, "the live harness runs in the order's repo")
	assert.Equal(t, "add a feature.go file", gotPrompt, "the order's prompt drives the live harness")
	assert.NotEqual(t, headBefore, gitOrder(t, repo, "rev-parse", "HEAD"), "the live run produced a real revision (HEAD moved)")
	assert.Equal(t, "done", statusOfOrder(t, log, 1), "the live order runs to done")
	assert.False(t, cycleCalled, "a prompt order must NOT run the pre-funded catch cycle")

	balAfter, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, balAfter, "firewall: the live run mints NO catch (the oracle/catch step is slice 4b)")
}

// A live run that fails (the harness errored — e.g. the agent crashed) must
// mark the order failed, never "done": a failed run is not a completed fill, and
// the order must reach a terminal status so it doesn't linger mid-flight.
func TestDrainQueuedOrders_marksALiveOrderFailedWhenTheHarnessErrors(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	headBefore := gitOrder(t, repo, "rev-parse", "HEAD")

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	runHarness = func(_ context.Context, _, _ string) ([]harness.Turn, error) {
		return nil, errors.New("harness crashed")
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "livefail", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: headBefore, Prompt: "do the task"}
	require.NoError(t, log.AppendDispatch("d1", live, own))
	registerSession("livefail", LiveConfig{RepoDir: repo, BaseRev: headBefore, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	drainQueuedOrders("livefail")

	assert.Equal(t, "failed", statusOfOrder(t, log, 1), "a harness error marks the order failed, not done")
}

// A work order WITHOUT a prompt must keep filling via the pre-funded catch
// cycle — the live-harness routing must not disturb the existing path.
func TestDrainQueuedOrders_keepsAPromptlessOrderOnThePrefundedCycle(t *testing.T) {
	resetConsumersForTest()

	restoreCycle := resolveCycle
	t.Cleanup(func() { resolveCycle = restoreCycle })
	cycleCalled := false
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		cycleCalled = true
		return Resolution{}, nil
	}

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	harnessCalled := false
	runHarness = func(_ context.Context, _, _ string) ([]harness.Turn, error) {
		harnessCalled = true
		return nil, nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "prefunded", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"})) // balance 1 to fund the dispatch

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	prefunded := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}
	require.NoError(t, log.AppendDispatch("d1", prefunded, own))
	registerSession("prefunded", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	drainQueuedOrders("prefunded")

	assert.True(t, cycleCalled, "a promptless order must run the pre-funded catch cycle")
	assert.False(t, harnessCalled, "a promptless order must NOT spawn the live harness")
	assert.Equal(t, "done", statusOfOrder(t, log, 1), "the promptless order completes via the existing path")
}
