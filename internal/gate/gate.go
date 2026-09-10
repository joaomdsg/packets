// Package gate implements the emit gate's Stage B: an LLM review of a
// packet.yaml that produces advisory warnings and suggested terminal
// predicates. It never blocks emit on its own failure.
package gate

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed gate-prompt.txt
var promptTemplate string

// LLM is the host-side "claude -p" boundary. Review sends prompt and
// returns the model's raw text response, which may be plain JSON, JSON
// wrapped in prose, or JSON wrapped in markdown fences.
type LLM interface {
	Review(prompt string) (string, error)
}

// Warning is one entry in the gate's warnings array.
type Warning struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// Result is the gate's parsed response: warnings plus suggested terminal
// predicates to fill a gap in the packet's terminal list.
type Result struct {
	Warnings          []Warning `json:"warnings"`
	SuggestedTerminal []string  `json:"suggested_terminal"`
}

// BuildPrompt substitutes packetYAML and repoFiles into the embedded
// gate-prompt.txt template.
func BuildPrompt(packetYAML, repoFiles string) string {
	prompt := strings.ReplaceAll(promptTemplate, "{{PACKET}}", packetYAML)
	return strings.ReplaceAll(prompt, "{{REPO_FILES}}", repoFiles)
}

// Run calls llm with the built prompt and parses its response, retrying
// once on failure (process error or malformed JSON). err is non-nil only
// after both attempts fail; callers must log gate_llm_error and proceed
// with zero warnings rather than block emit.
func Run(llm LLM, packetYAML, repoFiles string) (Result, error) {
	prompt := BuildPrompt(packetYAML, repoFiles)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := llm.Review(prompt)
		if err != nil {
			lastErr = err
			continue
		}
		result, err := parse(raw)
		if err != nil {
			lastErr = err
			continue
		}
		return result, nil
	}
	return Result{}, fmt.Errorf("gate: llm review failed: %s", lastErr)
}

// parse extracts JSON from raw, defensively stripping any prose or
// markdown fences the model added despite being told not to.
func parse(raw string) (Result, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return Result{}, fmt.Errorf("gate: no JSON object found in response")
	}

	var result Result
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return Result{}, fmt.Errorf("gate: parse response: %s", err)
	}
	return result, nil
}
