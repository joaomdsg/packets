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
	"github.com/joaomdsg/packets/internal/translate"
)

// editingStubHarness returns a runHarness-shaped stub that edits the repo + returns
// a minted turn (like the live-order stubs), recording the repoDir/prompt it got.
func editingStubHarness(t *testing.T, called *bool, gotRepo, gotPrompt *string) func(context.Context, string, string, func([]translate.UIEvent)) ([]harness.Turn, error) {
	return func(_ context.Context, repoDir, prompt string, _ func([]translate.UIEvent)) ([]harness.Turn, error) {
		*called, *gotRepo, *gotPrompt = true, repoDir, prompt
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "f.txt"), []byte("one\nTWO\nthree\n"), 0o644))
		gitOrder(t, repoDir, "add", "-A")
		gitOrder(t, repoDir, "commit", "-qm", "edit")
		sha := gitOrder(t, repoDir, "rev-parse", "HEAD")
		return []harness.Turn{{Outcome: orchestrator.TurnOutcome{Minted: true, SHA: sha}}}, nil
	}
}

func stubResolveNoCatch(t *testing.T) {
	t.Helper()
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil
	}
}

// A session marked UseContainer must run its live orders in the agent CONTAINER
// (harness.RunContainer), not the host subprocess — that's how a Lead opts a
// session into containerized execution without touching runLiveOrder.
func TestDrainQueuedOrders_runsAUseContainerOrderInTheContainerRunner(t *testing.T) {
	resetConsumersForTest()
	stubResolveNoCatch(t)
	repo := initGitRepoForOrder(t)
	base := gitOrder(t, repo, "rev-parse", "HEAD")

	var procCalled, ctrCalled bool
	var ctrRepo, ctrPrompt string
	restoreProc := runHarness
	t.Cleanup(func() { runHarness = restoreProc })
	runHarness = func(_ context.Context, _, _ string, _ func([]translate.UIEvent)) ([]harness.Turn, error) {
		procCalled = true
		return nil, nil
	}
	restoreCtr := runHarnessContainer
	t.Cleanup(func() { runHarnessContainer = restoreCtr })
	runHarnessContainer = editingStubHarness(t, &ctrCalled, &ctrRepo, &ctrPrompt)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "ctr", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: base, Path: "t.go", Line: 4, Prompt: "fix it"}, own))
	registerSession("ctr", LiveConfig{RepoDir: repo, BaseRev: base, Anchor: anchorForCap(), TestCmd: []string{"true"}, UseContainer: true}, log)

	drainQueuedOrders("ctr")

	assert.True(t, ctrCalled, "a UseContainer session runs its live order in the container runner")
	assert.False(t, procCalled, "the host-subprocess runner must NOT be used for a UseContainer session")
	assert.Equal(t, repo, ctrRepo, "the container runner gets the order's repo")
	assert.Equal(t, "fix it", ctrPrompt, "the container runner gets the order's prompt")
	assert.Equal(t, "done", statusOfOrder(t, log, 1), "the order runs to done")
}

// By default (UseContainer false) a live order runs on the host subprocess — the
// existing behavior is preserved; opting into the container is explicit.
func TestDrainQueuedOrders_runsAPromptOrderOnTheSubprocessRunnerByDefault(t *testing.T) {
	resetConsumersForTest()
	stubResolveNoCatch(t)
	repo := initGitRepoForOrder(t)
	base := gitOrder(t, repo, "rev-parse", "HEAD")

	var procCalled, ctrCalled bool
	var procRepo, procPrompt string
	restoreProc := runHarness
	t.Cleanup(func() { runHarness = restoreProc })
	runHarness = editingStubHarness(t, &procCalled, &procRepo, &procPrompt)
	restoreCtr := runHarnessContainer
	t.Cleanup(func() { runHarnessContainer = restoreCtr })
	runHarnessContainer = func(_ context.Context, _, _ string, _ func([]translate.UIEvent)) ([]harness.Turn, error) {
		ctrCalled = true
		return nil, nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "proc", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: base, Path: "t.go", Line: 4, Prompt: "fix it"}, own))
	registerSession("proc", LiveConfig{RepoDir: repo, BaseRev: base, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log) // UseContainer defaults false

	drainQueuedOrders("proc")

	assert.True(t, procCalled, "by default a live order runs on the host subprocess")
	assert.False(t, ctrCalled, "the container runner is not used unless opted in")
}
