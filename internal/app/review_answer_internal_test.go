package app

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/mutation"
)

// Answering a review question: the reviewer submits a test for a surviving-mutant
// line; the server re-runs the oracle with it injected, and if the mutant DIES the
// question disappears. This is the server contract behind the editable flow — and a
// hard FIREWALL: it updates only the off-economy findings cache, never the ledger
// (answering mints nothing; the vanishing question is the whole reward). NOT
// parallel (shared liveReg + the re-run seam).
func TestReviewCard_aKillingAnswerRemovesTheQuestionFromTheCache(t *testing.T) {
	resetConsumersForTest()
	restore := rerunWithOverlay
	t.Cleanup(func() { rerunWithOverlay = restore })
	var sawOverlay bool
	rerunWithOverlay = func(_ context.Context, _, _, file string, line int, _ []string, overlay map[string]string) ([]mutation.Finding, error) {
		// the reviewer's test is injected, scoped to the answered line
		if file == "main.go" && line == 6 && len(overlay) == 1 {
			for _, content := range overlay {
				if content != "" {
					sawOverlay = true
				}
			}
		}
		return nil, nil // the test killed the mutant — no surviving findings remain
	}

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived, Message: "mutated >= to >"}})
	balBefore, _ := log.Balance()

	tc := vt.NewClient(t, server, "/review")
	code := tc.Action((&ReviewCard{}).AnswerQuestion).
		WithSignal("answerfile", "main.go").
		WithSignal("answerline", "6").
		WithSignal("answertest", "package main\n\nimport \"testing\"\nfunc TestBoundary(t *testing.T){}\n").
		Fire()
	require.Equal(t, 200, code)

	require.True(t, sawOverlay, "the reviewer's test is injected into the re-run, scoped to the answered line")
	require.Empty(t, lookupLiveEntry(defaultSessionKey).openFindings(), "a killing answer removes the question from the cache")
	balAfter, _ := log.Balance()
	require.Equal(t, balBefore, balAfter, "FIREWALL: answering touches no balance — diagnostic only, off the economy")
}

// A test that does NOT kill the mutant leaves the question OPEN — the re-run still
// reports the survivor, so the cache (and the surface) keep it. Honest: you only
// clear a question by actually constraining the line. NOT parallel.
func TestReviewCard_aWeakAnswerLeavesTheQuestionOpen(t *testing.T) {
	resetConsumersForTest()
	restore := rerunWithOverlay
	t.Cleanup(func() { rerunWithOverlay = restore })
	still := []mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived, Message: "mutated >= to >"}}
	rerunWithOverlay = func(_ context.Context, _, _, _ string, _ int, _ []string, _ map[string]string) ([]mutation.Finding, error) {
		return still, nil // the test didn't kill it — the survivor remains
	}

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings(still)

	tc := vt.NewClient(t, server, "/review")
	code := tc.Action((&ReviewCard{}).AnswerQuestion).
		WithSignal("answerfile", "main.go").
		WithSignal("answerline", "6").
		WithSignal("answertest", "package main\n\nimport \"testing\"\nfunc TestWeak(t *testing.T){}\n").
		Fire()
	require.Equal(t, 200, code)
	require.Len(t, lookupLiveEntry(defaultSessionKey).openFindings(), 1, "a non-killing answer leaves the question open")
}

// The question is inert without a way to answer it from the surface: /review must
// render an answer form wired to AnswerQuestion — a test textarea bound to the
// answer signal + a submit that sets the answered file/line and fires the action.
// Without it the reviewer can read the question but never close it from the UI. NOT
// parallel (shared liveReg).
func TestReviewCard_rendersAnAnswerFormWiredToAnswerQuestion(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived, Message: "mutated >= to >"}})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, "review-answer", "an answer form is offered for the open question")
	require.Contains(t, body, `data-bind="answertest"`, "the test textarea is bound to the answer signal")
	require.Contains(t, body, "/_action/AnswerQuestion", "the submit fires the AnswerQuestion action")
	require.Contains(t, body, "answerfile", "the answered file is set before the post")
	require.Contains(t, body, "answerline", "the answered line is set before the post")
}

