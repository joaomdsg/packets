package app

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// approveServer wires a funded session whose live HEAD is a real repo, returning the
// log + a /-mounted client, so Approve tests drive the action over HTTP.
func approveServer(t *testing.T, key string) (*ledger.Log, *httptest.Server) {
	t.Helper()
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, key, "i")
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second)))
	registerSession(key, LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })
	return log, server
}

// Approve must REFUSE to open a PR when the tree can't land (red checks) and the Lead
// hasn't overridden — mirroring PR etiquette (you don't merge red CI). The PR
// subprocess is never invoked. NOT parallel (shared globals + openPR seam).
func TestApprove_refusesToOpenPRWhenBlocked(t *testing.T) {
	restore := openPR
	t.Cleanup(func() { openPR = restore })
	called := false
	openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
		called = true
		return "", "", nil
	}

	log, server := approveServer(t, "appblk")
	lookupLiveEntry("appblk").setLand("checks_red") // the integrated tree fails its checks
	_ = log

	tc := vt.NewClient(t, server, "/?key=appblk")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appblk"}).Approve).Fire(),
		"a blocked approve is a calm no-op, never a crash")
	assert.False(t, called, "a blocked tree never opens a PR")
}

// With an override, Approve opens the PR despite a block — the guard is deliberate
// friction, not a hard gate (DESIGN §16). NOT parallel (shared globals).
func TestApprove_overrideOpensThePRDespiteABlock(t *testing.T) {
	restore := openPR
	t.Cleanup(func() { openPR = restore })
	var gotBranch, gotTitle string
	openPR = func(_ context.Context, _, _, branch, title, _, _ string) (string, string, error) {
		gotBranch, gotTitle = branch, title
		return "https://example/pr/1", "sha1", nil
	}

	log, server := approveServer(t, "appovr")
	lookupLiveEntry("appovr").setLand("checks_red")
	require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "h", Prompt: "Add the widget."}, ledger.Target{}))

	tc := vt.NewClient(t, server, "/?key=appovr")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appovr"}).Approve).
		WithSignal("landoverride", "1").Fire())
	assert.Equal(t, prBranchName("appovr"), gotBranch, "the PR pushes the session's stable branch")
	assert.Equal(t, "Add the widget.", gotTitle, "the PR title comes from the session's task")
}

// A clean tree with no open threads opens the PR directly and surfaces the returned
// URL on the card, so the Lead sees their work landed. NOT parallel (shared globals).
func TestApprove_cleanTreeOpensPRAndSurfacesTheURL(t *testing.T) {
	restore := openPR
	t.Cleanup(func() { openPR = restore })
	openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
		return "https://example/pr/42", "sha42", nil
	}

	log, server := approveServer(t, "appok")
	require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "h", Prompt: "Do the task."}, ledger.Target{}))

	tc := vt.NewClient(t, server, "/?key=appok")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appok"}).Approve).Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=appok").HTML())
	assert.Contains(t, body, `href="https://example/pr/42"`,
		"the opened PR URL surfaces as a CLICKABLE link, not bare text — it's the finish line")
	assert.Equal(t, "sha42", lookupLiveEntry("appok").lastPushedSHASnapshot(),
		"a successful land caches the pushed SHA so the next re-land leases against it")
}

// The push can succeed while a LATER step (gh pr create / pr view) fails. The pushed SHA
// must STILL be cached — the remote branch already advanced to it, so the next re-land
// must lease against it. Caching only on full success would leave the cache stale and
// WEDGE the re-land (its lease would name the old SHA against a branch that moved). NOT
// parallel (shared globals).
func TestApprove_cachesThePushedSHAEvenWhenThePRStepFails(t *testing.T) {
	restore := openPR
	t.Cleanup(func() { openPR = restore })
	openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
		return "", "pushedsha1", errors.New("gh pr create: boom") // pushed, but PR-open failed
	}

	_, server := approveServer(t, "appwedge")
	tc := vt.NewClient(t, server, "/?key=appwedge")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appwedge"}).Approve).Fire())

	assert.Equal(t, "pushedsha1", lookupLiveEntry("appwedge").lastPushedSHASnapshot(),
		"the pushed SHA is cached despite the PR-open failure, so the next re-land's lease matches the remote")
	body := bodyOf(vt.NewClient(t, server, "/?key=appwedge").HTML())
	assert.Contains(t, body, "forward failed", "the failure still surfaces calmly on the card")
}

