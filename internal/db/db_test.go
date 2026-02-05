package db_test

import (
	"context"
	"testing"

	"github.com/joaomdsg/agntpr/internal/db"
)

func TestDB_CreateAndGetIssue(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	issue := &db.Issue{
		GitHubID: 123,
		Number:   1,
		Title:    "Test issue",
		Body:     "Issue body",
		State:    db.StateNew,
	}

	err := d.CreateIssue(ctx, issue)
	if err != nil {
		t.Fatalf("failed to create issue: %v", err)
	}

	if issue.ID == 0 {
		t.Error("expected issue ID to be set")
	}

	got, err := d.GetIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	if got.GitHubID != 123 {
		t.Errorf("expected GitHubID 123, got %d", got.GitHubID)
	}
	if got.Title != "Test issue" {
		t.Errorf("expected title 'Test issue', got %s", got.Title)
	}
	if got.State != db.StateNew {
		t.Errorf("expected state new, got %s", got.State)
	}
}

func TestDB_GetIssueByGitHubID(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	issue := &db.Issue{
		GitHubID: 456,
		Number:   2,
		Title:    "GitHub issue",
		State:    db.StateNew,
	}

	err := d.CreateIssue(ctx, issue)
	if err != nil {
		t.Fatalf("failed to create issue: %v", err)
	}

	got, err := d.GetIssueByGitHubID(ctx, 456)
	if err != nil {
		t.Fatalf("failed to get issue by GitHub ID: %v", err)
	}

	if got.ID != issue.ID {
		t.Errorf("expected ID %d, got %d", issue.ID, got.ID)
	}
}

func TestDB_UpdateIssueState(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	issue := &db.Issue{
		GitHubID: 789,
		Number:   3,
		Title:    "State test",
		State:    db.StateNew,
	}

	err := d.CreateIssue(ctx, issue)
	if err != nil {
		t.Fatalf("failed to create issue: %v", err)
	}

	err = d.UpdateIssueState(ctx, issue.ID, db.StatePlanning)
	if err != nil {
		t.Fatalf("failed to update state: %v", err)
	}

	got, _ := d.GetIssue(ctx, issue.ID)
	if got.State != db.StatePlanning {
		t.Errorf("expected state planning, got %s", got.State)
	}
}

func TestDB_CreateAndGetPlan(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	issue := &db.Issue{
		GitHubID: 100,
		Number:   4,
		Title:    "Plan test",
		State:    db.StateNew,
	}
	_ = d.CreateIssue(ctx, issue)

	plan := &db.Plan{
		IssueID: issue.ID,
		Version: 1,
		Content: "Implementation plan",
	}

	err := d.CreatePlan(ctx, plan)
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	got, err := d.GetLatestPlan(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}

	if got.Content != "Implementation plan" {
		t.Errorf("expected content 'Implementation plan', got %s", got.Content)
	}
}

func TestDB_ApprovePlan(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	issue := &db.Issue{
		GitHubID: 101,
		Number:   5,
		Title:    "Approval test",
		State:    db.StateNew,
	}
	_ = d.CreateIssue(ctx, issue)

	plan := &db.Plan{
		IssueID: issue.ID,
		Version: 1,
		Content: "Plan to approve",
	}
	_ = d.CreatePlan(ctx, plan)

	err := d.ApprovePlan(ctx, plan.ID)
	if err != nil {
		t.Fatalf("failed to approve plan: %v", err)
	}

	got, _ := d.GetLatestPlan(ctx, issue.ID)
	if !got.Approved {
		t.Error("expected plan to be approved")
	}
}

func TestDB_ListPendingIssues(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	_ = d.CreateIssue(ctx, &db.Issue{
		GitHubID: 200,
		Number:   10,
		Title:    "Pending 1",
		State:    db.StateNew,
	})
	_ = d.CreateIssue(ctx, &db.Issue{
		GitHubID: 201,
		Number:   11,
		Title:    "Pending 2",
		State:    db.StatePlanning,
	})
	_ = d.CreateIssue(ctx, &db.Issue{
		GitHubID: 202,
		Number:   12,
		Title:    "Done",
		State:    db.StateDone,
	})

	issues, err := d.ListActiveIssues(ctx)
	if err != nil {
		t.Fatalf("failed to list issues: %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("expected 2 active issues, got %d", len(issues))
	}
}

func TestDB_CreateAndGetPR(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	issue := &db.Issue{
		GitHubID: 300,
		Number:   20,
		Title:    "PR test",
		State:    db.StatePRCreated,
	}
	_ = d.CreateIssue(ctx, issue)

	pr := &db.PullRequest{
		IssueID:  issue.ID,
		GitHubID: 500,
		Number:   1,
		Title:    "Fix issue #20",
		State:    "open",
	}

	err := d.CreatePR(ctx, pr)
	if err != nil {
		t.Fatalf("failed to create PR: %v", err)
	}

	got, err := d.GetPRByIssueID(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get PR: %v", err)
	}

	if got.Number != 1 {
		t.Errorf("expected PR number 1, got %d", got.Number)
	}
}

func TestDB_UpdatePlanFeedback(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	issue := &db.Issue{
		GitHubID: 102,
		Number:   6,
		Title:    "Feedback test",
		State:    db.StateNew,
	}
	_ = d.CreateIssue(ctx, issue)

	plan := &db.Plan{
		IssueID: issue.ID,
		Version: 1,
		Content: "Initial plan",
	}
	_ = d.CreatePlan(ctx, plan)

	err := d.UpdatePlanFeedback(ctx, plan.ID, "Please add more tests")
	if err != nil {
		t.Fatalf("failed to update plan feedback: %v", err)
	}

	got, _ := d.GetLatestPlan(ctx, issue.ID)
	if got.Feedback != "Please add more tests" {
		t.Errorf("expected feedback 'Please add more tests', got %s", got.Feedback)
	}
}

func TestDB_GetPlanHistory(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	issue := &db.Issue{
		GitHubID: 103,
		Number:   7,
		Title:    "History test",
		State:    db.StateNew,
	}
	_ = d.CreateIssue(ctx, issue)

	plan1 := &db.Plan{
		IssueID: issue.ID,
		Version: 1,
		Content: "Plan v1",
	}
	_ = d.CreatePlan(ctx, plan1)
	_ = d.UpdatePlanFeedback(ctx, plan1.ID, "Add tests")

	plan2 := &db.Plan{
		IssueID: issue.ID,
		Version: 2,
		Content: "Plan v2",
	}
	_ = d.CreatePlan(ctx, plan2)

	plans, err := d.GetPlanHistory(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get plan history: %v", err)
	}

	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}

	if plans[0].Version != 1 {
		t.Errorf("expected first plan to be v1, got v%d", plans[0].Version)
	}

	if plans[0].Feedback != "Add tests" {
		t.Errorf("expected feedback 'Add tests', got %s", plans[0].Feedback)
	}

	if plans[1].Version != 2 {
		t.Errorf("expected second plan to be v2, got v%d", plans[1].Version)
	}
}

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}
