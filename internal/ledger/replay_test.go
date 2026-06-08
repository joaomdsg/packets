package ledger_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// evt is one logical ledger event in the characterization fixture: exactly the
// field matching kind is populated. The SAME marshaled payload is written to a
// JSONL file AND published to the stream, so the lock compares two substrates
// holding byte-identical events.
type evt struct {
	kind  string
	catch ledger.CatchRecord
	spend ledger.SpendRecord
	order ledger.WorkOrderRecord
	stat  ledger.StatusRecord
}

func (e evt) payload(t *testing.T) []byte {
	t.Helper()
	var v any
	switch e.kind {
	case "catch":
		v = e.catch
	case "spend":
		v = e.spend
	case "workorder":
		v = e.order
	case "wostatus":
		v = e.stat
	default:
		t.Fatalf("unknown kind %q", e.kind)
	}
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
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

// The migration's safety precondition: every projection the economy exposes must
// fold to IDENTICAL state whether the events live in the JSONL file (the proven
// scan) or on the JetStream (the new replay fold). If these two independent
// implementations ever disagree, the substrate swap would silently change the
// balance, the board, the hit-rate, or the work-order ledger — so this lock must
// be green before any swap, and stay green through it.
func TestEconomyProjectionsAreIdenticalFromJSONLAndFromStreamReplay(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tgt := func(line int) ledger.Target {
		return ledger.Target{BaseRev: "base", FixRev: "fix", TipRev: "fix", Path: "a.go", Line: line}
	}
	connectCatch := func(line int, reason string) ledger.CatchRecord {
		return ledger.CatchRecord{Outcome: catch.Catch, Path: "a.go", Line: line, BeforeRev: "base", AfterRev: "fix", ReasonTag: reason, Producer: "connect"}
	}

	// A worked economy: a connect win, a forged NON-catch catch-kind line (locks
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

	// --- JSONL substrate: write the raw lines, project via the proven file scan.
	path := filepath.Join(t.TempDir(), "catches.jsonl")
	var buf []byte
	for _, e := range fixture {
		buf = append(buf, e.payload(t)...)
		buf = append(buf, '\n')
	}
	require.NoError(t, os.WriteFile(path, buf, 0o644))
	log, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, log.Close()) })

	// --- Stream substrate: publish the same events, project via the replay fold.
	f := startFabric(t)
	for _, e := range fixture {
		e.publish(t, ctx, f, "s1", "i1")
	}
	proj, err := ledger.ReplayProjection(ctx, f, "s1", "i1")
	require.NoError(t, err)

	// Balance.
	fileBal, err := log.Balance()
	require.NoError(t, err)
	assert.Equal(t, fileBal, proj.Balance(), "balance must fold identically across substrates")
	assert.Equal(t, 1, proj.Balance(), "4 confirmed credits − 3 positive debits (the 0-amount spend never debits)")

	// Catch records (Records is unfiltered by ShouldRecord — the forged NoCatch
	// line survives the projection on BOTH substrates).
	fileRecs, err := log.Records()
	require.NoError(t, err)
	assert.Equal(t, fileRecs, proj.Records(), "the catch-kind record stream must fold identically")
	require.Len(t, proj.Records(), 5)

	// Stock / hit-rate inputs: the shared pure fold over each substrate's records.
	fileStock := ledger.ConfirmedCatches(fileRecs)
	streamStock := ledger.ConfirmedCatches(proj.Records())
	assert.Equal(t, fileStock, streamStock, "confirmed/reinvested stock must agree")
	assert.Equal(t, 4, streamStock.Count, "the forged NoCatch line is never counted as confirmed")
	assert.Equal(t, 1, streamStock.Reinvested, "exactly the wo:1 mint is reinvested")

	// Work orders.
	fileOrders, err := log.WorkOrders()
	require.NoError(t, err)
	assert.Equal(t, fileOrders, proj.WorkOrders(), "the funded work-order ledger must fold identically")
	require.Len(t, proj.WorkOrders(), 3)

	// Dispatch status counts (current status: last-writer-wins per id).
	fileCounts, err := log.DispatchStatusCounts()
	require.NoError(t, err)
	assert.Equal(t, fileCounts, proj.DispatchStatusCounts(), "dispatch status tally must fold identically")
	assert.Equal(t, ledger.DispatchCounts{Queued: 1, Running: 0, Done: 2}, proj.DispatchStatusCounts())

	// Queued work orders (the runner's input).
	fileQueued, err := log.QueuedWorkOrders()
	require.NoError(t, err)
	assert.Equal(t, fileQueued, proj.QueuedWorkOrders(), "the queued-order input must fold identically")
	require.Len(t, proj.QueuedWorkOrders(), 1)
	assert.Equal(t, 3, proj.QueuedWorkOrders()[0].ID, "only order 3 is still queued")

	// Board-derived honest losses + hit-rate: the same derivation over each
	// substrate's projections must agree (Misses = max(0, Done−Reinvested)).
	fileMisses := boardMisses(fileCounts.Done, fileStock.Reinvested)
	streamMisses := boardMisses(proj.DispatchStatusCounts().Done, streamStock.Reinvested)
	assert.Equal(t, fileMisses, streamMisses, "honest misses must fold identically")
	assert.Equal(t, 1, streamMisses, "order 2 is done but minted nothing — one honest miss")
}

// An order mid-flight (running, not yet done) must tally identically across
// substrates: the watchable queued→running→done shape the Lead sees is a
// projection too, and the swap must not silently miscount in-flight work.
func TestReplayProjection_countsAnOrderLeftRunningLikeTheFileScan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fixture := []evt{
		{kind: "workorder", order: ledger.WorkOrderRecord{Kind: "workorder", ID: 1, Producer: "in-process", Status: "queued", Reason: "r", Target: ledger.Target{BaseRev: "base", FixRev: "fix", TipRev: "fix", Path: "a.go", Line: 5}}},
		{kind: "wostatus", stat: ledger.StatusRecord{Kind: "wostatus", ID: 1, Status: "running"}},
	}

	path := filepath.Join(t.TempDir(), "catches.jsonl")
	var buf []byte
	for _, e := range fixture {
		buf = append(buf, e.payload(t)...)
		buf = append(buf, '\n')
	}
	require.NoError(t, os.WriteFile(path, buf, 0o644))
	log, err := ledger.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, log.Close()) })

	f := startFabric(t)
	for _, e := range fixture {
		e.publish(t, ctx, f, "s2", "i2")
	}
	proj, err := ledger.ReplayProjection(ctx, f, "s2", "i2")
	require.NoError(t, err)

	fileCounts, err := log.DispatchStatusCounts()
	require.NoError(t, err)
	assert.Equal(t, fileCounts, proj.DispatchStatusCounts(), "an in-flight order must tally identically")
	assert.Equal(t, ledger.DispatchCounts{Queued: 0, Running: 1, Done: 0}, proj.DispatchStatusCounts())
	assert.Empty(t, proj.QueuedWorkOrders(), "a running order is not queued")
}

func boardMisses(done, reinvested int) int {
	if m := done - reinvested; m > 0 {
		return m
	}
	return 0
}
