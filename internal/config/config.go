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
	GitHubToken  string
	ClaudeAPIKey string
	ClaudeModel  string
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
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		ClaudeAPIKey: os.Getenv("CLAUDE_API_KEY"),
		TargetRepo:   os.Getenv("TARGET_REPO"),
		WorkDir:      os.Getenv("WORK_DIR"),
		DatabasePath: os.Getenv("DATABASE_PATH"),
	}

	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required")
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

	cfg.ClaudeModel = os.Getenv("CLAUDE_MODEL")
	if cfg.ClaudeModel == "" {
		cfg.ClaudeModel = agent.DefaultModel
	}

	return cfg, nil
}
