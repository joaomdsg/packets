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
	// A live order is funded by attention bandwidth: clear a block fast to earn it.
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second))) // +3 bandwidth
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
	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 2, bw, "authoring a live order spends one attention bandwidth to fund it (3 earned − 1)")
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
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second))) // +3 bandwidth
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

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 3, bw, "an empty prompt funds nothing — the bandwidth meter is untouched")
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
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second))) // +3 bandwidth funds the control
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
	require.Contains(t, body, "authoring-editor", "with the editable Monaco editor as the draft source")
	require.Contains(t, body, "$orderprompt=evt.detail.draft", "whose value is lifted into the order-prompt signal at place time")
}
