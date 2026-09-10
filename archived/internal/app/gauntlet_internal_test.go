package app

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/pipe"
)

var gauntletGateNames = []string{
	"intent fidelity", "handshake conformance", "handshake tightness",
	"build · vet · lint", "test sensitivity", "independent check",
}

// Every gate row renders, including the ones with no real
// mechanic yet (G2/G5/G6 stay NotRun for now) — an absent gate is never
// hidden. G4 is a REAL exec seam over the fixture's own clean build. NOT
// parallel (shared liveReg/liveFabric).
func TestReviewCard_PacketScopedTimelineRendersAllSixGatesComputedOnRender(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletall", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletall", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=gauntletall&wo=1").HTML())
	assert.Contains(t, body, "gauntlet")
	for _, name := range gauntletGateNames {
		assert.Contains(t, body, name, "every gate row renders")
	}

	e := lookupLiveEntry("gauntletall")
	require.NotNil(t, e)
	g := e.cachedGauntlet(1)
	assert.Equal(t, packet.GatePassed, g.BuildVetLint.Status, "the fixture repo builds and vets clean")
	assert.Equal(t, packet.GateNotRun, g.HandshakeConformance.Status, "G2 has no handshake concept yet")
	assert.Equal(t, packet.GateNotRun, g.HandshakeTightness.Status, "no catch cycle ran for this order — G3 must not fabricate a pass or fail")
	assert.Equal(t, "not measured — no catch cycle run yet", g.HandshakeTightness.Detail)
	assert.Equal(t, packet.GateNotRun, g.TestSensitivity.Status, "G5 needs the handshake/agent-test split before it can measure")
	assert.Equal(t, packet.GateNotRun, g.IndependentCheck.Status, "G6 needs cage wired into local dispatch")
}

// G3 (handshake tightness) folds from the order's OWN cached catch-cycle
// outcome — never a re-run of the mutation oracle. NOT parallel.
func TestReviewCard_timelineDerivesHandshakeTightnessFromThePacketsCachedCatchOutcome(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletg3", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletg3", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	e := lookupLiveEntry("gauntletg3")
	require.NotNil(t, e)
	e.setOrderCatchOutcome(1, catch.NoCatch, 3, 5)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=gauntletg3&wo=1").HTML())
	assert.Contains(t, body, "3 survivors of 5 — gap found")
	assert.Contains(t, body, `data-status="held"`)
}

// The G1 human residual is a real ACTION, not a computed gate: confirming it
// flips the gate to Passed and the confirmation SURVIVES a later render (it
// is read fresh from its own store on every gauntletFor/cachedGauntlet call,
// never baked into the G3/G4 cache entry). NOT parallel.
func TestReviewCard_confirmIntentFidelityMarksItPassedAndPersistsAcrossReRender(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletconfirm", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletconfirm", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	before := bodyOf(vt.NewClient(t, server, "/review?key=gauntletconfirm&wo=1").HTML())
	assert.Contains(t, before, "/_action/ConfirmIntentFidelity", "the confirm affordance renders while intent fidelity is not-run")

	tc := vt.NewClient(t, server, "/review?key=gauntletconfirm&wo=1")
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "gauntletconfirm"}).ConfirmIntentFidelity).
		WithSignal("confirmwo", "1").Fire())

	after := bodyOf(vt.NewClient(t, server, "/review?key=gauntletconfirm&wo=1").HTML())
	assert.Contains(t, after, "confirmed by gauntletconfirm", "the confirmation names the session key as the confirming identity — no fabricated identity")
	assert.NotContains(t, after, "/_action/ConfirmIntentFidelity", "a confirmed gate no longer offers the confirm affordance")
}

// A SECOND confirmation, for a different order, must not clobber or be
// blocked by the first — intentFidelityConfirmed is a real per-order map,
// not a single slot reused across confirmations. NOT parallel.
func TestReviewCard_confirmIntentFidelityForASecondPacketDoesNotClobberTheFirst(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)
	target := ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletconfirmtwo", "i")
	fundSend(t, log, "d1", target)
	fundSend(t, log, "d2", target)
	registerSession("gauntletconfirmtwo", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/review?key=gauntletconfirmtwo&wo=1")
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "gauntletconfirmtwo"}).ConfirmIntentFidelity).
		WithSignal("confirmwo", "1").Fire(), "the first confirmation initializes the map")
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "gauntletconfirmtwo"}).ConfirmIntentFidelity).
		WithSignal("confirmwo", "2").Fire(), "the second confirmation, for a DIFFERENT order, hits the already-initialized map")

	order1 := bodyOf(vt.NewClient(t, server, "/review?key=gauntletconfirmtwo&wo=1").HTML())
	order2 := bodyOf(vt.NewClient(t, server, "/review?key=gauntletconfirmtwo&wo=2").HTML())
	assert.Contains(t, order1, "confirmed by gauntletconfirmtwo", "order 1's own confirmation stuck")
	assert.Contains(t, order2, "confirmed by gauntletconfirmtwo", "order 2's confirmation also stuck — neither clobbered the other")
}

