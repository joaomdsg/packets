package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// The producer-auth boundary, proven END TO END over a real socket: an EXTERNAL
// NATS client authenticates with grant credentials, JetStream-publishes a claim
// to its grant-confined claim subtree, and the host's consumer drains, verifies,
// and mints it — while a publish to the MINTED subtree (a forged catch) is denied
// by the grant. This closes the "tested by contract, not end-to-end" gap left by
// R81 (connect-only) and R82 (in-process publish). NOT parallel (shared globals).
func TestProducer_authenticatedExternalClaimPublishMintsAndCannotForgeAMint(t *testing.T) {
	resetConsumersForTest()
	grant := NewProducerGrant(defaultSessionKey, "prodA", "pwA")
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: filepath.Join(t.TempDir(), "default.jsonl"),
		ListenAddr: "127.0.0.1:0", Grants: []fabric.ProducerGrant{grant},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	StartClaimConsumers(ctx, func(LiveConfig) ledger.Verifier { return confirmingVerifier }, 30*time.Second, nil)

	// An external producer authenticates over the real listen socket.
	pc, err := nats.Connect(liveFabric.Addr(), nats.UserInfo("prodA", "pwA"))
	require.NoError(t, err)
	t.Cleanup(pc.Close)
	pjs, err := pc.JetStream()
	require.NoError(t, err)

	data, err := json.Marshal(ledger.ClaimRecord{Target: validClaimTarget})
	require.NoError(t, err)

	// ALLOWED: a claim to the producer's own claim subtree ("work" is the claim
	// kind subjectKindClaim publishes under). It is consumed, verified, and minted.
	pubCtx, pcancel := context.WithTimeout(ctx, 3*time.Second)
	defer pcancel()
	_, err = pjs.Publish(fabric.EventSubject(defaultSessionKey, ledgerInstance, fabric.StatusClaim, "work"), data, nats.Context(pubCtx))
	require.NoError(t, err, "a granted producer may publish to its own claim subtree")

	require.Eventually(t, func() bool {
		b, err := log.Balance()
		return err == nil && b == 1
	}, 5*time.Second, 25*time.Millisecond,
		"an authenticated external producer's claim is consumed and minted by the host")

	// DENIED: a producer cannot publish to the MINTED subtree — only the host mints.
	forgeCtx, fcancel := context.WithTimeout(ctx, 2*time.Second)
	defer fcancel()
	_, ferr := pjs.Publish(fabric.EventSubject(defaultSessionKey, ledgerInstance, fabric.StatusMinted, "catch"), data, nats.Context(forgeCtx))
	require.Error(t, ferr, "a producer publishing to the minted subtree must be denied — it can never forge a catch")
}
