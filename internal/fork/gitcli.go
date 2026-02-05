package fork

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type GitCli struct{}

func NewGitCli() *GitCli {
	return &GitCli{}
}

func (g *GitCli) Clone(ctx context.Context, url, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", url, dest)
	return runGitCmd(cmd)
}

func (g *GitCli) Fetch(ctx context.Context, dir, remote string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "fetch", remote)
	return runGitCmd(cmd)
}

func (g *GitCli) CheckoutBranch(
	ctx context.Context, dir, branch string, create bool,
) error {
	args := []string{"-C", dir, "checkout"}
	if create {
		args = append(args, "-B")
	}
	args = append(args, branch)

	cmd := exec.CommandContext(ctx, "git", args...)
	return runGitCmd(cmd)
}

func (g *GitCli) ResetHard(ctx context.Context, dir, ref string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "reset", "--hard", ref)
	return runGitCmd(cmd)
}

func (g *GitCli) Push(
	ctx context.Context, dir, remote, branch string, force bool,
) error {
	args := []string{"-C", dir, "push", "-u", remote, branch}
	if force {
		args = append(args[:3], append([]string{"--force"}, args[3:]...)...)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	return runGitCmd(cmd)
}

func (g *GitCli) AddRemote(ctx context.Context, dir, name, url string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "remote", "add", name, url)
	err := runGitCmd(cmd)
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func (g *GitCli) RemoteURL(
	ctx context.Context, dir, name string,
) (string, error) {
	cmd := exec.CommandContext(ctx,
		"git", "-C", dir, "remote", "get-url", name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (g *GitCli) Commit(
	ctx context.Context, dir, message string,
) error {
	cmd := exec.CommandContext(ctx,
		"git", "-C", dir, "commit", "-m", message)
	return runGitCmd(cmd)
}

func (g *GitCli) AddAll(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "add", "-A")
	return runGitCmd(cmd)
}

func (g *GitCli) HasCommitsAhead(
	ctx context.Context, dir, base string,
) (bool, error) {
	// Count commits ahead of base
	cmd := exec.CommandContext(ctx,
		"git", "-C", dir, "rev-list", "--count", base+"..HEAD")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("%w: %s", err, stderr.String())
	}

	count := strings.TrimSpace(stdout.String())
	return count != "0", nil
}

func runGitCmd(cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}
