package orchestrator_test

import (
	"context"
	"testing"

	"github.com/joaomdsg/agntpr/internal/db"
	"github.com/joaomdsg/agntpr/internal/orchestrator"
	"github.com/joaomdsg/agntpr/internal/watcher"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	return database
}

// TestClarificationWaitsForResponse tests that when bot asks a clarifying
// question, it stays in "new" state and waits for a response before planning
func TestClarificationWaitsForResponse(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	// Initial issue that needs clarification
	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 1001, Number: 20, Title: "Ambiguous request", Body: "Make it work"},
		},
		issueComments: []*watcher.Comment{}, // No comments initially
		mention:       "@test-bot",
	}

	fm := &mockForkManager{workDir: "/work/test"}

	// Agent returns NeedsClarify on initial evaluation
	agent := &mockAgent{
		intentResult: &orchestrator.Intent{
			IsQuestion:   false,
			NeedsClarify: true,
			SkipPlanning: false,
			SkipApproval: false,
			Question:     "What specific functionality do you need?",
		},
		planResult: "Test plan content",
	}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	// First poll - should ask clarifying question and stay in "new"
	err := orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("first poll failed: %v", err)
	}

	// Verify issue was created in "new" state
	issue, err := database.GetIssueByGitHubID(ctx, 1001)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	if issue.State != db.StateNew {
		t.Errorf("expected state 'new' after clarification, got %s", issue.State)
	}

	// Verify clarifying question was posted
	if len(w.posted) != 1 {
		t.Fatalf("expected 1 comment posted, got %d", len(w.posted))
	}

	if !contains(w.posted[0], "Before I start") {
		t.Errorf("expected clarifying question format, got: %s", w.posted[0])
	}

	// Second poll - no response yet, should stay in "new" and NOT start planning
	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("second poll failed: %v", err)
	}

	issue, err = database.GetIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get issue after second poll: %v", err)
	}

	if issue.State != db.StateNew {
		t.Errorf("expected to stay in 'new' without response, got %s", issue.State)
	}

	// Verify no plan was created in DB
	plans, err := database.GetPlanHistory(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get plan history: %v", err)
	}
	if len(plans) > 0 {
		t.Error("agent should not plan before receiving clarification response")
	}

	// Third poll - maintainer responds with clarification
	w.issueComments = []*watcher.Comment{
		{
			ID:   1,
			Body: "@test-bot I need user authentication with JWT",
		},
	}
	w.issueComments[0].User.Login = "maintainer"

	// Agent now returns normal intent (no clarification needed)
	agent.intentResult = &orchestrator.Intent{
		IsQuestion:   false,
		NeedsClarify: false,
		SkipPlanning: false,
	}

	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("third poll failed: %v", err)
	}

	// Now should transition to planning
	issue, err = database.GetIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get issue after response: %v", err)
	}

	if issue.State != db.StatePlanReview {
		t.Errorf("expected state 'plan_review' after clarification response, got %s", issue.State)
	}

	// Verify plan was created
	plans, err = database.GetPlanHistory(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get plan history: %v", err)
	}
	if len(plans) == 0 {
		t.Error("agent should have been asked to plan after clarification response")
	}
}

// TestQuestionInNewIssueWaitsForResponse tests that when a question is
// detected in the initial issue, bot answers and waits before planning
func TestQuestionInNewIssueWaitsForResponse(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{
				ID:     2001,
				Number: 25,
				Title:  "Question about feature",
				Body:   "@test-bot How does the caching system work?",
			},
		},
		issueComments: []*watcher.Comment{},
		mention:       "@test-bot",
	}

	fm := &mockForkManager{workDir: "/work/test"}

	// Agent detects this is a question
	agent := &mockAgent{
		intentResult: &orchestrator.Intent{
			IsQuestion:   true,
			NeedsClarify: false,
			Question:     "How does the caching system work?",
		},
		answerResult: "The caching system uses Redis...",
		planResult:   "Test plan",
	}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	// First poll - should answer question and stay in "new"
	err := orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("first poll failed: %v", err)
	}

	// Verify issue in "new" state
	issue, err := database.GetIssueByGitHubID(ctx, 2001)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	if issue.State != db.StateNew {
		t.Errorf("expected state 'new' after answering question, got %s", issue.State)
	}

	// Verify answer was posted
	if len(w.posted) != 1 {
		t.Fatalf("expected 1 comment (answer), got %d", len(w.posted))
	}

	if !contains(w.posted[0], "💬") {
		t.Errorf("expected answer format with emoji, got: %s", w.posted[0])
	}

	// Second poll - no follow-up yet, should NOT start planning
	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("second poll failed: %v", err)
	}

	// Verify no plan created yet
	plans, err := database.GetPlanHistory(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get plan history: %v", err)
	}
	if len(plans) > 0 {
		t.Error("agent should not plan without explicit follow-up mention")
	}

	// Third poll - user follows up with implementation request
	w.issueComments = []*watcher.Comment{
		{
			ID:   2,
			Body: "@test-bot Thanks! Can you implement this feature?",
		},
	}
	w.issueComments[0].User.Login = "user"

	// Agent now detects implementation request
	agent.intentResult = &orchestrator.Intent{
		IsQuestion:   false,
		NeedsClarify: false,
	}

	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("third poll failed: %v", err)
	}

	// Should now proceed to planning
	issue, err = database.GetIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	if issue.State != db.StatePlanReview {
		t.Errorf("expected state 'plan_review' after implementation request, got %s", issue.State)
	}

	// Verify plan was created
	plans, err = database.GetPlanHistory(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get plan history: %v", err)
	}
	if len(plans) == 0 {
		t.Error("agent should plan after explicit implementation request")
	}
}
