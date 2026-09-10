// Package tokenstore persists the single Anthropic API key the live server hands
// to the harness it spawns. The key is the one secret the system holds; it is
// written owner-only and is never returned to a browser — Configured reports only
// presence, and Load is for the server process (to inject ANTHROPIC_API_KEY into
// the harness it runs), not for rendering.
package tokenstore

import (
	"fmt"
	"os"
	"strings"
)

// Store is the on-disk home of the Anthropic API key, addressed by one file path.
type Store struct {
	path string
}

// New binds a store to the file the key lives in. The file need not exist yet —
// Load treats absence as the unconfigured state.
func New(path string) *Store {
	return &Store{path: path}
}

// Save writes token to disk owner-only (0600), trimming surrounding whitespace so
// a pasted key's stray newline never reaches the harness env. It replaces any
// previous token rather than appending.
func (s *Store) Save(token string) error {
	if err := os.WriteFile(s.path, []byte(strings.TrimSpace(token)), 0o600); err != nil {
		return fmt.Errorf("tokenstore: save: %v", err)
	}
	return nil
}

// Load returns the stored token, or "" with a nil error when the file is absent —
// a missing key is the unconfigured state, not a failure.
func (s *Store) Load() (string, error) {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("tokenstore: load: %v", err)
	}
	return strings.TrimSpace(string(b)), nil
}

// Configured reports whether a non-empty token is stored. A read error reads as
// unconfigured — the honest signal that the harness has no key to run with.
func (s *Store) Configured() bool {
	tok, err := s.Load()
	return err == nil && tok != ""
}
