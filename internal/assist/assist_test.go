package assist_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/assist"
)

const draft = "Add retry logic to the uploader so transient network errors recover."

// The producer's analysis arrives as one JSON object the agent prints amid its
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
	assert.False(t, got.Ready, "the producer judged the draft not yet ready")
	require.Len(t, got.Highlights, 1)
	assert.Equal(t, "how many retries?", got.Highlights[0].Note)
	assert.Equal(t, "question", got.Highlights[0].Severity)
	assert.Equal(t, []string{"What is the maximum retry count?", "Which errors count as transient?"}, got.Questions)
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
	for _, key := range []string{"summary", "ready", "highlights", "questions", "start", "end"} {
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
	assert.Equal(t, []string{"q1"}, got.Questions)
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
