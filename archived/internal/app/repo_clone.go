package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// cloneRepo is the seam session-create clones a remote repo through (process I/O —
// verified by build + manual run, like runHarness; tests swap it). It clones url into
// dest and, when baseRef is set, checks it out.
var cloneRepo = realCloneRepo

// realCloneRepo clones url into dest and (when baseRef is set) checks out that ref, so
// a session created from a URL works a fresh local tree (DESIGN §15.2). A short-lived
// token is injected into the environment at clone time in production — never a
// long-lived host credential.
func realCloneRepo(url, dest, baseRef string) error {
	clone := exec.Command("git", "clone", url, dest)
	if out, err := clone.CombinedOutput(); err != nil {
		return fmt.Errorf("clone: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if baseRef != "" {
		co := exec.Command("git", "-C", dest, "checkout", baseRef)
		if out, err := co.CombinedOutput(); err != nil {
			return fmt.Errorf("checkout %s: %v: %s", baseRef, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// CloneOnBoot is the CLI entry for the -repo flag (DESIGN §15.2): a local path passes
// through unchanged, while a clonable URL is cloned under reposRoot into a repo-named
// dir and the local dir is returned. Unlike the board path it returns the error rather
// than falling back — a bad -repo at boot should fail loudly, not start a treeless
// server.
func CloneOnBoot(repo, reposRoot string) (string, error) {
	if !isRepoURL(strings.TrimSpace(repo)) {
		return repo, nil
	}
	dest := filepath.Join(reposRoot, cloneDirName(repo))
	if err := cloneRepo(repo, dest, ""); err != nil {
		return "", err
	}
	return dest, nil
}

// resolveOrCloneRepo turns a session-create repo pick into a local repo dir: a clonable
// URL is cloned under reposRoot into a repo-named dir (DESIGN §15.2) and that dir is
// returned; a clone failure returns "" so the session falls back to inheriting the
// server's repo. A non-URL pick resolves as a local path (resolveRepoDir).
func resolveOrCloneRepo(reposRoot, pick string) string {
	pick = strings.TrimSpace(pick)
	if !isRepoURL(pick) {
		return resolveRepoDir(reposRoot, pick)
	}
	dest := filepath.Join(reposRoot, cloneDirName(pick))
	if err := cloneRepo(pick, dest, ""); err != nil {
		return ""
	}
	return dest
}

// isRepoURL reports whether a repo pick is a clonable REMOTE URL (vs a local path):
// session create clones a URL but uses a path in place (DESIGN §15.2). It recognizes
// the common remote forms — https/http, ssh, git protocol, and the scp-style git@host:
// — by prefix; anything else (an absolute or relative filesystem path) is a local path.
func isRepoURL(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "ssh://") ||
		strings.HasPrefix(s, "git://") ||
		strings.HasPrefix(s, "git@")
}

// cloneDirName derives the local directory name a remote URL clones into: the URL's
// last path segment with any ".git" suffix (and trailing slash) stripped, so the clone
// lands in a recognizable, repo-named folder regardless of remote form. An unparseable
// URL with no usable segment falls back to "repo" so the target is always a real,
// non-traversal folder name.
func cloneDirName(url string) string {
	s := strings.TrimSpace(url)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	if i := strings.LastIndexAny(s, "/:"); i >= 0 {
		s = s[i+1:]
	}
	if s == "" || s == "." || s == ".." {
		return "repo"
	}
	return s
}
