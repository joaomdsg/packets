// Package assist models the assist's live analysis of a packet draft — the
// structured highlights, clarifying questions, and readiness summary the authoring
// surface renders. The analysis is produced by a Claude Code harness run that
// prints one JSON object; ParseAnalysis extracts and validates it, so the rest of
// the system works against a typed result, never raw agent output.
package assist

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Highlight marks a span of the draft [Start,End) the assist flagged, with a note
// and a severity ("question", "gap", "note", …). Offsets are byte offsets into the
// draft so the editor can anchor a decoration on exactly that range.
type Highlight struct {
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Note     string `json:"note"`
	Severity string `json:"severity,omitempty"`
}

// Question is one clarifying question the assist raised, with up to four suggested
// answers the Lead can pick from and a flag for whether several may apply at once
// (multiselect). Suggestions let the Lead choose instead of free-typing; a question
// may still carry none (the Lead answers in the notes field the panel renders).
type Question struct {
	Q           string   `json:"q"`
	Suggestions []string `json:"suggestions,omitempty"`
	Multiselect bool     `json:"multiselect,omitempty"`
}

// maxSuggestions bounds how many suggested answers a question renders — a calm,
// scannable choice set, not an overwhelming wall. ParseAnalysis trims to this.
const maxSuggestions = 4

// UnmarshalJSON decodes a question tolerantly: a assist may answer with either a
// bare JSON string (the terse "just the question" shape) or the rich object shape
// {q, suggestions, multiselect}. A bare string becomes the question text with no
// suggestions, so a half-structured reply (mixed strings and objects) still decodes
// instead of failing the whole analysis.
func (q *Question) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*q = Question{Q: s}
		return nil
	}
	type alias Question // avoid recursing into this method
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		// Neither a string nor an object (a stray number/bool/null/array the assist
		// emitted by mistake): decode to a zero Question (empty Q) so ParseAnalysis drops
		// it, rather than failing the WHOLE analysis on one malformed element.
		*q = Question{}
		return nil
	}
	*q = Question(a)
	return nil
}

// Analysis is the assist's structured read of a draft: a one-line summary, a
// readiness verdict (is the goal clear and verifiable enough to run unattended),
// the flagged spans, and the clarifying questions worth answering before placing
// the packet.
type Analysis struct {
	Summary    string      `json:"summary"`
	Ready      bool        `json:"ready"`
	Highlights []Highlight `json:"highlights"`
	Questions  []Question  `json:"questions"`
}

// AnalysisPrompt builds the instruction a assist harness runs to analyze a
// packet draft: it carries the draft and pins the exact JSON shape
// ParseAnalysis decodes, so the prompt and parser are one contract. The agent is
// told to print ONLY the JSON object (the parser tolerates surrounding prose, but
// asking for clean output keeps a run cheap and unambiguous).
func AnalysisPrompt(draft string) string {
	return `You are a assist reviewing a packet draft a Lead is authoring.
Analyze it for clarity and whether its goal is verifiable enough to run
unattended. Identify spans worth flagging (ambiguities, gaps, undefined terms)
and the clarifying questions worth answering before the work is sent.

Respond with ONE JSON object and nothing else, in this exact shape:
{
  "summary": "<one line: the draft's goal and the biggest gap>",
  "ready": <true|false: is the goal clear and verifiable enough to run unattended>,
  "highlights": [
    {"start": <byte offset into the draft>, "end": <byte offset, exclusive>,
     "note": "<why this span is flagged>", "severity": "<question|gap|note>"}
  ],
  "questions": [
    {"q": "<clarifying question>",
     "suggestions": ["<up to 4 likely answers the Lead can pick from>"],
     "multiselect": <true|false: may several suggestions apply at once>}
  ]
}
For each question, offer at most 4 concrete suggested answers when you can, and set
multiselect true only when more than one suggestion could legitimately apply.
Offsets are byte offsets into the draft below; end is exclusive.

DRAFT:
` + draft
}

// ReviewTurnPrompt composes the harness turn for an anchored review comment (DESIGN
// §12.3): it names the commented file and line, quotes the line under review (when the
// orchestrator could read it), carries the maintainer's comment, and instructs the
// harness to address it against the working tree — so the agent re-edits in place and
// its edits fold into the next revision. This is the keystone of the review-thread
// loop: it turns "leave an adjustment" into "watch the agent fix it".
func ReviewTurnPrompt(file string, line int, codeLine, comment string) string {
	var b strings.Builder
	b.WriteString("[review comment on ")
	b.WriteString(file)
	b.WriteString(", line ")
	b.WriteString(strconv.Itoa(line))
	b.WriteString("]\n")
	if strings.TrimSpace(codeLine) != "" {
		b.WriteString("> ")
		b.WriteString(strconv.Itoa(line))
		b.WriteString("  ")
		b.WriteString(codeLine)
		b.WriteByte('\n')
	}
	b.WriteString("Maintainer: \"")
	b.WriteString(comment)
	b.WriteString("\"\n\nAddress this specifically. The surrounding code is in the working tree.")
	return b.String()
}

