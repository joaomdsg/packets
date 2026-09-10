package app

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/translate"
)

// The review turn quotes the line under comment, so the orchestrator must read that
// exact source line from the working tree. readSourceLine returns the 1-indexed line's
// content, and degrades to "" (no quote) for a missing file or an out-of-range line —
// never an error that would block dispatching the adjustment.
func TestReadSourceLine_returnsTheOneIndexedLineOrEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("line one\nline two\nline three\n"), 0o644))

	assert.Equal(t, "line two", readSourceLine(dir, "a.go", 2), "the 1-indexed line content is returned")
	assert.Equal(t, "", readSourceLine(dir, "a.go", 99), "an out-of-range line degrades to no quote")
	assert.Equal(t, "", readSourceLine(dir, "missing.go", 1), "a missing file degrades to no quote")
	assert.Equal(t, "", readSourceLine(dir, "a.go", 0), "a non-positive line degrades to no quote")
}

// The keystone of the review-thread loop: leaving an adjustment on a line dispatches a
// harness turn (DESIGN §12.3) that tells the agent what to fix, against the session's
// live HEAD — reusing the live-order pipe. The dispatched order must carry the composed
// review turn (the comment + the address-it instruction), so the agent re-edits in
// place. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_addAdjustmentSendsAReviewTurnToTheHarness(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	runHarness = func(_ context.Context, _, _ string, _ func([]translate.UIEvent)) ([]harness.Turn, error) {
		return nil, nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "adjust", "i")
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second))) // +3 bandwidth funds the turn
	registerSession("adjust", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/review?key=adjust")
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "adjust"}).AddAdjustment).
		WithSignal("adjfile", "main.go").
		WithSignal("adjline", "3").
		WithSignal("adjtext", "handle the nil case here").Fire(),
		"leaving an adjustment is a calm, valid action")

	got := orderRecordFor(t, log, 1)
	assert.Contains(t, got.Target.Prompt, "handle the nil case here", "the dispatched turn carries the comment")
	assert.Contains(t, got.Target.Prompt, "main.go", "the turn names the commented file")
	assert.Contains(t, got.Target.Prompt, "Address this specifically", "the turn instructs the agent to address it")
	assert.Equal(t, head, got.Target.BaseRev, "the turn runs against the session's live HEAD")
}

// After an adjustment is left, the surface must report whether it was addressed by
// relocating the anchor against the file's current content — so "leave an adjustment →
// watch it addressed" has a visible payoff instead of a write-only box. When the
// commented line still exists unchanged, the badge says it's still there. NOT parallel
// (shared globals).
func TestReviewCard_surfacesWhetherTheAdjustmentWasAddressed(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "adjstat", "i")
	registerSession("adjstat", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	// initGitRepoForPacket seeds base.txt whose line 1 is "base"; anchor there.
	lookupLiveEntry("adjstat").addAdjAnchor("base.txt", 1, "base", "address this")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=adjstat").HTML())
	assert.Contains(t, body, "still on line 1",
		"an adjustment whose line is unchanged surfaces as still-anchored, not a stale silent comment")
	assert.Contains(t, body, "address this", "the adjustment's own comment is shown beside its status")
	assert.Contains(t, body, "/_action/ResolveAdjustment",
		"each adjustment carries a resolve affordance so the Lead can clear it")
}

