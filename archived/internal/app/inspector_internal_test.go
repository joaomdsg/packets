package app

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// gitInitNoRemote initializes a bare local repo with no origin remote, so
// packet.ParseAddr on it deterministically falls back to the honest
// "local/<dir>" identity instead of resolving a real remote. Duplicated in
// the external test package, for the tests that moved there.
func gitInitNoRemote(t testing.TB, dir string) {
	t.Helper()
	require.NoError(t, exec.Command("git", "init", "-q", dir).Run())
}

// /review renders as the 3-column Inspector (252|1fr|312)
// rather than a flat stack — the changed-files tree, the Monaco
// island, and the annotation rail are three distinct, hairline-bounded
// regions. NOT parallel (shared liveReg/liveFabric).
func TestReviewCard_rendersTheInspectorGridWithThreeRegionsSessionScoped(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{{File: "auth.go", Line: 12, Outcome: mutation.Survived, Message: "mutated >= to >"}})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, `class="inspector"`, "the 3-column Inspector grid renders")
	require.Contains(t, body, `class="inspector__tree"`, "the changed-files tree region renders")
	require.Contains(t, body, `class="inspector__main"`, "the Monaco/main region renders")
	require.Contains(t, body, "inspector__rail", "the annotation rail region renders")
}

// The per-order review is the Inspector too — same three regions, now scoped to
// the funded work-order's own edits and questions. NOT parallel (shared
// liveReg + the resolveCycle seam).
func TestReviewCard_rendersTheInspectorGridWithThreeRegionsPacketScoped(t *testing.T) {
	resetConsumersForTest()
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Findings: []mutation.Finding{
			{File: "alpha.go", Line: 7, Outcome: mutation.Survived, Message: "mutated >= to >"},
		}}, nil
	}
	restoreReader := reviewFileReader
	t.Cleanup(func() { reviewFileReader = restoreReader })
	reviewFileReader = func(_ context.Context, _, _, _ string) (string, error) { return "package main\n", nil }

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "insp1", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	registerSession("insp1", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	drainQueuedPackets("insp1")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=insp1&wo=1").HTML())
	require.Contains(t, body, `class="inspector"`, "the 3-column Inspector grid renders for a per-order review")
	require.Contains(t, body, `class="inspector__tree"`, "the tree region renders")
	require.Contains(t, body, `class="inspector__main"`, "the main region renders")
	require.Contains(t, body, "inspector__rail", "the annotation rail renders")
	require.Contains(t, body, "file-tree", "the order's changed-files tree renders inside the tree region")
}

// The identity strip names the scope (pkt#<id> or the raw session key) and a
// base→fix rev chip in SHORT (7-char) SHAs when both revs are known — never a
// second brand mark (navHeader already carries it). NOT parallel.
func TestReviewCard_identityStripShowsScopeAndShortRevChipWhenRevsAreKnown(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "abcdef01234", FixRev: "98765432109", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, "default", "the identity strip names the raw session key when not order-scoped")
	require.Contains(t, body, "abcdef0", "the base rev is truncated to a 7-char short SHA")
	require.Contains(t, body, "9876543", "the fix rev is truncated to a 7-char short SHA")
	require.NotContains(t, body, "abcdef01234", "the full base rev never appears — short SHAs only")
}

// When either rev is unknown (a live/prompt session with no fix rev yet, or a
// per-order review whose target can't be resolved), the rev chip is OMITTED
// entirely rather than fabricated. NOT parallel.
func TestReviewCard_identityStripOmitsRevChipWhenARevIsUnknown(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "norev", "i")
	// FixRev deliberately unset — a live/prompt session with no produced fix yet.
	registerSession("norev", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=norev").HTML())
	require.NotContains(t, body, `class="inspector__rev"`, "the rev chip is omitted, never fabricated, when the fix rev is unknown")

	// A per-order review whose target cannot be resolved (an unfilled/unknown
	// order id) also has no known revs — the chip must be absent there too.
	unknownOrderBody := bodyOf(vt.NewClient(t, server, "/review?wo=999").HTML())
	require.NotContains(t, unknownOrderBody, `class="inspector__rev"`,
		"an unresolved per-order review omits the rev chip rather than showing zero-value revs")
}

