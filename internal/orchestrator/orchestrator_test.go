package orchestrator_test

import (
	"context"
	"testing"

	"github.com/joaomdsg/agntpr/internal/db"
	"github.com/joaomdsg/agntpr/internal/orchestrator"
	"github.com/joaomdsg/agntpr/internal/watcher"
)

type mockWatcher struct {
	issues        []*watcher.Issue
	prs           []*watcher.PR
	comments      []*watcher.Comment
	issueComments []*watcher.Comment
	posted        []string
	postedTo      []int // tracks which issue/PR number each comment was posted to
	prBodies      []string
	mention       string
}

func (m *mockWatcher) FetchMentionedIssues(
	ctx context.Context,
) ([]*watcher.Issue, error) {
	return m.issues, nil
}

func (m *mockWatcher) FetchIssueComments(
	ctx context.Context, issueNum int,
) ([]*watcher.Comment, error) {
	return m.issueComments, nil
}

func (m *mockWatcher) FetchOpenPRs(ctx context.Context) ([]*watcher.PR, error) {
	return m.prs, nil
}

func (m *mockWatcher) FetchPRComments(
	ctx context.Context, prNum int,
) ([]*watcher.Comment, error) {
	return m.comments, nil
}

func (m *mockWatcher) FetchAllPRComments(
	ctx context.Context, prNum int,
) ([]*watcher.Comment, error) {
	return m.comments, nil
}

func (m *mockWatcher) FilterMentionedComments(
	comments []*watcher.Comment,
) []*watcher.Comment {
	return comments
}

func (m *mockWatcher) PostComment(
	ctx context.Context, num int, body string,
) (int64, error) {
	m.posted = append(m.posted, body)
	m.postedTo = append(m.postedTo, num)
	return int64(len(m.posted)), nil
}

func (m *mockWatcher) Mention() string {
	if m.mention == "" {
		return "@ai-r-sentry"
	}
	return m.mention
}

func (m *mockWatcher) CreatePR(
	ctx context.Context, title, body, head, base string,
) (*watcher.PR, error) {
	m.prBodies = append(m.prBodies, body)
	return &watcher.PR{Number: 1, Title: title, State: "open"}, nil
}

func (m *mockWatcher) GetPR(
	ctx context.Context, prNum int,
) (*watcher.PR, error) {
	for _, pr := range m.prs {
		if pr.Number == prNum {
			return pr, nil
		}
	}
	return &watcher.PR{Number: prNum, State: "open"}, nil
}

type mockForkManager struct {
	workDir string
	branch  string
}

func (m *mockForkManager) SetupWorkDir(
	ctx context.Context, owner, repo string, issueNum int,
) (string, error) {
	return m.workDir, nil
}

func (m *mockForkManager) CreateBranch(
	ctx context.Context, workDir string, issueNum int,
) (string, error) {
	return m.branch, nil
}

func (m *mockForkManager) SyncWithUpstream(
	ctx context.Context, workDir, baseBranch string,
) error {
	return nil
}

func (m *mockForkManager) PushBranch(
	ctx context.Context, workDir, branch string, force bool,
) error {
	return nil
}

func (m *mockForkManager) HasChanges(
	ctx context.Context, workDir, baseBranch string,
) (bool, error) {
	return true, nil
}

type mockAgent struct {
	planResult      string
	summaryResult   string
	answerResult    string
	intentResult    *orchestrator.Intent
	implementCalled bool
	implementPlan   string
}

func (m *mockAgent) Plan(
	ctx context.Context, workDir string, issue *db.Issue,
) (string, error) {
	return m.planResult, nil
}

func (m *mockAgent) Implement(
	ctx context.Context, workDir string, issue *db.Issue, plan string,
) error {
	m.implementCalled = true
	m.implementPlan = plan
	return nil
}

func (m *mockAgent) RespondToReview(
	ctx context.Context, workDir, comment string,
) error {
	return nil
}

func (m *mockAgent) EvaluateIntent(
	ctx context.Context, issueTitle, issueBody string, labels []string, comments []string,
) (*orchestrator.Intent, error) {
	if m.intentResult != nil {
		return m.intentResult, nil
	}
	return &orchestrator.Intent{}, nil
}

