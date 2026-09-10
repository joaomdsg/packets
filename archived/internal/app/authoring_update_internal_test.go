package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"
)

// The whole point of the answer form: the assist rewrites the draft to fold in the
// Lead's answers, and the new draft is pushed back to the editor. UpdateDraft must run
// the rewrite over the draft + answers and emit the rewritten text in the editor's
// rewrite payload, so Monaco swaps to it. NOT parallel (shared globals).
func TestUpdateDraft_rewritesDraftAndPushesTheNewTextToTheEditor(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	var gotPrompt string
	analyzeDraft = func(_ context.Context, _, prompt, _ string) (string, error) {
		gotPrompt = prompt
		return "Add retry logic with a budget of 5 attempts, using Postgres.", nil
	}

	_, server := fundedAuthoringServer(t, "authupd")
	tc := vt.NewClient(t, server, "/?key=authupd")
	answers := `[{"Q":"Which datastore?","Answers":["Postgres"],"Note":"managed"}]`
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupd"}).UpdateDraft).
		WithSignal("draft", "Add retry logic.").
		WithSignal("draftanswers", answers).Fire())

	// The rewrite prompt carried the draft and the Lead's answer.
	assert.Contains(t, gotPrompt, "Add retry logic.", "the rewrite ran over the current draft")
	assert.Contains(t, gotPrompt, "Postgres", "the rewrite carried the Lead's picked answer")

	body := bodyOf(vt.NewClient(t, server, "/?key=authupd").HTML())
	assert.Contains(t, body, "authoring-rewrite-data", "the editor's rewrite payload is emitted")
	assert.Contains(t, body, "budget of 5 attempts", "the new draft text is pushed to the editor")
}

// Replace-only: once the draft is rewritten, the OLD questions are stale (they were
// answered), so the analysis panel must clear — the Lead re-analyzes the new draft
// when ready, never answers questions about a draft that no longer exists. NOT
// parallel (shared globals).
func TestUpdateDraft_clearsTheStaleAnalysisAfterRewrite(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	calls := 0
	analyzeDraft = func(_ context.Context, _, _, _ string) (string, error) {
		calls++
		if calls == 1 { // the analyze: produce a question
			return `{"summary":"s","ready":false,"highlights":[],"questions":[{"q":"Which datastore?","suggestions":["Postgres"]}]}`, nil
		}
		return "A clean rewritten draft.", nil // the rewrite
	}

	_, server := fundedAuthoringServer(t, "authupdclear")
	tc := vt.NewClient(t, server, "/?key=authupdclear")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdclear"}).AnalyzeDraft).
		WithSignal("draft", "Add retry logic.").Fire())
	// The question is present before the update.
	require.Contains(t, bodyOf(vt.NewClient(t, server, "/?key=authupdclear").HTML()), "Which datastore?")

	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdclear"}).UpdateDraft).
		WithSignal("draft", "Add retry logic.").
		WithSignal("draftanswers", `[{"Q":"Which datastore?","Answers":["Postgres"]}]`).Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=authupdclear").HTML())
	assert.NotContains(t, body, "Which datastore?", "the stale question is cleared after the rewrite")
	// Assert the rendered button markup, not the bare class — the editor JS literally
	// contains the ".analysis__update" delegated-click selector on every page.
	assert.NotContains(t, body, `class="pk-btn analysis__update"`, "the answer form is gone with the cleared analysis")
}

// No answers means nothing to fold in — UpdateDraft must be a silent no-op, never
// spawning a assist to rewrite a draft against an empty answer set. NOT parallel.
func TestUpdateDraft_isANoOpWhenThereAreNoAnswers(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	called := false
	analyzeDraft = func(_ context.Context, _, _, _ string) (string, error) {
		called = true
		return "x", nil
	}

	_, server := fundedAuthoringServer(t, "authupdnoans")
	tc := vt.NewClient(t, server, "/?key=authupdnoans")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdnoans"}).UpdateDraft).
		WithSignal("draft", "Add retry logic.").
		WithSignal("draftanswers", `[]`).Fire())

	assert.False(t, called, "an empty answer set never spawns a rewrite")
}

// An empty draft is nothing to rewrite — a silent no-op, like AnalyzeDraft. NOT
// parallel (shared globals).
func TestUpdateDraft_isANoOpOnAnEmptyDraft(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	called := false
	analyzeDraft = func(_ context.Context, _, _, _ string) (string, error) {
		called = true
		return "x", nil
	}

	_, server := fundedAuthoringServer(t, "authupdempty")
	tc := vt.NewClient(t, server, "/?key=authupdempty")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdempty"}).UpdateDraft).
		WithSignal("draft", "   ").
		WithSignal("draftanswers", `[{"Q":"q","Answers":["a"]}]`).Fire())

	assert.False(t, called, "an empty draft never spawns a rewrite")
}

