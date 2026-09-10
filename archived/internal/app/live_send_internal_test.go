package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
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

// orderRecordFor returns the dispatched work-order with the given id (zero value
// when absent), so a test can assert on the funded order's target/prompt.
// duplicated for the external test package (used by the tests that moved there).
func orderRecordFor(t *testing.T, log *ledger.Log, id int) ledger.SendView {
	t.Helper()
	views, err := log.RecentSends(0)
	require.NoError(t, err)
	for _, v := range views {
		if v.ID == id {
			return v
		}
	}
	return ledger.SendView{}
}

// A Lead must be able to AUTHOR a live order from the card — type a task prompt and
// place it — instead of only drawing a pre-baked target off the backlog. Placing an
// order funds it against the balance (one catch, like any spend) and dispatches a
// prompt-carrying target so the live harness runs the authored task.
// A handshake must be authored FIRST — placing without one is refused
// (proven separately). NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_sendFundsAndSendsTheAuthoredPrompt(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForPacket(t)
	head := gitPacket(t, repo, "rev-parse", "HEAD")

	// Stub the harness so the placed order's background drain neither spawns claude
	// nor errors — this test's subject is the AUTHORING + dispatch, not the run.
	restoreHarness := runHarness
	t.Cleanup(func() { runHarness = restoreHarness })
	runHarness = func(_ context.Context, _, _ string, _ func([]translate.UIEvent)) ([]harness.Turn, error) {
		return nil, nil
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "author", "i")
	// A live order is funded by attention bandwidth: clear a block fast to earn it.
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second))) // +3 bandwidth
	registerSession("author", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=author")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "author"}).AuthorHandshake).
		WithSignal("handshakedraft", "package handshake\n\nfunc TestSpec(t *testing.T) {}\n").
		WithSignal("handshakestrengthpick", "examples").
		Fire(), "a handshake must be authored before a live order can be placed")

	require.Equal(t, 200, tc.Action((&LiveCard{Key: "author"}).Send).
		WithSignal("draft", "add a feature.go file").Fire(),
		"authoring a live order is a calm, valid action")

	got := orderRecordFor(t, log, 1)
	assert.Equal(t, "add a feature.go file", got.Target.Prompt, "the order carries the authored prompt")
	assert.Equal(t, head, got.Target.BaseRev, "the order's base is the repo's live HEAD, so the agent works the current tree")
	assert.Equal(t, filepath.Join(repo, "handshake", "spec_test.go"), got.Target.HandshakePath, "the dispatched order carries the authored handshake")
	assert.NotEmpty(t, got.Target.HandshakeHash)
	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 2, bw, "authoring a live order spends one attention bandwidth to fund it (3 earned − 1)")
}

// Once a handshake exists, placing succeeds and the refusal message from an
// earlier attempt clears. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_sendSucceedsOnceAHandshakeExists(t *testing.T) {
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
	log := ledger.Bind(f, "authorretry", "i")
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second))) // +3 bandwidth
	registerSession("authorretry", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=authorretry")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authorretry"}).Send).
		WithSignal("draft", "add a feature.go file").Fire(), "the first attempt is refused")
	require.Empty(t, orderRecordFor(t, log, 1).Target.Prompt)

	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authorretry"}).AuthorHandshake).
		WithSignal("handshakedraft", "package handshake\n\nfunc TestSpec(t *testing.T) {}\n").
		WithSignal("handshakestrengthpick", "properties").
		Fire())

	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authorretry"}).Send).
		WithSignal("draft", "add a feature.go file").Fire(), "the retry succeeds now that a handshake exists")

	got := orderRecordFor(t, log, 1)
	assert.Equal(t, "add a feature.go file", got.Target.Prompt)

	body := bodyOf(vt.NewClient(t, server, "/?key=authorretry").HTML())
	assert.NotContains(t, body, "author a handshake before sending", "a successful placement clears the earlier refusal message")
}

// A handshake is CONSUMED by the order it's placed for — a second live order
// must author its own, never silently reuse a prior one. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_sendConsumesTheHandshakeSoASecondPacketMustAuthorItsOwn(t *testing.T) {
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
	log := ledger.Bind(f, "authorconsume", "i")
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(60*time.Second))) // +6 bandwidth, enough for two orders
	registerSession("authorconsume", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=authorconsume")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authorconsume"}).AuthorHandshake).
		WithSignal("handshakedraft", "package handshake\n\nfunc TestSpec(t *testing.T) {}\n").
		WithSignal("handshakestrengthpick", "examples").
		Fire())
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authorconsume"}).Send).
		WithSignal("draft", "first order").Fire(), "the first order consumes the authored handshake")
	require.NotEmpty(t, orderRecordFor(t, log, 1).Target.Prompt)

	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authorconsume"}).Send).
		WithSignal("draft", "second order").Fire(), "a refusal is still a calm 200")

	assert.Empty(t, orderRecordFor(t, log, 2).Target.Prompt, "the second order has no handshake of its own — it must be refused, not silently reuse the first's")
}
