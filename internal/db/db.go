package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS issues (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		github_id INTEGER UNIQUE NOT NULL,
		number INTEGER NOT NULL,
		title TEXT NOT NULL,
		body TEXT,
		state TEXT NOT NULL DEFAULT 'new',
		branch_name TEXT,
		work_dir TEXT,
		skip_planning BOOLEAN DEFAULT FALSE,
		skip_approval BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS plans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		issue_id INTEGER NOT NULL REFERENCES issues(id),
		version INTEGER NOT NULL,
		content TEXT NOT NULL,
		approved BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		issue_id INTEGER NOT NULL REFERENCES issues(id),
		github_id INTEGER UNIQUE NOT NULL,
		author TEXT NOT NULL,
		body TEXT NOT NULL,
		is_ours BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS pull_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		issue_id INTEGER NOT NULL REFERENCES issues(id),
		github_id INTEGER UNIQUE NOT NULL,
		number INTEGER NOT NULL,
		title TEXT NOT NULL,
		state TEXT NOT NULL DEFAULT 'open',
		merged BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_issues_github_id ON issues(github_id);
	CREATE INDEX IF NOT EXISTS idx_issues_state ON issues(state);
	CREATE INDEX IF NOT EXISTS idx_plans_issue_id ON plans(issue_id);
	CREATE INDEX IF NOT EXISTS idx_comments_issue_id ON comments(issue_id);
	CREATE INDEX IF NOT EXISTS idx_pull_requests_issue_id ON pull_requests(issue_id);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Run migrations for existing databases (errors ignored - columns may exist)
	migrations := []string{
		`ALTER TABLE issues ADD COLUMN skip_planning BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE issues ADD COLUMN skip_approval BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE plans ADD COLUMN feedback TEXT`,
	}
	for _, m := range migrations {
		db.conn.Exec(m)
	}

	return nil
}

func (db *DB) CreateIssue(ctx context.Context, issue *Issue) error {
	now := time.Now()
	result, err := db.conn.ExecContext(ctx,
		`INSERT INTO issues (github_id, number, title, body, state, branch_name,
		 work_dir, skip_planning, skip_approval, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		issue.GitHubID, issue.Number, issue.Title, issue.Body, issue.State,
		issue.BranchName, issue.WorkDir, issue.SkipPlanning, issue.SkipApproval,
		now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	issue.ID = id
	issue.CreatedAt = now
	issue.UpdatedAt = now
	return nil
}

func (db *DB) GetIssue(ctx context.Context, id int64) (*Issue, error) {
	issue := &Issue{}
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, github_id, number, title, body, state, branch_name,
		 work_dir, skip_planning, skip_approval, created_at, updated_at
		 FROM issues WHERE id = ?`, id).
		Scan(&issue.ID, &issue.GitHubID, &issue.Number, &issue.Title,
			&issue.Body, &issue.State, &issue.BranchName, &issue.WorkDir,
			&issue.SkipPlanning, &issue.SkipApproval,
			&issue.CreatedAt, &issue.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

func (db *DB) GetIssueByGitHubID(ctx context.Context, ghID int64) (*Issue, error) {
	issue := &Issue{}
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, github_id, number, title, body, state, branch_name,
		 work_dir, skip_planning, skip_approval, created_at, updated_at
		 FROM issues WHERE github_id = ?`, ghID).
		Scan(&issue.ID, &issue.GitHubID, &issue.Number, &issue.Title,
			&issue.Body, &issue.State, &issue.BranchName, &issue.WorkDir,
			&issue.SkipPlanning, &issue.SkipApproval,
			&issue.CreatedAt, &issue.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

func (db *DB) UpdateIssueState(ctx context.Context, id int64, state IssueState) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE issues SET state = ?, updated_at = ? WHERE id = ?`,
		state, time.Now(), id)
	return err
}

func (db *DB) UpdateIssueBranch(
	ctx context.Context, id int64, branch, workDir string,
) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE issues SET branch_name = ?, work_dir = ?, updated_at = ?
		 WHERE id = ?`,
		branch, workDir, time.Now(), id)
	return err
}

func (db *DB) UpdateIssueSkips(
	ctx context.Context, id int64, skipPlanning, skipApproval bool,
) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE issues SET skip_planning = ?, skip_approval = ?, updated_at = ?
		 WHERE id = ?`,
		skipPlanning, skipApproval, time.Now(), id)
	return err
}

