// Package packet's handshake authoring model is the pure record shape for
// the handshake: a runnable contract authored independently of, and
// before, the agent's own code, living under a protected directory the
// agent's turn cannot touch (enforced separately by internal/settle's
// handshake/** deny-rule). This file's only I/O is the file write/read a
// handshake fundamentally requires — everything else (validation, hashing
// via reanchor.HashLines) is pure.
package packet

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/joaomdsg/packets/internal/reanchor"
)

// HandshakeStrength is the gradient a handshake's author BUYS with their own
// effort (examples -> properties/contracts -> proof) — self-declared by the
// human authoring it, never computed or inferred from the content itself.
type HandshakeStrength int

const (
	// StrengthNone is the zero value: no strength has been declared.
	StrengthNone HandshakeStrength = iota
	// StrengthExamples: example-based tests (the cheapest honest rung).
	StrengthExamples
	// StrengthProperties: property/contract-based tests.
	StrengthProperties
)

// String renders the lowercase mono-voice name used across the UI, failing
// safe to "none" for any unrecognized value — an unknown strength is never
// read as a stronger one than was actually declared.
func (s HandshakeStrength) String() string {
	switch s {
	case StrengthExamples:
		return "examples"
	case StrengthProperties:
		return "properties"
	default:
		return "none"
	}
}

// Handshake is one authored contract: the file it lives in, that file's
// content hash at authoring time (VerifyHandshake's before-gates check), and
// its self-declared strength.
type Handshake struct {
	Path     string
	Hash     string
	Strength HandshakeStrength
}

// safeHandshakeName matches the only names WriteHandshake accepts — a caller
// passes an already-safe name or gets an error; this never re-slugs, so a
// name that would need slugging is a caller bug, not silently corrected.
var safeHandshakeName = regexp.MustCompile(`^[a-z0-9_-]+$`)

// HandshakePath is the one protected directory a repo's handshake lives
// under — the same path internal/settle's handshake/** deny-rule scopes to.
func HandshakePath(repoDir string) string {
	return filepath.Join(repoDir, "handshake")
}

// WriteHandshake authors (or re-authors) a handshake file under repoDir's
// protected directory and returns its content hash and declared strength.
// It refuses — writing nothing — an empty content (a handshake authored
// empty is dishonest, not a contract) or a name that isn't
// safeHandshakeName (never re-slugged). Re-authoring the same name
// overwrites the prior content, since strengthening a handshake before
// sending is a normal authoring step, not a distinct operation.
func WriteHandshake(repoDir, name, content string, strength HandshakeStrength) (Handshake, error) {
	if content == "" {
		return Handshake{}, fmt.Errorf("packet: refusing to write an empty handshake %q", name)
	}
	if !safeHandshakeName.MatchString(name) {
		return Handshake{}, fmt.Errorf("packet: unsafe handshake name %q — must match %s", name, safeHandshakeName)
	}
	dir := HandshakePath(repoDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Handshake{}, fmt.Errorf("packet: create handshake dir: %w", err)
	}
	path := filepath.Join(dir, name+".go")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return Handshake{}, fmt.Errorf("packet: write handshake %q: %w", name, err)
	}
	return Handshake{Path: path, Hash: reanchor.HashLines(content), Strength: strength}, nil
}

// VerifyHandshake re-reads h.Path and re-hashes it against h.Hash — the
// content-hash check before gates run (the handshake content-hash-before-gates
// invariant's belt-and-suspenders alongside the settle deny-rule). A mismatch is an
// honest (false, nil): the file changed, which is a real finding, not a
// programming error. A missing/unreadable file is (false, err): that is an
// infra failure distinct from "it changed".
func VerifyHandshake(h Handshake) (bool, error) {
	content, err := os.ReadFile(h.Path)
	if err != nil {
		return false, fmt.Errorf("packet: verify handshake: %w", err)
	}
	return reanchor.HashLines(string(content)) == h.Hash, nil
}
