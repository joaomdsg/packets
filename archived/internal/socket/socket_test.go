package socket_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/socket"
)

// newFabric stands up a host-owned in-process fabric for a test. The socket
// rides it; the host (here, the test) owns its lifecycle.
func newFabric(t *testing.T) *fabric.Fabric {
	t.Helper()
	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// originates publishes on the addr's subtree to prove the fabric is live and
// writable — the socket "originating" a send.
func originates(t *testing.T, f *fabric.Fabric) error {
	t.Helper()
	_, err := f.Publish(context.Background(), fabric.EventSubject("A", "i", fabric.StatusMinted, "catch"), []byte("m"))
	return err
}

// spyAttach is a fake for the socket's injected attachment. Each invocation
// signals runs on start, blocks until its context is cancelled, then signals
// stops on return — so a test can observe exactly when the socket starts and
// stops the addr's work.
type spyAttach struct {
	runs  chan struct{}
	stops chan struct{}
}

func newSpyAttach() *spyAttach {
	return &spyAttach{runs: make(chan struct{}, 8), stops: make(chan struct{}, 8)}
}

func (s *spyAttach) fn(ctx context.Context) error {
	s.runs <- struct{}{}
	<-ctx.Done()
	// Delay the stop signal so that a Park/Close which only cancels (without
	// WAITING for this goroutine to exit) is caught deterministically: the
	// caller's non-blocking assertSignalledNow runs before this send lands and
	// sees an empty channel. A correct impl joins the goroutine, so its
	// Park/Close returns only after this send has completed.
	time.Sleep(30 * time.Millisecond)
	s.stops <- struct{}{}
	return ctx.Err()
}

func awaitSignal(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal(msg)
	}
}

// assertSignalledNow requires the signal to have ALREADY fired (non-blocking):
// used to prove a call returned only AFTER the attach goroutine exited.
func assertSignalledNow(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	default:
		t.Fatal(msg)
	}
}

func TestOpenRunsTheAttach(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()

	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	awaitSignal(t, spy.runs, "Open must start the addr's attach")
	assert.Same(t, fab, s.Fabric(), "the socket exposes the host-owned fabric, not a new one")
	assert.Equal(t, "octocat/hello", s.Addr())
}

func TestOpenRejectsANilFabric(t *testing.T) {
	t.Parallel()
	spy := newSpyAttach()
	_, err := socket.Open(context.Background(), nil, "octocat/hello", spy.fn)
	assert.Error(t, err, "a socket cannot attach onto a nil fabric")
}

func TestOpenRejectsAnEmptyAddr(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()
	_, err := socket.Open(context.Background(), fab, "", spy.fn)
	assert.Error(t, err, "a socket needs a durable addr identity")
}

func TestOpenRejectsANilAttach(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	_, err := socket.Open(context.Background(), fab, "octocat/hello", nil)
	assert.Error(t, err, "a socket with nothing to run is meaningless")
}

// Park must stop the addr's work and Resume must start it again — the warm/idle
// cycle. Park must WAIT for the attach to exit so a resumed attach never races a
// still-running one on the same durable.
func TestParkStopsTheAttachAndResumeRestartsIt(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()

	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)
	awaitSignal(t, spy.runs, "Open must start the attach")

	ticket, err := s.Park()
	require.NoError(t, err)
	assertSignalledNow(t, spy.stops, "Park must WAIT for the attach to stop before returning")

	s2, err := socket.Resume(context.Background(), ticket)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s2.Close() })
	awaitSignal(t, spy.runs, "Resume must start the attach again")
}

// Parking releases the addr's work, never the network: the host-owned fabric
// must stay live and writable across park.
func TestParkLeavesTheSharedFabricLive(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()

	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	awaitSignal(t, spy.runs, "Open must start the attach")

	_, err = s.Park()
	require.NoError(t, err)

	f := s.Fabric()
	require.NotNil(t, f, "Fabric() must stay valid across park")
	assert.Same(t, fab, f)
	assert.NoError(t, originates(t, f), "the shared fabric must still be writable after park")
}

func TestCloseStopsTheAttach(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()

	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)
	awaitSignal(t, spy.runs, "Open must start the attach")

	require.NoError(t, s.Close())
	assertSignalledNow(t, spy.stops, "Close must WAIT for the attach to stop before returning")
}

// The socket never owns the fabric's lifecycle, so closing it must never take
// the host's network down.
func TestClosingASocketNeverClosesTheSharedFabric(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()

	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)

	require.NoError(t, s.Close())
	assert.Same(t, fab, s.Fabric(), "Fabric() must stay valid after Close")
	assert.NoError(t, originates(t, fab), "the host-owned fabric must survive the socket's Close")
}

// An attach can return on its own before the socket ever cancels it (a
// consumer that errors out, say). Close must still return cleanly — its join
// must not block forever waiting on a goroutine that has already exited.
func TestCloseReturnsCleanlyWhenTheAttachAlreadyExited(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	s, err := socket.Open(context.Background(), fab, "octocat/hello", func(ctx context.Context) error { return nil })
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- s.Close() }()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Close must not hang when the attach goroutine already exited")
	}
}

func TestAddrIsStableAcrossParkAndResume(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()
	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)
	addr := s.Addr()

	ticket, err := s.Park()
	require.NoError(t, err)
	assert.Equal(t, addr, s.Addr())

	s2, err := socket.Resume(context.Background(), ticket)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s2.Close() })
	assert.Equal(t, addr, s2.Addr())
}

func TestATicketRedeemsOnlyOnce(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()
	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)

	ticket, err := s.Park()
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	s2, err := socket.Resume(context.Background(), ticket)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s2.Close() })

	_, err = socket.Resume(context.Background(), ticket)
	assert.Error(t, err, "a second Resume on the same ticket must error")
}

func TestCloseIsIdempotentWhileListening(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()
	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)

	require.NoError(t, s.Close())
	assert.NoError(t, s.Close())
}

func TestCloseIsIdempotentWhileParked(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()
	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)

	_, err = s.Park()
	require.NoError(t, err)

	require.NoError(t, s.Close())
	assert.NoError(t, s.Close())
}

func TestParkAfterCloseIsRejected(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()
	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)
	require.NoError(t, s.Close())

	_, err = s.Park()
	assert.Error(t, err)
}

func TestParkingAnAlreadyParkedSocketIsRejected(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()
	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)

	ticket, err := s.Park()
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	_, err = s.Park()
	assert.Error(t, err)

	s2, err := socket.Resume(context.Background(), ticket)
	require.NoError(t, err)
	_ = s2.Close()
}

// Closing a parked socket invalidates its outstanding ticket, so a stale ticket
// can never resurrect an endpoint the owner already tore down.
func TestATicketFromAClosedSocketCannotResume(t *testing.T) {
	t.Parallel()
	fab := newFabric(t)
	spy := newSpyAttach()
	s, err := socket.Open(context.Background(), fab, "octocat/hello", spy.fn)
	require.NoError(t, err)

	ticket, err := s.Park()
	require.NoError(t, err)
	require.NoError(t, s.Close())

	_, err = socket.Resume(context.Background(), ticket)
	assert.Error(t, err, "a ticket from a closed socket must not resume")
}
