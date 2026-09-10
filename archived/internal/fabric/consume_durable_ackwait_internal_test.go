package fabric

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The configured AckWait must reach the real JetStream consumer — it is the
// governor's load-bearing value (it must exceed the per-claim verify deadline so
// a slow verify is never redelivered into a concurrent double run). A param that
// was silently dropped would default to 30s and reintroduce that race, so assert
// the durable's actual server-side config carries the value we passed.
func TestConsumeDurable_setsTheConfiguredAckWaitOnTheConsumer(t *testing.T) {
	t.Parallel()
	f, err := Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	filter := EventSubject("s", "i", StatusClaim, ">")
	const wantAckWait = 7 * time.Second
	go func() {
		_ = f.ConsumeDurable(ctx, "ackwait_probe", filter, wantAckWait, func(Event) error { return nil })
	}()

	require.Eventually(t, func() bool {
		info, err := f.js.ConsumerInfo(streamName, "ackwait_probe")
		return err == nil && info != nil
	}, 3*time.Second, 25*time.Millisecond, "the durable consumer must be created")

	info, err := f.js.ConsumerInfo(streamName, "ackwait_probe")
	require.NoError(t, err)
	assert.Equal(t, wantAckWait, info.Config.AckWait, "the consumer must carry the AckWait the caller configured, not the 30s default")
}