// Answer is the Lead's response to one clarifying Question: the question text, the
// suggestions they picked (zero or more), and any free-text note (a different answer,
// or extra context). RewritePrompt feeds these to the assist to revise the draft.
type Answer struct {
	Q       string
	Answers []string
	Note    string
}

// RewritePrompt builds the instruction a assist runs to REVISE a draft using the
// Lead's answers to the clarifying questions. It carries the current draft and each
// answer (question + picked suggestions + free note) and tells the assist to fold
// them in and return ONLY the rewritten draft — no commentary — so ParseRewrite gets
// draft text, not a chat reply.
func RewritePrompt(draft string, answers []Answer) string {
	var b strings.Builder
	b.WriteString(`You are a assist revising a packet draft a Lead is authoring.
The Lead answered your clarifying questions below. Fold their answers into the
draft: resolve the ambiguities, fill the gaps, and keep the Lead's intent and
voice. Output ONLY the rewritten draft as plain markdown — no preamble, no
commentary, no code fences around the whole thing.

ANSWERS:
`)
	for _, a := range answers {
		b.WriteString("- Q: ")
		b.WriteString(a.Q)
		b.WriteByte('\n')
		if len(a.Answers) > 0 {
			b.WriteString("  Chosen: ")
			b.WriteString(strings.Join(a.Answers, ", "))
			b.WriteByte('\n')
		}
		if strings.TrimSpace(a.Note) != "" {
			b.WriteString("  Note: ")
			b.WriteString(a.Note)
			b.WriteByte('\n')
		}
	}
	b.WriteString("\nDRAFT:\n")
	b.WriteString(draft)
	return b.String()
}

// ParseRewrite cleans the assist's rewrite reply into draft text ready for the
// editor: it trims surrounding whitespace, and if the WHOLE reply is wrapped in one
// outer ```fence (a common model habit) it strips just that fence — dropping the
// opening line's language info-string and the single closing fence — so an inner
// code block that is part of the draft survives untouched. Otherwise the trimmed
// reply is returned verbatim. Never errors: an empty reply is an empty draft.
func ParseRewrite(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") && strings.HasSuffix(s, "```") {
		if nl := strings.IndexByte(s, '\n'); nl >= 0 {
			inner := strings.TrimSuffix(s[nl+1:], "```")
			return strings.TrimRight(inner, "\n")
		}
	}
	return s
}

// ParseAnalysis extracts the one JSON object the assist printed from raw (tolerant
// of surrounding prose and ```json fences) and validates it against draft: a
// highlight whose range is inverted or falls outside the draft is DROPPED rather
// than returned as a range the editor can't anchor (an end exactly at len(draft) is
// valid — it marks through the last byte). Output with no JSON object is an error,
// never a silently empty analysis.
func ParseAnalysis(raw, draft string) (Analysis, error) {
	obj, ok := extractJSONObject(raw)
	if !ok {
		return Analysis{}, fmt.Errorf("assist: no JSON object found in assist output")
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
	// A question with no text is nothing to ask (a dead control); drop it. Trim the
	// text so it renders clean, and cap each question's suggestions at maxSuggestions
	// so the panel stays a scannable choice set.
	keptQ := a.Questions[:0]
	for _, q := range a.Questions {
		q.Q = strings.TrimSpace(q.Q)
		if q.Q == "" {
			continue
		}
		// Trim suggestions and drop blanks so a padded/empty choice never renders a
		// ragged or dead chip (the trimmed value is also what's echoed into the rewrite).
		if len(q.Suggestions) > 0 {
			cleaned := q.Suggestions[:0]
			for _, s := range q.Suggestions {
				if s = strings.TrimSpace(s); s != "" {
					cleaned = append(cleaned, s)
				}
			}
			q.Suggestions = cleaned
		}
		if len(q.Suggestions) > maxSuggestions {
			q.Suggestions = q.Suggestions[:maxSuggestions]
		}
		keptQ = append(keptQ, q)
	}
	a.Questions = keptQ
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
