package gate

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// ClaudeCLI is the real LLM implementation: it shells out to the host's
// installed "claude -p" binary. Never used in tests — see gate_test.go's
// stub, which is what CONVENTIONS.md's mocking preference requires at this
// external-process boundary.
type ClaudeCLI struct{}

// Review runs `claude -p <prompt> --output-format json` and returns the
// "result" field, which holds the model's text — the gate JSON is nested
// inside that, not at the top level of the CLI's own JSON envelope.
func (ClaudeCLI) Review(prompt string) (string, error) {
	out, err := exec.Command("claude", "-p", prompt, "--output-format", "json").Output()
	if err != nil {
		return "", fmt.Errorf("gate: claude -p: %s", err)
	}

	var envelope struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		return "", fmt.Errorf("gate: parse claude -p envelope: %s", err)
	}
	return envelope.Result, nil
}
