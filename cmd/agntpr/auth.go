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

	fmt.Printf("[AUTH] Starting auth initialization (mode: %s)\n", cfg.GitHubAuthMode)

	switch cfg.GitHubAuthMode {
	case "app":
		fmt.Println("[AUTH] Using GitHub App mode")

		// Get embedded credentials
		fmt.Println("[AUTH] Loading embedded credentials...")
		appID, err := auth.GetEmbeddedAppID()
		if err != nil {
			return nil, "", fmt.Errorf("get embedded app ID: %w", err)
		}
		fmt.Printf("[AUTH] ✓ App ID: %s\n", appID)

		privateKey, err := auth.GetEmbeddedPrivateKey()
		if err != nil {
			return nil, "", fmt.Errorf("get embedded private key: %w", err)
		}
		fmt.Printf("[AUTH] ✓ Private key loaded (%d bytes)\n", len(privateKey))

		// Create GitHub App auth
		fmt.Printf("[AUTH] Creating auth with installation ID: %d\n", cfg.GitHubAppInstallationID)
		a, err = auth.NewGitHubAppAuth(appID, cfg.GitHubAppInstallationID, privateKey)
		if err != nil {
			return nil, "", fmt.Errorf("create GitHub App auth: %w", err)
		}
		fmt.Println("[AUTH] ✓ GitHub App auth created")

	case "token":
		fmt.Println("[AUTH] Using Token mode")

		// Set token first so gh CLI can use it
		os.Setenv("GH_TOKEN", cfg.GitHubToken)
		os.Setenv("GITHUB_TOKEN", cfg.GitHubToken)
		fmt.Println("[AUTH] ✓ Token set in environment")

		// Get username via fetcher
		fmt.Println("[AUTH] Fetching username via gh CLI...")
		username, err := fetcher.getAuthenticatedUser(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("get authenticated user: %w", err)
		}
		fmt.Printf("[AUTH] ✓ Username: %s\n", username)

		a = auth.NewTokenAuth(cfg.GitHubToken, username)

	default:
		return nil, "", fmt.Errorf("invalid auth mode: %s", cfg.GitHubAuthMode)
	}

	// Get token and set environment variables for gh CLI
	fmt.Println("[AUTH] Fetching installation token from GitHub API...")
	token, err := a.GetToken(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("get token: %w", err)
	}
	fmt.Printf("[AUTH] ✓ Token fetched: %s...\n", token[:min(20, len(token))])

	// Set token for gh CLI and git
	os.Setenv("GH_TOKEN", token)
	os.Setenv("GITHUB_TOKEN", token)
	fmt.Println("[AUTH] ✓ Token set for gh CLI")

	// Get bot username
	fmt.Println("[AUTH] Getting bot username...")
	username, err := a.GetBotUsername(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("get bot username: %w", err)
	}
	fmt.Printf("[AUTH] ✓ Bot username: %s\n", username)

	fmt.Println("[AUTH] ✅ Authentication complete!")
	return a, username, nil
}
