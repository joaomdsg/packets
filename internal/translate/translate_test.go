package translate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/translate"
)

func TestTranslate_mapsAssistantTextToThinkingActivity(t *testing.T) {
	t.Parallel()
	raw := `{"type":"assistant","message":{"content":[{"type":"text","text":"considering the error path"}]}}`
	got, err := translate.Translate([]byte(raw))
	require.NoError(t, err)
	want := []translate.UIEvent{{Type: "activity.agent", Kind: "thinking", Detail: "considering the error path"}}
	assert.Equal(t, want, got)
}

func TestTranslate_mapsEditToolsToEditingActivityWithFilePath(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"Edit", "Write", "MultiEdit"} {
		raw := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"` + name + `","input":{"file_path":"src/auth.go"}}]}}`
		got, err := translate.Translate([]byte(raw))
		require.NoError(t, err, name)
		want := []translate.UIEvent{{Type: "activity.agent", Kind: "editing", Detail: "src/auth.go"}}
		assert.Equal(t, want, got, name)
	}
}

func TestTranslate_mapsBashToToolActivityWithCommandNotACheck(t *testing.T) {
	t.Parallel()
	raw := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}`
	got, err := translate.Translate([]byte(raw))
	require.NoError(t, err)
	want := []translate.UIEvent{{Type: "activity.agent", Kind: "tool", Detail: "go test ./..."}}
	assert.Equal(t, want, got)
	for _, e := range got {
		assert.Equal(t, "activity.agent", e.Type, "Bash must produce only activity, never a check")
	}
}

func TestTranslate_mapsUnknownToolToToolActivityWithItsName(t *testing.T) {
	t.Parallel()
	raw := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Grep","input":{"pattern":"foo"}}]}}`
	got, err := translate.Translate([]byte(raw))
	require.NoError(t, err)
	want := []translate.UIEvent{{Type: "activity.agent", Kind: "tool", Detail: "Grep"}}
	assert.Equal(t, want, got)
}

func TestTranslate_preservesOrderOfMultipleContentItems(t *testing.T) {
	t.Parallel()
	raw := `{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"editing now"},` +
		`{"type":"tool_use","name":"Edit","input":{"file_path":"a.go"}},` +
		`{"type":"tool_use","name":"Bash","input":{"command":"go build"}}` +
		`]}}`
	got, err := translate.Translate([]byte(raw))
	require.NoError(t, err)
	want := []translate.UIEvent{
		{Type: "activity.agent", Kind: "thinking", Detail: "editing now"},
		{Type: "activity.agent", Kind: "editing", Detail: "a.go"},
		{Type: "activity.agent", Kind: "tool", Detail: "go build"},
	}
	assert.Equal(t, want, got)
}

func TestTranslate_mapsResultToTurnEndedWithSubtype(t *testing.T) {
	t.Parallel()
	raw := `{"type":"result","subtype":"success"}`
	got, err := translate.Translate([]byte(raw))
	require.NoError(t, err)
	want := []translate.UIEvent{{Type: "turn.ended", Kind: "", Detail: "success"}}
	assert.Equal(t, want, got)
}

func TestTranslate_emitsNothingForToolResult(t *testing.T) {
	t.Parallel()
	raw := `{"type":"user","message":{"content":[{"type":"tool_result","content":"PASS"}]}}`
	got, err := translate.Translate([]byte(raw))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestTranslate_ignoresUnknownEventTypeWithoutError(t *testing.T) {
	t.Parallel()
	raw := `{"type":"system","subtype":"init","extra":123}`
	got, err := translate.Translate([]byte(raw))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestTranslate_returnsErrorOnMalformedJSON(t *testing.T) {
	t.Parallel()
	_, err := translate.Translate([]byte(`{"type":"assistant" this is not json`))
	assert.Error(t, err)
}

func TestTranslate_skipsUnknownContentTypeButOthersStillFlow(t *testing.T) {
	t.Parallel()
	raw := `{"type":"assistant","message":{"content":[` +
		`{"type":"image","source":{}},` +
		`{"type":"text","text":"hi"}` +
		`]}}`
	got, err := translate.Translate([]byte(raw))
	require.NoError(t, err)
	want := []translate.UIEvent{{Type: "activity.agent", Kind: "thinking", Detail: "hi"}}
	assert.Equal(t, want, got)
}

func TestTranslate_emitsNoEventsForEmptyContentArray(t *testing.T) {
	t.Parallel()
	got, err := translate.Translate([]byte(`{"type":"assistant","message":{"content":[]}}`))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestTranslate_degradesEditToolWithMissingFilePathToEmptyDetail(t *testing.T) {
	t.Parallel()
	got, err := translate.Translate([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{}}]}}`))
	require.NoError(t, err)
	want := []translate.UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}}
	assert.Equal(t, want, got)
}

func TestTranslate_signalsTurnEndForResultWithoutSubtype(t *testing.T) {
	t.Parallel()
	got, err := translate.Translate([]byte(`{"type":"result"}`))
	require.NoError(t, err)
	want := []translate.UIEvent{{Type: "turn.ended", Kind: "", Detail: ""}}
	assert.Equal(t, want, got)
}

func TestTranslate_degradesToolInputOfUnexpectedShapeNotErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
		want []translate.UIEvent
	}{
		{
			"input is a string",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":"oops"}]}}`,
			[]translate.UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}},
		},
		{
			"input is an array",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":[1,2]}]}}`,
			[]translate.UIEvent{{Type: "activity.agent", Kind: "tool", Detail: ""}},
		},
		{
			"input is a number",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":5}]}}`,
			[]translate.UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}},
		},
		{
			"input is null",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":null}]}}`,
			[]translate.UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}},
		},
		{
			"file_path is wrong type",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":123}}]}}`,
			[]translate.UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}},
		},
		{
			"good fields still parse",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}`,
			[]translate.UIEvent{{Type: "activity.agent", Kind: "tool", Detail: "go test ./..."}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := translate.Translate([]byte(tc.raw))
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
