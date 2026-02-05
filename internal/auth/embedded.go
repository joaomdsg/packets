package auth

import (
	_ "embed"
	"fmt"
	"strings"
)

// Embedded GitHub App credentials (set at build time)
// These are NOT configurable at runtime for security
//
//go:embed credentials.txt
var embeddedCredentials string

// GetEmbeddedAppID returns the baked-in GitHub App ID
func GetEmbeddedAppID() (string, error) {
	appID, _, err := parseEmbeddedCredentials()
	return appID, err
}

// GetEmbeddedPrivateKey returns the baked-in private key
func GetEmbeddedPrivateKey() ([]byte, error) {
	_, privateKey, err := parseEmbeddedCredentials()
	return privateKey, err
}

func parseEmbeddedCredentials() (string, []byte, error) {
	if embeddedCredentials == "" {
		return "", nil, fmt.Errorf(
			"no embedded credentials found - build with make build")
	}

	lines := strings.Split(strings.TrimSpace(embeddedCredentials), "\n")
	if len(lines) < 2 {
		return "", nil, fmt.Errorf("invalid embedded credentials format")
	}

	appID := strings.TrimSpace(lines[0])
	if appID == "" {
		return "", nil, fmt.Errorf("embedded App ID is empty")
	}

	// Rest of the lines are the private key
	privateKey := []byte(strings.Join(lines[1:], "\n"))
	if len(privateKey) == 0 {
		return "", nil, fmt.Errorf("embedded private key is empty")
	}

	return appID, privateKey, nil
}
