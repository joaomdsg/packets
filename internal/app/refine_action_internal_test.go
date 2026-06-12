package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/ledger"
)

func TestBuildRefinement_turnsTheLeadsInputIntoTheRightSharpeningFact(t *testing.T) {
	t.Parallel()
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}
	tests := []struct {
		name    string
		kind    string
		text    string
		wantOK  bool
		wantRec ledger.RefinedOrderRecord
	}{
		{
			name:   "criteria splits into one fact per non-blank line",
			kind:   "criteria",
			text:   "rejects a negative amount\n\n  caps at the daily ceiling  \n",
			wantOK: true,
			wantRec: ledger.RefinedOrderRecord{Target: tgt, Refine: "criteria",
				Criteria: []string{"rejects a negative amount", "caps at the daily ceiling"}},
		},
		{
			name:   "criteria with no non-blank line is not a refinement",
			kind:   "criteria",
			text:   "   \n\n",
			wantOK: false,
		},
		{
			name:   "convention carries the trimmed note",
			kind:   "convention",
			text:   "  wrap errors with an origin prefix  ",
			wantOK: true,
			wantRec: ledger.RefinedOrderRecord{Target: tgt, Refine: "convention",
				Note: "wrap errors with an origin prefix"},
		},
		{
			name:   "convention with empty text is not a refinement",
			kind:   "convention",
			text:   "   ",
			wantOK: false,
		},
		{
			name:   "split is built elsewhere (needs harvested sub-targets), not here",
			kind:   "split",
			text:   "anything",
			wantOK: false,
		},
		{
			name:   "an unknown kind is refused",
			kind:   "bogus",
			text:   "x",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec, ok := buildRefinement(tgt, tt.kind, tt.text)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantRec, rec)
			}
		})
	}
}

func TestLiveCard_refineChosenSharpensTheChosenBenchTarget(t *testing.T) {
	// The sharpen action: the Lead attaches acceptance criteria to a fundable bench
	// target during dead-air. It appends a worefine fact for THAT target, validated
	// against the fundable set like FundChosen. NOT parallel (shared globals).
	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&LiveCard{}).RefineChosen).
		WithSignal("refinetarget", "pay.go:88").
		WithSignal("refinekind", "criteria").
		WithSignal("refinetext", "rejects a negative amount").
		Fire())

	refs, err := log.Refinements()
	require.NoError(t, err)
	require.Len(t, refs, 1, "sharpening a fundable target appends exactly one worefine fact")
	assert.Equal(t, "criteria", refs[0].Refine)
	assert.Equal(t, "pay.go", refs[0].Target.Path)
	assert.Equal(t, []string{"rejects a negative amount"}, refs[0].Criteria)
}

func TestLiveCard_refineChosenRefusesAnOffBenchTarget(t *testing.T) {
	// Sharpening is constrained to real fundable work, exactly like funding: an
	// off-bench (unknown/consumed/own) target appends nothing. A valid sharpen first
	// proves the action DOES append, so the off-bench no-op is the membership gate
	// firing — not an unconditional no-op. NOT parallel.
	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&LiveCard{}).RefineChosen).
		WithSignal("refinetarget", "pay.go:88").WithSignal("refinekind", "convention").
		WithSignal("refinetext", "wrap errors with an origin prefix").Fire())
	require.Equal(t, 200, tc.Action((&LiveCard{}).RefineChosen).
		WithSignal("refinetarget", "nowhere.go:99").WithSignal("refinekind", "criteria").
		WithSignal("refinetext", "x").Fire())

	refs, err := log.Refinements()
	require.NoError(t, err)
	require.Len(t, refs, 1, "only the on-bench sharpen was appended; the off-bench target was refused by the membership gate")
	assert.Equal(t, "convention", refs[0].Refine)
}

func TestLiveCard_refineChosenWithEmptyTextAppendsNothing(t *testing.T) {
	// A contentless sharpen (no criteria lines / blank convention) is not a
	// refinement: buildRefinement refuses it and the action must append nothing,
	// even on a perfectly valid bench target. NOT parallel.
	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&LiveCard{}).RefineChosen).
		WithSignal("refinetarget", "pay.go:88").WithSignal("refinekind", "criteria").
		WithSignal("refinetext", "   \n\n").Fire())

	refs, err := log.Refinements()
	require.NoError(t, err)
	require.Empty(t, refs, "a contentless refinement is refused — the bench is not polluted with empty facts")
}
