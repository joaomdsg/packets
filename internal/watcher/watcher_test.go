package watcher_test

import (
	"context"
	"strings"
	"testing"

	"github.com/joaomdsg/agntpr/internal/watcher"
)

func TestParseIssue(t *testing.T) {
	json := `{
		"id": 12345,
		"number": 42,
		"title": "Test issue",
		"body": "Issue body content",
		"labels": [{"name": "ai-r-sentry"}, {"name": "bug"}],
		"state": "open"
	}`

	issue, err := watcher.ParseIssue([]byte(json))
	if err != nil {
		t.Fatalf("failed to parse issue: %v", err)
	}

	if issue.ID != 12345 {
		t.Errorf("expected ID 12345, got %d", issue.ID)
	}
	if issue.Number != 42 {
		t.Errorf("expected number 42, got %d", issue.Number)
	}
	if issue.Title != "Test issue" {
		t.Errorf("expected title 'Test issue', got %s", issue.Title)
	}
	if !issue.HasLabel("ai-r-sentry") {
		t.Error("expected issue to have ai-r-sentry label")
	}
}

func TestParseComment(t *testing.T) {
	json := `{
		"id": 99999,
		"body": "Hello @ai-r-sentry please help",
		"user": {"login": "testuser"}
	}`

	comment, err := watcher.ParseComment([]byte(json))
	if err != nil {
		t.Fatalf("failed to parse comment: %v", err)
	}

	if comment.ID != 99999 {
		t.Errorf("expected ID 99999, got %d", comment.ID)
	}
	if comment.Author != "testuser" {
		t.Errorf("expected author testuser, got %s", comment.Author)
	}
	if !comment.MentionsAgent("@ai-r-sentry") {
		t.Error("expected comment to mention agent")
	}
}

func TestParsePR(t *testing.T) {
	json := `{
		"number": 10,
		"title": "Fix bug #42",
		"state": "open",
		"merged": false,
		"body": "Closes #42"
	}`

	pr, err := watcher.ParsePR([]byte(json))
	if err != nil {
		t.Fatalf("failed to parse PR: %v", err)
	}

	if pr.Number != 10 {
		t.Errorf("expected number 10, got %d", pr.Number)
	}
	if pr.State != "open" {
		t.Errorf("expected state open, got %s", pr.State)
	}
}

type mockGH struct {
	issues        string
	comments      string
	issueComments string
	prs           string
}

func (m *mockGH) ListOpenIssues(
	ctx context.Context, owner, repo string,
) ([]byte, error) {
	return []byte(m.issues), nil
}

func (m *mockGH) ListIssueComments(
	ctx context.Context, owner, repo string, issueNum int,
) ([]byte, error) {
	return []byte(m.issueComments), nil
}

func (m *mockGH) ListPRComments(
	ctx context.Context, owner, repo string, prNum int,
) ([]byte, error) {
	return []byte(m.comments), nil
}

func (m *mockGH) ListPRReviewComments(
	ctx context.Context, owner, repo string, prNum int,
) ([]byte, error) {
	return []byte(`[]`), nil
}

func (m *mockGH) GetPR(
	ctx context.Context, owner, repo string, prNum int,
) ([]byte, error) {
	return []byte(`{"number": 1, "state": "open", "merged": false}`), nil
}

func (m *mockGH) ListOpenPRs(
	ctx context.Context, owner, repo string,
) ([]byte, error) {
	return []byte(m.prs), nil
}

func (m *mockGH) PostComment(
	ctx context.Context, owner, repo string, num int, body string,
) ([]byte, error) {
	return []byte(`{"id": 12345, "body": "test"}`), nil
}

func (m *mockGH) CreatePR(
	ctx context.Context, owner, repo, title, body, head, base string,
) ([]byte, error) {
	return []byte(`{"id": 1, "number": 1, "title": "Test PR", "state": "open"}`), nil
}

func TestComment_IsApproval(t *testing.T) {
	tests := []struct {
		body     string
		expected bool
	}{
		{"@ai-r-sentry approve", true},
		{"@ai-r-sentry APPROVE", true},
		{"@ai-r-sentry Approve", true},
		{"LGTM! @ai-r-sentry approve", true},
		{"@ai-r-sentry revise: needs more tests", false},
		{"looks good", false},
		{"@someone-else approve", false},
	}

	for _, tc := range tests {
		comment := &watcher.Comment{Body: tc.body}
		got := comment.IsApproval("@ai-r-sentry")
		if got != tc.expected {
			t.Errorf("IsApproval(%q) = %v, want %v", tc.body, got, tc.expected)
		}
	}
}

func TestComment_IsRevision(t *testing.T) {
	tests := []struct {
		body         string
		expectRevise bool
		expectFB     string
	}{
		{"@ai-r-sentry revise: add more tests", true, "add more tests"},
		{"@ai-r-sentry REVISE: fix the bug", true, "fix the bug"},
		{"Please @ai-r-sentry revise: needs work", true, "needs work"},
		{"@ai-r-sentry approve", false, ""},
		{"looks good", false, ""},
	}

	for _, tc := range tests {
		comment := &watcher.Comment{Body: tc.body}
		gotRevise, gotFB := comment.IsRevision("@ai-r-sentry")
		if gotRevise != tc.expectRevise {
			t.Errorf("IsRevision(%q) revise = %v, want %v", tc.body, gotRevise, tc.expectRevise)
		}
		if gotFB != tc.expectFB {
			t.Errorf("IsRevision(%q) feedback = %q, want %q", tc.body, gotFB, tc.expectFB)
		}
	}
}

