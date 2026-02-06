package watcher

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TokenProvider provides fresh GitHub tokens
type TokenProvider interface {
	GetToken(ctx context.Context) (string, error)
}

type GHCli struct {
	auth TokenProvider
}

func NewGHCli(auth TokenProvider) *GHCli {
	return &GHCli{auth: auth}
}

func (g *GHCli) GetAuthenticatedUser(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", "/user", "-q", ".login")
	output, err := g.runCmdWithAuth(ctx, cmd)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}

func (g *GHCli) GetUserPermission(
	ctx context.Context, owner, repo, username string,
) (string, error) {
	// First check if user is the repo owner (for personal repos)
	if username == owner {
		return "admin", nil
	}

	// Try collaborators API (works for direct collaborators)
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/collaborators/%s/permission", owner, repo, username),
		"-q", ".permission")
	output, err := g.runCmdWithAuth(ctx, cmd)
	if err == nil {
		return string(bytes.TrimSpace(output)), nil
	}

	// For orgs, check if user is an org member with write access
	// Try checking org membership
	cmd = exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/orgs/%s/memberships/%s", owner, username),
		"-q", ".role")
	output, err = g.runCmdWithAuth(ctx, cmd)
	if err == nil {
		role := strings.TrimSpace(string(output))
		if role == "admin" {
			return "admin", nil
		}
		if role == "member" {
			// Org members typically have write access to org repos
			return "write", nil
		}
	}

	return "read", nil
}

func (g *GHCli) CloneRepo(ctx context.Context, owner, repo, dest string) error {
	cmd := exec.CommandContext(ctx, "gh", "repo", "clone",
		fmt.Sprintf("%s/%s", owner, repo), dest)
	_, err := g.runCmdWithAuth(ctx, cmd)
	return err
}

func (g *GHCli) ForkRepo(ctx context.Context, owner, repo string) error {
	cmd := exec.CommandContext(ctx, "gh", "repo", "fork",
		fmt.Sprintf("%s/%s", owner, repo), "--clone=false")
	_, err := g.runCmdWithAuth(ctx, cmd)
	return err
}

func (g *GHCli) ListOpenIssues(
	ctx context.Context, owner, repo string,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/issues?state=open", owner, repo),
		"--paginate")
	return g.runCmdWithAuth(ctx, cmd)
}

func (g *GHCli) ListIssueComments(
	ctx context.Context, owner, repo string, issueNum int,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, issueNum),
		"--paginate")
	return g.runCmdWithAuth(ctx, cmd)
}

func (g *GHCli) ListPRComments(
	ctx context.Context, owner, repo string, prNum int,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, prNum),
		"--paginate")
	return g.runCmdWithAuth(ctx, cmd)
}

func (g *GHCli) ListPRReviewComments(
	ctx context.Context, owner, repo string, prNum int,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, prNum),
		"--paginate")
	return g.runCmdWithAuth(ctx, cmd)
}

func (g *GHCli) ListOpenPRs(
	ctx context.Context, owner, repo string,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/pulls?state=open", owner, repo),
		"--paginate")
	return g.runCmdWithAuth(ctx, cmd)
}

func (g *GHCli) GetPR(
	ctx context.Context, owner, repo string, prNum int,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, prNum))
	return g.runCmdWithAuth(ctx, cmd)
}

func (g *GHCli) PostComment(
	ctx context.Context, owner, repo string, num int, body string,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, num),
		"-f", fmt.Sprintf("body=%s", body))
	return g.runCmdWithAuth(ctx, cmd)
}

func (g *GHCli) CreatePR(
	ctx context.Context, owner, repo, title, body, head, base string,
) ([]byte, error) {
	// Create the PR and get the URL
	cmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"-R", fmt.Sprintf("%s/%s", owner, repo),
		"--title", title,
		"--body", body,
		"--head", head,
		"--base", base)
	output, err := g.runCmdWithAuth(ctx, cmd)
	if err != nil {
		return nil, err
	}

	// Extract PR number from URL (e.g., https://github.com/owner/repo/pull/123)
	url := strings.TrimSpace(string(output))
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("unexpected PR URL format: %s", url)
	}
	prNum := parts[len(parts)-1]

	// Fetch PR details as JSON
	cmd = exec.CommandContext(ctx, "gh", "pr", "view", prNum,
		"-R", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "number,title,state")
	return g.runCmdWithAuth(ctx, cmd)
}

func runCmd(cmd *exec.Cmd) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// runCmdWithAuth gets a fresh token and runs the command with it
func (g *GHCli) runCmdWithAuth(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	// Get fresh token from auth provider
	token, err := g.auth.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Set fresh token in command environment
	env := os.Environ()
	env = append(env,
		"GH_TOKEN="+token,
		"GITHUB_TOKEN="+token,
	)
	cmd.Env = env

	return runCmd(cmd)
}
