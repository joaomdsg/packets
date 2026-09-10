package assist_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/assist"
)

const draft = "Add retry logic to the uploader so transient network errors recover."

// The assist's analysis arrives as one JSON object the agent prints amid its
// prose/stream chatter. ParseAnalysis must find and decode that object — tolerant
// of leading/trailing noise — into the structured highlights/questions/summary the
// authoring surface renders.
func TestParseAnalysis_extractsTheJSONBlockFromNoisyOutput(t *testing.T) {
	t.Parallel()
	raw := "Let me analyze the draft.\n" +
		`{"summary":"Clear goal, missing the retry budget.",` +
		`"ready":false,` +
		`"highlights":[{"start":4,"end":15,"note":"how many retries?","severity":"question"}],` +
		`"questions":["What is the maximum retry count?","Which errors count as transient?"]}` +
		"\nDone."

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)

	assert.Equal(t, "Clear goal, missing the retry budget.", got.Summary)
	assert.False(t, got.Ready, "the assist judged the draft not yet ready")
	require.Len(t, got.Highlights, 1)
	assert.Equal(t, "how many retries?", got.Highlights[0].Note)
	assert.Equal(t, "question", got.Highlights[0].Severity)
	assert.Equal(t, []assist.Question{
		{Q: "What is the maximum retry count?"},
		{Q: "Which errors count as transient?"},
	}, got.Questions, "bare-string questions decode into Question.Q")
}

// A assist that answers in the RICH question shape — each question carrying its own
// suggested answers and a multiselect flag — must decode into the structured form the
// authoring panel renders (a choice list per question, single- or multi-select). This
// is the whole point of the suggestions feature: the Lead picks, not free-types.
func TestParseAnalysis_decodesStructuredQuestionsWithSuggestionsAndMultiselect(t *testing.T) {
	t.Parallel()
	raw := `{"summary":"s","ready":false,"highlights":[],"questions":[` +
		`{"q":"Which datastore?","suggestions":["Postgres","SQLite"],"multiselect":false},` +
		`{"q":"Which error classes are transient?","suggestions":["timeouts","5xx","connection reset"],"multiselect":true}` +
		`]}`

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)
	require.Len(t, got.Questions, 2)
	assert.Equal(t, "Which datastore?", got.Questions[0].Q)
	assert.Equal(t, []string{"Postgres", "SQLite"}, got.Questions[0].Suggestions)
	assert.False(t, got.Questions[0].Multiselect, "a single-answer question is not multiselect")
	assert.True(t, got.Questions[1].Multiselect, "a question that can take several answers is multiselect")
}

// The panel renders AT MOST four suggestions per question (a calm, scannable choice
// set, not an overwhelming wall) — so a assist that returns more must be trimmed to
// the first four, never rendered in full.
func TestParseAnalysis_capsSuggestionsAtFourPerQuestion(t *testing.T) {
	t.Parallel()
	raw := `{"summary":"s","ready":false,"highlights":[],"questions":[` +
		`{"q":"Pick one","suggestions":["a","b","c","d","e","f"],"multiselect":false}` +
		`]}`

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)
	require.Len(t, got.Questions, 1)
	assert.Equal(t, []string{"a", "b", "c", "d"}, got.Questions[0].Suggestions,
		"only the first four suggestions are kept")
}

// A assist may mix shapes in one array — a terse bare-string question beside a
// rich object one — and may omit suggestions on a question entirely. The tolerant
// decode must handle the mix (so a half-structured reply isn't rejected), default a
// suggestion-less question to no choices, and trim padding off the question text so
// it renders clean.
func TestParseAnalysis_toleratesMixedAndSparseQuestionShapes(t *testing.T) {
	t.Parallel()
	raw := `{"summary":"s","ready":false,"highlights":[],"questions":[` +
		`"A bare string question",` +
		`{"q":"  Padded question?  "},` +
		`{"q":"Rich one","suggestions":["x"],"multiselect":true}` +
		`]}`

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)
	require.Len(t, got.Questions, 3)
	assert.Equal(t, "A bare string question", got.Questions[0].Q)
	assert.Nil(t, got.Questions[0].Suggestions, "a bare-string question carries no suggestions")
	assert.Equal(t, "Padded question?", got.Questions[1].Q, "the question text is trimmed of padding")
	assert.Nil(t, got.Questions[1].Suggestions, "an object without a suggestions key has no choices")
	assert.Equal(t, []string{"x"}, got.Questions[2].Suggestions)
	assert.True(t, got.Questions[2].Multiselect)
}

// A question with no text is nothing to ask — rendering an empty choice block would
// be a confusing dead control, so such questions are dropped entirely.
func TestParseAnalysis_dropsQuestionsWithEmptyText(t *testing.T) {
	t.Parallel()
	raw := `{"summary":"s","ready":false,"highlights":[],"questions":[` +
		`{"q":"","suggestions":["a"]},` +
		`{"q":"   ","suggestions":["b"]},` +
		`{"q":"Real question?","suggestions":[]}` +
		`]}`

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)
	require.Len(t, got.Questions, 1, "only the question with real text survives")
	assert.Equal(t, "Real question?", got.Questions[0].Q)
}

