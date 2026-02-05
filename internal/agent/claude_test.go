package agent_test

import (
	"testing"
	"time"

	"github.com/joaomdsg/agntpr/internal/agent"
)

func TestNewClaudeRunner_WithModel(t *testing.T) {
	runner := agent.NewClaudeRunner(30*time.Second, "claude-3-haiku-20240307")

	if runner.Model() != "claude-3-haiku-20240307" {
		t.Errorf("expected model claude-3-haiku-20240307, got %s", runner.Model())
	}
}

func TestNewClaudeRunner_DefaultModel(t *testing.T) {
	runner := agent.NewClaudeRunner(30*time.Second, "")

	if runner.Model() != agent.DefaultModel {
		t.Errorf("expected default model %s, got %s", agent.DefaultModel, runner.Model())
	}
}
