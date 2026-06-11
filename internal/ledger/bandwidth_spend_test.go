package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// earnBandwidth clears one fast block so the log holds a known bandwidth (3 per
// fast clear), the funding a live dispatch draws down.
func earnBandwidth(t *testing.T, log *ledger.Log, id string) {
	t.Helper()
	base := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock(id, base))
	require.NoError(t, log.AppendUnblock(id, base.Add(30*time.Second)))
}

func TestLog_bandwidthSpendDebitsTheMeter(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	earnBandwidth(t, log, "q1") // +3

	require.NoError(t, log.AppendBandwidthSpend(1, "live order"))

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 2, bw, "a bandwidth spend debits the meter")
}

func TestLog_bandwidthSpendRefusesOverdraft(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	earnBandwidth(t, log, "q1") // +3

	require.Error(t, log.AppendBandwidthSpend(4, "too big"), "a spend the meter can't cover is refused")

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 3, bw, "a refused spend leaves the meter untouched")
}

// A live order authored from the UI is funded by ATTENTION bandwidth, not a catch:
// AppendLiveDispatch debits the bandwidth meter and queues the prompt-carrying
// order in one write, leaving the catch balance untouched (the two meters, both
// used — the division the Lead chose).
func TestLog_appendLiveDispatchSpendsBandwidthAndQueuesTheOrder(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	earnBandwidth(t, log, "q1") // +3 bandwidth
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: "head", Prompt: "do the task"}
	require.NoError(t, log.AppendLiveDispatch("liveorder", live, own))

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 2, bw, "the live dispatch spent one bandwidth")
	bal, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, 1, bal, "the catch balance is untouched — a live order is funded by attention, not a catch")

	queued, err := log.QueuedWorkOrders()
	require.NoError(t, err)
	require.Len(t, queued, 1)
	assert.Equal(t, "do the task", queued[0].Target.Prompt, "the order carries the authored prompt")
}

func TestLog_appendLiveDispatchRefusesWithoutBandwidth(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)

	own := ledger.Target{BaseRev: "ob", Path: "own.go", Line: 1}
	live := ledger.Target{BaseRev: "head", Prompt: "do the task"}
	require.Error(t, log.AppendLiveDispatch("liveorder", live, own),
		"with no earned bandwidth there is no attention to fund a live order")

	queued, err := log.QueuedWorkOrders()
	require.NoError(t, err)
	assert.Empty(t, queued, "a refused live dispatch queues no order")
}

func TestLog_appendLiveDispatchRefusesItsOwnTarget(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	earnBandwidth(t, log, "q1")

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.Error(t, log.AppendLiveDispatch("liveorder", own, own),
		"a live order can no more fund the card's own work than a catch-funded one")
}
