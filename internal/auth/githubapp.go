package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GitHubAppAuth authenticates as a GitHub App installation
type GitHubAppAuth struct {
	appID          string
	installationID int64
	privateKey     *rsa.PrivateKey
	token          string
	tokenExpiry    time.Time
	botUsername    string
	apiBaseURL     string
	mu             sync.Mutex
}

// NewGitHubAppAuth creates a new GitHub App authenticator
func NewGitHubAppAuth(
	appID string, installationID int64, privateKeyPEM []byte,
) (*GitHubAppAuth, error) {
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return &GitHubAppAuth{
		appID:          appID,
		installationID: installationID,
		privateKey:     privateKey,
		apiBaseURL:     "https://api.github.com",
	}, nil
}

func parsePrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKCS1 first
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}

	// Try PKCS8
	keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse key (tried PKCS1 and PKCS8): %w", err)
	}

	rsaKey, ok := keyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA private key")
	}

	return rsaKey, nil
}

func (g *GitHubAppAuth) generateJWT() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		Issuer:    g.appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(g.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signed, nil
}

func (g *GitHubAppAuth) GetToken(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Return cached token if still valid (with 5-minute buffer)
	if g.token != "" && time.Now().Before(g.tokenExpiry.Add(-5*time.Minute)) {
		return g.token, nil
	}

	// Generate JWT
	jwtToken, err := g.generateJWT()
	if err != nil {
		return "", fmt.Errorf("generate JWT: %w", err)
	}

	// Exchange JWT for installation access token
	token, expiry, username, err := g.getInstallationToken(ctx, jwtToken)
	if err != nil {
		return "", fmt.Errorf("get installation token: %w", err)
	}

	g.token = token
	g.tokenExpiry = expiry
	g.botUsername = username

	return token, nil
}

func (g *GitHubAppAuth) GetBotUsername(ctx context.Context) (string, error) {
	g.mu.Lock()
	username := g.botUsername
	g.mu.Unlock()

	if username != "" {
		return username, nil
	}

	// Trigger token fetch which also sets username
	_, err := g.GetToken(ctx)
	if err != nil {
		return "", err
	}

	g.mu.Lock()
	username = g.botUsername
	g.mu.Unlock()

	return username, nil
}

func (g *GitHubAppAuth) getInstallationToken(
	ctx context.Context, jwtToken string,
) (string, time.Time, string, error) {
	url := fmt.Sprintf(
		"%s/app/installations/%d/access_tokens",
		g.apiBaseURL,
		g.installationID,
	)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", time.Time{}, "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", time.Time{}, "", fmt.Errorf(
			"unexpected status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", time.Time{}, "", fmt.Errorf("parse response: %w", err)
	}

	// Get bot username
	username, err := g.fetchBotUsername(ctx, jwtToken)
	if err != nil {
		return "", time.Time{}, "", fmt.Errorf("fetch bot username: %w", err)
	}

	return result.Token, result.ExpiresAt, username, nil
}

func (g *GitHubAppAuth) fetchBotUsername(
	ctx context.Context, jwtToken string,
) (string, error) {
	url := fmt.Sprintf("%s/app", g.apiBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	// Bot username is slug + "[bot]"
	return result.Slug + "[bot]", nil
}
