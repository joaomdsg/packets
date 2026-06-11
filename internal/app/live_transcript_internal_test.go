package app

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/orchestrator"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/translate"
)

// The latest-move line shows only the agent's CURRENT beat; to watch a run unfold a
// Lead needs the accruing TRANSCRIPT — every beat, in order. As the harness streams,
// the session's activity transcript must accumulate each formatted beat (not just
// replace the latest). NOT parallel (shared globals).
func TestRunLiveOrder_accumulatesTheActivityTranscriptAsTheHarnessStreams(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	base := gitOrder(t, repo, "rev-parse", "HEAD")

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	runHarness = func(_ context.Context, repoDir, _ string, onActivity func([]translate.UIEvent)) ([]harness.Turn, error) {
		onActivity([]translate.UIEvent{{Type: "activity.agent", Kind: "thinking", Detail: "considering"}})
		onActivity([]translate.UIEvent{{Type: "activity.agent", Kind: "editing", Detail: "auth.go"}})
		onActivity([]translate.UIEvent{{Type: "activity.agent", Kind: "tool", Detail: "go test ./..."}})
		// The transcript holds every beat in stream order — the run made legible, not
		// just its final move (the stub runs synchronously during the drain = mid-run).
		assert.Equal(t, []string{"thinking", "editing auth.go", "running go test ./..."},
			lookupLiveEntry("transcript").activityTranscript(),
			"the transcript accumulates each streamed beat in order")

		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "auth.go"), []byte("package main\n"), 0o644))
		gitOrder(t, repoDir, "add", "-A")
		gitOrder(t, repoDir, "commit", "-qm", "live fix")
		sha := gitOrder(t, repoDir, "rev-parse", "HEAD")
		return []harness.Turn{{Outcome: orchestrator.TurnOutcome{Minted: true, SHA: sha}}}, nil
	}

	restoreCycle := resolveCycle
	t.Cleanup(func() { resolveCycle = restoreCycle })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "transcript", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "seed.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: base, Path: "target.go", Line: 42, Prompt: "fix the bug"}
	require.NoError(t, log.AppendDispatch("d1", live, own))
	registerSession("transcript", LiveConfig{RepoDir: repo, BaseRev: base, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	drainQueuedOrders("transcript")
}

// endFill clears the transcript with the rest of the live-fill buffer when the
// order resolves — a finished order's transcript does not linger past the run.
// NOT parallel (shared globals).
func TestLiveEntry_endFillClearsTheActivityTranscript(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "clear", "i")
	registerSession("clear", LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap()}, log)

	e := lookupLiveEntry("clear")
	e.startFill(1)
	e.addActivityBeat("thinking")
	require.NotEmpty(t, e.activityTranscript())

	e.endFill()
	assert.Empty(t, e.activityTranscript(), "a resolved order's transcript is cleared, not left to linger")
}

// While an order is in flight the card must render the scrolling activity
// transcript — every beat so far, in order — so the Lead watches the run unfold.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_rendersTheScrollingActivityTranscriptForAnInFlightOrder(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "showt", "i")
	registerSession("showt", LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	// Stand the session in an in-flight fill with two streamed beats, then render — the
	// transcript shows while fillingOrder>0 (endFill, which clears it, never runs here).
	e := lookupLiveEntry("showt")
	e.startFill(1)
	e.addActivityBeat("thinking")
	e.addActivityBeat("editing auth.go")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/?key=showt").HTML())
	require.Contains(t, body, `data-state="transcript"`, "the in-flight order renders a transcript region")
	require.Contains(t, body, "thinking", "the transcript shows the first streamed beat")
	require.Contains(t, body, "editing auth.go", "and the latest beat, in order")
}
