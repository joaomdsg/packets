package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/translate"
)

// The container runner is built and wired, but was reachable only via the boot-time
// -container flag. The Lead must be able to switch a session between host-subprocess
// and hardened-container execution from the card. NOT parallel (shared globals).
func TestLiveCard_toggleRunnerSwitchesTheLiveRunnerMode(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath, // UseContainer defaults false
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	require.False(t, lookupLiveEntry(defaultSessionKey).useContainerMode(), "a session defaults to the host subprocess")

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&LiveCard{}).ToggleRunner).Fire())
	require.True(t, lookupLiveEntry(defaultSessionKey).useContainerMode(), "toggling opts the session into the container")
	require.Equal(t, 200, tc.Action((&LiveCard{}).ToggleRunner).Fire())
	require.False(t, lookupLiveEntry(defaultSessionKey).useContainerMode(), "toggling again returns to the host subprocess")
}

// The runtime toggle must actually change which runner a subsequent live order uses
// — not just a cosmetic label. A session toggled into the container runs its next
// live order in the container runner, overriding the boot default. NOT parallel.
func TestDrainQueuedPackets_honorsARuntimeRunnerToggle(t *testing.T) {
	resetConsumersForTest()
	stubResolveNoCatch(t)
	repo := initGitRepoForPacket(t)
	base := gitPacket(t, repo, "rev-parse", "HEAD")

	var procCalled, ctrCalled bool
	restoreProc := runHarness
	t.Cleanup(func() { runHarness = restoreProc })
	runHarness = func(_ context.Context, _, _ string, _ func([]translate.UIEvent)) ([]harness.Turn, error) {
		procCalled = true
		return nil, nil
	}
	var ctrRepo, ctrPrompt string
	restoreCtr := runHarnessContainer
	t.Cleanup(func() { runHarnessContainer = restoreCtr })
	runHarnessContainer = editingStubHarness(t, &ctrCalled, &ctrRepo, &ctrPrompt)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "rt", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: base, Path: "t.go", Line: 4, Prompt: "fix it"}, own))
	// Boot default is host; the Lead toggles this session into the container at runtime.
	registerSession("rt", LiveConfig{RepoDir: repo, BaseRev: base, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	lookupLiveEntry("rt").toggleRunner()

	drainQueuedPackets("rt")

	assert.True(t, ctrCalled, "a session toggled into the container runs its next live order in the container runner")
	assert.False(t, procCalled, "the host runner is not used once the session is toggled to the container")
}
