package auth

import (
	"strings"
	"testing"
)

func TestParseEmbeddedCredentials(t *testing.T) {
	// Save original and restore after test
	originalCreds := embeddedCredentials
	defer func() { embeddedCredentials = originalCreds }()

	t.Run("valid credentials", func(t *testing.T) {
		embeddedCredentials = `123456
-----BEGIN RSA PRIVATE KEY-----
MIIBogIBAAJBALRiMLAA...
-----END RSA PRIVATE KEY-----`

		appID, privateKey, err := parseEmbeddedCredentials()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if appID != "123456" {
			t.Errorf("expected app ID 123456, got %s", appID)
		}

		expectedKey := `-----BEGIN RSA PRIVATE KEY-----
MIIBogIBAAJBALRiMLAA...
-----END RSA PRIVATE KEY-----`
		if string(privateKey) != expectedKey {
			t.Errorf("private key mismatch")
		}
	})

	t.Run("empty credentials", func(t *testing.T) {
		embeddedCredentials = ""

		_, _, err := parseEmbeddedCredentials()
		if err == nil {
			t.Error("expected error for empty credentials")
		}
		if !strings.Contains(err.Error(), "no embedded credentials") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("invalid format - only one line", func(t *testing.T) {
		embeddedCredentials = "123456"

		_, _, err := parseEmbeddedCredentials()
		if err == nil {
			t.Error("expected error for invalid format")
		}
		if !strings.Contains(err.Error(), "invalid embedded credentials format") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestGetEmbeddedAppID(t *testing.T) {
	// Save original and restore after test
	originalCreds := embeddedCredentials
	defer func() { embeddedCredentials = originalCreds }()

	embeddedCredentials = `789012
-----BEGIN RSA PRIVATE KEY-----
test
-----END RSA PRIVATE KEY-----`

	appID, err := GetEmbeddedAppID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if appID != "789012" {
		t.Errorf("expected app ID 789012, got %s", appID)
	}
}

func TestGetEmbeddedPrivateKey(t *testing.T) {
	// Save original and restore after test
	originalCreds := embeddedCredentials
	defer func() { embeddedCredentials = originalCreds }()

	embeddedCredentials = `123456
-----BEGIN RSA PRIVATE KEY-----
test-key-data
-----END RSA PRIVATE KEY-----`

	privateKey, err := GetEmbeddedPrivateKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `-----BEGIN RSA PRIVATE KEY-----
test-key-data
-----END RSA PRIVATE KEY-----`

	if string(privateKey) != expected {
		t.Errorf("expected key:\n%s\ngot:\n%s", expected, privateKey)
	}
}
