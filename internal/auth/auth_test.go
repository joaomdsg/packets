package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Test TokenAuth implementation
func TestTokenAuth_GetToken(t *testing.T) {
	auth := NewTokenAuth("ghp_test123", "test-user")

	ctx := context.Background()
	token, err := auth.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "ghp_test123" {
		t.Errorf("expected ghp_test123, got %s", token)
	}
}

func TestTokenAuth_GetBotUsername(t *testing.T) {
	auth := NewTokenAuth("ghp_test123", "test-user")

	ctx := context.Background()
	username, err := auth.GetBotUsername(ctx)
	if err != nil {
		t.Fatalf("GetBotUsername failed: %v", err)
	}
	if username != "test-user" {
		t.Errorf("expected test-user, got %s", username)
	}
}

// Test private key parsing
func TestParsePrivateKey_PKCS1(t *testing.T) {
	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Encode as PKCS1 PEM
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	parsed, err := parsePrivateKey(pemData)
	if err != nil {
		t.Fatalf("parse PKCS1 key: %v", err)
	}

	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus doesn't match original")
	}
}

func TestParsePrivateKey_PKCS8(t *testing.T) {
	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Encode as PKCS8 PEM
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal PKCS8: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	parsed, err := parsePrivateKey(pemData)
	if err != nil {
		t.Fatalf("parse PKCS8 key: %v", err)
	}

	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus doesn't match original")
	}
}

func TestParsePrivateKey_Invalid(t *testing.T) {
	_, err := parsePrivateKey([]byte("not a pem"))
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

// Test JWT generation
func TestGitHubAppAuth_GenerateJWT(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	auth := &GitHubAppAuth{
		appID:      "123456",
		privateKey: privateKey,
	}

	jwtToken, err := auth.generateJWT()
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}

	// Parse and verify JWT
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			t.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &privateKey.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse JWT: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims not MapClaims")
	}

	if claims["iss"] != "123456" {
		t.Errorf("wrong issuer: %v", claims["iss"])
	}

	// Check expiry is ~10 minutes from now
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp not float64")
	}
	expTime := time.Unix(int64(exp), 0)
	expectedExp := time.Now().Add(10 * time.Minute)
	if expTime.Before(expectedExp.Add(-1*time.Minute)) ||
		expTime.After(expectedExp.Add(1*time.Minute)) {
		t.Errorf("expiry not ~10 minutes: %v", expTime)
	}
}

// Test token caching
func TestGitHubAppAuth_TokenCaching(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	auth := &GitHubAppAuth{
		appID:          "123456",
		installationID: 789,
		privateKey:     privateKey,
		token:          "cached_token",
		tokenExpiry:    time.Now().Add(1 * time.Hour),
		botUsername:    "test-bot[bot]",
	}

	ctx := context.Background()

	// Should return cached token
	token, err := auth.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "cached_token" {
		t.Errorf("expected cached_token, got %s", token)
	}

	// Should return cached username
	username, err := auth.GetBotUsername(ctx)
	if err != nil {
		t.Fatalf("GetBotUsername failed: %v", err)
	}
	if username != "test-bot[bot]" {
		t.Errorf("expected test-bot[bot], got %s", username)
	}
}

// Test expired token refresh
func TestGitHubAppAuth_ExpiredTokenRefresh(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	auth := &GitHubAppAuth{
		appID:          "123456",
		installationID: 789,
		privateKey:     privateKey,
		token:          "expired_token",
		tokenExpiry:    time.Now().Add(-1 * time.Hour), // Expired
		botUsername:    "test-bot[bot]",
	}

	ctx := context.Background()

	// Should try to refresh (will fail without real API)
	_, err = auth.GetToken(ctx)
	if err == nil {
		t.Error("expected error when refreshing expired token without API")
	}
}

// Test NewGitHubAppAuth
func TestNewGitHubAppAuth_Success(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	auth, err := NewGitHubAppAuth("123456", 789, pemData)
	if err != nil {
		t.Fatalf("NewGitHubAppAuth failed: %v", err)
	}

	if auth.appID != "123456" {
		t.Errorf("expected appID 123456, got %s", auth.appID)
	}
	if auth.installationID != 789 {
		t.Errorf("expected installationID 789, got %d", auth.installationID)
	}
}

func TestNewGitHubAppAuth_InvalidKey(t *testing.T) {
	_, err := NewGitHubAppAuth("123456", 789, []byte("invalid key"))
	if err == nil {
		t.Error("expected error for invalid private key")
	}
}
