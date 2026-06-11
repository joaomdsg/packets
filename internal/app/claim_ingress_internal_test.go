package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"

	"github.com/joaomdsg/packets/internal/ledger"
)

// validClaimTarget is the canonical in-flight claim the lifecycle tests submit
// through the in-process ingress (the host-side equivalent of an authenticated
// producer publishing the same encoded ClaimRecord over the NATS socket).
var validClaimTarget = ledger.Target{BaseRev: "basesha", FixRev: "fixsha", TipRev: "fixsha", Path: "adult.go", Line: 4}

// publishClaim submits a claim to key's grant-confined claim subtree via the
// authenticated NATS ingress's in-process equivalent (ledger.PublishClaim on the
// shared fabric). Claim submission is NATS-only since R82 — there is no HTTP
// claim edge to post to — so tests publish here exactly as the host does.
func publishClaim(t *testing.T, key string, tgt ledger.Target) {
	t.Helper()
	_, err := ledger.PublishClaim(context.Background(), liveFabric, key, ledgerInstance, ledger.ClaimRecord{Target: tgt})
	require.NoError(t, err)
}

// The unauthenticated HTTP POST /claim is RETIRED (R82): claims arrive ONLY
// through the authenticated NATS ingress, so anyone who can merely reach the port
// can no longer inject a claim. The route must be gone (not 202).
func TestPostClaim_isRetiredFromTheUnauthenticatedHTTPSurface(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	resp, err := http.Post(server.URL+"/claim", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NotEqual(t, http.StatusAccepted, resp.StatusCode, "the unauthenticated HTTP claim edge must no longer accept claims")
	require.GreaterOrEqual(t, resp.StatusCode, 400, "POST /claim must be a client error now that the route is retired")
}