// The identity strip's repo cell is the honest packet addr
// (packet.ParseAddr(cfg.RepoDir).String()), not a raw folder name — a
// remote-less repo falls back to "local/<dir>", never a fabricated owner.
// NOT parallel.
func TestReviewCard_identityStripShowsTheAddr(t *testing.T) {
	resetConsumersForTest()
	repoDir := filepath.Join(t.TempDir(), "acme-widgets")
	require.NoError(t, os.MkdirAll(repoDir, 0o755))
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: repoDir, BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, "local/acme-widgets", "a remote-less repo renders the honest local/<dir> addr")
}

// An order-scoped review that folds to a packet shows the
// packet's own Name beside pkt#<id> — the identity the packet model gives the
// order, not just its raw numeric id. NOT parallel (shared liveReg + the
// resolveCycle seam).
func TestReviewCard_identityStripShowsThePacketNameWhenPacketScoped(t *testing.T) {
	resetConsumersForTest()
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{
			Verdict: string(catch.Catch),
			Record:  &ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch"},
		}, nil
	}
	restoreReader := reviewFileReader
	t.Cleanup(func() { reviewFileReader = restoreReader })
	reviewFileReader = func(_ context.Context, _, _, _ string) (string, error) { return "package main\n", nil }

	repoDir := t.TempDir()
	gitInitNoRemote(t, repoDir)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "insppkt", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	registerSession("insppkt", LiveConfig{RepoDir: repoDir, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)
	drainQueuedPackets("insppkt")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=insppkt&wo=1").HTML())
	require.Contains(t, body, "local/"+filepath.Base(repoDir), "the identity strip's addr slot uses the real repo identity")
	require.Contains(t, body, "pkt-1", "the order's own packet name renders beside pkt#<id> once it folds to a packet")
}

// The annotation rail renders each open thread as an annotation card: an
// "agent" author chip (every open thread here is oracle-authored) and a
// "question" severity chip, while KEEPING the data-file/data-line attributes
// and the review-thread class the answer-form anchor flow and island payload
// depend on. NOT parallel.
func TestReviewCard_annotationRailRendersACardWithChipsAndRetainsTheAnchor(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{{File: "auth.go", Line: 12, Outcome: mutation.Survived, Message: "mutated >= to >"}})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, "annotation-card", "the thread renders as an annotation card")
	require.Contains(t, body, "review-thread annotation-card", "the annotation-card class is additive to review-thread")
	require.Contains(t, body, `data-file="auth.go"`, "the data-file anchor attribute is retained")
	require.Contains(t, body, `data-line="12"`, "the data-line anchor attribute is retained")
	require.Contains(t, body, ">agent<", "the author chip reads agent — every open thread is oracle-authored")
	require.Contains(t, body, ">question<", "the severity chip carries the thread's Conventional-Comment tag")
	require.Contains(t, body, "annotations · 1", "the rail kicker counts the open threads (label · N voice)")
}

// The timeline footer is the gauntlet record: the six gates,
// every one honestly NotRun for a session-scoped review with no single
// packet to gauntlet — never faked. Supersedes the earlier dashed-empty stub,
// which this footer content replaces. NOT parallel.
func TestReviewCard_timelineFooterRendersTheHonestlyUnmeasuredGauntlet(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, `class="inspector__timeline gauntlet"`, "the timeline footer region renders")
	require.Contains(t, body, "gauntlet", "the gauntlet kicker names itself")
	for _, name := range gauntletGateNames {
		require.Contains(t, body, name, "every gate row renders, an honest absence never hidden")
	}
}

// The session (non-drilled-in) review has no single base→fix diff to scope a
// file tree to, so the left rail shows the honest scope empty rather than an
// arbitrary tree — the open threads still list on the right. NOT parallel.
func TestReviewCard_sessionScopedLeftRailShowsTheHonestScopeEmpty(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{{File: "auth.go", Line: 12, Outcome: mutation.Survived, Message: "mutated >= to >"}})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, "pick a packet to scope the file tree",
		"the session-scoped tree region shows the honest empty, not an arbitrary tree")
	require.NotContains(t, body, `class="file-tree"`, "no file tree renders when the review isn't order-scoped")
	require.Contains(t, body, "annotation-card", "the open threads still list on the right")
}
