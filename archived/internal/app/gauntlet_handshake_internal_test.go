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
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// initHandshakeRepo builds a real git repo whose fix commit carries a
// handshake/spec_test.go — passing or failing per want — plus the base
// commit before it existed. The working tree is left AT the fix commit, and
// the handshake file's live content matches what was committed (the caller
// may then dirty it to simulate post-authoring tampering).
func initHandshakeRepo(t *testing.T, passing bool) (dir, base, fix, handshakeContent string) {
	t.Helper()
	dir = t.TempDir()
	gitPacket(t, dir, "init", "-q")
	gitPacket(t, dir, "config", "user.email", "t@t")
	gitPacket(t, dir, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture.test/handshake\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	gitPacket(t, dir, "add", "-A")
	gitPacket(t, dir, "commit", "-qm", "base")
	base = gitPacket(t, dir, "rev-parse", "HEAD")

	body := "_ = t"
	if !passing {
		body = `t.Fatal("handshake failed")`
	}
	handshakeContent = "package handshake\n\nimport \"testing\"\n\nfunc TestSpec(t *testing.T) { " + body + " }\n"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "handshake"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handshake", "spec_test.go"), []byte(handshakeContent), 0o644))
	gitPacket(t, dir, "add", "-A")
	gitPacket(t, dir, "commit", "-qm", "fix with handshake")
	fix = gitPacket(t, dir, "rev-parse", "HEAD")
	return dir, base, fix, handshakeContent
}

// Once an order's ledger.Target carries a HandshakePath/Hash,
// G2 becomes a REAL exec seam (RunHandshakeGate), not the earlier placeholder.
// NOT parallel (shared liveReg/liveFabric).
func TestReviewCard_gauntletRunsARealG2WhenThePacketHasAnAuthoredPassingHandshake(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix, content := initHandshakeRepo(t, true)
	handshakePath := filepath.Join(repo, "handshake", "spec_test.go")

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntlethsok", "i")
	fundSend(t, log, "d1", ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1,
		HandshakePath: handshakePath, HandshakeHash: reanchor.HashLines(content),
	})
	registerSession("gauntlethsok", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	_ = bodyOf(vt.NewClient(t, server, "/review?key=gauntlethsok&wo=1").HTML())

	e := lookupLiveEntry("gauntlethsok")
	require.NotNil(t, e)
	g := e.cachedGauntlet(1)
	assert.Equal(t, "passed", g.HandshakeConformance.Status.String())
}

// NOT parallel (shared liveReg/liveFabric).
func TestReviewCard_gauntletReportsG2FailedWhenTheAuthoredHandshakeTestsFail(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix, content := initHandshakeRepo(t, false)
	handshakePath := filepath.Join(repo, "handshake", "spec_test.go")

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntlethsfail", "i")
	fundSend(t, log, "d1", ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1,
		HandshakePath: handshakePath, HandshakeHash: reanchor.HashLines(content),
	})
	registerSession("gauntlethsfail", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	_ = bodyOf(vt.NewClient(t, server, "/review?key=gauntlethsfail&wo=1").HTML())

	e := lookupLiveEntry("gauntlethsfail")
	require.NotNil(t, e)
	g := e.cachedGauntlet(1)
	assert.Equal(t, "failed", g.HandshakeConformance.Status.String())
	assert.Contains(t, g.HandshakeConformance.Detail, "handshake failed")
}

// Integrity wins over a stale pass: if the LIVE handshake file (the one under
// repoDir, outside any per-revision worktree) no longer matches the hash
// recorded at authoring time, G2 is overridden to a hard fail regardless of
// what the fix revision's own committed handshake tests reported. NOT
// parallel (shared liveReg/liveFabric).
func TestReviewCard_gauntletOverridesG2ToFailedWhenTheHandshakeChangedAfterAuthoring(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix, content := initHandshakeRepo(t, true)
	handshakePath := filepath.Join(repo, "handshake", "spec_test.go")
	recordedHash := reanchor.HashLines(content)

	// Tamper with the LIVE file (outside the throwaway worktree RunHandshakeGate
	// checks out) — the committed fix revision's handshake still passes.
	require.NoError(t, os.WriteFile(handshakePath, []byte(content+"\n// tampered\n"), 0o644))

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntlethstamper", "i")
	fundSend(t, log, "d1", ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1,
		HandshakePath: handshakePath, HandshakeHash: recordedHash,
	})
	registerSession("gauntlethstamper", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	_ = bodyOf(vt.NewClient(t, server, "/review?key=gauntlethstamper&wo=1").HTML())

	e := lookupLiveEntry("gauntlethstamper")
	require.NotNil(t, e)
	g := e.cachedGauntlet(1)
	assert.Equal(t, "failed", g.HandshakeConformance.Status.String())
	assert.Contains(t, g.HandshakeConformance.Detail, "handshake changed after authoring")
}

// A handshake authored before the live order even fills (FixRev=="") must
// report an honest "no revision to build yet" for G2, not silently regress
// to the "no handshake yet" sentinel now that one actually exists — the
// live handshake file lives directly under repoDir, independent of any
// particular fix revision. NOT parallel (shared liveReg/liveFabric).
func TestReviewCard_gauntletReportsG2HonestlyForAHandshakeBearingPacketWithNoFixRevYet(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	handshakePath := filepath.Join(repo, "handshake", "spec_test.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(handshakePath), 0o755))
	content := "package handshake\n\nimport \"testing\"\n\nfunc TestSpec(t *testing.T) { _ = t }\n"
	require.NoError(t, os.WriteFile(handshakePath, []byte(content), 0o644))

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntlethsnorev", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: "catch", Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{
		BaseRev: head, Prompt: "do the thing",
		HandshakePath: handshakePath, HandshakeHash: reanchor.HashLines(content),
	}, own))
	registerSession("gauntlethsnorev", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	// The rendered body reflects gauntletFor's live (uncached) computation for a
	// revless packet — cachedGauntlet would instead read the (empty, since
	// revless never caches) cache slot, which proves nothing about what this
	// render actually computed. G5 (test sensitivity) is UNCHANGED this slice
	// and still renders the shared "not measured — no handshake yet" sentinel
	// unconditionally, so the assertion counts occurrences rather than using a
	// blanket NotContains — exactly one (G5's), never two (G2's old sentinel).
	body := bodyOf(vt.NewClient(t, server, "/review?key=gauntlethsnorev&wo=1").HTML())
	assert.Equal(t, 1, strings.Count(body, "not measured — no handshake yet"), "only G5 (unchanged, out of scope this slice) renders the no-handshake sentinel — G2 has an authored handshake and must not")
	assert.Contains(t, body, "no revision to build yet", "G2 is honestly not-run because there is no fix revision to gate yet, not because no handshake exists")

	e := lookupLiveEntry("gauntlethsnorev")
	require.NotNil(t, e)
	e.gauntletMu.Lock()
	_, cached := e.gauntletCache[1]
	e.gauntletMu.Unlock()
	assert.False(t, cached, "a revless packet is never cached, even when it carries a handshake")
}

