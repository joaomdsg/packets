package context_test

import (
	"testing"

	"github.com/joaomdsg/agntpr/internal/context"
	"github.com/joaomdsg/agntpr/internal/db"
)

func TestBuildContext_Issue(t *testing.T) {
	issue := &db.Issue{
		Number: 42,
		Title:  "Add feature X",
		Body:   "We need feature X",
	}

	comments := []*context.Comment{
		{Author: "user1", Body: "Sounds good"},
		{Author: "ai-r-sentry", Body: "I'll work on this"},
	}

	ctx := context.Build(issue, comments, nil, nil)

	if ctx == "" {
		t.Error("expected non-empty context")
	}

	if !contains(ctx, "# Issue #42: Add feature X") {
		t.Error("expected issue title")
	}

	if !contains(ctx, "Sounds good") {
		t.Error("expected comment")
	}
}

func TestBuildContext_WithPRComments(t *testing.T) {
	issue := &db.Issue{
		Number: 42,
		Title:  "Fix bug",
		Body:   "Bug description",
	}

	prComments := []*context.PRComment{
		{
			Author:   "reviewer",
			Body:     "Fix this",
			Path:     "main.go",
			Line:     10,
			DiffHunk: "@@ ...",
		},
	}

	ctx := context.Build(issue, nil, prComments, nil)

	if !contains(ctx, "main.go:10") {
		t.Error("expected file:line reference")
	}
}

func TestBuildContext_WithPlanHistory(t *testing.T) {
	issue := &db.Issue{
		Number: 42,
		Title:  "Add feature",
		Body:   "Feature description",
	}

	plans := []*context.Plan{
		{
			Version:  1,
			Content:  "Plan v1",
			Feedback: "Add more tests",
			Approved: false,
		},
		{
			Version:  2,
			Content:  "Plan v2 with more tests",
			Feedback: "",
			Approved: true,
		},
	}

	ctx := context.Build(issue, nil, nil, plans)

	if !contains(ctx, "Plan History") {
		t.Error("expected 'Plan History' section")
	}

	if !contains(ctx, "Plan v1") {
		t.Error("expected plan v1 content")
	}

	if !contains(ctx, "Add more tests") {
		t.Error("expected feedback from v1")
	}

	if !contains(ctx, "Plan v2 with more tests") {
		t.Error("expected plan v2 content")
	}
}

func TestBuildContext_NoPlanHistory(t *testing.T) {
	issue := &db.Issue{
		Number: 42,
		Title:  "Add feature",
		Body:   "Feature description",
	}

	ctx := context.Build(issue, nil, nil, nil)

	if contains(ctx, "Plan History") {
		t.Error("should not have 'Plan History' section when no plans")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr, 0)
}

func findSubstr(s, substr string, start int) bool {
	if start > len(s)-len(substr) {
		return false
	}
	if s[start:start+len(substr)] == substr {
		return true
	}
	return findSubstr(s, substr, start+1)
}
