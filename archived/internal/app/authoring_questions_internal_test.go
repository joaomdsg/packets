package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/assist"
)

// panelFor renders the analysis panel for a set of questions, so the tests can assert
// the per-question answer form the assist's questions drive.
func panelFor(t *testing.T, qs []assist.Question) string {
	t.Helper()
	da := &draftAnalysis{Draft: "d", Result: &assist.Analysis{Summary: "s", Questions: qs}}
	return renderHTML(t, renderAnalysisPanel(da))
}

// A single-answer question must render its suggestions as RADIO inputs grouped under
// one name — so picking one suggestion deselects the others (the Lead gives exactly
// one answer). This is the difference between "pick one" and "pick any".
func TestAnalysisPanel_singleSelectQuestionRendersGroupedRadios(t *testing.T) {
	out := panelFor(t, []assist.Question{
		{Q: "Which datastore?", Suggestions: []string{"Postgres", "SQLite"}, Multiselect: false},
	})
	assert.Contains(t, out, `type="radio"`, "a single-answer question uses radios")
	assert.NotContains(t, out, `type="checkbox"`, "a single-answer question is not multiselect")
	assert.Contains(t, out, "Postgres", "the suggestions are rendered as choices")
	assert.Contains(t, out, "SQLite")
	// Both radios share one group name so they are mutually exclusive.
	assert.GreaterOrEqual(t, strings.Count(out, `name="qans-0"`), 2,
		"the question's radios share one group name (mutually exclusive)")
}

// A multiselect question must render CHECKBOXES, so the Lead can pick several answers
// that legitimately apply at once (e.g. multiple transient error classes).
func TestAnalysisPanel_multiSelectQuestionRendersCheckboxes(t *testing.T) {
	out := panelFor(t, []assist.Question{
		{Q: "Which errors are transient?", Suggestions: []string{"timeouts", "5xx", "reset"}, Multiselect: true},
	})
	assert.Contains(t, out, `type="checkbox"`, "a multiselect question uses checkboxes")
	assert.NotContains(t, out, `type="radio"`, "a multiselect question is not single-select")
	for _, s := range []string{"timeouts", "5xx", "reset"} {
		assert.Contains(t, out, s, "every suggestion is rendered as a checkbox choice")
	}
}

// Every question must offer a free-text note input — so the Lead can add context or
// give a different answer entirely, not only pick from the suggestions.
func TestAnalysisPanel_everyQuestionHasANotesInput(t *testing.T) {
	out := panelFor(t, []assist.Question{
		{Q: "With suggestions?", Suggestions: []string{"a"}, Multiselect: false},
		{Q: "Without any suggestions?"}, // no suggestions at all
	})
	assert.GreaterOrEqual(t, strings.Count(out, `class="analysis__note"`), 2,
		"each question (even one with no suggestions) carries a notes/free-answer input")
	assert.Contains(t, out, "Without any suggestions?",
		"a question with no suggestions still renders (answerable via the note)")
}

// The panel must offer ONE 'Update intent' control at the end (not per question), so
// the Lead answers all the questions and then updates the intent in a single action.
func TestAnalysisPanel_offersASingleUpdateDraftControl(t *testing.T) {
	out := panelFor(t, []assist.Question{
		{Q: "Q1?", Suggestions: []string{"a"}},
		{Q: "Q2?", Suggestions: []string{"b"}},
	})
	assert.Contains(t, out, "analysis__update", "the panel offers an Update-intent control")
	assert.Contains(t, out, "Update intent", "labelled so the Lead knows it applies their answers")
	assert.Equal(t, 1, strings.Count(out, "analysis__update"),
		"exactly one update control, at the end — not one per question")
}

// Each question's inputs must be scoped to that question (distinct group names), so a
// radio pick for Q1 never clobbers Q2's pick — without this the answers would bleed
// across questions.
func TestAnalysisPanel_questionInputsAreScopedPerQuestion(t *testing.T) {
	out := panelFor(t, []assist.Question{
		{Q: "Q1?", Suggestions: []string{"a", "b"}},
		{Q: "Q2?", Suggestions: []string{"c", "d"}},
	})
	assert.Contains(t, out, `name="qans-0"`, "the first question's choices group under qans-0")
	assert.Contains(t, out, `name="qans-1"`, "the second question's choices group under qans-1")
}

// With no questions there is nothing to answer — the panel must NOT render an update
// control (a dead button that would rewrite a draft against zero answers).
func TestAnalysisPanel_noUpdateControlWhenThereAreNoQuestions(t *testing.T) {
	out := panelFor(t, nil)
	assert.NotContains(t, out, "analysis__update", "no questions → no update control")
}
