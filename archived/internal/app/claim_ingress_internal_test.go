package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

// validClaimTarget is the canonical in-flight claim the lifecycle tests submit
// through the in-process ingress (the host-side equivalent of an authenticated
// peer publishing the same encoded ClaimRecord over the NATS socket). Used by
// the internal lifecycle tests (which swap/read unexported seams, so they stay in
// the internal test package), so this helper stays here rather than moving with
// the one test that no longer needs it (the test asserting the unauthenticated
// HTTP claim edge is retired).
var validClaimTarget = ledger.Target{BaseRev: "basesha", FixRev: "fixsha", TipRev: "fixsha", Path: "adult.go", Line: 4}

// publishClaim submits a claim to key's grant-confined claim subtree via the
// authenticated NATS ingress's in-process equivalent (ledger.PublishClaim on the
// shared fabric). Claim submission is NATS-only — there is no HTTP
// claim edge to post to — so tests publish here exactly as the host does.
func publishClaim(t *testing.T, key string, tgt ledger.Target) {
	t.Helper()
	_, err := ledger.PublishClaim(context.Background(), liveFabric, key, LedgerInstance, ledger.ClaimRecord{Target: tgt})
	require.NoError(t, err)
}
