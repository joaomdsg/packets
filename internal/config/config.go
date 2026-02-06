package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joaomdsg/agntpr/internal/agent"
)

type Config struct {
	// GitHub authentication
	// For app mode: App ID and private key are embedded at build time
	// Only installation ID is configurable at runtime
	GitHubToken             string // For token mode (legacy)
	GitHubAppInstallationID int64  // For app mode (runtime configurable)
	GitHubAuthMode          string // "token" or "app", auto-detected

	// AI Backend configuration
	AIBackend    string // "claude" or "opencode"
	ClaudeAPIKey string
	ClaudeModel  string
	OpenCodeModel string

	TargetRepo   string
	RepoOwner    string
	RepoName     string
	PollInterval time.Duration
	WorkDir      string
	DatabasePath string
	ResetDB      bool
	Debug        bool
}

func Load() (*Config, error) {
	cfg := &Config{
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
		GitHubAuthMode: os.Getenv("GITHUB_AUTH_MODE"),
		AIBackend:      os.Getenv("AI_BACKEND"),
		ClaudeAPIKey:   os.Getenv("CLAUDE_API_KEY"),
		OpenCodeModel:  os.Getenv("OPENCODE_MODEL"),
		TargetRepo:     os.Getenv("TARGET_REPO"),
		WorkDir:        os.Getenv("WORK_DIR"),
		DatabasePath:   os.Getenv("DATABASE_PATH"),
	}

	// Parse installation ID if set (for app mode)
	if installIDStr := os.Getenv("GITHUB_APP_INSTALLATION_ID"); installIDStr != "" {
		installID, err := strconv.ParseInt(installIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid GITHUB_APP_INSTALLATION_ID: %w", err)
		}
		cfg.GitHubAppInstallationID = installID
	}

	// Auto-detect auth mode if not specified
	if cfg.GitHubAuthMode == "" {
		if cfg.GitHubAppInstallationID > 0 {
			cfg.GitHubAuthMode = "app"
		} else if cfg.GitHubToken != "" {
			cfg.GitHubAuthMode = "token"
		} else {
			return nil, fmt.Errorf(
				"either GITHUB_TOKEN or GITHUB_APP_INSTALLATION_ID is required")
		}
	}

	// Validate auth config based on mode
	if cfg.GitHubAuthMode == "app" {
		if cfg.GitHubAppInstallationID == 0 {
			return nil, fmt.Errorf(
				"GITHUB_APP_INSTALLATION_ID is required for app auth")
		}
	} else {
		if cfg.GitHubToken == "" {
			return nil, fmt.Errorf("GITHUB_TOKEN is required for token auth")
		}
	}

	if cfg.TargetRepo == "" {
		return nil, fmt.Errorf("TARGET_REPO is required")
	}

	parts := strings.Split(cfg.TargetRepo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("TARGET_REPO must be in format owner/repo")
	}
	cfg.RepoOwner = parts[0]
	cfg.RepoName = parts[1]

	pollStr := os.Getenv("POLL_INTERVAL")
	if pollStr == "" {
		cfg.PollInterval = 60 * time.Second
	} else {
		poll, err := strconv.Atoi(pollStr)
		if err != nil {
			return nil, fmt.Errorf("invalid POLL_INTERVAL: %w", err)
		}
		cfg.PollInterval = time.Duration(poll) * time.Second
	}

	if cfg.WorkDir == "" {
		cfg.WorkDir = "/work"
	}

	if cfg.DatabasePath == "" {
		cfg.DatabasePath = cfg.WorkDir + "/agntpr.db"
	}

	resetDB := os.Getenv("RESET_DB")
	cfg.ResetDB = resetDB == "1" || resetDB == "true"

	debug := os.Getenv("DEBUG")
	cfg.Debug = debug == "1" || debug == "true"

	// Default to opencode if not specified
	if cfg.AIBackend == "" {
		cfg.AIBackend = "opencode"
	}

	// Validate AI backend
	if cfg.AIBackend != "claude" && cfg.AIBackend != "opencode" {
		return nil, fmt.Errorf("AI_BACKEND must be 'claude' or 'opencode', got: %s", cfg.AIBackend)
	}

	cfg.ClaudeModel = os.Getenv("CLAUDE_MODEL")
	if cfg.ClaudeModel == "" {
		cfg.ClaudeModel = agent.DefaultModel
	}

	return cfg, nil
}
