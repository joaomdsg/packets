package ledger_test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// One embedded fabric is shared across the whole package: each test isolates its
// economy by binding to its OWN session token (t.Name()), so parallel tests never
// see each other's events without each paying for its own embedded server. This
// mirrors production — one fabric, many sessions demuxed by the subject token.
// The server is leaked at process exit (the OS reclaims it); the package never
// closes it, so a test's no-op Close can't tear it down under a parallel peer.
var (
	sharedFabOnce sync.Once
	sharedFab     *fabric.Fabric
)

func fab(t *testing.T) *fabric.Fabric {
	t.Helper()
	sharedFabOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ledger-fabric-*")
		require.NoError(t, err)
		f, err := fabric.Start(context.Background(), dir)
		require.NoError(t, err)
		sharedFab = f
	})
	return sharedFab
}

// openLog binds a Log to this test's isolated session subtree on the shared
// fabric. The second return is the fabric, for a test that re-binds (the restart
// analogue). The Log's Close is a no-op (non-owning), so a deferred close is safe.
func openLog(t *testing.T) (*ledger.Log, *fabric.Fabric) {
	t.Helper()
	f := fab(t)
	l := ledger.Bind(f, t.Name(), "i")
	t.Cleanup(func() { _ = l.Close() })
	return l, f
}

// boundLog is openLog when the test does not need the fabric handle.
func boundLog(t *testing.T) *ledger.Log {
	t.Helper()
	l, _ := openLog(t)
	return l
}

// eventCount replays a session's whole minted economy subtree and returns the
// number of stored events — the stream analogue of "how many lines were written".
func eventCount(t *testing.T, f *fabric.Fabric, session string) int {
	t.Helper()
	events, err := f.ReplaySubject(context.Background(), fabric.EventSubject(session, "i", fabric.StatusMinted, "*"))
	require.NoError(t, err)
	return len(events)
}
