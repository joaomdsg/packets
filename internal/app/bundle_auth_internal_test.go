package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"

	"github.com/joaomdsg/packets/internal/fabric"
)

// postBundle issues a POST /bundle with optional Basic credentials (empty user
// skips auth) and returns the status code.
func postBundle(t *testing.T, url, user, pass string, body []byte) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+"/bundle", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/octet-stream")
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

// When producers are configured, the HTTP bundle blob is authenticated against
// the SAME grant table as the NATS claim ingress: no credentials and wrong
// credentials are refused (401); the granted producer's credentials are accepted
// and the bundle ingests. Producer == session key. NOT parallel (shared globals).
func TestPostBundle_requiresGrantCredentialsWhenProducersAreConfigured(t *testing.T) {
	repoDir := freshGitRepo(t)
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: repoDir, BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(t.TempDir(), "default.jsonl"),
		Grants: []fabric.ProducerGrant{NewProducerGrant(defaultSessionKey, "prodA", "pw")},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	bundle, sha := producerCommitBundle(t)

	require.Equal(t, http.StatusUnauthorized, postBundle(t, server.URL, "", "", bundle),
		"no credentials must be refused when a grant table is configured")
	require.Equal(t, http.StatusUnauthorized, postBundle(t, server.URL, "prodA", "wrong", bundle),
		"a wrong password must be refused")
	require.Equal(t, http.StatusUnauthorized, postBundle(t, server.URL, "ghost", "pw", bundle),
		"an unknown user must be refused")

	require.Equal(t, http.StatusAccepted, postBundle(t, server.URL, "prodA", "pw", bundle),
		"the granted producer's credentials must be accepted")
	resolved := exec.Command("git", "-C", repoDir, "rev-parse", "--verify", "--quiet", sha+"^{commit}").Run()
	require.NoError(t, resolved, "an authenticated upload ingests the producer's commit")
}
