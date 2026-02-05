package fork_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/agntpr/internal/fork"
)

// Test direct mode (GitHub App) - no forking, clone directly
func TestManager_SetupWorkDir_DirectMode(t *testing.T) {
	tmpDir := t.TempDir()
	git := &mockGit{}
	gh := &mockGitHub{}

	// Create manager in direct mode (useFork=false)
	mgr := fork.NewManager(git, gh, tmpDir, "upstream", "origin", "agntpr[bot]", false)

	ctx := context.Background()
	workDir, err := mgr.SetupWorkDir(ctx, "owner", "repo", 1)
	if err != nil {
		t.Fatalf("failed to setup work dir: %v", err)
	}

	// Should NOT fork
	if gh.forked {
		t.Error("should not fork in direct mode")
	}

	// Should clone directly from owner/repo
	if !gh.cloned {
		t.Error("expected clone to be called")
	}

	expectedPath := filepath.Join(tmpDir, "owner", "repo", "issue-1")
	if workDir != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, workDir)
	}
}

// Test sync with upstream in direct mode
func TestManager_SyncWithUpstream_DirectMode(t *testing.T) {
	tmpDir := t.TempDir()
	git := &mockGit{}
	gh := &mockGitHub{}

	mgr := fork.NewManager(git, gh, tmpDir, "upstream", "origin", "agntpr[bot]", false)

	workDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	ctx := context.Background()
	err := mgr.SyncWithUpstream(ctx, workDir, "main")
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if !git.fetched {
		t.Error("expected fetch to be called")
	}
}

// Test branch creation in direct mode
func TestManager_CreateBranch_DirectMode(t *testing.T) {
	tmpDir := t.TempDir()
	git := &mockGit{}
	gh := &mockGitHub{}

	mgr := fork.NewManager(git, gh, tmpDir, "upstream", "origin", "agntpr[bot]", false)

	workDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	ctx := context.Background()
	branch, err := mgr.CreateBranch(ctx, workDir, 42)
	if err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Both modes use same branch naming now
	if branch != "agntpr/issue-42" {
		t.Errorf("expected branch agntpr/issue-42, got %s", branch)
	}

	if !git.checkedOut {
		t.Error("expected checkout to be called")
	}
}

// Test push in direct mode
func TestManager_PushBranch_DirectMode(t *testing.T) {
	tmpDir := t.TempDir()
	git := &mockGit{}
	gh := &mockGitHub{}

	mgr := fork.NewManager(git, gh, tmpDir, "upstream", "origin", "agntpr[bot]", false)

	workDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	ctx := context.Background()
	err := mgr.PushBranch(ctx, workDir, "agntpr/issue-42", false)
	if err != nil {
		t.Fatalf("push failed: %v", err)
	}

	if !git.pushed {
		t.Error("expected push to be called")
	}
}