// A cache HIT for a handshake-bearing order must never recompute G2 — proven
// by tampering with the handshake AFTER the first visit cached a passed
// result; a recompute would see the tamper and fail, so a still-passed
// second visit is direct evidence the cache served the render. NOT parallel
// (shared liveReg/liveFabric).
func TestReviewCard_secondVisitServesTheCachedG2WithoutRecomputing(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix, content := initHandshakeRepo(t, true)
	handshakePath := filepath.Join(repo, "handshake", "spec_test.go")

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntlethscache", "i")
	fundSend(t, log, "d1", ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1,
		HandshakePath: handshakePath, HandshakeHash: reanchor.HashLines(content),
	})
	registerSession("gauntlethscache", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	_ = bodyOf(vt.NewClient(t, server, "/review?key=gauntlethscache&wo=1").HTML())
	e := lookupLiveEntry("gauntlethscache")
	require.NotNil(t, e)
	require.Equal(t, "passed", e.cachedGauntlet(1).HandshakeConformance.Status.String())

	require.NoError(t, os.WriteFile(handshakePath, []byte(content+"\n// tampered after first visit\n"), 0o644))

	_ = bodyOf(vt.NewClient(t, server, "/review?key=gauntlethscache&wo=1").HTML())
	assert.Equal(t, "passed", e.cachedGauntlet(1).HandshakeConformance.Status.String(), "the cached value served the render — a recompute would have seen the tamper and failed")
}

// When the live handshake file is gone (VerifyHandshake errors rather than
// reporting a clean mismatch), G2 falls through to the fix revision's own
// test-run result rather than fabricating a failure from an infra error —
// the plan's override applies only to a confirmed (false, nil) mismatch.
// NOT parallel (shared liveReg/liveFabric).
func TestReviewCard_gauntletFallsBackToTheTestRunResultWhenTheLiveHandshakeFileIsGone(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix, content := initHandshakeRepo(t, true)
	handshakePath := filepath.Join(repo, "handshake", "spec_test.go")
	recordedHash := reanchor.HashLines(content)
	require.NoError(t, os.Remove(handshakePath))

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntlethsgone", "i")
	fundSend(t, log, "d1", ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1,
		HandshakePath: handshakePath, HandshakeHash: recordedHash,
	})
	registerSession("gauntlethsgone", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	_ = bodyOf(vt.NewClient(t, server, "/review?key=gauntlethsgone&wo=1").HTML())

	e := lookupLiveEntry("gauntlethsgone")
	require.NotNil(t, e)
	g := e.cachedGauntlet(1)
	assert.Equal(t, "passed", g.HandshakeConformance.Status.String(), "the committed fix revision's own handshake test run still passed — a gone live file is an infra fact, not a fabricated fail")
}

// The 100ms Stream poll must never exec go test to compute G2, exactly like
// G3/G4 — proven here with an order that DOES carry a real,
// runnable handshake, so an errant poll-time computation would be caught.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_streamPollNeverComputesG2ForAPacketWithARealHandshake(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix, content := initHandshakeRepo(t, true)
	handshakePath := filepath.Join(repo, "handshake", "spec_test.go")

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "gauntlethspoll", "i")
	fundSend(t, log, "d1", ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1,
		HandshakePath: handshakePath, HandshakeHash: reanchor.HashLines(content),
	})
	registerSession("gauntlethspoll", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=gauntlethspoll")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `console__hero-addr`)

	require.NoError(t, log.AppendStatus(1, "running"))
	vt.AwaitFrame(t, frames, 10*time.Second, "in flight · 1")

	e := lookupLiveEntry("gauntlethspoll")
	require.NotNil(t, e)
	e.gauntletMu.Lock()
	_, cached := e.gauntletCache[1]
	e.gauntletMu.Unlock()
	assert.False(t, cached, "the poll observably re-rendered over this connection yet never computed the handshake-bearing order's gauntlet")
}