func (m *mockAgent) SummarizeChanges(
	ctx context.Context, workDir string,
) (string, error) {
	if m.summaryResult != "" {
		return m.summaryResult, nil
	}
	return "- Changes made", nil
}

func (m *mockAgent) AnswerQuestion(
	ctx context.Context, workDir, question, issueContext string,
) (string, error) {
	if m.answerResult != "" {
		return m.answerResult, nil
	}
	return "Here is the answer.", nil
}

func TestOrchestrator_ProcessNewIssue(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 100, Number: 1, Title: "Test Issue", Body: "Body"},
		},
	}
	fm := &mockForkManager{workDir: "/work/test", branch: "ai-r-sentry/issue-1"}
	agent := &mockAgent{planResult: "## Plan\n1. Do thing"}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	ctx := context.Background()
	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process issues: %v", err)
	}

	// Check issue was created in database
	issue, err := database.GetIssueByGitHubID(ctx, 100)
	if err != nil {
		t.Fatalf("issue not found in db: %v", err)
	}

	if issue.State != db.StatePlanReview {
		t.Errorf("expected state plan_review, got %s", issue.State)
	}

	// Check plan was posted as comment
	if len(w.posted) == 0 {
		t.Error("expected plan to be posted as comment")
	}
}

func TestOrchestrator_SkipsExistingIssues(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Create existing issue
	existing := &db.Issue{
		GitHubID: 200,
		Number:   2,
		Title:    "Existing",
		State:    db.StatePlanning,
	}
	if err := database.CreateIssue(ctx, existing); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 200, Number: 2, Title: "Existing", Body: "Body"},
		},
	}
	fm := &mockForkManager{}
	agent := &mockAgent{}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// Verify no new issues created
	issues, _ := database.ListActiveIssues(ctx)
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestOrchestrator_SkipPlanning(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 300, Number: 3, Title: "Skip Plan Issue", Body: "Just do it"},
		},
	}
	fm := &mockForkManager{workDir: "/work/test", branch: "ai-r-sentry/issue-3"}
	agent := &mockAgent{
		intentResult: &orchestrator.Intent{SkipPlanning: true},
	}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	ctx := context.Background()
	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process issues: %v", err)
	}

	// Check issue went directly to pr_created (skipped planning)
	issue, err := database.GetIssueByGitHubID(ctx, 300)
	if err != nil {
		t.Fatalf("issue not found in db: %v", err)
	}

	if issue.State != db.StatePRCreated {
		t.Errorf("expected state pr_created, got %s", issue.State)
	}

	if !issue.SkipPlanning {
		t.Error("expected SkipPlanning to be true")
	}

	// Verify Implement was called with empty plan
	if !agent.implementCalled {
		t.Error("expected Implement to be called")
	}
	if agent.implementPlan != "" {
		t.Errorf("expected empty plan when skipping, got %q", agent.implementPlan)
	}

	// Verify status message doesn't mention "plan approved"
	foundPlanApproved := false
	for _, comment := range w.posted {
		if contains(comment, "Plan approved") {
			foundPlanApproved = true
			break
		}
	}
	if foundPlanApproved {
		t.Error("should not mention 'Plan approved' when planning was skipped")
	}
}

func TestOrchestrator_NormalPlanningFlow(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 400, Number: 4, Title: "Normal Issue", Body: "Need a plan"},
		},
	}
	fm := &mockForkManager{workDir: "/work/test", branch: "ai-r-sentry/issue-4"}
	agent := &mockAgent{
		planResult:   "## Plan\n1. Step one\n2. Step two",
		intentResult: &orchestrator.Intent{SkipPlanning: false},
	}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	ctx := context.Background()
	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process issues: %v", err)
	}

	// Check issue is in plan_review state (waiting for approval)
	issue, err := database.GetIssueByGitHubID(ctx, 400)
	if err != nil {
		t.Fatalf("issue not found in db: %v", err)
	}

	if issue.State != db.StatePlanReview {
		t.Errorf("expected state plan_review, got %s", issue.State)
	}

	// Verify plan was posted
	foundPlan := false
	for _, comment := range w.posted {
		if contains(comment, "Implementation Plan") {
			foundPlan = true
			break
		}
	}
	if !foundPlan {
		t.Error("expected plan to be posted as comment")
	}
}

