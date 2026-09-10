package packet

import (
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
)

// Addr is the repo identity in owner/repo form (concepts.md: addr = repo).
type Addr struct {
	Owner string
	Name  string
}

// String renders the addr as "owner/name".
func (a Addr) String() string {
	return a.Owner + "/" + a.Name
}

// ParseRemoteURL parses a git remote URL into an Addr, without touching the
// filesystem or a subprocess. It accepts the scp-like ssh form
// (git@host:owner/repo[.git]) and the http(s) form
// (scheme://host[:port]/owner/.../repo[.git][/]), including nested path
// groups (e.g. GitLab's owner/subgroup/repo), where every path segment but
// the last becomes Owner and the last becomes Name. It reports false for
// input with no owner+name pair to extract (empty, garbage, or a path with
// fewer than two segments) — callers must never treat a false result as a
// zero-value Addr worth trusting.
func ParseRemoteURL(raw string) (Addr, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Addr{}, false
	}

	var pathPart string
	switch {
	case strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "https://"):
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return Addr{}, false
		}
		pathPart = u.Path
	default:
		// scp-like ssh form: user@host:path. The colon separating host from
		// path must follow the '@', or this isn't that form at all.
		at := strings.Index(raw, "@")
		colon := strings.Index(raw, ":")
		if at < 0 || colon < 0 || colon < at {
			return Addr{}, false
		}
		pathPart = raw[colon+1:]
	}

	pathPart = strings.Trim(pathPart, "/")
	pathPart = strings.TrimSuffix(pathPart, ".git")
	if pathPart == "" {
		return Addr{}, false
	}

	segments := strings.Split(pathPart, "/")
	if len(segments) < 2 {
		return Addr{}, false
	}
	for _, s := range segments {
		if s == "" {
			return Addr{}, false
		}
	}

	name := segments[len(segments)-1]
	owner := strings.Join(segments[:len(segments)-1], "/")
	return Addr{Owner: owner, Name: name}, true
}

// ParseAddr derives the repo identity from repoDir's git origin remote. When
// repoDir has no origin remote (or isn't a git repo at all), it falls back to
// the HONEST local identity Addr{Owner: "local", Name: filepath.Base(repoDir)}
// — it never fabricates an owner for a repo that doesn't declare one.
func ParseAddr(repoDir string) Addr {
	fallback := Addr{Owner: "local", Name: filepath.Base(repoDir)}

	out, err := exec.Command("git", "-C", repoDir, "remote", "get-url", "origin").Output()
	if err != nil {
		return fallback
	}

	addr, ok := ParseRemoteURL(string(out))
	if !ok {
		return fallback
	}
	return addr
}
