package app

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/packet"
)

// initMeasurableRepo builds a real one-package git repo (a go.mod + a single
// main.go) with a base and a fix commit that touches that one package — the
// smallest fixture whose blast radius is deterministic: touching the module's
// only package always ripples through 100% of a 1-package graph, LaneStrict.
// Working tree is left AT the fix commit (Measure reads the on-disk tree).
func initMeasurableRepo(t *testing.T) (dir, base, fix string) {
	t.Helper()
	dir = t.TempDir()
	gitPacket(t, dir, "init", "-q")
	gitPacket(t, dir, "config", "user.email", "t@t")
	gitPacket(t, dir, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture.test/lane\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	gitPacket(t, dir, "add", "-A")
	gitPacket(t, dir, "commit", "-qm", "base")
	base = gitPacket(t, dir, "rev-parse", "HEAD")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { _ = 1 }\n"), 0o644))
	gitPacket(t, dir, "add", "-A")
	gitPacket(t, dir, "commit", "-qm", "fix")
	fix = gitPacket(t, dir, "rev-parse", "HEAD")
	return dir, base, fix
}

// fundSend funds one distinct work-order (one catch debits one dispatch)
// against target, returning nothing — a thin helper shared by this file's
// lane tests to avoid repeating the catch+dispatch boilerplate. The funding
// catch's AfterRev is set to id so two calls in the same test never collide
// on the ledger's (BeforeRev, AfterRev, Path, Line, ReasonTag) catch identity.
func fundSend(t *testing.T, log *ledger.Log, id string, target ledger.Target) {
	t.Helper()
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: "catch", Path: "c.go", Line: 1, AfterRev: id, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend(id, target, own))
}

// The Inspector's identity strip names a measured lane for
// an order-scoped packet, computed ON RENDER (never self-reported) — the
// single-package fixture's own change is a 100%-of-graph ripple, LaneStrict.
// NOT parallel (shared liveReg/liveFabric).
func TestReviewCard_PacketScopedTitlebarShowsTheMeasuredLaneChipComputedOnRender(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "lanechip", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("lanechip", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=lanechip&wo=1").HTML())
	assert.Contains(t, body, "lane strict", "the single-package repo's own change ripples through 100% of its 1-package graph")

	e := lookupLiveEntry("lanechip")
	require.NotNil(t, e)
	assert.Equal(t, packet.LaneStrict, e.cachedLane(1), "rendering the titlebar computed AND cached the lane")
}

// A live PROMPT order has no produced fix revision yet (FixRev stays empty
// until the harness runs) — nothing to measure, so the chip must say so
// honestly WITHOUT shelling out (there is no revision to diff). NOT parallel.
func TestReviewCard_PacketScopedTitlebarShowsUnmeasuredWithoutCachingWhenRevsAreUnknown(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "laneunmeasured", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: "catch", Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", Prompt: "do the thing"}, own))
	registerSession("laneunmeasured", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=laneunmeasured&wo=1").HTML())
	assert.Contains(t, body, "lane unmeasured")

	e := lookupLiveEntry("laneunmeasured")
	require.NotNil(t, e)
	e.laneMu.Lock()
	_, cached := e.laneCache[1]
	e.laneMu.Unlock()
	assert.False(t, cached, "a revless packet is never cached — a later render (once revs exist) must get a real measurement")
}

// The 100ms via.Stream poll (OnConnect) must NEVER exec `go list`/git to
// compute a lane — only a render (View) may, and only for the packet being
// shown in detail. Proven causally: force the poll to observably re-render
// (a dispatch status change over the SAME connection), then assert the lane
// cache still holds nothing for the measurable order that was live in
// sessionPackets the whole time. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_streamPollNeverComputesTheLaneCacheForAnUnvisitedPacket(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "lanepoll", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("lanepoll", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=lanepoll")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `console__hero-addr`)

	// A status change the poll must NOTICE and re-render for (proves the
	// 100ms tick body actually ran its dispatch-tally logic over this
	// connection) — the in-flight strip picks up the now-running order.
	require.NoError(t, log.AppendStatus(1, "running"))
	vt.AwaitFrame(t, frames, 10*time.Second, "in flight · 1")

	e := lookupLiveEntry("lanepoll")
	require.NotNil(t, e)
	e.laneMu.Lock()
	_, cached := e.laneCache[1]
	e.laneMu.Unlock()
	assert.False(t, cached, "the poll observably re-rendered over this connection yet never computed the measurable order's lane")
}

// The Console's lane-health grid counts ONLY lanes already in the cache — an
// order never opened in the Inspector stays "unmeasured" there until it is,
// and the grid never computes to fill a gap. Honest zero counts for the
// untouched buckets, never hidden. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_laneHealthGridCountsOnlyLanesAlreadyCachedByAPriorInspectorVisit(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "lanegrid", "i")
	target := ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1}
	fundSend(t, log, "d1", target) // order 1 — will be visited
	fundSend(t, log, "d2", target) // order 2 — left unvisited
	registerSession("lanegrid", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	// Visiting order 1's Inspector computes and caches its lane; order 2 is
	// never visited, so it stays uncached (the honest "unmeasured" bucket).
	_ = bodyOf(vt.NewClient(t, server, "/review?key=lanegrid&wo=1").HTML())

	body := bodyOf(vt.NewClient(t, server, "/?key=lanegrid").HTML())
	assert.Contains(t, body, `class="console__lane-health"`, "the lane-health grid renders on the Console")
	assert.Contains(t, body, `data-lane="strict">1<`, "order 1's cached strict lane is counted")
	assert.Contains(t, body, `data-lane="unmeasured">1<`, "order 2's never-visited lane is honestly unmeasured")
	assert.Contains(t, body, `data-lane="best-effort">0<`, "an empty bucket renders an honest zero, never hidden")
	assert.Contains(t, body, `data-lane="standard">0<`, "an empty bucket renders an honest zero, never hidden")
}

