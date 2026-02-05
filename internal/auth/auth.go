package auth

import (
	"context"
)

// Auth provides GitHub authentication tokens
type Auth interface {
	// GetToken returns a valid GitHub token for API calls
	GetToken(ctx context.Context) (string, error)
	// GetBotUsername returns the bot's GitHub username
	GetBotUsername(ctx context.Context) (string, error)
}
