package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/socket"
)

// parkedAddrs reads the durable parked-socket registry on the live fabric.
func parkedAddrs(t *testing.T) []string {
	t.Helper()
	r, err := socket.OpenParkedRegistry(liveFabric)
	require.NoError(t, err)
	entries, err := r.List()
	require.NoError(t, err)
	addrs := make([]string, 0, len(entries))
	for _, e := range entries {
		addrs = append(addrs, e.Addr)
	}
	return addrs
}

// Parking a session must persist its ticket's routing identity to the durable
// registry — a parked session is known across a restart, not just in memory.
func TestPark_persistsTheParkedIdentity(t *testing.T) {
	clk := &testClock{now: time.Unix(5_000_000, 0)}
	autoparkServer(t, clk, 15*time.Minute)

	expLog, err := AddSession("experiment", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = expLog.Close() })

	activeLog, err := AddSession("active", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = activeLog.Close() })

	clk.Advance(20 * time.Minute)
	consumerSpawner.noteActivity("active")
	consumerSpawner.parkIdle()
	require.True(t, sessionParked("experiment"))
	require.True(t, sessionWarm("active"))

	addrs := parkedAddrs(t)
	assert.Contains(t, addrs, "experiment", "a parked session must be recorded in the durable registry")
	assert.NotContains(t, addrs, "active", "a warm session must NOT be persisted — persistence records parking, not spawning")
}

// Resuming a parked session must drop its persisted record — it is no longer parked.
func TestResume_dropsThePersistedParkedIdentity(t *testing.T) {
	clk := &testClock{now: time.Unix(6_000_000, 0)}
	autoparkServer(t, clk, 15*time.Minute)

	expLog, err := AddSession("experiment", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = expLog.Close() })

	clk.Advance(20 * time.Minute)
	consumerSpawner.parkIdle()
	require.Contains(t, parkedAddrs(t), "experiment")

	publishClaim(t, "experiment", validClaimTarget)
	require.Eventually(t, func() bool { b, err := expLog.Balance(); return err == nil && b == 1 }, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool { return !contains(parkedAddrs(t), "experiment") },
		2*time.Second, 20*time.Millisecond, "resuming a session drops its persisted parked record")
}

// Retiring a parked session must drop its persisted record.
func TestRetire_dropsThePersistedParkedIdentity(t *testing.T) {
	clk := &testClock{now: time.Unix(7_000_000, 0)}
	autoparkServer(t, clk, 15*time.Minute)

	expLog, err := AddSession("experiment", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = expLog.Close() })

	clk.Advance(20 * time.Minute)
	consumerSpawner.parkIdle()
	require.Contains(t, parkedAddrs(t), "experiment")

	liveReg.Delete("experiment")
	consumerSpawner.stopConsumer("experiment")
	assert.NotContains(t, parkedAddrs(t), "experiment", "retiring a session drops its persisted parked record")
}

// The reader that makes persistence load-bearing: a session persisted as parked
// comes back PARKED (not warm) at boot, and a stale entry for a session that no
// longer exists is pruned.
func TestBoot_restoresPersistedParkedSessionsAndPrunesStale(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// Pre-seed the durable registry as if a prior process had parked "default"
	// and left a stale "ghost" that is no longer a registered session.
	r, err := socket.OpenParkedRegistry(liveFabric)
	require.NoError(t, err)
	require.NoError(t, r.Put(socket.ParkedEntry{Addr: "default", Session: "default", Instance: LedgerInstance}))
	require.NoError(t, r.Put(socket.ParkedEntry{Addr: "ghost", Session: "ghost", Instance: LedgerInstance}))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	StartClaimConsumers(ctx, func(LiveConfig) ledger.Verifier { return confirmingVerifier }, 30*time.Second, nil)

	assert.True(t, sessionParked("default"), "a session persisted as parked comes back parked, not warm")
	assert.False(t, sessionWarm("default"))
	assert.Contains(t, parkedAddrs(t), "default")
	assert.NotContains(t, parkedAddrs(t), "ghost", "a stale entry for an unregistered session is pruned at boot")

	// A restored parked session is genuinely armed: a claim still wakes it.
	publishClaim(t, "default", validClaimTarget)
	require.Eventually(t, func() bool { b, err := log.Balance(); return err == nil && b == 1 }, 5*time.Second, 20*time.Millisecond,
		"a boot-restored parked session must still wake and mint on a claim")
}

// The real durability proof: park a session, TEAR THE SERVER DOWN (close the
// fabric), then re-boot on the SAME store dir. The parked identity must survive
// on disk and the session must come back parked — not warm, not lost. This is
// what "persist" actually means, vs a same-fabric simulation.
func TestParkedIdentitySurvivesARealServerRestart(t *testing.T) {
	ledgerPath := filepath.Join(t.TempDir(), "catches") // stable across both boots
	cfg := LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: ledgerPath,
	}
	verifierFor := func(LiveConfig) ledger.Verifier { return confirmingVerifier }

	// --- Boot 1: park the default session, then tear the server down. ---
	resetConsumersForTest()
	clk := &testClock{now: time.Unix(9_000_000, 0)}
	_, log1, err := NewServer(cfg)
	require.NoError(t, err)
	consumerSpawner.now = clk.Now
	consumerSpawner.idleAfter = 15 * time.Minute
	ctx1, cancel1 := context.WithCancel(context.Background())
	StartClaimConsumers(ctx1, verifierFor, 30*time.Second, nil)

	clk.Advance(20 * time.Minute)
	consumerSpawner.parkIdle()
	require.True(t, sessionParked("default"), "default must be parked before the restart")
	require.Contains(t, parkedAddrs(t), "default", "the park must be persisted before the restart")

	cancel1()
	require.NoError(t, log1.Close()) // tears the fabric (and its KV) down to disk

	// --- Boot 2: same store dir, fresh process state. ---
	resetConsumersForTest()
	_, log2, err := NewServer(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = log2.Close() })
	ctx2, cancel2 := context.WithCancel(context.Background())
	t.Cleanup(cancel2)
	StartClaimConsumers(ctx2, verifierFor, 30*time.Second, nil)

	assert.Contains(t, parkedAddrs(t), "default", "the parked identity must survive the fabric teardown on disk")
	assert.True(t, sessionParked("default"), "a session parked before a real restart comes back parked after it")
	assert.False(t, sessionWarm("default"))
}

// The crash consequence: a claim can be on the stream unprocessed when the
// process dies (published, not yet verdicted). On restart, restoring that
// session to PARKED would strand the claim — the arrival watcher only fires on
// NEW claims, so a pre-existing backlog would sit until the next claim. Restore
// must drain a backlog instead of parking blind.
func TestBoot_restoredSessionWithABacklogDrainsItNotStrands(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// A claim is already on the stream, unprocessed (no consumer has run yet) —
	// the mid-flight state a crash leaves behind.
	publishClaim(t, "default", validClaimTarget)

	// And the registry says "default" was parked before the restart.
	r, err := socket.OpenParkedRegistry(liveFabric)
	require.NoError(t, err)
	require.NoError(t, r.Put(socket.ParkedEntry{Addr: "default", Session: "default", Instance: LedgerInstance}))

	// Boot the consumers: restore must NOT strand the buffered claim.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	StartClaimConsumers(ctx, func(LiveConfig) ledger.Verifier { return confirmingVerifier }, 30*time.Second, nil)

	require.Eventually(t, func() bool { b, err := log.Balance(); return err == nil && b == 1 }, 5*time.Second, 20*time.Millisecond,
		"a claim buffered while parked must drain on restore, not sit stranded until the next claim")
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