// A cache HIT must never recompute: proven by breaking the repo's git history
// AFTER the first visit cached "strict" — a recompute would hit a git error
// and flip the cache to "unmeasured" (laneFor's documented error-caches
// behavior), so the chip staying "lane strict" on the second visit is direct
// evidence the cached value, not a fresh Measure, served the render. NOT
// parallel (shared liveReg/liveFabric).
func TestReviewCard_secondVisitServesTheCachedLaneWithoutRecomputing(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "lanecachehit", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("lanecachehit", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	first := bodyOf(vt.NewClient(t, server, "/review?key=lanecachehit&wo=1").HTML())
	require.Contains(t, first, "lane strict")

	require.NoError(t, os.RemoveAll(filepath.Join(repo, ".git")), "a recompute against this repo would now error")

	second := bodyOf(vt.NewClient(t, server, "/review?key=lanecachehit&wo=1").HTML())
	assert.Contains(t, second, "lane strict", "the cached value served the render — a recompute would have errored and shown unmeasured")
}

// A Measure ERROR is cached as LaneUnmeasured (not left as a miss) — distinct
// from the "revs unknown" skip-cache path proven above. Verified by checking
// cachedLaneEntry's own hit/miss bool, not just the resulting lane value. NOT
// parallel (shared liveReg/liveFabric).
func TestReviewCard_laneComputationErrorIsCachedNotLeftAsAMiss(t *testing.T) {
	resetConsumersForTest()
	repo, base, _ := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "lanemeasureerr", "i")
	// Both revs are set (so laneFor attempts to compute) but the fix rev
	// doesn't resolve — Measure errors.
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: "not-a-real-rev", TipRev: base, Path: "main.go", Line: 1})
	registerSession("lanemeasureerr", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=lanemeasureerr&wo=1").HTML())
	assert.Contains(t, body, "lane unmeasured")

	e := lookupLiveEntry("lanemeasureerr")
	require.NotNil(t, e)
	lane, cached := e.cachedLaneEntry(1)
	assert.True(t, cached, "a Measure error must be CACHED (as unmeasured), not left as a retryable miss")
	assert.Equal(t, packet.LaneUnmeasured, lane)
}
