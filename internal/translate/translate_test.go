package translate

import (
	"testing"
)

// The reviewer must SEE the agent working between turns (DESIGN risk #3: a
// 3-minute reply must feel alive). Assistant prose becomes a "thinking"
// activity carrying the text.
func TestAssistantTextBecomesThinkingActivity(t *testing.T) {
	raw := `{"type":"assistant","message":{"content":[{"type":"text","text":"considering the error path"}]}}`
	got, err := Translate([]byte(raw))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := []UIEvent{{Type: "activity.agent", Kind: "thinking", Detail: "considering the error path"}}
	assertEvents(t, got, want)
}

// An edit tool tells the reviewer WHICH file is changing, live — the editing
// activity must carry the file path for Edit/Write/MultiEdit alike.
func TestEditToolsBecomeEditingActivityWithFilePath(t *testing.T) {
	for _, name := range []string{"Edit", "Write", "MultiEdit"} {
		raw := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"` + name + `","input":{"file_path":"src/auth.go"}}]}}`
		got, err := Translate([]byte(raw))
		if err != nil {
			t.Fatalf("%s: Translate: %v", name, err)
		}
		want := []UIEvent{{Type: "activity.agent", Kind: "editing", Detail: "src/auth.go"}}
		assertEvents(t, got, want)
	}
}

// A Bash command surfaces as a "tool" activity carrying the command — but NOT
// as a check verdict: per RISKS.md iter-9, the translator must never infer
// pass/fail from agent Bash. It is activity only.
func TestBashBecomesToolActivityWithCommandNotACheck(t *testing.T) {
	raw := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}`
	got, err := Translate([]byte(raw))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := []UIEvent{{Type: "activity.agent", Kind: "tool", Detail: "go test ./..."}}
	assertEvents(t, got, want)
	for _, e := range got {
		if e.Type != "activity.agent" {
			t.Errorf("Bash must produce only activity, never a check; got %+v", e)
		}
	}
}

// Tools the translator doesn't special-case still surface as activity, labelled
// by tool name, so the reviewer isn't blind to them.
func TestUnknownToolBecomesToolActivityWithItsName(t *testing.T) {
	raw := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Grep","input":{"pattern":"foo"}}]}}`
	got, err := Translate([]byte(raw))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := []UIEvent{{Type: "activity.agent", Kind: "tool", Detail: "Grep"}}
	assertEvents(t, got, want)
}

// One assistant message may carry several content items; their activity events
// must come out in the same order the agent produced them.
func TestMultipleContentItemsPreserveOrder(t *testing.T) {
	raw := `{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"editing now"},` +
		`{"type":"tool_use","name":"Edit","input":{"file_path":"a.go"}},` +
		`{"type":"tool_use","name":"Bash","input":{"command":"go build"}}` +
		`]}}`
	got, err := Translate([]byte(raw))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := []UIEvent{
		{Type: "activity.agent", Kind: "thinking", Detail: "editing now"},
		{Type: "activity.agent", Kind: "editing", Detail: "a.go"},
		{Type: "activity.agent", Kind: "tool", Detail: "go build"},
	}
	assertEvents(t, got, want)
}

// The turn boundary is the signal the orchestrator settles on. A result event
// becomes turn.ended carrying the subtype, so the orchestrator knows whether
// the turn succeeded before committing a revision.
func TestResultBecomesTurnEndedWithSubtype(t *testing.T) {
	raw := `{"type":"result","subtype":"success"}`
	got, err := Translate([]byte(raw))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := []UIEvent{{Type: "turn.ended", Kind: "", Detail: "success"}}
	assertEvents(t, got, want)
}

// Tool results (the harness echoing tool output back) are not part of this
// slice — turning them into checks is deferred (iter-9). They must produce
// nothing rather than be misread.
func TestToolResultEmitsNothing(t *testing.T) {
	raw := `{"type":"user","message":{"content":[{"type":"tool_result","content":"PASS"}]}}`
	got, err := Translate([]byte(raw))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("tool_result must produce no UI events in this slice, got %+v", got)
	}
}