// A push/PR failure must surface calmly on the card, never crash the action. NOT
// parallel (shared globals).
func TestApprove_pushFailureSurfacesCalmly(t *testing.T) {
	restore := openPR
	t.Cleanup(func() { openPR = restore })
	openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
		return "", "", errors.New("push rejected")
	}

	_, server := approveServer(t, "appfail")
	tc := vt.NewClient(t, server, "/?key=appfail")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appfail"}).Approve).Fire(),
		"a failed push is still a calm 200")

	body := bodyOf(vt.NewClient(t, server, "/?key=appfail").HTML())
	assert.Contains(t, body, "push rejected", "the failure reason surfaces calmly on the card")
	assert.Equal(t, "", lookupLiveEntry("appfail").lastPushedSHASnapshot(),
		"a PRE-push failure pushed nothing, so there is no SHA to cache")
}

// Two concurrent Approve calls (a double-clicked "forward →", or two tabs) both spawn a
// git push + gh pr create against the SAME session repo — a second call must be dropped
// as a calm no-op while the first is in flight, never race the shared worktree/branch
// ops (the same class of race beginAnswer/endAnswer already closes for answer re-runs).
// NOT parallel (shared globals + openPR seam).
func TestApprove_dropsAConcurrentReLandRatherThanRacingTheSharedWorktree(t *testing.T) {
	restore := openPR
	t.Cleanup(func() { openPR = restore })
	started := make(chan struct{})
	var startedOnce sync.Once
	release := make(chan struct{})
	var calls int32
	openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
		atomic.AddInt32(&calls, 1)
		startedOnce.Do(func() { close(started) })
		<-release
		return "https://example/pr/1", "sha1", nil
	}

	log, server := approveServer(t, "applock")
	require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "h", Prompt: "Do it."}, ledger.Target{}))

	done := make(chan struct{})
	go func() {
		defer close(done)
		vt.NewClient(t, server, "/?key=applock").Action((&LiveCard{Key: "applock"}).Approve).Fire()
	}()
	<-started // the first call is in flight, holding the land slot

	require.Equal(t, 200, vt.NewClient(t, server, "/?key=applock").Action((&LiveCard{Key: "applock"}).Approve).Fire(),
		"a dropped duplicate is still a calm 200, never an error")

	close(release)
	<-done

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls),
		"a concurrent Approve must never invoke openPR a second time while one is already in flight")

	// The slot must release once the in-flight call finishes — a LATER, non-concurrent
	// Approve is not a duplicate and must run normally.
	require.Equal(t, 200, vt.NewClient(t, server, "/?key=applock").Action((&LiveCard{Key: "applock"}).Approve).Fire())
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls),
		"once the guard releases, a later Approve is a fresh call, not a dropped duplicate")
}

// "Landed ≠ Merged" (DESIGN §29.2): an opened PR is NOT a merge. classifyLandLifecycle
// maps gh's PR state to the post-open lifecycle, failing CLOSED — it never claims Merged
// unless gh definitively says so (an unknown/empty state reads as the conservative
// not-yet-merged Landed, never a false Merged).
func TestClassifyLandLifecycle_failsClosedNeverFalselyMerged(t *testing.T) {
	cases := []struct {
		state string
		want  landLifecycle
	}{
		{"MERGED", lifecycleMerged},
		{"merged", lifecycleMerged},   // case-insensitive
		{" MERGED ", lifecycleMerged}, // trimmed
		{"CLOSED", lifecycleBounced},  // closed unmerged
		{"OPEN", lifecycleLanded},     // landed, not yet merged
		{"", lifecycleLanded},         // unknown → conservative not-yet-merged
		{"WeIrD", lifecycleLanded},    // unknown → never a false Merged
	}
	for _, tc := range cases {
		if got := classifyLandLifecycle(tc.state); got != tc.want {
			t.Errorf("classifyLandLifecycle(%q) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

// CheckMergeState (DESIGN §29.2) refreshes an opened PR's merge lifecycle through the
// mergeState seam: a definitive gh state caches the right lifecycle; a seam error is a
// calm no-op; and it never runs (nor caches) when no PR was opened. NOT parallel.
func TestCheckMergeState_refreshesLifecycleOnlyForAnOpenedPR(t *testing.T) {
	restoreOpen := openPR
	restoreMerge := mergeState
	t.Cleanup(func() { openPR = restoreOpen; mergeState = restoreMerge })
	openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
		return "https://example/pr/7", "shaX", nil
	}

	log, server := approveServer(t, "appmerge")
	require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "h", Prompt: "Do it."}, ledger.Target{}))
	tc := vt.NewClient(t, server, "/?key=appmerge")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appmerge"}).Approve).Fire())
	// A freshly opened PR is landed-not-merged immediately.
	assert.Equal(t, string(lifecycleLanded), lookupLiveEntry("appmerge").landLifecycleSnapshot())

	// gh says MERGED → lifecycle refreshes to merged.
	mergeState = func(_ context.Context, _, _ string) (string, error) { return "MERGED", nil }
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appmerge"}).CheckMergeState).Fire())
	assert.Equal(t, string(lifecycleMerged), lookupLiveEntry("appmerge").landLifecycleSnapshot(),
		"a definitive merged state is surfaced")

	// A seam error is a calm no-op — the prior (merged) lifecycle stands, never a false downgrade.
	mergeState = func(_ context.Context, _, _ string) (string, error) { return "", errors.New("gh down") }
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appmerge"}).CheckMergeState).Fire())
	assert.Equal(t, string(lifecycleMerged), lookupLiveEntry("appmerge").landLifecycleSnapshot(),
		"a transient gh error leaves the prior lifecycle, never a false claim")
}

