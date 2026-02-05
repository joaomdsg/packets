package main

import (
	"context"
	"os"
	"testing"

	"github.com/joaomdsg/agntpr/internal/config"
)

type mockUserFetcher struct {
	username string
	err      error
}

func (m *mockUserFetcher) getAuthenticatedUser(ctx context.Context) (string, error) {
	return m.username, m.err
}

func TestInitAuth_TokenMode(t *testing.T) {
	cfg := &config.Config{
		GitHubAuthMode: "token",
		GitHubToken:    "ghp_test123",
	}

	mockFetcher := &mockUserFetcher{username: "test-user"}

	ctx := context.Background()
	auth, username, err := initAuthWithFetcher(ctx, cfg, mockFetcher)
	if err != nil {
		t.Fatalf("initAuth failed: %v", err)
	}

	if auth == nil {
		t.Fatal("auth is nil")
	}

	if username != "test-user" {
		t.Errorf("expected username test-user, got %s", username)
	}

	// Verify token was set in environment
	if os.Getenv("GH_TOKEN") != "ghp_test123" {
		t.Errorf("GH_TOKEN not set correctly: %s", os.Getenv("GH_TOKEN"))
	}

	if os.Getenv("GITHUB_TOKEN") != "ghp_test123" {
		t.Errorf("GITHUB_TOKEN not set correctly: %s", os.Getenv("GITHUB_TOKEN"))
	}
}

func TestInitAuth_InvalidMode(t *testing.T) {
	cfg := &config.Config{
		GitHubAuthMode: "invalid",
	}

	mockFetcher := &mockUserFetcher{}

	ctx := context.Background()
	_, _, err := initAuthWithFetcher(ctx, cfg, mockFetcher)
	if err == nil {
		t.Error("expected error for invalid auth mode")
	}
}

func TestInitAuth_AppMode_MissingCredentials(t *testing.T) {
	cfg := &config.Config{
		GitHubAuthMode:          "app",
		GitHubAppInstallationID: 789,
	}

	mockFetcher := &mockUserFetcher{}

	ctx := context.Background()
	_, _, err := initAuthWithFetcher(ctx, cfg, mockFetcher)
	if err == nil {
		t.Error("expected error for missing embedded credentials")
	}
}
