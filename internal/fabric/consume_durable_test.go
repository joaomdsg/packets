package fabric_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
)

func startFab(t *testing.T) *fabric.Fabric {
	t.Helper()
	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// A durable consumer must RESUME past the events it already acked when it is
// rebound — so a restart of the claim consumer does NOT replay (and re-verify)
// every claim from the start of the log. This is the re-verification-storm fix.
func TestConsumeDurable_resumesPastAckedEventsOnRebind(t *testing.T) {
	t.Parallel()
	f := startFab(t)
	subj := fabric.EventSubject("s", "i", fabric.StatusClaim, "work")
	filter := fabric.EventSubject("s", "i", fabric.StatusClaim, ">")

	var seqs []uint64
	for i := 0; i < 3; i++ {
		seq, err := f.Publish(context.Background(), subj, []byte(strconv.Itoa(i)))
		require.NoError(t, err)
		seqs = append(seqs, seq)
	}

	// First bind: process (ack) the first two, then stop. handle does the counting
	// and cancels itself, so the ack of the 2nd happens before ConsumeDurable returns.
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	var got1 []uint64
	done1 := make(chan error, 1)
	go func() {
		done1 <- f.ConsumeDurable(ctx1, "claims_s_i", filter, func(e fabric.Event) error {
			got1 = append(got1, e.Seq)
			if len(got1) == 2 {
				cancel1()
			}
			return nil
		})
	}()
	<-done1 // happens-before: got1 is safe to read, and the 2nd ack completed
	require.Equal(t, []uint64{seqs[0], seqs[1]}, got1, "the first bind acks events 1 and 2")

	// Second bind, SAME durable: must deliver only the un-acked 3rd event.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	var got2 []uint64
	done2 := make(chan error, 1)
	go func() {
		done2 <- f.ConsumeDurable(ctx2, "claims_s_i", filter, func(e fabric.Event) error {
			got2 = append(got2, e.Seq)
			cancel2()
			return nil
		})
	}()
	<-done2
	require.Equal(t, []uint64{seqs[2]}, got2, "a rebind resumes past the acked events — only the un-acked 3rd is delivered, never a replay of 1 and 2")
}

// A message the handle does NOT finish (returns an error for) must not be acked:
// it is redelivered so the work is not silently lost. Tested in-run via Nak so it
// does not depend on the ack-wait timer.
func TestConsumeDurable_redeliversAMessageTheHandleRejects(t *testing.T) {
	t.Parallel()
	f := startFab(t)
	subj := fabric.EventSubject("s", "i", fabric.StatusClaim, "work")
	filter := fabric.EventSubject("s", "i", fabric.StatusClaim, ">")

	_, err := f.Publish(context.Background(), subj, []byte("x"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var deliveries int
	done := make(chan error, 1)
	go func() {
		done <- f.ConsumeDurable(ctx, "claims_s_i", filter, func(e fabric.Event) error {
			deliveries++
			if deliveries == 1 {
				return assertErr{} // reject the first delivery → must be redelivered, not acked
			}
			cancel() // accepted on redelivery → ack and stop
			return nil
		})
	}()
	<-done
	assert.GreaterOrEqual(t, deliveries, 2, "a rejected (un-acked) message must be redelivered, not dropped")
}

// A Fetch failure that is NOT a ctx cancellation (the stream/connection died
// under a still-live ctx) must surface as a real error, never be reported as a
// clean nil shutdown — otherwise the supervisor that runs ConsumeDurable can't
// tell "asked to stop" from "the log fell out from under me" and silently stops
// consuming forever.
func TestConsumeDurable_surfacesAStreamFailureRatherThanMaskingItAsCleanShutdown(t *testing.T) {
	t.Parallel()
	f := startFab(t)
	subj := fabric.EventSubject("s", "i", fabric.StatusClaim, "work")
	filter := fabric.EventSubject("s", "i", fabric.StatusClaim, ">")

	_, err := f.Publish(context.Background(), subj, []byte("x"))
	require.NoError(t, err)

	// ctx stays live for the whole test — teardown is the fabric dying, not cancel.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// inLoop closes once handle has run: the durable is created, the bind
	// succeeded, and the FIRST Fetch returned — so the consumer is now parked in
	// the Fetch loop. Killing the fabric here makes the NEXT Fetch fail, which is
	// the path under test (not the setup-time AddConsumer error).
	inLoop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- f.ConsumeDurable(ctx, "claims_s_i", filter, func(e fabric.Event) error {
			close(inLoop)
			return nil
		})
	}()

	<-inLoop
	// Kill the fabric out from under the running consumer; its next Fetch fails
	// with a connection/stream error while ctx is still live.
	require.NoError(t, f.Close())

	select {
	case got := <-done:
		require.Error(t, got, "a stream/connection failure under a live ctx must be returned, not masked as nil")
	case <-time.After(5 * time.Second):
		t.Fatal("ConsumeDurable did not return after the fabric was closed")
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "handle rejected" }