// New harness event types must not break the translator — it ignores what it
// doesn't recognize rather than erroring, so the stream keeps flowing.
func TestUnknownEventTypeIsIgnoredWithoutError(t *testing.T) {
	raw := `{"type":"system","subtype":"init","extra":123}`
	got, err := Translate([]byte(raw))
	if err != nil {
		t.Fatalf("unknown event types must not error, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("unknown event type must produce no events, got %+v", got)
	}
}

// Malformed input is a real error the caller must see — never silently
// swallowed as "no activity".
func TestMalformedJSONReturnsError(t *testing.T) {
	if _, err := Translate([]byte(`{"type":"assistant" this is not json`)); err == nil {
		t.Fatal("malformed JSON must return an error, got nil")
	}
}

// New content-block types (the harness can emit e.g. "image" or thinking
// blocks) must be skipped, not break the stream — recognized items in the same
// message still flow. Forward-compatibility at the content level, mirroring the
// unknown-event-type behavior.
func TestUnknownContentTypeIsSkippedButOthersStillFlow(t *testing.T) {
	raw := `{"type":"assistant","message":{"content":[` +
		`{"type":"image","source":{}},` +
		`{"type":"text","text":"hi"}` +
		`]}}`
	got, err := Translate([]byte(raw))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := []UIEvent{{Type: "activity.agent", Kind: "thinking", Detail: "hi"}}
	assertEvents(t, got, want)
}

// An assistant message with no content items is benign (e.g. a pause) — it
// produces no activity rather than an error.
func TestEmptyContentArrayProducesNoEvents(t *testing.T) {
	got, err := Translate([]byte(`{"type":"assistant","message":{"content":[]}}`))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty content must produce no events, got %+v", got)
	}
}

// A tool_use whose input lacks the expected field must NOT panic or error — it
// degrades to an activity with an empty Detail, so a malformed/partial event
// still flows as best-effort activity.
func TestEditToolWithMissingFilePathDegradesToEmptyDetail(t *testing.T) {
	got, err := Translate([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{}}]}}`))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := []UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}}
	assertEvents(t, got, want)
}

// A result without a subtype must still signal turn end (with empty Detail),
// not error — the orchestrator still needs to know the turn boundary.
func TestResultWithoutSubtypeStillSignalsTurnEnd(t *testing.T) {
	got, err := Translate([]byte(`{"type":"result"}`))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := []UIEvent{{Type: "turn.ended", Kind: "", Detail: ""}}
	assertEvents(t, got, want)
}

// A tool_use whose "input" is valid JSON but not the expected object shape
// (e.g. a string, an array, or a field of the wrong type) must NOT error the
// whole event — it degrades to an activity with empty Detail, mirroring the
// missing-field case. Erroring here would kill the live stream on a merely-odd
// event, contradicting the package's forward-compatible intent.
func TestToolInputOfUnexpectedShapeDegradesNotErrors(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []UIEvent
	}{
		{
			"input is a string",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":"oops"}]}}`,
			[]UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}},
		},
		{
			"input is an array",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":[1,2]}]}}`,
			[]UIEvent{{Type: "activity.agent", Kind: "tool", Detail: ""}},
		},
		{
			"input is a number",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":5}]}}`,
			[]UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}},
		},
		{
			"input is null",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":null}]}}`,
			[]UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}},
		},
		{
			"file_path is wrong type",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":123}}]}}`,
			[]UIEvent{{Type: "activity.agent", Kind: "editing", Detail: ""}},
		},
		{
			"good fields still parse",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}`,
			[]UIEvent{{Type: "activity.agent", Kind: "tool", Detail: "go test ./..."}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Translate([]byte(tc.raw))
			if err != nil {
				t.Fatalf("odd-but-valid input must not error, got %v", err)
			}
			assertEvents(t, got, tc.want)
		})
	}
}

func assertEvents(t *testing.T, got, want []UIEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: got=%+v want=%+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