// With no open questions there is nothing to answer, so the answer form is omitted —
// the surface stays calm. NOT parallel.
func TestReviewCard_omitsTheAnswerFormWhenNoOpenQuestions(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.NotContains(t, body, "review-answer", "no answer form when there is nothing to answer")
}

// Submitting an answer re-runs the oracle (seconds of real work), so the form must
// show a calm in-flight "running…" affordance rather than dead-air: the submit
// carries a datastar indicator signal, and a sibling reveals a "re-running the
// oracle…" line while the request is in flight. Declarative (datastar), so it
// survives the surface's re-render. NOT parallel (shared liveReg).
func TestReviewCard_showsACalmRunningAffordanceWhileTheAnswerReRuns(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived, Message: "mutated >= to >"}})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, `data-indicator="answering"`, "the submit marks the request in-flight via a datastar indicator signal")
	require.Contains(t, body, `data-show="$answering"`, "a sibling reveals the running affordance only while in flight")
	require.Contains(t, body, "re-running the oracle", "the calm running message, not dead-air")
}

// The flaky-truth fence: a transient re-run failure (oracle timeout, git error,
// an Undetermined run) must NEVER clear the question — it stays open, retryable, so
// a flake can't silently mark a line "answered" that isn't. NOT parallel.
func TestReviewCard_aTransientRerunErrorLeavesTheQuestionOpen(t *testing.T) {
	resetConsumersForTest()
	restore := rerunWithOverlay
	t.Cleanup(func() { rerunWithOverlay = restore })
	rerunWithOverlay = func(_ context.Context, _, _, _ string, _ int, _ []string, _ map[string]string) ([]mutation.Finding, error) {
		return nil, errors.New("oracle re-run timed out")
	}

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived, Message: "x"}})

	tc := vt.NewClient(t, server, "/review")
	require.Equal(t, 200, tc.Action((&ReviewCard{}).AnswerQuestion).
		WithSignal("answerfile", "main.go").WithSignal("answerline", "6").
		WithSignal("answertest", "package main\nfunc x(){}\n").Fire())
	require.Len(t, lookupLiveEntry(defaultSessionKey).openFindings(), 1,
		"a transient re-run failure never clears the question — it stays open, retryable")
}

// Answering must update the SESSION the reviewer is on, never bleed across sessions:
// a killing answer on /review?key=other clears OTHER's question and leaves the
// default session's untouched. This also proves the key threads through the action
// POST (not silently falling back to default). NOT parallel.
func TestReviewCard_answersTheKeyedSessionNotTheDefault(t *testing.T) {
	resetConsumersForTest()
	restore := rerunWithOverlay
	t.Cleanup(func() { rerunWithOverlay = restore })
	rerunWithOverlay = func(_ context.Context, _, _, _ string, _ int, _ []string, _ map[string]string) ([]mutation.Finding, error) {
		return nil, nil // killing
	}

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	other, err := AddSession("other", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = other.Close() })

	finding := []mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived, Message: "x"}}
	lookupLiveEntry("other").setFindings(finding)
	lookupLiveEntry(defaultSessionKey).setFindings(finding)

	tc := vt.NewClient(t, server, "/review?key=other")
	require.Equal(t, 200, tc.Action((&ReviewCard{}).AnswerQuestion).
		WithSignal("answerfile", "main.go").WithSignal("answerline", "6").
		WithSignal("answertest", "package main\nfunc x(){}\n").Fire())

	require.Empty(t, lookupLiveEntry("other").openFindings(), "the keyed session's question is answered")
	require.Len(t, lookupLiveEntry(defaultSessionKey).openFindings(), 1, "the default session is untouched — no cross-session bleed")
}