func TestParseAnalysis_acceptsAFencedCodeBlock(t *testing.T) {
	t.Parallel()
	raw := "Here is the analysis:\n```json\n" +
		`{"summary":"ok","ready":true,"highlights":[],"questions":[]}` +
		"\n```\n"

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)
	assert.Equal(t, "ok", got.Summary)
	assert.True(t, got.Ready)
}

func TestParseAnalysis_errorsWhenNoJSONObjectPresent(t *testing.T) {
	t.Parallel()
	_, err := assist.ParseAnalysis("I could not analyze this.", draft)
	assert.Error(t, err, "output with no JSON object is an error, not an empty analysis")
}

// A highlight whose range falls outside the draft (or is inverted) would mis-decorate
// the editor; ParseAnalysis drops such highlights rather than return a range Monaco
// can't anchor. The rest of the analysis survives.
func TestParseAnalysis_dropsOutOfBoundsHighlights(t *testing.T) {
	t.Parallel()
	raw := `{"summary":"s","ready":false,` +
		`"highlights":[` +
		`{"start":0,"end":3,"note":"keep"},` + // in bounds
		`{"start":5,"end":2,"note":"inverted"},` + // end<start
		`{"start":1,"end":9999,"note":"past end"}` + // beyond draft
		`],"questions":[]}`

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)
	require.Len(t, got.Highlights, 1, "only the in-bounds highlight survives")
	assert.Equal(t, "keep", got.Highlights[0].Note)
}

func TestParseAnalysis_clampsEndToDraftLengthBoundary(t *testing.T) {
	t.Parallel()
	n := len(draft)
	raw := `{"summary":"s","ready":true,` +
		`"highlights":[{"start":0,"end":` + itoa(n) + `,"note":"whole draft"}],"questions":[]}`

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)
	require.Len(t, got.Highlights, 1, "a highlight ending exactly at the draft length is valid")
	assert.Equal(t, n, got.Highlights[0].End)
}

// The analysis prompt is the contract's input side: it must carry the draft and
// instruct the agent to emit exactly the JSON shape ParseAnalysis decodes, so a
// round-trip (prompt → agent → parse) is coherent.
func TestAnalysisPrompt_carriesTheDraftAndTheOutputContract(t *testing.T) {
	t.Parallel()
	p := assist.AnalysisPrompt(draft)

	assert.Contains(t, p, draft, "the prompt includes the draft to analyze")
	for _, key := range []string{"summary", "ready", "highlights", "questions", "start", "end", "suggestions", "multiselect"} {
		assert.Containsf(t, p, key, "the prompt names the %q field of the JSON contract", key)
	}
}

// A prompt's own output is a valid round-trip: a JSON object matching the shape the
// prompt asks for parses back through ParseAnalysis without loss.
func TestAnalysisPrompt_outputRoundTripsThroughParse(t *testing.T) {
	t.Parallel()
	// Stand in for the agent's reply: the exact shape the prompt requests.
	reply := `{"summary":"ok","ready":true,"highlights":[{"start":0,"end":3,"note":"n","severity":"note"}],"questions":["q1"]}`
	got, err := assist.ParseAnalysis(reply, draft)
	require.NoError(t, err)
	assert.Equal(t, "ok", got.Summary)
	require.Len(t, got.Highlights, 1)
	assert.Equal(t, []assist.Question{{Q: "q1"}}, got.Questions)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// The rewrite prompt is the input side of the draft-update contract: it must carry
// the current draft AND every answer the Lead gave (the question, the picked
// suggestions, and any free-text note), and instruct the assist to return ONLY the
// rewritten draft — so the round-trip (answers → assist → new draft) is coherent
// and the reply is the draft text, not commentary.
func TestRewritePrompt_carriesDraftAndAnswers(t *testing.T) {
	t.Parallel()
	answers := []assist.Answer{
		{Q: "Which datastore?", Answers: []string{"Postgres"}, Note: "managed instance"},
		{Q: "Which errors are transient?", Answers: []string{"timeouts", "5xx"}},
	}
	p := assist.RewritePrompt(draft, answers)

	assert.Contains(t, p, draft, "the rewrite prompt includes the draft to revise")
	assert.Contains(t, p, "Which datastore?", "it carries each question")
	assert.Contains(t, p, "Postgres", "it carries the picked suggestions")
	assert.Contains(t, p, "timeouts", "it carries multi-select picks")
	assert.Contains(t, p, "5xx")
	assert.Contains(t, p, "managed instance", "it carries the free-text note")
	assert.Contains(t, strings.ToLower(p), "only", "it tells the assist to return ONLY the rewritten draft")
}

// The assist is asked for the bare draft, but models often wrap it in a ```fenced
// block or add a stray prefix. ParseRewrite must return the clean draft text so it
// can drop straight into the editor — fences stripped, whitespace trimmed.
func TestParseRewrite_returnsTheCleanDraftText(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Add retry logic with a budget of 5.",
		assist.ParseRewrite("  Add retry logic with a budget of 5.\n\n"),
		"plain output is just trimmed")
	assert.Equal(t, "Line one\nLine two",
		assist.ParseRewrite("```\nLine one\nLine two\n```"),
		"a bare fenced block unwraps to its contents")
	assert.Equal(t, "# Heading\n\nbody",
		assist.ParseRewrite("```markdown\n# Heading\n\nbody\n```\n"),
		"a language-tagged fence unwraps, dropping the info string")
}