func TestGetAuthenticatedUser(t *testing.T) {
	// This test requires gh CLI to be authenticated
	// Skip if not in CI/testing environment with gh auth
	ghCli := watcher.NewGHCli()
	ctx := context.Background()

	username, err := ghCli.GetAuthenticatedUser(ctx)
	if err != nil {
		t.Skipf("gh not authenticated, skipping: %v", err)
	}

	if username == "" {
		t.Error("expected non-empty username")
	}
}

func TestWatcher_FetchMentionedIssues(t *testing.T) {
	mock := &mockGH{
		issues: `[
			{"id": 1, "number": 1, "title": "Issue 1", "body": "Hey @ai-r-sentry please help", "state": "open"},
			{"id": 2, "number": 2, "title": "Issue 2", "body": "No mention here", "state": "open"}
		]`,
	}

	w := watcher.New(mock, "owner", "repo", "@ai-r-sentry")
	issues, err := w.FetchMentionedIssues(context.Background())
	if err != nil {
		t.Fatalf("failed to fetch issues: %v", err)
	}

	if len(issues) != 1 {
		t.Errorf("expected 1 mentioned issue, got %d", len(issues))
	}
}

// Test PR parsing with actual gh pr view --json output format
func TestParsePR_GHOutputFormat(t *testing.T) {
	// This is the actual format returned by: gh pr view --json number,title,state
	ghOutput := `{"number":42,"state":"OPEN","title":"Fix issue #10"}`

	pr, err := watcher.ParsePR([]byte(ghOutput))
	if err != nil {
		t.Fatalf("failed to parse PR: %v", err)
	}

	if pr.Number != 42 {
		t.Errorf("expected number 42, got %d", pr.Number)
	}
	if pr.Title != "Fix issue #10" {
		t.Errorf("expected title 'Fix issue #10', got %s", pr.Title)
	}
}

// Test that PR parsing doesn't fail when extra fields are present
func TestParsePR_ExtraFields(t *testing.T) {
	// gh might return additional fields we don't use
	ghOutput := `{
		"number": 10,
		"title": "Test PR",
		"state": "open",
		"url": "https://github.com/owner/repo/pull/10",
		"headRefName": "feature-branch",
		"baseRefName": "main"
	}`

	pr, err := watcher.ParsePR([]byte(ghOutput))
	if err != nil {
		t.Fatalf("failed to parse PR with extra fields: %v", err)
	}

	if pr.Number != 10 {
		t.Errorf("expected number 10, got %d", pr.Number)
	}
}

// Test comment parsing with actual gh output
func TestParseComment_GHOutputFormat(t *testing.T) {
	ghOutput := `{
		"id": 123456789,
		"body": "@ai-r-sentry approve",
		"user": {"login": "maintainer"},
		"created_at": "2024-01-15T10:30:00Z"
	}`

	comment, err := watcher.ParseComment([]byte(ghOutput))
	if err != nil {
		t.Fatalf("failed to parse comment: %v", err)
	}

	if comment.ID != 123456789 {
		t.Errorf("expected ID 123456789, got %d", comment.ID)
	}
	if comment.Author != "maintainer" {
		t.Errorf("expected author 'maintainer', got %s", comment.Author)
	}
	if comment.Body != "@ai-r-sentry approve" {
		t.Errorf("unexpected body: %s", comment.Body)
	}
}

// Test issue parsing with labels
func TestParseIssue_WithLabels(t *testing.T) {
	ghOutput := `{
		"id": 999,
		"number": 15,
		"title": "Bug report",
		"body": "Something is broken",
		"state": "open",
		"labels": [
			{"name": "bug"},
			{"name": "priority-high"}
		]
	}`

	issue, err := watcher.ParseIssue([]byte(ghOutput))
	if err != nil {
		t.Fatalf("failed to parse issue: %v", err)
	}

	if !issue.HasLabel("bug") {
		t.Error("expected issue to have 'bug' label")
	}
	if !issue.HasLabel("priority-high") {
		t.Error("expected issue to have 'priority-high' label")
	}
	if issue.HasLabel("nonexistent") {
		t.Error("issue should not have 'nonexistent' label")
	}
}

// Test review comment context formatting
func TestComment_Context(t *testing.T) {
	// Regular comment (no file/line)
	regular := &watcher.Comment{
		Body: "Please fix this",
	}
	if regular.IsReviewComment() {
		t.Error("regular comment should not be a review comment")
	}
	if regular.Context() != "Please fix this" {
		t.Errorf("expected body only, got %s", regular.Context())
	}

	// Code review comment (with file/line)
	review := &watcher.Comment{
		Body:     "This variable name is unclear",
		Path:     "internal/handler/api.go",
		Line:     42,
		DiffHunk: "@@ -40,6 +40,8 @@ func HandleRequest() {\n+    x := getData()\n+    return x",
	}
	if !review.IsReviewComment() {
		t.Error("review comment should be identified as review comment")
	}
	ctx := review.Context()
	if !strings.Contains(ctx, "File: internal/handler/api.go") {
		t.Error("context should contain file path")
	}
	if !strings.Contains(ctx, "Line: 42") {
		t.Error("context should contain line number")
	}
	if !strings.Contains(ctx, "This variable name is unclear") {
		t.Error("context should contain comment body")
	}
}

// Test PostComment returns comment ID correctly
func TestWatcher_PostComment_ReturnsID(t *testing.T) {
	mock := &mockGH{}
	w := watcher.New(mock, "owner", "repo", "@ai-r-sentry")

	id, err := w.PostComment(context.Background(), 1, "test comment")
	if err != nil {
		t.Fatalf("failed to post comment: %v", err)
	}

	if id != 12345 {
		t.Errorf("expected comment ID 12345, got %d", id)
	}
}