// The 100ms via.Stream poll (OnConnect) must NEVER exec git/go to compute a
// gauntlet — only a render (View) may, scoped to the packet shown in detail.
// Mirrors the lane poll-safety proof exactly. NOT
// parallel (shared liveReg/liveFabric).
func TestLiveCard_streamPollNeverComputesTheGauntletCacheForAnUnvisitedPacket(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletpoll", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletpoll", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=gauntletpoll")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `console__hero-addr`)

	require.NoError(t, log.AppendStatus(1, "running"))
	vt.AwaitFrame(t, frames, 10*time.Second, "in flight · 1")

	e := lookupLiveEntry("gauntletpoll")
	require.NotNil(t, e)
	e.gauntletMu.Lock()
	_, cached := e.gauntletCache[1]
	e.gauntletMu.Unlock()
	assert.False(t, cached, "the poll observably re-rendered over this connection yet never computed the measurable order's gauntlet")
}

// A live PROMPT order has no produced fix revision yet — nothing to gauntlet,
// so every gate must answer honestly WITHOUT caching (a later render, once
// the order fills, must get a real computation). Mirrors the lane revless
// test. NOT parallel.
func TestReviewCard_PacketScopedTimelineShowsUnmeasuredGatesWithoutCachingWhenFixRevIsUnknown(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletnorev", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: "catch", Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", Prompt: "do the thing"}, own))
	registerSession("gauntletnorev", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=gauntletnorev&wo=1").HTML())
	assert.Equal(t, 6, strings.Count(body, `data-status="not-run"`), "with no fix rev, every one of the six gates is honestly not-run")

	e := lookupLiveEntry("gauntletnorev")
	require.NotNil(t, e)
	e.gauntletMu.Lock()
	_, cached := e.gauntletCache[1]
	e.gauntletMu.Unlock()
	assert.False(t, cached, "a revless packet is never cached — a later render (once revs exist) must get a real gauntlet")
}

// A cache HIT must never recompute: proven by breaking the repo's git history
// AFTER the first visit cached a passed build/vet gate — a recompute would
// see no .git and answer not-run, so the chip staying passed on the second
// visit is direct evidence the cache, not a fresh exec, served the render.
// Mirrors the lane cache-hit proof. NOT parallel.
func TestReviewCard_secondVisitServesTheCachedGauntletWithoutRecomputing(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletcachehit", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletcachehit", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	first := bodyOf(vt.NewClient(t, server, "/review?key=gauntletcachehit&wo=1").HTML())
	require.Contains(t, first, `data-status="passed"`, "build/vet passed on the clean fixture")

	require.NoError(t, os.RemoveAll(filepath.Join(repo, ".git")), "a recompute against this repo would now see no .git and answer not-run")

	second := bodyOf(vt.NewClient(t, server, "/review?key=gauntletcachehit&wo=1").HTML())
	assert.Contains(t, second, `data-status="passed"`, "the cached value served the render — a recompute would have found no .git")
}

// Each order's gauntlet is cached at its OWN id — visiting one order must
// never populate or corrupt a different order's cache slot. Mirrors the
// lane sibling's per-order cache-isolation proof (the lane-health-grid test).
// NOT parallel.
func TestReviewCard_visitingOnePacketNeverCachesAGauntletForADifferentPacket(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)
	target := ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletisolate", "i")
	fundSend(t, log, "d1", target) // order 1 — will be visited
	fundSend(t, log, "d2", target) // order 2 — left unvisited
	registerSession("gauntletisolate", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	_ = bodyOf(vt.NewClient(t, server, "/review?key=gauntletisolate&wo=1").HTML())

	e := lookupLiveEntry("gauntletisolate")
	require.NotNil(t, e)
	e.gauntletMu.Lock()
	_, order1Cached := e.gauntletCache[1]
	_, order2Cached := e.gauntletCache[2]
	e.gauntletMu.Unlock()
	assert.True(t, order1Cached, "the visited order's gauntlet is cached")
	assert.False(t, order2Cached, "the never-visited order's cache slot stays empty — visiting order 1 must not populate it")
}

// A blank/zero/invalid confirmwo signal is a calm no-op, mirroring how
// ResolveAdjustment/AddAdjustment handle blank input. NOT parallel.
func TestReviewCard_confirmIntentFidelityWithNoPacketIDIsASilentNoOp(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletconfirmnoop", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletconfirmnoop", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/review?key=gauntletconfirmnoop&wo=1")
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "gauntletconfirmnoop"}).ConfirmIntentFidelity).Fire(),
		"a blank confirmwo signal is a calm no-op, not an error")
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "gauntletconfirmnoop"}).ConfirmIntentFidelity).
		WithSignal("confirmwo", "0").Fire())
	require.Equal(t, 200, tc.Action((&ReviewCard{Key: "gauntletconfirmnoop"}).ConfirmIntentFidelity).
		WithSignal("confirmwo", "not-a-number").Fire())

	body := bodyOf(vt.NewClient(t, server, "/review?key=gauntletconfirmnoop&wo=1").HTML())
	assert.Contains(t, body, "/_action/ConfirmIntentFidelity", "intent fidelity is still not-run — none of the no-op posts confirmed it")
	assert.NotContains(t, body, "confirmed by", "a blank/invalid order id must never confirm ANY order's intent fidelity")
}

