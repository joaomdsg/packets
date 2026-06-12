package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"
)

// The warm-up establishes the session's resumable id and explores the repo headlessly
// so later analyze/order requests resume a warm context. It pins the id (--session-id),
// reads the repo with tool access (bypassPermissions), and is a one-shot text run.
func TestWarmArgs_pinsTheSessionIdAndExploresHeadlessly(t *testing.T) {
	t.Parallel()
	args := warmArgs("sess-xyz")
	assert.Contains(t, args, "--session-id")
	assert.Contains(t, args, "sess-xyz")
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "--permission-mode")
	assert.Contains(t, args, "bypassPermissions")
	assert.Contains(t, args, "text")
}

// A session's harness id is only usable AFTER the warm-up explore completes: until
// then, requests run cold (resume nothing), so they never try to --resume a session
// that is still being established (which would fail). Once warm, requests resume it.
func TestResumeSessionID_isEmptyUntilTheWarmUpCompletes(t *testing.T) {
	t.Parallel()
	e := &liveEntry{harnessSessionID: "sid-1"}
	assert.Equal(t, "", e.resumeSessionID(), "a not-yet-warm session resumes nothing (runs cold)")
	e.markWarm()
	assert.Equal(t, "sid-1", e.resumeSessionID(), "once warm, requests resume the explored session")
}

// A Board-created session warms a harness IMMEDIATELY (explores the repo) and
// remembers its session id, so the first analyze/order already resumes warm context.
// NOT parallel (shared globals).
func TestCreateSession_warmsAndRemembersTheHarnessSession(t *testing.T) {
	restore := warmHarnessRun
	t.Cleanup(func() { warmHarnessRun = restore })
	gotRepo := make(chan string, 1)
	var gotSession string
	warmHarnessRun = func(_ context.Context, repoDir, sessionID string) error {
		gotSession = sessionID
		gotRepo <- repoDir
		return nil
	}

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/board")
	require.Equal(t, 200, tc.Action((&BoardCard{}).CreateSession).WithSignal("newkey", "warmed").Fire())

	select {
	case r := <-gotRepo:
		assert.Equal(t, ".", r, "the warm-up explores the new session's repo")
	case <-time.After(5 * time.Second):
		t.Fatal("creating a session did not start the warm-up harness")
	}
	require.Eventually(t, func() bool {
		e := lookupLiveEntry("warmed")
		return e != nil && e.resumeSessionID() == gotSession && gotSession != ""
	}, 5*time.Second, 10*time.Millisecond, "the session remembers its warm harness id once explored")
}
