package app

import (
	"context"
	"net/http/httptest"
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
	"github.com/joaomdsg/packets/internal/translate"
)

// orderRecordFor returns the dispatched work-order with the given id (zero value
// when absent), so a test can assert on the funded order's target/prompt.
func orderRecordFor(t *testing.T, log *ledger.Log, id int) ledger.DispatchView {
	t.Helper()
	views, err := log.RecentDispatches(0)
	require.NoError(t, err)
	for _, v := range views {
		if v.ID == id {
			return v
		}
	}
	return ledger.DispatchView{}
}

// A Lead must be able to AUTHOR a live order from the card — type a task prompt and
// place it — instead of only drawing a pre-baked target off the backlog. Placing an
// order funds it against the balance (one catch, like any spend) and dispatches a
// prompt-carrying target so the live harness runs the authored task. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_placeOrderFundsAndDispatchesTheAuthoredPrompt(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	head := gitOrder(t, repo, "rev-parse", "HEAD")

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
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	registerSession("author", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=author")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "author"}).PlaceOrder).
		WithSignal("orderprompt", "add a feature.go file").Fire(),
		"authoring a live order is a calm, valid action")

	got := orderRecordFor(t, log, 1)
	assert.Equal(t, "add a feature.go file", got.Target.Prompt, "the order carries the authored prompt")
	assert.Equal(t, head, got.Target.BaseRev, "the order's base is the repo's live HEAD, so the agent works the current tree")
	bal, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "authoring an order spends one catch to fund it, like any dispatch")
}

// An empty prompt is not an order: placing one must be a silent no-op, never a
// funded work-order with no task. NOT parallel (shared globals).
func TestLiveCard_placeOrderIsANoOpOnAnEmptyPrompt(t *testing.T) {
	resetConsumersForTest()
	repo := initGitRepoForOrder(t)
	head := gitOrder(t, repo, "rev-parse", "HEAD")

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "empty", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	registerSession("empty", LiveConfig{RepoDir: repo, BaseRev: head, Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=empty")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "empty"}).PlaceOrder).WithSignal("orderprompt", "   ").Fire())

	bal, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "an empty prompt funds nothing — the balance is untouched")
}

// The card must render the order-authoring control (a prompt input bound to the
// order signal + the place-order action) when there is balance to fund it, so the
// Lead has a way to author and place a live order. NOT parallel (shared globals).
func TestLiveCard_rendersTheOrderAuthoringControlWhenFunded(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "compose", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	registerSession("compose", LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/?key=compose").HTML())
	require.Contains(t, body, "/_action/PlaceOrder", "the card renders the place-order action binding")
	require.Contains(t, body, `data-bind="orderprompt"`, "with an input bound to the order-prompt signal")
}