// A real review leaves SEVERAL adjustments — each must surface its OWN status badge with
// its OWN comment, not just the last one. NOT parallel (shared globals).
func TestReviewCard_surfacesEveryAdjustmentNotJustTheLast(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")
	// Seed a second line so two distinct anchors exist.
	require.NoError(t, os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\nsecond\n"), 0o644))
	gitPacket(t, repo, "add", "-A")
	gitPacket(t, repo, "commit", "-qm", "two lines")
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "adjmulti", "i")
	registerSession("adjmulti", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	e := lookupLiveEntry("adjmulti")
	e.addAdjAnchor("base.txt", 1, "base", "guard the first line")
	e.addAdjAnchor("base.txt", 2, "second", "and the second line")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=adjmulti").HTML())
	assert.Contains(t, body, "guard the first line", "the first adjustment is not clobbered by the second")
	assert.Contains(t, body, "and the second line", "the second adjustment also surfaces")
	assert.Equal(t, 2, strings.Count(body, "still on line"), "both adjustments render their own status badge")
}

// When the agent's settled revision shifted the commented line, the badge must say it
// moved (the addressed-payoff), and when the line was edited away it must say so —
// locking the moved/outdated badge text the Lead reads. NOT parallel (shared globals).
func TestReviewCard_surfacesMovedAndOutdatedAdjustments(t *testing.T) {
	render := func(t *testing.T, key, fileContent string) string {
		resetConsumersForTest()
		repo := initGitRepoForPacket(t)
		head := gitPacket(t, repo, "rev-parse", "HEAD")
		ctx := context.Background()
		f, err := fabric.Start(ctx, t.TempDir())
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })
		log := ledger.Bind(f, key, "i")
		registerSession(key, LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
		lookupLiveEntry(key).addAdjAnchor("base.txt", 1, "base", "address this")
		// The settled revision: rewrite the working-tree file the badge reads.
		require.NoError(t, os.WriteFile(filepath.Join(repo, "base.txt"), []byte(fileContent), 0o644))

		defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
		var server *httptest.Server
		viaApp, defLog, err := NewServer(LiveConfig{
			RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
			TestCmd: []string{"true"}, LedgerPath: defLogPath,
		})
		require.NoError(t, err)
		server = httptest.NewServer(viaApp)
		t.Cleanup(func() { _ = defLog.Close() })
		return bodyOf(vt.NewClient(t, server, "/review?key="+key).HTML())
	}

	moved := render(t, "adjmoved", "x\ny\nbase\n") // "base" shifted to line 3
	assert.Contains(t, moved, "addressed — moved to line 3", "a shifted line surfaces as addressed/moved")

	outdated := render(t, "adjgone", "nothing here now\n") // "base" edited away
	assert.Contains(t, outdated, "addressed — line edited", "an edited-away line surfaces as addressed/outdated")
}

// Re-commenting the SAME file:line replaces that adjustment rather than stacking a
// duplicate badge — the surface shows one entry per commented line with the latest
// comment. NOT parallel (shared globals).
func TestLiveEntry_reCommentingTheSameLineReplacesNotStacks(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "adjdedupe", "i")
	registerSession("adjdedupe", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	e := lookupLiveEntry("adjdedupe")

	e.addAdjAnchor("base.txt", 1, "base", "first thought")
	e.addAdjAnchor("base.txt", 1, "base", "actually, this instead") // same line, re-commented

	got := e.adjAnchorsSnapshot()
	require.Len(t, got, 1, "re-commenting the same line replaces, never stacks a duplicate")
	assert.Equal(t, "actually, this instead", got[0].comment, "the latest comment wins")
}

// The Lead can RESOLVE an addressed adjustment — it leaves the list (closing the loop),
// while other adjustments stay. An unknown file:line is a calm no-op. NOT parallel.
func TestReviewCard_resolveAdjustmentClearsOneAnchor(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "adjresolve", "i")
	registerSession("adjresolve", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	e := lookupLiveEntry("adjresolve")
	e.addAdjAnchor("base.txt", 1, "base", "first")
	e.addAdjAnchor("other.go", 7, "x := 1", "second")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	// An unknown file:line is a calm no-op (both kept).
	tc := vt.NewClient(t, server, "/review?key=adjresolve")
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "adjresolve"}).ResolveAdjustment).
		WithSignal("adjfile", "nope.go").WithSignal("adjline", "99").Fire())
	require.Len(t, e.adjAnchorsSnapshot(), 2, "an unknown anchor resolves nothing")

	// Resolve the first → it leaves, the second stays.
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "adjresolve"}).ResolveAdjustment).
		WithSignal("adjfile", "base.txt").WithSignal("adjline", "1").Fire())
	got := e.adjAnchorsSnapshot()
	require.Len(t, got, 1, "the resolved adjustment leaves the list")
	assert.Equal(t, "other.go", got[0].file, "the other adjustment stays")
}
