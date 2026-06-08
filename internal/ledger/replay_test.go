package ledger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// evt is one logical ledger event in the characterization fixture: exactly the
// field matching kind is populated. Publishing forged events directly (a
// non-catch on the catch subject, a zero-amount spend) exercises the fold over
// sequences the public writers would refuse — the substrate-tolerance the old
// hand-edited-JSONL tests covered, now at the stream level.
type evt struct {
	kind  string
	catch ledger.CatchRecord
	spend ledger.SpendRecord
	order ledger.WorkOrderRecord
	stat  ledger.StatusRecord
}

func (e evt) publish(t *testing.T, ctx context.Context, f *fabric.Fabric, session, instance string) {
	t.Helper()
	var err error
	switch e.kind {
	case "catch":
		_, err = ledger.PublishCatch(ctx, f, session, instance, e.catch)
	case "spend":
		_, err = ledger.PublishSpend(ctx, f, session, instance, e.spend)
	case "workorder":
		_, err = ledger.PublishWorkOrder(ctx, f, session, instance, e.order)
	case "wostatus":
		_, err = ledger.PublishStatus(ctx, f, session, instance, e.stat)
	}
	require.NoError(t, err)
}

// The economy fold is the live read path: every projecting Log method delegates
// to ReplayProjection, so this pins the economy logic against a concrete worked
// economy folded from the stream.
func TestReplayProjection_foldsTheWorkedEconomyFromTheStream(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tgt := func(line int) ledger.Target {
		return ledger.Target{BaseRev: "base", FixRev: "fix", TipRev: "fix", Path: "a.go", Line: line}
	}
	connectCatch := func(line int, reason string) ledger.CatchRecord {
		return ledger.CatchRecord{Outcome: catch.Catch, Path: "a.go", Line: line, BeforeRev: "base", AfterRev: "fix", ReasonTag: reason, Producer: "connect"}
	}

	// A worked economy: a connect win, a forged NON-catch catch-kind event (locks
	// the ShouldRecord asymmetry — counted by Records, never by Balance), a
	// zero-amount spend (locks the amount<=0 guard), three dispatched orders where
	// order 1 mints back (a reinvested HIT), order 2 mints nothing (an honest
	// MISS), and order 3 stays queued (a pending bet).
	fixture := []evt{
		{kind: "catch", catch: connectCatch(4, "boundary")},
		{kind: "catch", catch: ledger.CatchRecord{Outcome: catch.NoCatch, Path: "a.go", Line: 9, Producer: "connect"}},
		{kind: "spend", spend: ledger.SpendRecord{Kind: "spend", Amount: 0, Reason: "noop"}},
		{kind: "spend", spend: ledger.SpendRecord{Kind: "spend", Amount: 1, Reason: "d1"}},
		{kind: "workorder", order: ledger.WorkOrderRecord{Kind: "workorder", ID: 1, Producer: "in-process", Status: "queued", Reason: "d1", Target: tgt(5)}},
		{kind: "wostatus", stat: ledger.StatusRecord{Kind: "wostatus", ID: 1, Status: "running"}},
		{kind: "catch", catch: ledger.CatchRecord{Outcome: catch.Catch, Path: "a.go", Line: 5, BeforeRev: "base", AfterRev: "fix", ReasonTag: "boundary", Producer: "wo:1"}},
		{kind: "wostatus", stat: ledger.StatusRecord{Kind: "wostatus", ID: 1, Status: "done"}},
		{kind: "spend", spend: ledger.SpendRecord{Kind: "spend", Amount: 1, Reason: "d2"}},
		{kind: "workorder", order: ledger.WorkOrderRecord{Kind: "workorder", ID: 2, Producer: "in-process", Status: "queued", Reason: "d2", Target: tgt(6)}},
		{kind: "wostatus", stat: ledger.StatusRecord{Kind: "wostatus", ID: 2, Status: "done"}},
		{kind: "spend", spend: ledger.SpendRecord{Kind: "spend", Amount: 1, Reason: "d3"}},
		{kind: "workorder", order: ledger.WorkOrderRecord{Kind: "workorder", ID: 3, Producer: "in-process", Status: "queued", Reason: "d3", Target: tgt(7)}},
		{kind: "catch", catch: connectCatch(10, "nil")},
		{kind: "catch", catch: connectCatch(11, "bounds")},
	}

	f := startFabric(t)
	for _, e := range fixture {
		e.publish(t, ctx, f, "s1", "i1")
	}
	proj, err := ledger.ReplayProjection(ctx, f, "s1", "i1")
	require.NoError(t, err)

	assert.Equal(t, 1, proj.Balance(), "4 confirmed credits − 3 positive debits (the 0-amount spend never debits)")

	require.Len(t, proj.Records(), 5, "Records is unfiltered: the forged NoCatch event survives the fold")
	stock := ledger.ConfirmedCatches(proj.Records())
	assert.Equal(t, 4, stock.Count, "the forged NoCatch event is never counted as confirmed")
	assert.Equal(t, 1, stock.Reinvested, "exactly the wo:1 mint is reinvested")

	require.Len(t, proj.WorkOrders(), 3)
	assert.Equal(t, ledger.DispatchCounts{Queued: 1, Running: 0, Done: 2}, proj.DispatchStatusCounts())

	require.Len(t, proj.QueuedWorkOrders(), 1)
	assert.Equal(t, 3, proj.QueuedWorkOrders()[0].ID, "only order 3 is still queued")

	assert.Equal(t, 1, boardMisses(proj.DispatchStatusCounts().Done, stock.Reinvested), "order 2 is done but minted nothing — one honest miss")
}

// An order mid-flight (running, not yet done) must tally correctly: the watchable
// queued→running→done shape the Lead sees is a projection too.
func TestReplayProjection_countsAnOrderStillRunning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fixture := []evt{
		{kind: "workorder", order: ledger.WorkOrderRecord{Kind: "workorder", ID: 1, Producer: "in-process", Status: "queued", Reason: "r", Target: ledger.Target{BaseRev: "base", FixRev: "fix", TipRev: "fix", Path: "a.go", Line: 5}}},
		{kind: "wostatus", stat: ledger.StatusRecord{Kind: "wostatus", ID: 1, Status: "running"}},
	}

	f := startFabric(t)
	for _, e := range fixture {
		e.publish(t, ctx, f, "s2", "i2")
	}
	proj, err := ledger.ReplayProjection(ctx, f, "s2", "i2")
	require.NoError(t, err)

	assert.Equal(t, ledger.DispatchCounts{Queued: 0, Running: 1, Done: 0}, proj.DispatchStatusCounts())
	assert.Empty(t, proj.QueuedWorkOrders(), "a running order is not queued")
}

func boardMisses(done, reinvested int) int {
	if m := done - reinvested; m > 0 {
		return m
	}
	return 0
}
