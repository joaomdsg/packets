package app

import (
	"path/filepath"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
)

// NewServer with a ListenAddr binds an AUTHENTICATED NATS socket so cross-process
// producers can submit claims as authenticated clients — the producer-auth
// boundary. A granted producer's credentials connect; a wrong credential is
// refused at connect; and the in-process host economy is unaffected (it still
// reads its own ledger off the same fabric). NOT parallel (shared
// liveReg/liveFabric).
func TestNewServer_bindsAnAuthenticatedListenerForGrantedProducers(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	grant := NewProducerGrant("default", "prodA", "pwA")
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
		ListenAddr: "127.0.0.1:0", Grants: []fabric.ProducerGrant{grant},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	addr := liveFabric.Addr()
	require.NotEmpty(t, addr, "a ListenAddr must bind a real socket the producer can reach")

	// The granted producer authenticates and connects.
	pc, err := nats.Connect(addr, nats.UserInfo("prodA", "pwA"))
	require.NoError(t, err, "the granted producer's credentials must be accepted")
	pc.Close()

	// A wrong credential is refused at the boundary.
	_, err = nats.Connect(addr, nats.UserInfo("prodA", "wrong"))
	assert.Error(t, err, "a wrong credential must be refused — the socket is authenticated, not open")

	// The in-process host economy is unaffected: it still reads its own ledger.
	bal, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, 0, bal, "the in-process host path is unchanged by binding the listener")
}