func TestOrchestrator_QuestionNoStateChange(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Create issue in plan_review state
	issue := &db.Issue{
		GitHubID: 1000,
		Number:   10,
		Title:    "Test Issue",
		Body:     "Issue body",
		State:    db.StatePlanReview,
		WorkDir:  "/work/test",
	}
	if err := database.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}

	// Create a plan
	plan := &db.Plan{
		IssueID: issue.ID,
		Version: 1,
		Content: "Test plan with caching",
	}
	if err := database.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("failed to create test plan: %v", err)
	}

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 1000, Number: 10, Title: "Test Issue", Body: "Issue body"},
		},
		issueComments: []*watcher.Comment{
			{ID: 2001, Body: "How does the caching work?", Author: "maintainer"},
		},
	}
	fm := &mockForkManager{workDir: "/work/test"}
	agent := &mockAgent{
		intentResult: &orchestrator.Intent{IsQuestion: true},
		answerResult: "The caching uses an LRU strategy...",
	}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// Verify state did NOT change
	updated, err := database.GetIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	if updated.State != db.StatePlanReview {
		t.Errorf("expected state to remain plan_review, got %s", updated.State)
	}

	// Verify answer was posted
	if len(w.posted) == 0 {
		t.Error("expected answer to be posted")
	}

	foundAnswer := false
	for _, comment := range w.posted {
		if contains(comment, "caching uses an LRU") {
			foundAnswer = true
			break
		}
	}
	if !foundAnswer {
		t.Error("expected answer about caching to be posted")
	}
}

func TestOrchestrator_PRQuestionNoStateChange(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Create issue in pr_review state with PR
	issue := &db.Issue{
		GitHubID:   1100,
		Number:     11,
		Title:      "Test Issue",
		State:      db.StatePRReview,
		BranchName: "ai-r-sentry/issue-11",
		WorkDir:    "/work/test",
	}
	if err := database.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}

	pr := &db.PullRequest{
		IssueID:  issue.ID,
		GitHubID: 200,
		Number:   20,
		Title:    "Fix #11",
		State:    "open",
	}
	if err := database.CreatePR(ctx, pr); err != nil {
		t.Fatalf("failed to create test PR: %v", err)
	}

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 1100, Number: 11, Title: "Test Issue", Body: "Body"},
		},
		prs: []*watcher.PR{
			{Number: 20, State: "open", Merged: false},
		},
		comments: []*watcher.Comment{
			{ID: 3001, Body: "@ai-r-sentry why did you use a map here?", Author: "maintainer"},
		},
	}
	fm := &mockForkManager{workDir: "/work/test"}
	agent := &mockAgent{
		intentResult: &orchestrator.Intent{IsQuestion: true},
		answerResult: "I used a map for O(1) lookups...",
	}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// Verify state did NOT change
	updated, err := database.GetIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	if updated.State != db.StatePRReview {
		t.Errorf("expected state to remain pr_review, got %s", updated.State)
	}

	// Verify answer was posted to the PR (not the issue)
	foundAnswer := false
	for i, comment := range w.posted {
		if contains(comment, "O(1) lookups") {
			foundAnswer = true
			// Verify it was posted to PR #20, not issue #11
			if w.postedTo[i] != 20 {
				t.Errorf("expected answer to be posted to PR #20, got #%d", w.postedTo[i])
			}
			break
		}
	}
	if !foundAnswer {
		t.Error("expected answer about map usage to be posted")
	}
}

