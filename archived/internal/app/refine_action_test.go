package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestLiveCard_refineChosenSharpensTheChosenBenchTarget(t *testing.T) {
	// The sharpen action: the Lead attaches acceptance criteria to a fundable bench
	// target during dead-air. It appends a worefine fact for THAT target, validated
	// against the fundable set like FundChosen. NOT parallel (shared globals).
	server, log := bootDefaultServer(t, app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd:     []string{"true"},
		SendBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	})

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&app.LiveCard{}).RefineChosen).
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
	server, log := bootDefaultServer(t, app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd:     []string{"true"},
		SendBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	})

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&app.LiveCard{}).RefineChosen).
		WithSignal("refinetarget", "pay.go:88").WithSignal("refinekind", "convention").
		WithSignal("refinetext", "wrap errors with an origin prefix").Fire())
	require.Equal(t, 200, tc.Action((&app.LiveCard{}).RefineChosen).
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
	server, log := bootDefaultServer(t, app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd:     []string{"true"},
		SendBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	})

	tc := vt.NewClient(t, server, "/")
	require.Equal(t, 200, tc.Action((&app.LiveCard{}).RefineChosen).
		WithSignal("refinetarget", "pay.go:88").WithSignal("refinekind", "criteria").
		WithSignal("refinetext", "   \n\n").Fire())

	refs, err := log.Refinements()
	require.NoError(t, err)
	require.Empty(t, refs, "a contentless refinement is refused — the bench is not polluted with empty facts")
}
