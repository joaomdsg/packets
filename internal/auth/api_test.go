package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test successful token exchange
func TestGitHubAppAuth_GetInstallationToken_Success(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Mock GitHub API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/app" {
			// Return app info
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"slug": "test-app",
			})
		} else if r.URL.Path == "/app/installations/789/access_tokens" {
			// Return installation token
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"token":      "ghs_test_installation_token",
				"expires_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			})
		} else {
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	auth := &GitHubAppAuth{
		appID:          "123456",
		installationID: 789,
		privateKey:     privateKey,
		apiBaseURL:     server.URL,
	}

	ctx := context.Background()
	token, err := auth.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	if token != "ghs_test_installation_token" {
		t.Errorf("expected ghs_test_installation_token, got %s", token)
	}

	// Verify username was set
	username, err := auth.GetBotUsername(ctx)
	if err != nil {
		t.Fatalf("GetBotUsername failed: %v", err)
	}
	if username != "test-app[bot]" {
		t.Errorf("expected test-app[bot], got %s", username)
	}
}

// Test API error handling
func TestGitHubAppAuth_GetInstallationToken_APIError(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Mock GitHub API with error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Bad credentials"}`))
	}))
	defer server.Close()

	auth := &GitHubAppAuth{
		appID:          "123456",
		installationID: 789,
		privateKey:     privateKey,
		apiBaseURL:     server.URL,
	}

	ctx := context.Background()
	_, err = auth.GetToken(ctx)
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Test token refresh when near expiry
func TestGitHubAppAuth_RefreshNearExpiry(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/app" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"slug": "test-app",
			})
		} else if r.URL.Path == "/app/installations/789/access_tokens" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"token":      "ghs_new_token",
				"expires_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			})
		}
	}))
	defer server.Close()

	auth := &GitHubAppAuth{
		appID:          "123456",
		installationID: 789,
		privateKey:     privateKey,
		apiBaseURL:     server.URL,
		token:          "old_token",
		tokenExpiry:    time.Now().Add(4 * time.Minute), // Within 5-minute buffer
	}

	ctx := context.Background()
	token, err := auth.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	if token != "ghs_new_token" {
		t.Errorf("expected ghs_new_token (refreshed), got %s", token)
	}

	if callCount < 1 {
		t.Error("expected API call for token refresh")
	}
}
