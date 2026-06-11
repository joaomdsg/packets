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
	"github.com/joaomdsg/packets/internal/ledger"
)

// fundedAuthoringServer boots a session with attention bandwidth (so the compose +
// analyze controls render) and the default server. It returns the session's log and
// the test server. NOT-parallel callers only (shared liveReg/liveFabric).
func fundedAuthoringServer(t *testing.T, key string) (*ledger.Log, *httptest.Server) {
	t.Helper()
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, key, "i")
	bbase := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("q1", bbase))
	require.NoError(t, log.AppendUnblock("q1", bbase.Add(30*time.Second))) // +3 bandwidth funds authoring
	registerSession(key, LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })
	return log, server
}

// The Lead authors a live order's prompt blind today. The authoring assist must run
// a producer over the draft and surface its structured read — the one-line summary,
// the readiness verdict, and the clarifying questions worth answering — on the card,
// so the Lead sharpens the order before placing it. NOT parallel (shared globals).
func TestLiveCard_analyzeDraftRendersTheProducersStructuredRead(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	var gotPrompt string
	analyzeDraft = func(_ context.Context, _, prompt string) (string, error) {
		gotPrompt = prompt
		return `{"summary":"Clear goal, missing the retry budget.","ready":false,` +
			`"highlights":[{"start":0,"end":3,"note":"how many retries?","severity":"question"}],` +
			`"questions":["What is the maximum retry count?","Which errors count as transient?"]}`, nil
	}

	_, server := fundedAuthoringServer(t, "authz")

	tc := vt.NewClient(t, server, "/?key=authz")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authz"}).AnalyzeDraft).
		WithSignal("orderprompt", "Add retry logic to the uploader.").Fire(),
		"analyzing a draft is a calm, valid action")

	assert.Contains(t, gotPrompt, "Add retry logic to the uploader.",
		"the analysis harness runs on the authored draft")

	body := bodyOf(vt.NewClient(t, server, "/?key=authz").HTML())
	assert.Contains(t, body, "Clear goal, missing the retry budget.", "the producer's summary is shown")
	assert.Contains(t, body, `data-state="blocked"`, "the not-ready verdict is surfaced as a readiness hook")
	assert.Contains(t, body, "What is the maximum retry count?", "the clarifying questions are shown")
	assert.Contains(t, body, "Which errors count as transient?", "every clarifying question is shown")
}

// The analysis feeds Monaco: the analyzed draft + the flagged spans must be emitted
// as a machine-readable payload the editor decorates against, so the producer's
// highlights anchor on exactly the bytes it flagged. NOT parallel (shared globals).
func TestLiveCard_analyzeDraftEmitsTheHighlightPayloadForMonaco(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	analyzeDraft = func(_ context.Context, _, _ string) (string, error) {
		return `{"summary":"s","ready":true,"highlights":[{"start":0,"end":3,"note":"flagged","severity":"gap"}],"questions":[]}`, nil
	}

	_, server := fundedAuthoringServer(t, "authpay")

	tc := vt.NewClient(t, server, "/?key=authpay")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authpay"}).AnalyzeDraft).
		WithSignal("orderprompt", "Add a thing.").Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=authpay").HTML())
	assert.Contains(t, body, "authoring-analysis-data", "the Monaco authoring island emits its JSON payload")
	assert.Contains(t, body, `"flagged"`, "the highlight note is in the payload the editor decorates with")
	assert.Contains(t, body, `data-state="ready"`, "a ready draft surfaces the ready readiness hook")
}

// A producer run that fails or returns unreadable output must NOT break the card:
// it degrades to a calm "analysis unavailable" state, leaving the draft and the
// place control intact (the Lead can still place the order). NOT parallel (globals).
func TestLiveCard_analyzeDraftDegradesCalmlyOnUnreadableOutput(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	analyzeDraft = func(_ context.Context, _, _ string) (string, error) {
		return "I could not produce an analysis.", nil // no JSON object → ParseAnalysis errors
	}

	_, server := fundedAuthoringServer(t, "authbad")

	tc := vt.NewClient(t, server, "/?key=authbad")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authbad"}).AnalyzeDraft).
		WithSignal("orderprompt", "Do the task.").Fire(),
		"an unreadable analysis is still a calm 200, never a crash")

	body := bodyOf(vt.NewClient(t, server, "/?key=authbad").HTML())
	assert.Contains(t, body, `data-state="unavailable"`, "the card degrades to a calm analysis-unavailable state")
	assert.Contains(t, body, "/_action/PlaceOrder", "the place control survives a failed analysis")
}

// An empty draft is nothing to analyze: AnalyzeDraft must be a silent no-op, never
// spawning a producer over a blank prompt. NOT parallel (shared globals).
func TestLiveCard_analyzeDraftIsANoOpOnAnEmptyDraft(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	called := false
	analyzeDraft = func(_ context.Context, _, _ string) (string, error) {
		called = true
		return "", nil
	}

	_, server := fundedAuthoringServer(t, "authempty")

	tc := vt.NewClient(t, server, "/?key=authempty")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authempty"}).AnalyzeDraft).WithSignal("orderprompt", "   ").Fire())

	assert.False(t, called, "an empty draft never spawns a producer analysis")
}

// The card must render the analyze control (an action binding) alongside the compose
// control when there is bandwidth to fund authoring, so the Lead can invoke the
// assist. NOT parallel (shared globals).
func TestLiveCard_rendersTheAnalyzeControlWhenFunded(t *testing.T) {
	_, server := fundedAuthoringServer(t, "authctl")
	body := bodyOf(vt.NewClient(t, server, "/?key=authctl").HTML())
	assert.Contains(t, body, "/_action/AnalyzeDraft", "the card renders the analyze-draft action binding")
}
