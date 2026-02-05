package auth

import (
	"context"
)

// TokenAuth uses a static personal access token
type TokenAuth struct {
	token       string
	botUsername string
}

// NewTokenAuth creates a new token-based authenticator
func NewTokenAuth(token, botUsername string) *TokenAuth {
	return &TokenAuth{
		token:       token,
		botUsername: botUsername,
	}
}

func (t *TokenAuth) GetToken(ctx context.Context) (string, error) {
	return t.token, nil
}

func (t *TokenAuth) GetBotUsername(ctx context.Context) (string, error) {
	return t.botUsername, nil
}
