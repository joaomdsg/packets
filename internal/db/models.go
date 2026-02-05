package db

import "time"

type IssueState string

const (
	StateNew          IssueState = "new"
	StatePlanning     IssueState = "planning"
	StatePlanReview   IssueState = "plan_review"
	StateImplementing IssueState = "implementing"
	StatePRCreated    IssueState = "pr_created"
	StatePRReview     IssueState = "pr_review"
	StateDone         IssueState = "done"
	StateRejected     IssueState = "rejected"
	StateErrored      IssueState = "errored"
)

type Issue struct {
	ID           int64
	GitHubID     int64
	Number       int
	Title        string
	Body         string
	State        IssueState
	BranchName   string
	WorkDir      string
	SkipPlanning bool
	SkipApproval bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Plan struct {
	ID         int64
	IssueID    int64
	Version    int
	Content    string
	Feedback   string
	Approved   bool
	CreatedAt  time.Time
}

type Comment struct {
	ID        int64
	IssueID   int64
	GitHubID  int64
	Author    string
	Body      string
	IsOurs    bool
	CreatedAt time.Time
}

type PullRequest struct {
	ID        int64
	IssueID   int64
	GitHubID  int64
	Number    int
	Title     string
	State     string
	Merged    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
