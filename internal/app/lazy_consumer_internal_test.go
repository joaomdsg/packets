package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

// R53 lets a Lead create a session at runtime, but claim consumers were spawned
// only once at boot (a liveReg snapshot), so a runtime-created session got NO
// consumer — its producer claims would publish and never verify. A session
// registered AFTER the consumers start must still get a consumer, so the create
// flow isn't a dead end for the producer path. NOT parallel (shared globals).
func TestClaimConsumers_aRuntimeCreatedSessionStillGetsAConsumer(t *testing.T) {
	claimConsumerServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Consumers start with only the boot (default) session registered.
	StartClaimConsumers(ctx, func(LiveConfig) ledger.Verifier { return confirmingVerifier }, 30*time.Second, nil)

	// A session created AFTER consumers started — the case R53 introduced.
	newLog, err := AddSession("runtimesess", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = newLog.Close() })

	publishClaim(t, "runtimesess", validClaimTarget)

	require.Eventually(t, func() bool {
		b, err := newLog.Balance()
		return err == nil && b == 1
	}, 3*time.Second, 20*time.Millisecond,
		"a session created after consumers started still gets a consumer that verifies + mints its claim")
}
