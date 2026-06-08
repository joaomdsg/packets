// Package translate maps the Claude Code harness's stream-json events into the
// review UI's events (the event-translation layer). This is the
// smallest slice: a PURE, stateless per-event translation producing live
// "activity.agent" events plus a "turn.ended" signal — no git, no I/O. The
// stateful reducer (cross-turn dirty tracking, threads, checks, permissions)
// and the orchestrator wiring (turn.ended -> settle+diff -> revision.created)
// are deferred.
package translate

import (
	"encoding/json"
	"fmt"
)

// UIEvent is one event surfaced to the review UI. Type is the event kind
// ("activity.agent" or "turn.ended"); Kind sub-classifies an activity
// ("thinking"|"editing"|"tool"); Detail carries the text, file path, command,
// or result subtype.
type UIEvent struct {
	Type   string
	Kind   string
	Detail string
}

// harnessEvent is the subset of the stream-json shape we consume.
type harnessEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Message struct {
		Content []contentItem `json:"content"`
	} `json:"message"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
	// Input is decoded lazily as raw JSON rather than into a fixed struct so
	// that an odd-but-valid tool input (a string, an array, or a field of the
	// wrong type) degrades to empty Detail instead of erroring the whole event
	// — the harness owns the tool-input schema, so we stay forward-compatible.
	Input json.RawMessage `json:"input"`
}

// toolInput is the subset of a tool_use's input we surface. It is parsed
// best-effort: a non-object or wrong-typed input yields the zero value.
type toolInput struct {
	FilePath string `json:"file_path"`
	Command  string `json:"command"`
}

// parseToolInput decodes input leniently. Any decode failure (input is not an
// object, or a field is the wrong type) is swallowed, yielding empty fields so
// the activity still flows.
func parseToolInput(raw json.RawMessage) toolInput {
	var in toolInput
	if len(raw) == 0 {
		return in
	}
	// Ignore the error on purpose: an odd input degrades, never fails the event.
	_ = json.Unmarshal(raw, &in)
	return in
}

// Translate maps one harness stream-json event to the UI events it produces.
// Unrecognized event types yield no events and no error (forward-compatible);
// only malformed JSON is an error. It performs no git or filesystem work.
func Translate(raw []byte) ([]UIEvent, error) {
	var ev harnessEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return nil, fmt.Errorf("translate: parse harness event: %w", err)
	}

	switch ev.Type {
	case "assistant":
		var out []UIEvent
		for _, c := range ev.Message.Content {
			switch c.Type {
			case "text":
				out = append(out, UIEvent{Type: "activity.agent", Kind: "thinking", Detail: c.Text})
			case "tool_use":
				in := parseToolInput(c.Input)
				switch c.Name {
				case "Edit", "Write", "MultiEdit":
					out = append(out, UIEvent{Type: "activity.agent", Kind: "editing", Detail: in.FilePath})
				case "Bash":
					out = append(out, UIEvent{Type: "activity.agent", Kind: "tool", Detail: in.Command})
				default:
					out = append(out, UIEvent{Type: "activity.agent", Kind: "tool", Detail: c.Name})
				}
			}
		}
		return out, nil
	case "result":
		return []UIEvent{{Type: "turn.ended", Detail: ev.Subtype}}, nil
	default:
		// Unrecognized event type (incl. user/tool_result): no UI events.
		return nil, nil
	}
}
