package agent_test

import (
	"context"
	"testing"

	"github.com/joaomdsg/agntpr/internal/agent"
)

func TestBuildPlanningPrompt(t *testing.T) {
	req := &agent.Request{
		IssueNumber: 42,
		IssueTitle:  "Add feature X",
		IssueBody:   "We need feature X to do Y",
	}

	prompt := agent.BuildPlanningPrompt(req)

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}

	if !containsAll(prompt, ".agntpr-context.md", "plan") {
		t.Error("prompt should mention context file")
	}
}

func TestBuildImplementationPrompt(t *testing.T) {
	req := &agent.Request{
		IssueNumber: 42,
		IssueTitle:  "Add feature X",
		IssueBody:   "We need feature X to do Y",
		Plan:        "1. Add function\n2. Add tests",
	}

	prompt := agent.BuildImplementationPrompt(req)

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}

	if !containsAll(prompt, ".agntpr-context.md", "TDD", "brew install", "docker") {
		t.Error("prompt should mention context, TDD, brew, and docker")
	}
}

func TestBuildReviewResponsePrompt(t *testing.T) {
	req := &agent.Request{
		IssueNumber:   42,
		ReviewComment: "Please add error handling",
	}

	prompt := agent.BuildReviewResponsePrompt(req)

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}

	if !containsAll(prompt, "error handling", "brew install", "docker") {
		t.Error("prompt should contain review comment, brew, and docker")
	}
}

type mockRunner struct {
	output string
	err    error
}

func (m *mockRunner) Run(
	ctx context.Context, workDir, prompt string,
) (*agent.Result, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &agent.Result{
		Output:  m.output,
		Success: true,
	}, nil
}

func TestInvoker_Plan(t *testing.T) {
	runner := &mockRunner{
		output: "## Plan\n1. Do thing\n2. Do other thing",
	}

	inv := agent.NewInvoker(runner)
	ctx := context.Background()

	req := &agent.Request{
		WorkDir:     "/work",
		IssueNumber: 1,
		IssueTitle:  "Test",
		IssueBody:   "Body",
	}

	result, err := inv.Plan(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}

	if result.Output == "" {
		t.Error("expected output")
	}
}

func TestInvoker_Implement(t *testing.T) {
	runner := &mockRunner{
		output: "Implementation complete",
	}

	inv := agent.NewInvoker(runner)
	ctx := context.Background()

	req := &agent.Request{
		WorkDir:     "/work",
		IssueNumber: 1,
		IssueTitle:  "Test",
		Plan:        "1. Do thing",
	}

	result, err := inv.Implement(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
}

func TestBuildSummaryPrompt(t *testing.T) {
	prompt := agent.BuildSummaryPrompt()

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}

	if !containsAll(prompt, "git diff", "summary", "bullet") {
		t.Error("prompt should mention git diff and bullet point summary")
	}

	if !containsAll(prompt, "origin/main") {
		t.Error("prompt should reference base branch")
	}
}

func TestInvoker_SummarizeChanges(t *testing.T) {
	runner := &mockRunner{
		output: "- Added new feature\n- Fixed bug",
	}

	inv := agent.NewInvoker(runner)
	ctx := context.Background()

	result, err := inv.SummarizeChanges(ctx, "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}

	if result.Output == "" {
		t.Error("expected summary output")
	}
}

func TestBuildAnswerPrompt(t *testing.T) {
	req := &agent.AnswerRequest{
		Question:     "How does the caching work?",
		IssueContext: "Issue about adding caching",
	}

	prompt := agent.BuildAnswerPrompt(req)

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}

	if !containsAll(prompt, "caching", "Question") {
		t.Error("prompt should contain question and context")
	}
}

func TestInvoker_AnswerQuestion(t *testing.T) {
	runner := &mockRunner{
		output: "The caching uses an LRU eviction policy.",
	}

	inv := agent.NewInvoker(runner)
	ctx := context.Background()

	req := &agent.AnswerRequest{
		WorkDir:      "/work",
		Question:     "How does caching work?",
		IssueContext: "Adding cache feature",
	}

	result, err := inv.AnswerQuestion(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}

	if result.Output == "" {
		t.Error("expected answer output")
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && (s[0:len(substr)] == substr ||
			contains(s[1:], substr)))
}
