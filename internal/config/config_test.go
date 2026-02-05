package config_test

import (
	"testing"
	"time"

	"github.com/joaomdsg/agntpr/internal/agent"
	"github.com/joaomdsg/agntpr/internal/config"
)

func TestLoad_RequiredEnvVars(t *testing.T) {
	t.Run("fails without GITHUB_TOKEN", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("TARGET_REPO", "owner/repo")

		_, err := config.Load()
		if err == nil {
			t.Fatal("expected error for missing GITHUB_TOKEN")
		}
	})

	t.Run("fails without TARGET_REPO", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "ghp_test")
		t.Setenv("TARGET_REPO", "")

		_, err := config.Load()
		if err == nil {
			t.Fatal("expected error for missing TARGET_REPO")
		}
	})

	t.Run("fails with invalid TARGET_REPO format", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "ghp_test")
		t.Setenv("TARGET_REPO", "invalid-format")

		_, err := config.Load()
		if err == nil {
			t.Fatal("expected error for invalid TARGET_REPO format")
		}
	})
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("TARGET_REPO", "owner/repo")
	t.Setenv("POLL_INTERVAL", "")
	t.Setenv("WORK_DIR", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PollInterval != 60*time.Second {
		t.Errorf("expected default PollInterval 60s, got %v", cfg.PollInterval)
	}

	if cfg.WorkDir != "/work" {
		t.Errorf("expected default WorkDir /work, got %s", cfg.WorkDir)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_custom")
	t.Setenv("TARGET_REPO", "myorg/myrepo")
	t.Setenv("POLL_INTERVAL", "120")
	t.Setenv("WORK_DIR", "/custom/work")
	t.Setenv("CLAUDE_API_KEY", "sk-ant-test")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GitHubToken != "ghp_custom" {
		t.Errorf("expected GitHubToken ghp_custom, got %s", cfg.GitHubToken)
	}

	if cfg.TargetRepo != "myorg/myrepo" {
		t.Errorf("expected TargetRepo myorg/myrepo, got %s", cfg.TargetRepo)
	}

	if cfg.RepoOwner != "myorg" {
		t.Errorf("expected RepoOwner myorg, got %s", cfg.RepoOwner)
	}

	if cfg.RepoName != "myrepo" {
		t.Errorf("expected RepoName myrepo, got %s", cfg.RepoName)
	}

	if cfg.PollInterval != 120*time.Second {
		t.Errorf("expected PollInterval 120s, got %v", cfg.PollInterval)
	}

	if cfg.WorkDir != "/custom/work" {
		t.Errorf("expected WorkDir /custom/work, got %s", cfg.WorkDir)
	}

	if cfg.ClaudeAPIKey != "sk-ant-test" {
		t.Errorf("expected ClaudeAPIKey sk-ant-test, got %s", cfg.ClaudeAPIKey)
	}
}

func TestLoad_InvalidPollInterval(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("TARGET_REPO", "owner/repo")
	t.Setenv("POLL_INTERVAL", "invalid")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid POLL_INTERVAL")
	}
}

func TestLoad_ClaudeModel(t *testing.T) {
	t.Run("defaults to agent.DefaultModel", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "ghp_test")
		t.Setenv("TARGET_REPO", "owner/repo")
		t.Setenv("CLAUDE_MODEL", "")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.ClaudeModel != agent.DefaultModel {
			t.Errorf("expected default model %s, got %s", agent.DefaultModel, cfg.ClaudeModel)
		}
	})

	t.Run("accepts custom model", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "ghp_test")
		t.Setenv("TARGET_REPO", "owner/repo")
		t.Setenv("CLAUDE_MODEL", "claude-3-haiku-20240307")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.ClaudeModel != "claude-3-haiku-20240307" {
			t.Errorf("expected model claude-3-haiku-20240307, got %s", cfg.ClaudeModel)
		}
	})
}
