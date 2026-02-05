package fork_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/agntpr/internal/fork"
)

func TestBranchName(t *testing.T) {
	name := fork.BranchName(42)
	expected := "agntpr/issue-42"

	if name != expected {
		t.Errorf("expected %s, got %s", expected, name)
	}
}

func TestWorkDirPath(t *testing.T) {
	path := fork.WorkDirPath("/work", "owner", "repo", 42)
	expected := "/work/owner/repo/issue-42"

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

type mockGit struct {
	cloned      bool
	fetched     bool
	checkedOut  bool
	pushed      bool
	currentDir  string
}

func (m *mockGit) Clone(ctx context.Context, url, dest string) error {
	m.cloned = true
	m.currentDir = dest
	return os.MkdirAll(dest, 0755)
}

func (m *mockGit) Fetch(ctx context.Context, dir, remote string) error {
	m.fetched = true
	return nil
}

func (m *mockGit) CheckoutBranch(
	ctx context.Context, dir, branch string, create bool,
) error {
	m.checkedOut = true
	return nil
}

func (m *mockGit) ResetHard(ctx context.Context, dir, ref string) error {
	return nil
}

func (m *mockGit) Push(
	ctx context.Context, dir, remote, branch string, force bool,
) error {
	m.pushed = true
	return nil
}

func (m *mockGit) AddRemote(ctx context.Context, dir, name, url string) error {
	return nil
}

func (m *mockGit) RemoteURL(ctx context.Context, dir, name string) (string, error) {
	return "https://github.com/fork/repo.git", nil
}

func (m *mockGit) HasCommitsAhead(ctx context.Context, dir, base string) (bool, error) {
	return true, nil
}

type mockGitHub struct {
	forked bool
	cloned bool
}

func (m *mockGitHub) ForkRepo(ctx context.Context, owner, repo string) error {
	m.forked = true
	return nil
}

func (m *mockGitHub) CloneRepo(ctx context.Context, owner, repo, dest string) error {
	m.cloned = true
	return os.MkdirAll(dest, 0755)
}

func TestManager_SetupWorkDir_NewClone(t *testing.T) {
	tmpDir := t.TempDir()
	git := &mockGit{}
	gh := &mockGitHub{}

	mgr := fork.NewManager(git, gh, tmpDir, "upstream", "origin", "ai-r-sentry", true)

	ctx := context.Background()
	workDir, err := mgr.SetupWorkDir(ctx, "owner", "repo", 1)
	if err != nil {
		t.Fatalf("failed to setup work dir: %v", err)
	}

	if !gh.forked {
		t.Error("expected fork to be called")
	}

	if !gh.cloned {
		t.Error("expected clone to be called")
	}

	expectedPath := filepath.Join(tmpDir, "owner", "repo", "issue-1")
	if workDir != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, workDir)
	}
}

func TestManager_SetupWorkDir_ExistingClone(t *testing.T) {
	tmpDir := t.TempDir()
	git := &mockGit{}
	gh := &mockGitHub{}

	existingDir := filepath.Join(tmpDir, "owner", "repo", "issue-2")
	if err := os.MkdirAll(filepath.Join(existingDir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	mgr := fork.NewManager(git, gh, tmpDir, "upstream", "origin", "ai-r-sentry", true)

	ctx := context.Background()
	_, err := mgr.SetupWorkDir(ctx, "owner", "repo", 2)
	if err != nil {
		t.Fatalf("failed to setup work dir: %v", err)
	}

	if gh.cloned {
		t.Error("should not clone when directory exists")
	}

	if !git.fetched {
		t.Error("expected fetch to be called")
	}
}

func TestManager_CreateBranch(t *testing.T) {
	tmpDir := t.TempDir()
	git := &mockGit{}
	gh := &mockGitHub{}

	mgr := fork.NewManager(git, gh, tmpDir, "upstream", "origin", "ai-r-sentry", true)

	workDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	ctx := context.Background()
	branch, err := mgr.CreateBranch(ctx, workDir, 5)
	if err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	if branch != "agntpr/issue-5" {
		t.Errorf("expected branch agntpr/issue-5, got %s", branch)
	}

	if !git.checkedOut {
		t.Error("expected checkout to be called")
	}
}