// A draft is markdown and may legitimately contain its OWN fenced code block. The
// unwrap must only strip an OUTER fence that wraps the whole reply — never mangle an
// inner code block that is part of the draft.
func TestParseRewrite_preservesInnerCodeBlocks(t *testing.T) {
	t.Parallel()
	draftWithCode := "Do this:\n\n```go\nfmt.Println(\"hi\")\n```\n\nThen ship it."
	assert.Equal(t, draftWithCode, assist.ParseRewrite(draftWithCode),
		"an unfenced reply that contains a code block is returned verbatim (only trimmed)")

	// The reply IS wrapped in an outer fence AND contains an inner code block: only the
	// outer fence is stripped, the inner block survives intact (outer-only semantics).
	assert.Equal(t, "Code:\n\n```js\nalert('x')\n```",
		assist.ParseRewrite("```\nCode:\n\n```js\nalert('x')\n```\n```"),
		"only the outer wrapping fence is stripped; the inner block is preserved")
}

// An empty or whitespace-only reply is an empty draft — never a panic, never stray
// whitespace dropped into the editor.
func TestParseRewrite_emptyReplyIsEmptyDraft(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", assist.ParseRewrite("   \n\n  "), "whitespace-only output trims to empty")
	assert.Equal(t, "", assist.ParseRewrite(""), "empty output stays empty")
}

// The decode must be TOLERANT of a malformed question element (a stray number, bool,
// or null the assist emitted by mistake): such an element is dropped, never failing
// the WHOLE analysis — a half-bad questions array still yields the good questions.
func TestParseAnalysis_skipsMalformedQuestionElements(t *testing.T) {
	t.Parallel()
	raw := `{"summary":"s","ready":false,"highlights":[],"questions":[` +
		`"Good one",` +
		`42,` +
		`null,` +
		`true,` +
		`[1,2],` +
		`{"q":"Also good","suggestions":["x"]}` +
		`]}`

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err, "a stray scalar/array element must not fail the whole analysis")
	require.Len(t, got.Questions, 2, "only the two real questions survive; the malformed elements are dropped")
	assert.Equal(t, "Good one", got.Questions[0].Q)
	assert.Equal(t, "Also good", got.Questions[1].Q)
}

// Suggestion text must be trimmed (and blank suggestions dropped) — a padded or empty
// suggestion would render a ragged or empty choice chip, and the trimmed value is what
// gets echoed back into the rewrite, so it must be clean.
func TestParseAnalysis_trimsAndDropsBlankSuggestions(t *testing.T) {
	t.Parallel()
	raw := `{"summary":"s","ready":false,"highlights":[],"questions":[` +
		`{"q":"Pick","suggestions":["  Postgres  ","","   ","SQLite"]}` +
		`]}`

	got, err := assist.ParseAnalysis(raw, draft)
	require.NoError(t, err)
	require.Len(t, got.Questions, 1)
	assert.Equal(t, []string{"Postgres", "SQLite"}, got.Questions[0].Suggestions,
		"suggestions are trimmed and blank ones dropped")
}

// Routing a review comment back into the harness (DESIGN §12.3): an anchored comment
// {file, line, code, text} composes a turn that tells the harness exactly what to
// address, quotes the line in question, and reminds it the code is in the working
// tree — so the agent re-edits in place. This is the keystone of the review-thread
// loop (the comment→harness round-trip).
func TestReviewTurnPrompt_composesTheAnchoredCommentTurn(t *testing.T) {
	t.Parallel()
	p := assist.ReviewTurnPrompt("src/auth.go", 42, "return validate(tok)", "this ignores the expired-token case — handle it.")

	assert.Contains(t, p, "src/auth.go", "the turn names the commented file")
	assert.Contains(t, p, "42", "the turn names the commented line")
	assert.Contains(t, p, "return validate(tok)", "the turn quotes the line under review")
	assert.Contains(t, p, "this ignores the expired-token case", "the turn carries the maintainer's comment")
	assert.Contains(t, strings.ToLower(p), "address this", "the turn instructs the harness to address it specifically")
	assert.Contains(t, strings.ToLower(p), "working tree", "the turn reminds the harness the code is in the working tree")
}

// A comment with no quoted source line (the orchestrator couldn't read it) must still
// compose a coherent turn — the file:line and the instruction carry it, no dangling
// empty quote line.
func TestReviewTurnPrompt_handlesAMissingSourceLine(t *testing.T) {
	t.Parallel()
	p := assist.ReviewTurnPrompt("a.go", 7, "", "rename this")

	assert.Contains(t, p, "a.go", "the file is still named")
	assert.Contains(t, p, "rename this", "the comment is still carried")
	assert.NotContains(t, p, "> 7  \n", "no dangling empty quote line when there is no source")
}