// A failed rewrite must NOT wipe the editor nor lose the questions — the Lead can
// retry. UpdateDraft degrades calmly: no rewrite payload pushed, the analysis (and its
// answer form) preserved. NOT parallel (shared globals).
func TestUpdateDraft_failureKeepsTheDraftAndQuestions(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	calls := 0
	analyzeDraft = func(_ context.Context, _, _, _ string) (string, error) {
		calls++
		if calls == 1 {
			return `{"summary":"s","ready":false,"highlights":[],"questions":[{"q":"Keep me?","suggestions":["yes"]}]}`, nil
		}
		return "", assert.AnError // the rewrite fails
	}

	_, server := fundedAuthoringServer(t, "authupdfail")
	tc := vt.NewClient(t, server, "/?key=authupdfail")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdfail"}).AnalyzeDraft).
		WithSignal("draft", "Add retry logic.").Fire())

	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdfail"}).UpdateDraft).
		WithSignal("draft", "Add retry logic.").
		WithSignal("draftanswers", `[{"Q":"Keep me?","Answers":["yes"]}]`).Fire(),
		"a failed rewrite is still a calm 200, never a crash")

	body := bodyOf(vt.NewClient(t, server, "/?key=authupdfail").HTML())
	assert.Contains(t, body, "Keep me?", "a failed rewrite preserves the questions so the Lead can retry")
}

// A successful run that yields EMPTY draft text is a failure in disguise — pushing it
// would wipe the editor. UpdateDraft must treat an empty rewrite like a failure: keep
// the draft and the questions. NOT parallel (shared globals).
func TestUpdateDraft_emptyRewriteIsTreatedAsFailure(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	calls := 0
	analyzeDraft = func(_ context.Context, _, _, _ string) (string, error) {
		calls++
		if calls == 1 {
			return `{"summary":"s","ready":false,"highlights":[],"questions":[{"q":"Survive?","suggestions":["yes"]}]}`, nil
		}
		return "   \n", nil // a "successful" run that is effectively empty
	}

	_, server := fundedAuthoringServer(t, "authupdblank")
	tc := vt.NewClient(t, server, "/?key=authupdblank")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdblank"}).AnalyzeDraft).
		WithSignal("draft", "Add retry logic.").Fire())

	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdblank"}).UpdateDraft).
		WithSignal("draft", "Add retry logic.").
		WithSignal("draftanswers", `[{"Q":"Survive?","Answers":["yes"]}]`).Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=authupdblank").HTML())
	assert.Contains(t, body, "Survive?", "an empty rewrite is treated as failure — the questions survive")
}

// Malformed answer JSON must be a silent no-op, never a crash — a corrupted/partial
// signal can't take down the card. NOT parallel (shared globals).
func TestUpdateDraft_malformedAnswersIsANoOp(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	called := false
	analyzeDraft = func(_ context.Context, _, _, _ string) (string, error) {
		called = true
		return "x", nil
	}

	_, server := fundedAuthoringServer(t, "authupdbadjson")
	tc := vt.NewClient(t, server, "/?key=authupdbadjson")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authupdbadjson"}).UpdateDraft).
		WithSignal("draft", "Add retry logic.").
		WithSignal("draftanswers", `[not valid json`).Fire(),
		"malformed answers is a calm 200, never a crash")

	assert.False(t, called, "malformed answers never spawn a rewrite")
}

// A pending rewrite must not outlive a fresh analysis: once the Lead re-analyzes (a
// new read of the current draft), any stashed rewrite from a prior update is stale and
// must be cleared, so it can never be re-pushed into the editor later. NOT parallel.
func TestAnalyzeDraft_clearsAStalePendingRewrite(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	calls := 0
	analyzeDraft = func(_ context.Context, _, _, _ string) (string, error) {
		calls++
		if calls == 1 {
			return "A REWRITTEN draft from the update.", nil // the UpdateDraft rewrite
		}
		return `{"summary":"s","ready":true,"highlights":[],"questions":[]}`, nil // the re-analyze
	}

	_, server := fundedAuthoringServer(t, "authrwclear")
	tc := vt.NewClient(t, server, "/?key=authrwclear")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authrwclear"}).UpdateDraft).
		WithSignal("draft", "Add retry logic.").
		WithSignal("draftanswers", `[{"Q":"q","Answers":["a"]}]`).Fire())
	// The rewrite is pending in the payload after the update.
	require.Contains(t, bodyOf(vt.NewClient(t, server, "/?key=authrwclear").HTML()), "A REWRITTEN draft from the update.")

	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authrwclear"}).AnalyzeDraft).
		WithSignal("draft", "A REWRITTEN draft from the update.").Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=authrwclear").HTML())
	assert.NotContains(t, body, "A REWRITTEN draft from the update.",
		"a fresh analysis clears the stale pending rewrite so it can't be re-pushed")
}
