package app_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/ledger"
)

// Dispatch is a blind auto-FIFO pick today — the Lead can't see WHAT work is on
// deck to fund. The prep bench surfaces the fundable backlog on the card so the
// Lead sees what a Spend would fund (and, in a later slice, curates it), killing
// the compute dead-air. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_showsTheFundableWorkOnTheBench(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
		SendBacklog: []ledger.Target{
			{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7},
			{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9},
		},
	})

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "bench", "the card surfaces the fundable work on the bench")
	require.Contains(t, body, "alpha.go:7", "the next fundable target is on the bench")
	require.Contains(t, body, "beta.go:9", "and the one behind it")
	require.Contains(t, body, "(next)", "the FIFO-next target a plain Spend would fund is marked")
}

// A session with no fundable work renders no bench — no empty block implying there
// is work on deck. NOT parallel (shared globals).
func TestLiveCard_omitsTheBenchWhenNoFundableWork(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, // no SendBacklog, no catches → no fundable work
	})

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.NotContains(t, body, `class="bench"`, "no bench when there is no fundable work")
}

// The bench shows only the next few fundable targets — the Lead curates what's on
// deck, not an unbounded wall. A backlog past the cap renders exactly benchCapMirror
// items. NOT parallel (shared globals).
func TestLiveCard_benchCapsTheVisibleTargets(t *testing.T) {
	var many []ledger.Target
	for i := 0; i < benchCapMirror+3; i++ {
		many = append(many, ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "f.go", Line: i + 1})
	}
	server, _ := bootDefaultServer(t, app.LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, SendBacklog: many,
	})

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Equal(t, benchCapMirror, strings.Count(body, "bench__item"),
		"the bench shows exactly benchCap items, not the whole backlog")
}
