package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
)

func claimServer(t *testing.T) *httptest.Server {
	t.Helper()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	return server
}

const validClaimBody = `{"base_rev":"basesha","fix_rev":"fixsha","tip_rev":"fixsha","path":"adult.go","line":4}`

// A producer submits a unit of work to a session's claim subtree via POST /claim:
// the claim is accepted and shows as IN FLIGHT, but mints NOTHING — a submitted
// claim is a pending bet, never a confirmed catch (two-scores at the HTTP edge).
func TestPostClaim_acceptsAClaimAsInFlightNeverAsAConfirmedCatch(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	resp, err := http.Post(server.URL+"/claim", "application/json", strings.NewReader(validClaimBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode, "a valid claim is accepted (202)")

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 1, inflight, "the submitted claim landed on the claim subtree, in flight")
	bal, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, 0, bal, "a submitted claim mints NOTHING — it is a pending bet, never a confirmed catch")
}

// An untrusted producer cannot stream an unbounded body: a payload past the
// claim body cap is rejected at the boundary (400) and publishes nothing, so it
// can neither exhaust server memory nor flood the claim subtree.
func TestPostClaim_rejectsAnOversizedBody(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// A syntactically valid Target prefix followed by megabytes of filler: the
	// MaxBytesReader truncates the stream so the decode fails before it ever
	// finishes reading the body.
	huge := `{"base_rev":"b","fix_rev":"f","path":"a.go","line":1,"line_hash":"` +
		strings.Repeat("A", 2<<20) + `"}`
	resp, err := http.Post(server.URL+"/claim", "application/json", strings.NewReader(huge))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "an oversized body is refused at the boundary")

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 0, inflight, "an oversized body published nothing")
}

// A phantom (unregistered) session cannot receive claims.
func TestPostClaim_refusesAnUnregisteredSession(t *testing.T) {
	server := claimServer(t)
	resp, err := http.Post(server.URL+"/claim?key=ghost", "application/json", strings.NewReader(validClaimBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "an unregistered session key is refused")
}

// A malformed or obviously-incomplete claim is rejected at the boundary and
// nothing is published — a producer can't flood the subtree with garbage.
func TestPostClaim_rejectsMalformedOrIncompleteClaims(t *testing.T) {
	bad := []struct{ name, body string }{
		{"not json", "{not json"},
		{"missing base_rev", `{"fix_rev":"f","path":"a.go","line":4}`},
		{"missing fix_rev", `{"base_rev":"b","path":"a.go","line":4}`},
		{"missing path", `{"base_rev":"b","fix_rev":"f","line":4}`},
		{"non-positive line", `{"base_rev":"b","fix_rev":"f","path":"a.go","line":0}`},
	}
	for _, b := range bad {
		t.Run(b.name, func(t *testing.T) {
			defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
			var server *httptest.Server
			_, log, err := NewServer(LiveConfig{
				RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
				TestCmd: []string{"true"}, LedgerPath: defLogPath,
			}, via.WithTestServer(&server))
			require.NoError(t, err)
			t.Cleanup(func() { _ = log.Close() })

			resp, err := http.Post(server.URL+"/claim", "application/json", strings.NewReader(b.body))
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusBadRequest, resp.StatusCode, "%s must be rejected at the boundary", b.name)

			inflight, err := log.ClaimsInFlight()
			require.NoError(t, err)
			require.Equal(t, 0, inflight, "%s published nothing", b.name)
		})
	}
}
