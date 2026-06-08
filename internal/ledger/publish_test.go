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

// Every ledger event kind must travel producer → taxonomy → bus → consumer
// intact: it lands on the canonical minted subject for its kind, and its payload
// decodes back to the same record a projection rebuilds state from. These are the
// wire primitives the substrate swap rebuilds the economy from, so a lossy
// round-trip on ANY kind would silently corrupt the migrated ledger.
func TestPublishCatch_roundTripsCatchRecordThroughTheBus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f := startFabric(t)

	want := ledger.CatchRecord{
		Outcome:           catch.Catch,
		Path:              "adult.go",
		Line:              4,
		BeforeRev:         "base",
		AfterRev:          "fix",
		BeforeInventory:   []string{">=", "&&"},
		AfterInventory:    []string{">=", "&&"},
		MutantsConsidered: 2,
		ReasonTag:         "boundary",
		SelfFlagged:       true,
		WouldHaveShipped:  true,
		Producer:          "wo:3",
	}

	seq, err := ledger.PublishCatch(ctx, f, "s1", "i1", want)
	require.NoError(t, err)

	events, err := f.ReplaySubject(ctx, fabric.EventSubject("s1", "i1", fabric.StatusMinted, "catch"))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, fabric.EventSubject("s1", "i1", fabric.StatusMinted, "catch"), events[0].Subject)
	assert.Equal(t, seq, events[0].Seq, "the returned sequence is the event's authoritative stream position")

	got, err := ledger.DecodeCatch(events[0].Data)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPublishSpend_roundTripsSpendRecordThroughTheBus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f := startFabric(t)
	want := ledger.SpendRecord{Kind: "spend", Amount: 3, Reason: "dispatch line 5"}

	seq, err := ledger.PublishSpend(ctx, f, "s1", "i1", want)
	require.NoError(t, err)

	events, err := f.ReplaySubject(ctx, fabric.EventSubject("s1", "i1", fabric.StatusMinted, "spend"))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, seq, events[0].Seq)

	got, err := ledger.DecodeSpend(events[0].Data)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPublishWorkOrder_roundTripsWorkOrderRecordThroughTheBus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f := startFabric(t)
	want := ledger.WorkOrderRecord{
		Kind:     "workorder",
		ID:       2,
		Producer: "in-process",
		Status:   "queued",
		Reason:   "fund distinct work",
		Target: ledger.Target{
			BaseRev: "base", FixRev: "fix", TipRev: "fix",
			Path: "adult.go", Line: 5, LineHash: "abc",
		},
	}

	seq, err := ledger.PublishWorkOrder(ctx, f, "s1", "i1", want)
	require.NoError(t, err)

	events, err := f.ReplaySubject(ctx, fabric.EventSubject("s1", "i1", fabric.StatusMinted, "workorder"))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, seq, events[0].Seq)

	got, err := ledger.DecodeWorkOrder(events[0].Data)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPublishStatus_roundTripsStatusTransitionThroughTheBus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f := startFabric(t)
	want := ledger.StatusRecord{Kind: "wostatus", ID: 2, Status: "done"}

	seq, err := ledger.PublishStatus(ctx, f, "s1", "i1", want)
	require.NoError(t, err)

	events, err := f.ReplaySubject(ctx, fabric.EventSubject("s1", "i1", fabric.StatusMinted, "wostatus"))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, seq, events[0].Seq)

	got, err := ledger.DecodeStatus(events[0].Data)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// The taxonomy must demux the kinds: each publisher writes ONLY its own kind's
// subject, so a per-kind filter sees exactly one event and a rebuild never folds
// a spend where it expected a catch.
func TestPublishKinds_eachLandsOnlyOnItsOwnKindSubject(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f := startFabric(t)

	_, err := ledger.PublishCatch(ctx, f, "s1", "i1", ledger.CatchRecord{Outcome: catch.Catch, Path: "a.go", Line: 1, ReasonTag: "r"})
	require.NoError(t, err)
	_, err = ledger.PublishSpend(ctx, f, "s1", "i1", ledger.SpendRecord{Kind: "spend", Amount: 1})
	require.NoError(t, err)
	_, err = ledger.PublishWorkOrder(ctx, f, "s1", "i1", ledger.WorkOrderRecord{Kind: "workorder", ID: 1})
	require.NoError(t, err)
	_, err = ledger.PublishStatus(ctx, f, "s1", "i1", ledger.StatusRecord{Kind: "wostatus", ID: 1, Status: "running"})
	require.NoError(t, err)

	for _, kind := range []string{"catch", "spend", "workorder", "wostatus"} {
		events, err := f.ReplaySubject(ctx, fabric.EventSubject("s1", "i1", fabric.StatusMinted, kind))
		require.NoError(t, err)
		assert.Len(t, events, 1, "kind %q must carry exactly its own one event", kind)
	}
}

func TestDecoders_returnErrorOnMalformedPayload(t *testing.T) {
	t.Parallel()
	decoders := []struct {
		name string
		fn   func([]byte) error
	}{
		{"catch", func(b []byte) error { _, err := ledger.DecodeCatch(b); return err }},
		{"spend", func(b []byte) error { _, err := ledger.DecodeSpend(b); return err }},
		{"workorder", func(b []byte) error { _, err := ledger.DecodeWorkOrder(b); return err }},
		{"wostatus", func(b []byte) error { _, err := ledger.DecodeStatus(b); return err }},
	}
	for _, d := range decoders {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, d.fn([]byte("not json")))
		})
	}
}

func startFabric(t *testing.T) *fabric.Fabric {
	t.Helper()
	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })
	return f
}
