package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type GitHubClient interface {
	ListOpenIssues(ctx context.Context, owner, repo string) ([]byte, error)
	ListIssueComments(ctx context.Context, owner, repo string, issueNum int) ([]byte, error)
	ListPRComments(ctx context.Context, owner, repo string, prNum int) ([]byte, error)
	ListPRReviewComments(ctx context.Context, owner, repo string, prNum int) ([]byte, error)
	ListOpenPRs(ctx context.Context, owner, repo string) ([]byte, error)
	GetPR(ctx context.Context, owner, repo string, prNum int) ([]byte, error)
	PostComment(ctx context.Context, owner, repo string, num int, body string) ([]byte, error)
	CreatePR(ctx context.Context, owner, repo, title, body, head, base string) ([]byte, error)
}

type Issue struct {
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (i *Issue) HasLabel(label string) bool {
	for _, l := range i.Labels {
		if l.Name == label {
			return true
		}
	}
	return false
}

type Comment struct {
	ID     int64  `json:"id"`
	Body   string `json:"body"`
	Author string
	User   struct {
		Login string `json:"login"`
	} `json:"user"`
	// Review comment fields (only populated for code review comments)
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
	DiffHunk string `json:"diff_hunk,omitempty"`
}

func (c *Comment) IsReviewComment() bool {
	return c.Path != ""
}

func (c *Comment) Context() string {
	if !c.IsReviewComment() {
		return c.Body
	}
	return fmt.Sprintf("File: %s\nLine: %d\nDiff context:\n```\n%s\n```\n\nComment: %s",
		c.Path, c.Line, c.DiffHunk, c.Body)
}

func (c *Comment) MentionsAgent(mention string) bool {
	return strings.Contains(c.Body, mention)
}

func (c *Comment) IsApproval(mention string) bool {
	body := strings.ToLower(c.Body)
	pattern := strings.ToLower(mention + " approve")
	return strings.Contains(body, pattern)
}

func (c *Comment) IsRevision(mention string) (bool, string) {
	body := c.Body
	pattern := mention + " revise:"
	idx := strings.Index(strings.ToLower(body), strings.ToLower(pattern))
	if idx != -1 {
		feedback := strings.TrimSpace(body[idx+len(pattern):])
		return true, feedback
	}
	return false, ""
}

type PR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Merged bool   `json:"merged"`
}

func ParseIssue(data []byte) (*Issue, error) {
	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

func ParseComment(data []byte) (*Comment, error) {
	var comment Comment
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, err
	}
	comment.Author = comment.User.Login
	return &comment, nil
}

func ParsePR(data []byte) (*PR, error) {
	var pr PR
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

type Watcher struct {
	client  GitHubClient
	owner   string
	repo    string
	mention string
}

func New(
	client GitHubClient, owner, repo, mention string,
) *Watcher {
	return &Watcher{
		client:  client,
		owner:   owner,
		repo:    repo,
		mention: mention,
	}
}

func (w *Watcher) FetchMentionedIssues(ctx context.Context) ([]*Issue, error) {
	data, err := w.client.ListOpenIssues(ctx, w.owner, w.repo)
	if err != nil {
		return nil, err
	}

	var allIssues []*Issue
	if err := json.Unmarshal(data, &allIssues); err != nil {
		return nil, err
	}

	// Filter for issues that mention the agent
	var mentioned []*Issue
	for _, issue := range allIssues {
		if strings.Contains(issue.Body, w.mention) {
			mentioned = append(mentioned, issue)
		}
	}
	return mentioned, nil
}

func (w *Watcher) FetchOpenPRs(ctx context.Context) ([]*PR, error) {
	data, err := w.client.ListOpenPRs(ctx, w.owner, w.repo)
	if err != nil {
		return nil, err
	}

	var prs []*PR
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

func (w *Watcher) FetchPRComments(
	ctx context.Context, prNum int,
) ([]*Comment, error) {
	data, err := w.client.ListPRComments(ctx, w.owner, w.repo, prNum)
	if err != nil {
		return nil, err
	}
	return parseComments(data)
}

func (w *Watcher) FetchPRReviewComments(
	ctx context.Context, prNum int,
) ([]*Comment, error) {
	data, err := w.client.ListPRReviewComments(ctx, w.owner, w.repo, prNum)
	if err != nil {
		return nil, err
	}
	return parseComments(data)
}

func (w *Watcher) FetchAllPRComments(
	ctx context.Context, prNum int,
) ([]*Comment, error) {
	// Fetch regular PR comments (issue-style comments)
	regular, err := w.FetchPRComments(ctx, prNum)
	if err != nil {
		return nil, err
	}

	// Fetch code review comments
	review, err := w.FetchPRReviewComments(ctx, prNum)
	if err != nil {
		return nil, err
	}

	// Combine both
	all := append(regular, review...)
	return all, nil
}

func (w *Watcher) FilterMentionedComments(comments []*Comment) []*Comment {
	var mentioned []*Comment
	for _, c := range comments {
		if c.MentionsAgent(w.mention) {
			mentioned = append(mentioned, c)
		}
	}
	return mentioned
}

func (w *Watcher) FetchIssueComments(
	ctx context.Context, issueNum int,
) ([]*Comment, error) {
	data, err := w.client.ListIssueComments(ctx, w.owner, w.repo, issueNum)
	if err != nil {
		return nil, err
	}
	return parseComments(data)
}

func parseComments(data []byte) ([]*Comment, error) {
	var comments []*Comment
	if err := json.Unmarshal(data, &comments); err != nil {
		return nil, err
	}
	for _, c := range comments {
		c.Author = c.User.Login
	}
	return comments, nil
}

func (w *Watcher) PostComment(
	ctx context.Context, num int, body string,
) (int64, error) {
	data, err := w.client.PostComment(ctx, w.owner, w.repo, num, body)
	if err != nil {
		return 0, err
	}
	comment, err := ParseComment(data)
	if err != nil {
		return 0, err
	}
	return comment.ID, nil
}

func (w *Watcher) Mention() string {
	return w.mention
}

func (w *Watcher) CreatePR(
	ctx context.Context, title, body, head, base string,
) (*PR, error) {
	data, err := w.client.CreatePR(
		ctx, w.owner, w.repo, title, body, head, base)
	if err != nil {
		return nil, err
	}
	return ParsePR(data)
}

func (w *Watcher) GetPR(ctx context.Context, prNum int) (*PR, error) {
	data, err := w.client.GetPR(ctx, w.owner, w.repo, prNum)
	if err != nil {
		return nil, err
	}
	return ParsePR(data)
}