func (db *DB) ListActiveIssues(ctx context.Context) ([]*Issue, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, github_id, number, title, body, state, branch_name,
		 work_dir, skip_planning, skip_approval, created_at, updated_at
		 FROM issues WHERE state NOT IN ('done', 'rejected', 'errored')
		 ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []*Issue
	for rows.Next() {
		issue := &Issue{}
		if err := rows.Scan(&issue.ID, &issue.GitHubID, &issue.Number,
			&issue.Title, &issue.Body, &issue.State, &issue.BranchName,
			&issue.WorkDir, &issue.SkipPlanning, &issue.SkipApproval,
			&issue.CreatedAt, &issue.UpdatedAt); err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}
	return issues, rows.Err()
}

func (db *DB) CreatePlan(ctx context.Context, plan *Plan) error {
	now := time.Now()
	result, err := db.conn.ExecContext(ctx,
		`INSERT INTO plans (issue_id, version, content, approved, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		plan.IssueID, plan.Version, plan.Content, plan.Approved, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	plan.ID = id
	plan.CreatedAt = now
	return nil
}

func (db *DB) GetLatestPlan(ctx context.Context, issueID int64) (*Plan, error) {
	plan := &Plan{}
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, issue_id, version, content, COALESCE(feedback, ''),
		 approved, created_at
		 FROM plans WHERE issue_id = ? ORDER BY version DESC LIMIT 1`, issueID).
		Scan(&plan.ID, &plan.IssueID, &plan.Version, &plan.Content,
			&plan.Feedback, &plan.Approved, &plan.CreatedAt)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func (db *DB) ApprovePlan(ctx context.Context, planID int64) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE plans SET approved = TRUE WHERE id = ?`, planID)
	return err
}

func (db *DB) CreateComment(ctx context.Context, comment *Comment) error {
	now := time.Now()
	result, err := db.conn.ExecContext(ctx,
		`INSERT INTO comments (issue_id, github_id, author, body, is_ours,
		 created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		comment.IssueID, comment.GitHubID, comment.Author, comment.Body,
		comment.IsOurs, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	comment.ID = id
	comment.CreatedAt = now
	return nil
}

func (db *DB) GetCommentByGitHubID(
	ctx context.Context, ghID int64,
) (*Comment, error) {
	comment := &Comment{}
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, issue_id, github_id, author, body, is_ours, created_at
		 FROM comments WHERE github_id = ?`, ghID).
		Scan(&comment.ID, &comment.IssueID, &comment.GitHubID, &comment.Author,
			&comment.Body, &comment.IsOurs, &comment.CreatedAt)
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (db *DB) CreatePR(ctx context.Context, pr *PullRequest) error {
	now := time.Now()
	result, err := db.conn.ExecContext(ctx,
		`INSERT INTO pull_requests (issue_id, github_id, number, title, state,
		 merged, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		pr.IssueID, pr.GitHubID, pr.Number, pr.Title, pr.State, pr.Merged,
		now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	pr.ID = id
	pr.CreatedAt = now
	pr.UpdatedAt = now
	return nil
}

func (db *DB) GetPRByIssueID(ctx context.Context, issueID int64) (*PullRequest, error) {
	pr := &PullRequest{}
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, issue_id, github_id, number, title, state, merged,
		 created_at, updated_at FROM pull_requests WHERE issue_id = ?`, issueID).
		Scan(&pr.ID, &pr.IssueID, &pr.GitHubID, &pr.Number, &pr.Title,
			&pr.State, &pr.Merged, &pr.CreatedAt, &pr.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (db *DB) UpdatePRState(
	ctx context.Context, id int64, state string, merged bool,
) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE pull_requests SET state = ?, merged = ?, updated_at = ?
		 WHERE id = ?`,
		state, merged, time.Now(), id)
	return err
}

func (db *DB) UpdatePlanFeedback(
	ctx context.Context, planID int64, feedback string,
) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE plans SET feedback = ? WHERE id = ?`,
		feedback, planID)
	return err
}

func (db *DB) GetPlanHistory(
	ctx context.Context, issueID int64,
) ([]*Plan, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, issue_id, version, content, COALESCE(feedback, ''),
		 approved, created_at
		 FROM plans WHERE issue_id = ? ORDER BY version ASC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		plan := &Plan{}
		if err := rows.Scan(&plan.ID, &plan.IssueID, &plan.Version,
			&plan.Content, &plan.Feedback, &plan.Approved,
			&plan.CreatedAt); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}
