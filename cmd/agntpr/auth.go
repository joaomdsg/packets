package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/joaomdsg/agntpr/internal/auth"
	"github.com/joaomdsg/agntpr/internal/config"
)

// userFetcher fetches authenticated user info
type userFetcher interface {
	getAuthenticatedUser(ctx context.Context) (string, error)
}

// ghCliWrapper wraps gh CLI commands for auth
type ghCliWrapper struct{}

func (g *ghCliWrapper) getAuthenticatedUser(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", "/user", "-q", ".login")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh api failed: %w", err)
	}
	return string(bytes.TrimSpace(output)), nil
}

// initAuth initializes authentication based on config and returns the
// auth instance and bot username
func initAuth(ctx context.Context, cfg *config.Config) (auth.Auth, string, error) {
	return initAuthWithFetcher(ctx, cfg, &ghCliWrapper{})
}

func initAuthWithFetcher(ctx context.Context, cfg *config.Config, fetcher userFetcher) (auth.Auth, string, error) {
	var a auth.Auth
	var err error

	switch cfg.GitHubAuthMode {
	case "app":
		// Get embedded credentials
		appID, err := auth.GetEmbeddedAppID()
		if err != nil {
			return nil, "", fmt.Errorf("get embedded app ID: %w", err)
		}

		privateKey, err := auth.GetEmbeddedPrivateKey()
		if err != nil {
			return nil, "", fmt.Errorf("get embedded private key: %w", err)
		}

		// Create GitHub App auth
		a, err = auth.NewGitHubAppAuth(appID, cfg.GitHubAppInstallationID, privateKey)
		if err != nil {
			return nil, "", fmt.Errorf("create GitHub App auth: %w", err)
		}

	case "token":
		// Set token first so gh CLI can use it
		os.Setenv("GH_TOKEN", cfg.GitHubToken)
		os.Setenv("GITHUB_TOKEN", cfg.GitHubToken)

		// Get username via fetcher
		username, err := fetcher.getAuthenticatedUser(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("get authenticated user: %w", err)
		}

		a = auth.NewTokenAuth(cfg.GitHubToken, username)

	default:
		return nil, "", fmt.Errorf("invalid auth mode: %s", cfg.GitHubAuthMode)
	}

	// Get token and set environment variables for gh CLI
	token, err := a.GetToken(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("get token: %w", err)
	}

	// Set token for gh CLI and git
	os.Setenv("GH_TOKEN", token)
	os.Setenv("GITHUB_TOKEN", token)

	// Get bot username
	username, err := a.GetBotUsername(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("get bot username: %w", err)
	}

	return a, username, nil
}
