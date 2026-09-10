package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Session create must tell a clonable URL from a local path, so a URL triggers a clone
// while a path is used in place (DESIGN §15.2). The common remote forms — https, scp-
// style git@, ssh://, and a bare .git — are URLs; an absolute or relative filesystem
// path is not.
func TestIsRepoURL_distinguishesRemotesFromLocalPaths(t *testing.T) {
	urls := []string{
		"https://github.com/owner/repo.git",
		"https://github.com/owner/repo",
		"git@github.com:owner/repo.git",
		"ssh://git@host/owner/repo.git",
		"http://example.com/r.git",
	}
	for _, u := range urls {
		assert.True(t, isRepoURL(u), "%q is a clonable URL", u)
	}
	paths := []string{"/home/jgonc/repos/packets", "./relative", "myrepo", "", "../up"}
	for _, p := range paths {
		assert.False(t, isRepoURL(p), "%q is a local path, not a URL", p)
	}
}

// The clone lands under a directory named for the repo, so two clones don't collide and
// the local dir is recognizable — derived from the URL's last segment with any .git
// suffix stripped, regardless of remote form.
func TestCloneDirName_derivesTheRepoNameFromAnyRemoteForm(t *testing.T) {
	assert.Equal(t, "repo", cloneDirName("https://github.com/owner/repo.git"))
	assert.Equal(t, "repo", cloneDirName("https://github.com/owner/repo"))
	assert.Equal(t, "repo", cloneDirName("git@github.com:owner/repo.git"))
	assert.Equal(t, "repo", cloneDirName("ssh://git@host/owner/repo.git"))
	assert.Equal(t, "repo", cloneDirName("https://github.com/owner/repo/")) // trailing slash tolerated
}

// A URL with no recognizable repo segment must not yield an empty (or traversal) dir
// name — it falls back to a safe default so the clone target is always a real folder.
func TestCloneDirName_fallsBackOnAnUnparseableURL(t *testing.T) {
	assert.NotEqual(t, "", cloneDirName("https://"))
	assert.NotContains(t, cloneDirName("https://host/.git"), "..")
}

// A URL pick on session create must be CLONED under the repos root into a repo-named
// dir, and that local dir becomes the session's repo (DESIGN §15.2). The clone runs
// through a swappable seam so the test asserts the orchestration without a network.
func TestResolveOrCloneRepo_clonesAURLUnderTheReposRoot(t *testing.T) {
	restore := cloneRepo
	t.Cleanup(func() { cloneRepo = restore })
	var gotURL, gotDest string
	cloneRepo = func(url, dest, _ string) error { gotURL, gotDest = url, dest; return nil }

	got := resolveOrCloneRepo("/srv/repos", "https://github.com/owner/repo.git")
	assert.Equal(t, "/srv/repos/repo", got, "the session points at the cloned local dir")
	assert.Equal(t, "https://github.com/owner/repo.git", gotURL, "the URL is cloned")
	assert.Equal(t, "/srv/repos/repo", gotDest, "into a repo-named dir under the repos root")
}

// A failed clone must not strand the session at a phantom dir — it returns "" so the
// session falls back to inheriting the server's repo. NOT parallel (clone seam).
func TestResolveOrCloneRepo_cloneFailureReturnsEmpty(t *testing.T) {
	restore := cloneRepo
	t.Cleanup(func() { cloneRepo = restore })
	cloneRepo = func(_, _, _ string) error { return assertErr() }

	assert.Equal(t, "", resolveOrCloneRepo("/srv/repos", "https://h/o/repo.git"),
		"a failed clone yields no repo dir")
}

// A local path pick is used in place — never cloned. NOT parallel (clone seam).
func TestResolveOrCloneRepo_localPathIsNotCloned(t *testing.T) {
	restore := cloneRepo
	t.Cleanup(func() { cloneRepo = restore })
	called := false
	cloneRepo = func(_, _, _ string) error { called = true; return nil }

	got := resolveOrCloneRepo("/srv/repos", "/abs/local/path")
	assert.Equal(t, "/abs/local/path", got, "an absolute local path is used as-is")
	assert.False(t, called, "a local path is never cloned")
}

func assertErr() error { return errClone }

var errClone = errorString("clone failed")

type errorString string

func (e errorString) Error() string { return string(e) }

// CloneOnBoot is the CLI entry for -repo: a local path passes through untouched, a URL
// is cloned (returning the local dir), and a clone failure surfaces as an error the CLI
// can fatal on (not a silent fallback — at boot a bad -repo should fail loudly). NOT
// parallel (clone seam).
func TestCloneOnBoot_passesThroughLocalAndClonesURL(t *testing.T) {
	restore := cloneRepo
	t.Cleanup(func() { cloneRepo = restore })

	cloneRepo = func(_, _, _ string) error { t.Fatal("a local path must not be cloned"); return nil }
	got, err := CloneOnBoot("/abs/local", "/srv/repos")
	if err == nil {
		assert.Equal(t, "/abs/local", got, "a local path passes through unchanged")
	}

	var gotDest string
	cloneRepo = func(_, dest, _ string) error { gotDest = dest; return nil }
	got, err = CloneOnBoot("https://h/o/repo.git", "/srv/repos")
	assert.NoError(t, err)
	assert.Equal(t, "/srv/repos/repo", got, "a URL clones and returns the local dir")
	assert.Equal(t, "/srv/repos/repo", gotDest)

	cloneRepo = func(_, _, _ string) error { return errClone }
	_, err = CloneOnBoot("https://h/o/repo.git", "/srv/repos")
	assert.Error(t, err, "a clone failure surfaces as an error to fatal on at boot")
}