// The session-scoped review has no single packet to gauntlet — every gate is
// the honest zero value (NotRun), never hidden, and there is no order id to
// confirm intent fidelity against. NOT parallel.
func TestReviewCard_sessionScopedTimelineRendersAllSixGatesHonestlyUnmeasured(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletsession", "i")
	registerSession("gauntletsession", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=gauntletsession").HTML())
	for _, name := range gauntletGateNames {
		assert.Contains(t, body, name)
	}
	assert.Equal(t, 6, strings.Count(body, `data-status="not-run"`), "a session-scoped review has no single packet to gauntlet — every gate is honestly not-run")
	assert.NotContains(t, body, "/_action/ConfirmIntentFidelity", "there is no order id to confirm against in the session-scoped view")
}

// catchTranscriptJSON is the verdict bytes a cage emits for a genuine catch on
// the anchored line — mirrors internal/cage's own test double (unexported
// there, so this file needs its own copy rather than a cross-package import).
func catchTranscriptJSON(t *testing.T, path string, line int) string {
	t.Helper()
	b, err := json.Marshal(pipe.Transcript{
		Outcome: catch.Catch, Reason: pipe.ReasonNone, Path: path, Line: line, Land: pipe.LandClean,
		Before: catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}},
		After:  catch.LineState{Inventory: []string{">="}, Survivors: nil},
	})
	require.NoError(t, err)
	return string(b)
}

// G6 (independent check) is cage re-derivation wired into local dispatch: a
// filled order, with cage configured process-wide via
// StartCageClaimConsumers, gets a REAL re-verify through the same
// cage-backed Verifier the claim consumers run — not a re-read of any
// in-process catch outcome. The passed gate is cached alongside G3/G4. NOT
// parallel (shared liveReg/liveFabric + the process-global cage wiring).
func TestReviewCard_independentCheckReRunsTheCageAndCachesAPassedG6(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletg6", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletg6", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	var invoked atomic.Int32
	cageCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	StartCageClaimConsumers(cageCtx, "img", blessingRunner{output: catchTranscriptJSON(t, "main.go", 1), invoked: &invoked})

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=gauntletg6&wo=1").HTML())
	assert.Contains(t, body, "handshake tightened — 0 survivors of 1", "G6 re-derived a real passed catch, not the honest not-run default")
	assert.GreaterOrEqual(t, invoked.Load(), int32(1), "G6 must actually invoke the cage runner, not fabricate a result")

	e := lookupLiveEntry("gauntletg6")
	require.NotNil(t, e)
	g := e.cachedGauntlet(1)
	assert.Equal(t, packet.GatePassed, g.IndependentCheck.Status)

	e.gauntletMu.Lock()
	cached, ok := e.gauntletCache[1]
	e.gauntletMu.Unlock()
	require.True(t, ok, "G6 is cached alongside G3/G4 on a filled order")
	assert.Equal(t, packet.GatePassed, cached.IndependentCheck.Status)
}

// A claim whose target can never resolve (ledger.ErrClaimUnverifiable) must
// leave G6 honestly not-run — a permanent verify failure is not itself a
// proven finding about the fix, so it must never render as a pass or fail.
// NOT parallel.
func TestReviewCard_independentCheckStaysNotRunOnAnUnverifiableTarget(t *testing.T) {
	resetConsumersForTest()
	repo, _, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletg6bad", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: "0000000000000000000000000000000000bad0", FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletg6bad", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	var invoked atomic.Int32
	cageCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	StartCageClaimConsumers(cageCtx, "img", blessingRunner{output: catchTranscriptJSON(t, "main.go", 1), invoked: &invoked})

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	_ = bodyOf(vt.NewClient(t, server, "/review?key=gauntletg6bad&wo=1").HTML()) // triggers the render that computes+caches G6

	e := lookupLiveEntry("gauntletg6bad")
	require.NotNil(t, e)
	g6 := e.cachedGauntlet(1).IndependentCheck
	assert.Equal(t, packet.GateNotRun, g6.Status, "a permanent verify failure is never a fabricated pass or fail")
	assert.Equal(t, "not measured — cage could not verify this claim's target", g6.Detail)
	assert.Equal(t, int32(0), invoked.Load(), "an unresolvable target never reaches the runner — Materialize refuses it first")
}

// G6 stays the honest notMeasuredNoCage default when cage was never
// configured for this process (StartCageClaimConsumers not called) — even for
// an otherwise fully filled, real-repo order. NOT parallel.
func TestReviewCard_independentCheckStaysTheHonestDefaultWhenCageIsUnconfigured(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntletg6off", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1})
	registerSession("gauntletg6off", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=gauntletg6off&wo=1").HTML())
	assert.Contains(t, body, "not measured — cage not wired to local send")

	e := lookupLiveEntry("gauntletg6off")
	require.NotNil(t, e)
	assert.Equal(t, notMeasuredNoCage, e.cachedGauntlet(1).IndependentCheck)
}