// CheckMergeState must NOT call the gh seam when no PR was opened (a blocked/none land) —
// there's nothing to check. NOT parallel.
func TestCheckMergeState_isANoOpWhenNoPRWasOpened(t *testing.T) {
	restoreMerge := mergeState
	t.Cleanup(func() { mergeState = restoreMerge })
	called := false
	mergeState = func(_ context.Context, _, _ string) (string, error) { called = true; return "MERGED", nil }

	_, server := approveServer(t, "appnopr") // no Approve fired → no opened PR
	tc := vt.NewClient(t, server, "/?key=appnopr")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appnopr"}).CheckMergeState).Fire())
	assert.False(t, called, "with no opened PR, the merge-state seam is never invoked")
	assert.Equal(t, "", lookupLiveEntry("appnopr").landLifecycleSnapshot())
}

// The land control surfaces the "Landed ≠ Merged" thesis on an opened PR (DESIGN §29.2):
// the badge says not-yet-merged and a check-merge-state action is wired. NOT parallel.
func TestRenderLandControl_surfacesLandedNotMergedOnAnOpenedPR(t *testing.T) {
	restore := openPR
	t.Cleanup(func() { openPR = restore })
	openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
		return "https://example/pr/9", "shaY", nil
	}
	log, server := approveServer(t, "applife")
	require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "h", Prompt: "Do it."}, ledger.Target{}))
	tc := vt.NewClient(t, server, "/?key=applife")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "applife"}).Approve).Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=applife").HTML())
	assert.Contains(t, body, "not yet forwarded", "an opened PR is shown as landed, NOT merged (§29.2)")
	assert.Contains(t, body, "/_action/CheckMergeState", "a check-merge-state affordance is wired")
}

// A stale "merged" badge must not linger under a new blocked/error outcome: a re-land that
// gets BLOCKED clears the lifecycle, so the badge disappears. NOT parallel.
func TestApprove_clearsLifecycleWhenAReLandIsBlocked(t *testing.T) {
	restore := openPR
	t.Cleanup(func() { openPR = restore })
	openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
		return "https://example/pr/3", "shaZ", nil
	}
	log, server := approveServer(t, "appclear")
	require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "h", Prompt: "Do it."}, ledger.Target{}))
	tc := vt.NewClient(t, server, "/?key=appclear")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appclear"}).Approve).Fire())
	require.Equal(t, string(lifecycleLanded), lookupLiveEntry("appclear").landLifecycleSnapshot())

	// A later land is now blocked (red checks) → the lifecycle badge is cleared.
	lookupLiveEntry("appclear").setLand("checks_red")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "appclear"}).Approve).Fire())
	assert.Equal(t, "", lookupLiveEntry("appclear").landLifecycleSnapshot(),
		"a blocked re-land clears the lifecycle so no stale 'merged'/'landed' badge lingers")
}

// The merged and bounced lifecycle badges render their honest copy after a merge check.
// NOT parallel.
func TestRenderLandControl_surfacesMergedAndBouncedLifecycle(t *testing.T) {
	render := func(t *testing.T, key, ghState, want string) {
		restoreOpen := openPR
		restoreMerge := mergeState
		t.Cleanup(func() { openPR = restoreOpen; mergeState = restoreMerge })
		openPR = func(_ context.Context, _, _, _, _, _, _ string) (string, string, error) {
			return "https://example/pr/1", "s", nil
		}
		log, server := approveServer(t, key)
		require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "h", Prompt: "Do it."}, ledger.Target{}))
		tc := vt.NewClient(t, server, "/?key="+key)
		require.Equal(t, 200, tc.Action((&LiveCard{Key: key}).Approve).Fire())
		mergeState = func(_ context.Context, _, _ string) (string, error) { return ghState, nil }
		require.Equal(t, 200, tc.Action((&LiveCard{Key: key}).CheckMergeState).Fire())
		body := bodyOf(vt.NewClient(t, server, "/?key="+key).HTML())
		assert.Contains(t, body, want)
	}
	render(t, "appmrg", "MERGED", "forwarded")
	render(t, "appbnc", "CLOSED", "closed, not forwarded")
}
