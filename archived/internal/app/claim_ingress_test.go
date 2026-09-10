package app_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// The unauthenticated HTTP POST /claim is RETIRED: claims arrive ONLY
// through the authenticated NATS ingress, so anyone who can merely reach the port
// can no longer inject a claim. The route must be gone (not 202).
func TestPostClaim_isRetiredFromTheUnauthenticatedHTTPSurface(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	resp, err := http.Post(server.URL+"/claim", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NotEqual(t, http.StatusAccepted, resp.StatusCode, "the unauthenticated HTTP claim edge must no longer accept claims")
	require.GreaterOrEqual(t, resp.StatusCode, 400, "POST /claim must be a client error now that the route is retired")
}
