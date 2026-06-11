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

// The Lead must see a live agent's latest activity update AS THE RUN STREAMS, not
// just after it finishes. As the harness streams thinking→editing, the session's
// activity snapshot must reflect the latest beat each time — so the card can show
// the agent working in real time.
func TestRunLiveOrder_updatesTheActivitySnapshotLiveAsTheHarnessStreams(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	base := gitOrder(t, repo, "rev-parse", "HEAD")

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	var seen []string
	runHarness = func(_ context.Context, repoDir, _ string, onActivity func([]translate.UIEvent)) ([]harness.Turn, error) {
		// Stream activity as a live run would; after each batch the per-session
		// snapshot must already reflect the latest beat (the stub runs on the test
		// goroutine during the synchronous drain, standing in for "mid-run").
		onActivity([]translate.UIEvent{{Type: "activity.agent", Kind: "thinking", Detail: "considering the bug"}})
		seen = append(seen, lookupLiveEntry("liveact").activitySnapshot())
		onActivity([]translate.UIEvent{{Type: "activity.agent", Kind: "editing", Detail: "auth.go"}})
		seen = append(seen, lookupLiveEntry("liveact").activitySnapshot())
		// A batch with several events surfaces only the LATEST (the agent's most
		// recent move), not the first or all of them.
		onActivity([]translate.UIEvent{
			{Type: "activity.agent", Kind: "thinking", Detail: "first"},
			{Type: "activity.agent", Kind: "tool", Detail: "go test ./..."},
		})
		seen = append(seen, lookupLiveEntry("liveact").activitySnapshot())

		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "auth.go"), []byte("package main\n"), 0o644))
		gitOrder(t, repoDir, "add", "-A")
		gitOrder(t, repoDir, "commit", "-qm", "live fix")
		sha := gitOrder(t, repoDir, "rev-parse", "HEAD")
		return []harness.Turn{{Outcome: orchestrator.TurnOutcome{Minted: true, SHA: sha}}}, nil
	}

	restoreCycle := resolveCycle
	t.Cleanup(func() { resolveCycle = restoreCycle })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil // no-catch: keep the test fast and focused on activity
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "liveact", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "seed.go", Line: 1, ReasonTag: "catch"}))

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: base, Path: "target.go", Line: 42, Prompt: "fix the bug"}
	require.NoError(t, log.AppendDispatch("d1", live, own))
	registerSession("liveact", LiveConfig{RepoDir: repo, BaseRev: base, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	drainQueuedOrders("liveact")

	assert.Equal(t, []string{"thinking", "editing auth.go", "running go test ./..."}, seen,
		"the per-session activity snapshot updates live to the LATEST streamed beat of each batch")
}

// The activity line is a human-legible summary of the agent's latest beat — the
// card shows "editing auth.go" / "running go test", not a raw event kind.
func TestFormatActivity_rendersAHumanLegibleBeat(t *testing.T) {
	tests := []struct {
		name string
		ev   translate.UIEvent
		want string
	}{
		{"thinking", translate.UIEvent{Type: "activity.agent", Kind: "thinking", Detail: "weighing the error path"}, "thinking"},
		{"editing names the file", translate.UIEvent{Type: "activity.agent", Kind: "editing", Detail: "internal/auth/token.go"}, "editing internal/auth/token.go"},
		{"tool names the command", translate.UIEvent{Type: "activity.agent", Kind: "tool", Detail: "go test ./..."}, "running go test ./..."},
		{"unknown kind falls back to detail", translate.UIEvent{Type: "activity.agent", Kind: "other", Detail: "something"}, "something"},
		{"unknown kind empty detail falls back to kind", translate.UIEvent{Type: "activity.agent", Kind: "weird", Detail: ""}, "weird"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatActivity(tt.ev))
		})
	}
}
