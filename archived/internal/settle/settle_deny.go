package settle

import (
	"path/filepath"
	"strings"
)

// PathHit reports a staged touch (add, modify, or delete) to a path denied by
// a protected-path rule — a path-level finding, distinct in kind from
// SecretHit: it fires on the touch itself, never on added-line content.
type PathHit struct {
	File string
	Rule string
}

// handshakeDenyGlobs is the protected-path rule set: the handshake lives under
// handshake/ and the agent's turn must never touch it,
// however it touches it. A package-level var so it can be extended without
// touching the scan logic, mirroring secretRules.
var handshakeDenyGlobs = []string{"handshake/**"}

// scanStagedPaths parses a unified `git diff --cached` and returns a PathHit
// for every file the diff TOUCHES (added, modified, or deleted — both the
// "+++ b/<path>" and "--- a/<path>" headers are read) that matches one of
// denyGlobs. It is a pure function over the diff text, mirroring
// scanStagedDiff's header parsing shape but scanning PATHS rather than added
// LINE CONTENT, so it belongs in its own function rather than folded into
// scanStagedDiff (a content scan and a path scan are different in kind, and
// scanStagedDiff's tested behavior must not be disturbed).
func scanStagedPaths(diff string, denyGlobs []string) []PathHit {
	var hits []PathHit
	seen := map[string]bool{}
	for _, line := range strings.Split(diff, "\n") {
		var file string
		switch {
		case strings.HasPrefix(line, "+++ "):
			file = strings.TrimPrefix(strings.TrimPrefix(line, "+++ "), "b/")
		case strings.HasPrefix(line, "--- "):
			file = strings.TrimPrefix(strings.TrimPrefix(line, "--- "), "a/")
		default:
			continue
		}
		if file == "" || file == "/dev/null" || seen[file] {
			continue
		}
		for _, g := range denyGlobs {
			if matchDenyGlob(g, file) {
				hits = append(hits, PathHit{File: file, Rule: g})
				seen[file] = true
				break
			}
		}
	}
	return hits
}

// matchDenyGlob reports whether path matches glob. A glob ending in "/**"
// is a DIRECTORY-PREFIX match ("handshake/**" matches anything under
// "handshake/", never a bare substring like "handshake-notes.md") since
// path/filepath.Match has no "**" support; any other glob falls back to
// plain path/filepath.Match semantics.
func matchDenyGlob(glob, path string) bool {
	if prefix, ok := strings.CutSuffix(glob, "/**"); ok {
		return strings.HasPrefix(path, prefix+"/")
	}
	ok, err := filepath.Match(glob, path)
	return err == nil && ok
}
