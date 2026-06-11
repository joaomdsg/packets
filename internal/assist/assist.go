// Package assist models the producer's live analysis of a work-order draft — the
// structured highlights, clarifying questions, and readiness summary the authoring
// surface renders. The analysis is produced by a Claude Code harness run that
// prints one JSON object; ParseAnalysis extracts and validates it, so the rest of
// the system works against a typed result, never raw agent output.
package assist

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Highlight marks a span of the draft [Start,End) the producer flagged, with a note
// and a severity ("question", "gap", "note", …). Offsets are byte offsets into the
// draft so the editor can anchor a decoration on exactly that range.
type Highlight struct {
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Note     string `json:"note"`
	Severity string `json:"severity,omitempty"`
}

// Analysis is the producer's structured read of a draft: a one-line summary, a
// readiness verdict (is the goal clear and verifiable enough to run unattended),
// the flagged spans, and the clarifying questions worth answering before placing
// the order.
type Analysis struct {
	Summary    string      `json:"summary"`
	Ready      bool        `json:"ready"`
	Highlights []Highlight `json:"highlights"`
	Questions  []string    `json:"questions"`
}

// AnalysisPrompt builds the instruction a producer harness runs to analyze a
// work-order draft: it carries the draft and pins the exact JSON shape
// ParseAnalysis decodes, so the prompt and parser are one contract. The agent is
// told to print ONLY the JSON object (the parser tolerates surrounding prose, but
// asking for clean output keeps a run cheap and unambiguous).
func AnalysisPrompt(draft string) string {
	return `You are a producer reviewing a work-order draft a Lead is authoring.
Analyze it for clarity and whether its goal is verifiable enough to run
unattended. Identify spans worth flagging (ambiguities, gaps, undefined terms)
and the clarifying questions worth answering before the work is dispatched.

Respond with ONE JSON object and nothing else, in this exact shape:
{
  "summary": "<one line: the draft's goal and the biggest gap>",
  "ready": <true|false: is the goal clear and verifiable enough to run unattended>,
  "highlights": [
    {"start": <byte offset into the draft>, "end": <byte offset, exclusive>,
     "note": "<why this span is flagged>", "severity": "<question|gap|note>"}
  ],
  "questions": ["<clarifying question>", "..."]
}
Offsets are byte offsets into the draft below; end is exclusive.

DRAFT:
` + draft
}

// ParseAnalysis extracts the one JSON object the producer printed from raw (tolerant
// of surrounding prose and ```json fences) and validates it against draft: a
// highlight whose range is inverted or falls outside the draft is DROPPED rather
// than returned as a range the editor can't anchor (an end exactly at len(draft) is
// valid — it marks through the last byte). Output with no JSON object is an error,
// never a silently empty analysis.
func ParseAnalysis(raw, draft string) (Analysis, error) {
	obj, ok := extractJSONObject(raw)
	if !ok {
		return Analysis{}, fmt.Errorf("assist: no JSON object found in producer output")
	}
	var a Analysis
	if err := json.Unmarshal([]byte(obj), &a); err != nil {
		return Analysis{}, fmt.Errorf("assist: decode analysis: %v", err)
	}
	kept := a.Highlights[:0]
	for _, h := range a.Highlights {
		if h.Start >= 0 && h.End >= h.Start && h.End <= len(draft) {
			kept = append(kept, h)
		}
	}
	a.Highlights = kept
	return a, nil
}

// extractJSONObject returns the first balanced top-level {...} object in s,
// scanning past prose and code fences. It tracks string literals so a brace inside
// a JSON string never throws off the balance count.
func extractJSONObject(s string) (string, bool) {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return "", false
	}
	depth, inStr, esc := 0, false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		switch {
		case esc:
			esc = false
		case c == '\\' && inStr:
			esc = true
		case c == '"':
			inStr = !inStr
		case inStr:
			// inside a string literal — braces don't count
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}
	return "", false
}
