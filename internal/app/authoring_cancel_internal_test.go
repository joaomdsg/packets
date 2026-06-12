package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"
)

// As the Lead types fast, the assist fires repeatedly; a superseded run must be
// CANCELLED (so the slow model call is abandoned, not left racing) and must NEVER
// overwrite the latest read. This is the "cancel in-flight before launching a new
// one" guarantee, enforced server-side so a stale run can't win a write race. NOT
// parallel (shared globals).
func TestLiveCard_analyzeDraftCancelsASupersededInFlightRun(t *testing.T) {
	restore := analyzeDraft
	t.Cleanup(func() { analyzeDraft = restore })
	var calls int32
	firstStarted := make(chan struct{})
	firstCancelled := make(chan struct{})
	analyzeDraft = func(ctx context.Context, _, _ string) (string, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(firstStarted)
			<-ctx.Done() // block until a newer analyze supersedes (cancels) this one
			close(firstCancelled)
			return `{"summary":"STALE first read","ready":true,"highlights":[],"questions":[]}`, ctx.Err()
		}
		return `{"summary":"FRESH second read","ready":true,"highlights":[],"questions":[]}`, nil
	}

	_, server := fundedAuthoringServer(t, "authcancel")

	go func() {
		c := vt.NewClient(t, server, "/?key=authcancel")
		c.Action((&LiveCard{Key: "authcancel"}).AnalyzeDraft).WithSignal("orderprompt", "first draft").Fire()
	}()
	<-firstStarted // the first run is in flight, blocked

	tc := vt.NewClient(t, server, "/?key=authcancel")
	require.Equal(t, 200, tc.Action((&LiveCard{Key: "authcancel"}).AnalyzeDraft).
		WithSignal("orderprompt", "second draft").Fire())

	select {
	case <-firstCancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("the superseded first analysis was never cancelled")
	}

	body := bodyOf(vt.NewClient(t, server, "/?key=authcancel").HTML())
	assert.Contains(t, body, "FRESH second read", "the latest analysis owns the cache")
	assert.NotContains(t, body, "STALE first read", "a superseded (cancelled) run never overwrites the latest read")
}