func TestOrchestrator_PRMergeDetection(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Create issue in pr_review state with a PR
	issue := &db.Issue{
		GitHubID:   600,
		Number:     6,
		Title:      "Test Issue",
		State:      db.StatePRReview,
		BranchName: "ai-r-sentry/issue-6",
		WorkDir:    "/work/test",
	}
	if err := database.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}

	pr := &db.PullRequest{
		IssueID:  issue.ID,
		GitHubID: 100,
		Number:   10,
		Title:    "Fix #6",
		State:    "open",
	}
	if err := database.CreatePR(ctx, pr); err != nil {
		t.Fatalf("failed to create test PR: %v", err)
	}

	// Mock returns merged PR
	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 600, Number: 6, Title: "Test Issue", Body: "Body"},
		},
		prs: []*watcher.PR{
			{Number: 10, State: "closed", Merged: true},
		},
	}
	fm := &mockForkManager{workDir: "/work/test"}
	agent := &mockAgent{}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// Verify issue transitioned to done
	updated, err := database.GetIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	if updated.State != db.StateDone {
		t.Errorf("expected state done, got %s", updated.State)
	}
}

func TestOrchestrator_PRBodyWithSummary(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 800, Number: 8, Title: "Add feature X", Body: "Please add X"},
		},
	}
	fm := &mockForkManager{workDir: "/work/test", branch: "ai-r-sentry/issue-8"}
	agent := &mockAgent{
		planResult:    "## Plan\n1. Do thing",
		summaryResult: "- Added feature X\n- Updated tests",
	}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	ctx := context.Background()
	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// Simulate plan approval
	issue, _ := database.GetIssueByGitHubID(ctx, 800)
	if err := orch.ProcessApproval(ctx, issue.ID); err != nil {
		t.Fatalf("failed to process approval: %v", err)
	}

	// Check PR body contains summary
	if len(w.prBodies) == 0 {
		t.Fatal("expected PR to be created")
	}
	prBody := w.prBodies[0]

	if !contains(prBody, "Added feature X") {
		t.Error("expected PR body to contain summary")
	}
	if !contains(prBody, "approved plan") {
		t.Error("expected PR body to mention approved plan")
	}
}

func TestOrchestrator_PRBodySkippedPlanning(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 900, Number: 9, Title: "Quick fix", Body: "Fix typo"},
		},
	}
	fm := &mockForkManager{workDir: "/work/test", branch: "ai-r-sentry/issue-9"}
	agent := &mockAgent{
		intentResult:  &orchestrator.Intent{SkipPlanning: true},
		summaryResult: "- Fixed typo in README",
	}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	ctx := context.Background()
	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// Check PR body
	if len(w.prBodies) == 0 {
		t.Fatal("expected PR to be created")
	}
	prBody := w.prBodies[0]

	if !contains(prBody, "Fixed typo") {
		t.Error("expected PR body to contain summary")
	}
	if contains(prBody, "approved plan") {
		t.Error("should NOT mention approved plan when skipped")
	}
	if !contains(prBody, "directly from issue") {
		t.Error("expected PR body to mention implemented from issue")
	}
}

func TestOrchestrator_PRCloseDetection(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Create issue in pr_review state with a PR
	issue := &db.Issue{
		GitHubID:   700,
		Number:     7,
		Title:      "Test Issue",
		State:      db.StatePRReview,
		BranchName: "ai-r-sentry/issue-7",
		WorkDir:    "/work/test",
	}
	if err := database.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}

	pr := &db.PullRequest{
		IssueID:  issue.ID,
		GitHubID: 101,
		Number:   11,
		Title:    "Fix #7",
		State:    "open",
	}
	if err := database.CreatePR(ctx, pr); err != nil {
		t.Fatalf("failed to create test PR: %v", err)
	}

	// Mock returns closed (not merged) PR
	w := &mockWatcher{
		issues: []*watcher.Issue{
			{ID: 700, Number: 7, Title: "Test Issue", Body: "Body"},
		},
		prs: []*watcher.PR{
			{Number: 11, State: "closed", Merged: false},
		},
	}
	fm := &mockForkManager{workDir: "/work/test"}
	agent := &mockAgent{}

	orch := orchestrator.New(database, w, fm, agent, "owner", "repo")

	err = orch.ProcessIssues(ctx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// Verify issue transitioned to rejected
	updated, err := database.GetIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	if updated.State != db.StateRejected {
		t.Errorf("expected state rejected, got %s", updated.State)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
