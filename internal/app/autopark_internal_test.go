package app

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

// testClock is an advanceable clock for driving idle decisions deterministically.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *testClock) Advance(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.now = c.now.Add(d) }

func sessionParked(key string) bool {
	consumerSpawner.mu.Lock()
	defer consumerSpawner.mu.Unlock()
	return consumerSpawner.parked[key] != nil
}

func sessionWarm(key string) bool {
	consumerSpawner.mu.Lock()
	defer consumerSpawner.mu.Unlock()
	return consumerSpawner.socks[key] != nil
}

// autoparkServer boots a live server (setting liveFabric + the default session)
// with claim consumers started under a controllable clock and idle threshold.
func autoparkServer(t *testing.T, clk *testClock, idleAfter time.Duration) {
	t.Helper()
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	consumerSpawner.now = clk.Now
	consumerSpawner.idleAfter = idleAfter
	StartClaimConsumers(ctx, func(LiveConfig) ledger.Verifier { return confirmingVerifier }, 30*time.Second, nil)
}

// noteActivity fires from the claim-resolve hook, which runs inside the verify
// consumer goroutine. stopConsumer/parkIdle JOIN that goroutine while holding
// the spawner lock, so if noteActivity needed the spawner lock, a claim
// resolving during a retire/park would deadlock. It must not take that lock.
func TestNoteActivityNeverBlocksOnTheSpawnerLock(t *testing.T) {
	resetConsumersForTest()
	consumerSpawner.lastActive = map[string]time.Time{}

	consumerSpawner.mu.Lock() // stand in for a stopConsumer/parkIdle join holding the lock
	done := make(chan struct{})
	go func() { consumerSpawner.noteActivity("k"); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		consumerSpawner.mu.Unlock()
		t.Fatal("noteActivity blocked on the spawner lock — a resolving claim during a retire/park join would deadlock")
	}
	consumerSpawner.mu.Unlock()
}

// StartAutoPark's ticker must actually drive the sweep: a session idle past the
// threshold gets parked without anyone calling parkIdle by hand. This is the
// production entry point wired at boot.
func TestStartAutoPark_parksIdleSessionsOnItsTicker(t *testing.T) {
	clk := &testClock{now: time.Unix(8_000_000, 0)}
	autoparkServer(t, clk, 15*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	StartAutoPark(ctx, 15*time.Minute, 10*time.Millisecond)

	clk.Advance(20 * time.Minute)
	require.Eventually(t, func() bool { return sessionParked("default") }, 2*time.Second, 10*time.Millisecond,
		"the auto-park ticker must park a session idle past the threshold on its own")
}

// A session whose consumer has sat idle past the threshold must be parked to
// free it, while one with recent activity stays warm — the whole point of
// auto-park is to shed idle consumers without dropping active ones.
func TestAutoPark_parksAnIdleSessionButKeepsAnActiveOneWarm(t *testing.T) {
	clk := &testClock{now: time.Unix(1_000_000, 0)}
	autoparkServer(t, clk, 15*time.Minute)

	activeLog, err := AddSession("active", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = activeLog.Close() })

	require.True(t, sessionWarm("default"))
	require.True(t, sessionWarm("active"))

	clk.Advance(20 * time.Minute)
	consumerSpawner.noteActivity("active") // "active" was just touched; "default" was not
	consumerSpawner.parkIdle()

	assert.True(t, sessionParked("default"), "a session idle past the threshold must be parked")
	assert.False(t, sessionWarm("default"))
	assert.True(t, sessionWarm("active"), "a recently-active session must stay warm")
	assert.False(t, sessionParked("active"))
}

// A parked session must WAKE and process a claim that arrives while it is
// parked — auto-park must never silently drop peer work; the buffered claim is
// drained once the arrival watcher resumes the consumer.
func TestAutoPark_wakesAParkedSessionWhenAClaimArrives(t *testing.T) {
	clk := &testClock{now: time.Unix(2_000_000, 0)}
	autoparkServer(t, clk, 15*time.Minute)

	expLog, err := AddSession("experiment", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = expLog.Close() })

	clk.Advance(20 * time.Minute)
	consumerSpawner.parkIdle()
	require.True(t, sessionParked("experiment"), "the idle session must be parked before the wake test")
	require.False(t, sessionWarm("experiment"))

	publishClaim(t, "experiment", validClaimTarget)

	require.Eventually(t, func() bool {
		b, err := expLog.Balance()
		return err == nil && b == 1
	}, 5*time.Second, 20*time.Millisecond, "a claim arriving at a parked session must wake it and mint the catch")
	assert.Eventually(t, func() bool { return sessionWarm("experiment") }, 2*time.Second, 20*time.Millisecond,
		"the woken session is warm again")
}

// Tearing the spawner down (reset/shutdown) must stop parked sessions' watchers
// and drop their tickets, not just the warm sockets — otherwise a parked
// watcher outlives the server it belonged to.
func TestAutoPark_resetStopsParkedWatchers(t *testing.T) {
	clk := &testClock{now: time.Unix(4_000_000, 0)}
	autoparkServer(t, clk, 15*time.Minute)

	expLog, err := AddSession("experiment", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = expLog.Close() })

	clk.Advance(20 * time.Minute)
	consumerSpawner.parkIdle()
	require.True(t, sessionParked("experiment"))

	resetConsumersForTest()
	assert.False(t, sessionParked("experiment"), "reset must stop parked watchers and drop their tickets")
	assert.False(t, sessionWarm("experiment"))
}

// Retiring a parked session must stop its arrival watcher and drop its ticket,
// so a later claim can never silently resurrect a session the Lead retired.
func TestAutoPark_retiringAParkedSessionDoesNotResurrectIt(t *testing.T) {
	clk := &testClock{now: time.Unix(3_000_000, 0)}
	autoparkServer(t, clk, 15*time.Minute)

	expLog, err := AddSession("experiment", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = expLog.Close() })

	clk.Advance(20 * time.Minute)
	consumerSpawner.parkIdle()
	require.True(t, sessionParked("experiment"))

	liveReg.Delete("experiment")
	consumerSpawner.stopConsumer("experiment")
	require.False(t, sessionParked("experiment"), "retiring a parked session clears it")
	require.False(t, sessionWarm("experiment"))

	// A claim arriving after retire must not resume the retired session.
	publishClaim(t, "experiment", validClaimTarget)
	assert.Never(t, func() bool { return sessionWarm("experiment") }, 700*time.Millisecond, 50*time.Millisecond,
		"a retired session's stopped watcher must not resurrect it on a later claim")
}
