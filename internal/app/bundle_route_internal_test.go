package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
)

// gitIn runs git in dir, failing the test on error (offline, no network).
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

// freshGitRepo is an empty host store the producer's bundle ingests INTO — the
// same dir the session's cage Materialize would clone.
func freshGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	return dir
}

// producerCommitBundle builds a real one-commit producer repo and returns a
// `git bundle --all` of it plus the commit SHA — the bytes a cross-process
// producer would upload before claiming.
func producerCommitBundle(t *testing.T) (bundle []byte, sha string) {
	t.Helper()
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "config", "user.email", "p@p")
	gitIn(t, repo, "config", "user.name", "p")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "work.go"), []byte("package work\n"), 0o644))
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-qm", "producer work")
	sha = gitIn(t, repo, "rev-parse", "HEAD")
	bundlePath := filepath.Join(t.TempDir(), "p.bundle")
	gitIn(t, repo, "bundle", "create", bundlePath, "--all")
	b, err := os.ReadFile(bundlePath)
	require.NoError(t, err)
	return b, sha
}

func bundleServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	resetBundleGuardsForTest() // isolate per-producer rate/quota state from prior tests
	repoDir := freshGitRepo(t)
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: repoDir, BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	return server, repoDir
}

// The wire that makes a cross-process producer's commits reachable to the cage: a
// producer uploads a git bundle, the host ingests it into the session repo, and
// the producer's commit SHA is then held by that repo — the precondition for the
// cage to verify a claim naming those revisions (no host egress, council R38).
func TestPostBundle_ingestsAProducerBundleSoItsCommitsResolveInTheSessionRepo(t *testing.T) {
	server, repoDir := bundleServer(t)
	bundle, sha := producerCommitBundle(t)

	resp, err := http.Post(server.URL+"/bundle", "application/octet-stream", bytes.NewReader(bundle))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode, "a valid producer bundle is accepted (202)")

	resolved := exec.Command("git", "-C", repoDir, "rev-parse", "--verify", "--quiet", sha+"^{commit}").Run()
	require.NoError(t, resolved, "after upload the host store holds the producer's commit, ready for a claim")
	// And it landed CONFINED to the session's producer namespace (not the host's
	// own refs) — the session key is the producer id.
	require.Equal(t, sha, gitIn(t, repoDir, "rev-parse", "refs/producers/default/heads/main"),
		"the producer's commit is namespaced under refs/producers/<sessionKey>/")
}

// The producer namespace is keyed by the SESSION, not hardcoded to "default": a
// bundle posted to a keyed session lands under THAT key's namespace and in THAT
// session's repo, so two producers' uploads never collide.
func TestPostBundle_namespacesByTheSessionKeyNotADefault(t *testing.T) {
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: freshGitRepo(t), BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(t.TempDir(), "default.jsonl"),
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	aliceRepo := freshGitRepo(t)
	aliceLog, err := AddSession("alice", LiveConfig{
		RepoDir: aliceRepo, BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = aliceLog.Close() })

	bundle, sha := producerCommitBundle(t)
	resp, err := http.Post(server.URL+"/bundle?key=alice", "application/octet-stream", bytes.NewReader(bundle))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	require.Equal(t, sha, gitIn(t, aliceRepo, "rev-parse", "refs/producers/alice/heads/main"),
		"a keyed session's bundle lands under ITS namespace, in ITS repo")
}

// A phantom (unregistered) session cannot receive a bundle upload — the same gate
// as the claim route.
func TestPostBundle_refusesAnUnregisteredSession(t *testing.T) {
	server, _ := bundleServer(t)
	bundle, _ := producerCommitBundle(t)

	resp, err := http.Post(server.URL+"/bundle?key=ghost", "application/octet-stream", bytes.NewReader(bundle))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "an unregistered session key is refused")
}

// A session with no configured RepoDir must REFUSE a bundle, not let ingest fall
// back to the server process's cwd: an empty store means git runs in the cwd, so
// an unguarded upload would silently write the producer's commits into
// refs/producers/<key>/* of whatever repo the server was launched from. The guard
// rejects it (400) and writes no producer refs to the cwd.
func TestPostBundle_refusesASessionWithNoRepoDirRatherThanWritingToTheProcessCwd(t *testing.T) {
	cwdRepo := freshGitRepo(t)
	t.Chdir(cwdRepo) // the server process cwd is now a real git repo — the trap an empty store would write into

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: "", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(t.TempDir(), "default.jsonl"),
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	bundle, _ := producerCommitBundle(t)
	resp, err := http.Post(server.URL+"/bundle", "application/octet-stream", bytes.NewReader(bundle))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "a session with no RepoDir refuses the upload")

	refs := gitIn(t, cwdRepo, "for-each-ref", "--format=%(refname)")
	require.Empty(t, strings.TrimSpace(refs), "no producer refs leaked into the process cwd repo")
}

// A malformed (non-bundle) payload is refused at the boundary and nothing is
// written to the host store — garbage can't poison the session repo.
func TestPostBundle_rejectsAMalformedBundle(t *testing.T) {
	server, repoDir := bundleServer(t)

	resp, err := http.Post(server.URL+"/bundle", "application/octet-stream", strings.NewReader("not a git bundle"))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "a non-bundle payload is rejected")

	refs := gitIn(t, repoDir, "for-each-ref", "--format=%(refname)")
	require.Empty(t, strings.TrimSpace(refs), "a rejected bundle writes no refs to the host store")
}
