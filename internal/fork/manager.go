package fork

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Git interface {
	Clone(ctx context.Context, url, dest string) error
	Fetch(ctx context.Context, dir, remote string) error
	CheckoutBranch(ctx context.Context, dir, branch string, create bool) error
	ResetHard(ctx context.Context, dir, ref string) error
	Push(ctx context.Context, dir, remote, branch string, force bool) error
	AddRemote(ctx context.Context, dir, name, url string) error
	RemoteURL(ctx context.Context, dir, name string) (string, error)
	HasCommitsAhead(ctx context.Context, dir, base string) (bool, error)
}

type GitHub interface {
	ForkRepo(ctx context.Context, owner, repo string) error
	CloneRepo(ctx context.Context, owner, repo, dest string) error
}

type Manager struct {
	git        Git
	gh         GitHub
	baseDir    string
	upstream   string
	forkRemote string
	forkOwner  string
}

func NewManager(
	git Git, gh GitHub, baseDir, upstream, forkRemote, forkOwner string,
) *Manager {
	return &Manager{
		git:        git,
		gh:         gh,
		baseDir:    baseDir,
		upstream:   upstream,
		forkRemote: forkRemote,
		forkOwner:  forkOwner,
	}
}

func BranchName(issueNum int) string {
	return fmt.Sprintf("ai-r-sentry/issue-%d", issueNum)
}

func WorkDirPath(baseDir, owner, repo string, issueNum int) string {
	return filepath.Join(baseDir, owner, repo, fmt.Sprintf("issue-%d", issueNum))
}

func (m *Manager) SetupWorkDir(
	ctx context.Context, owner, repo string, issueNum int,
) (string, error) {
	workDir := WorkDirPath(m.baseDir, owner, repo, issueNum)

	if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
		// Fork the repo to our account (idempotent - gh handles existing forks)
		if err := m.gh.ForkRepo(ctx, owner, repo); err != nil {
			return "", fmt.Errorf("fork failed: %w", err)
		}

		// Clone from our fork
		if err := m.gh.CloneRepo(ctx, m.forkOwner, repo, workDir); err != nil {
			return "", fmt.Errorf("clone fork failed: %w", err)
		}

		// Add upstream remote pointing to the original repo
		upstreamURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
		if err := m.git.AddRemote(ctx, workDir, m.upstream, upstreamURL); err != nil {
			return "", fmt.Errorf("add upstream remote failed: %w", err)
		}
	} else {
		if err := m.git.Fetch(ctx, workDir, m.upstream); err != nil {
			return "", fmt.Errorf("fetch failed: %w", err)
		}
	}

	return workDir, nil
}

func (m *Manager) SyncWithUpstream(
	ctx context.Context, workDir, baseBranch string,
) error {
	if err := m.git.Fetch(ctx, workDir, m.upstream); err != nil {
		return fmt.Errorf("fetch upstream failed: %w", err)
	}

	ref := fmt.Sprintf("%s/%s", m.upstream, baseBranch)
	if err := m.git.ResetHard(ctx, workDir, ref); err != nil {
		return fmt.Errorf("reset to upstream failed: %w", err)
	}

	return nil
}

func (m *Manager) CreateBranch(
	ctx context.Context, workDir string, issueNum int,
) (string, error) {
	branch := BranchName(issueNum)

	if err := m.git.CheckoutBranch(ctx, workDir, branch, true); err != nil {
		return "", fmt.Errorf("checkout failed: %w", err)
	}

	return branch, nil
}

func (m *Manager) PushBranch(
	ctx context.Context, workDir, branch string, force bool,
) error {
	if err := m.git.Push(ctx, workDir, m.forkRemote, branch, force); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	return nil
}

func (m *Manager) SetupRemotes(
	ctx context.Context, workDir, upstreamOwner, forkOwner, repo string,
) error {
	upstreamURL := fmt.Sprintf(
		"https://github.com/%s/%s.git", upstreamOwner, repo)
	if err := m.git.AddRemote(ctx, workDir, m.upstream, upstreamURL); err != nil {
		return fmt.Errorf("add upstream remote failed: %w", err)
	}

	forkURL := fmt.Sprintf("https://github.com/%s/%s.git", forkOwner, repo)
	if err := m.git.AddRemote(ctx, workDir, m.forkRemote, forkURL); err != nil {
		return fmt.Errorf("add fork remote failed: %w", err)
	}

	return nil
}

func (m *Manager) HasChanges(
	ctx context.Context, workDir, baseBranch string,
) (bool, error) {
	base := fmt.Sprintf("%s/%s", m.upstream, baseBranch)
	return m.git.HasCommitsAhead(ctx, workDir, base)
}
