package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// A missed order today reads only "missed" — the Lead can't tell a no-catch (the
// oracle ran, nothing to catch) from a lost-via-rename (the anchor moved) from a
// no-oracle-signal. The oracle's persisted per-order verdict is surfaced here
// so the miss is DIAGNOSABLE on the surface, not just a bare outcome. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_showsThePerPacketVerdictSoAMissIsDiagnosable(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "whyc", app.LiveConfig{BaseRev: "own-b-whyc", FixRev: "own-f", Anchor: anchorForCap()})

	// One catch funds one order that ran, MISSED, and the oracle's honest verdict
	// (no-catch) was persisted for it.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.AppendPacketVerdict(1, "no-catch"))

	body := bodyOf(vt.NewClient(t, server, "/?key=whyc").HTML())
	require.Contains(t, body, "no-catch", "the missed order shows the oracle's verdict — the WHY behind the miss")
	require.Contains(t, body, "board-row__send-why", "the verdict carries its own calm hook so it reads as secondary detail")
}

// The board's "why" tag must render a HUMAN label for a real verdict token, not the raw
// snake_case the surface persists — so the Lead reads "Anchor lost: file renamed", never
// "lost_via_rename". NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_rendersTheVerdictWhyAsAHumanLabelNotARawToken(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "whylabel", app.LiveConfig{BaseRev: "own-b-whylabel", FixRev: "own-f", Anchor: anchorForCap()})

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.AppendPacketVerdict(1, "lost_via_rename")) // a real persisted verdict token

	body := bodyOf(vt.NewClient(t, server, "/?key=whylabel").HTML())
	require.Contains(t, body, "Anchor lost: file renamed", "the why tag reads as a human label")
	require.NotContains(t, body, "lost_via_rename", "the raw snake_case token must not leak to the board")
}

// An order with no persisted verdict (queued, or pre-diagnostics data) renders no
// verdict element — never an empty "why" tag implying the oracle said something.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_omitsTheVerdictWhenNonePersisted(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "noverdictc", app.LiveConfig{BaseRev: "own-b-noverdictc", FixRev: "own-f", Anchor: anchorForCap()})

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own)) // queued, no verdict

	// bodyOf scopes past the head — the stylesheet carries .board-row__send-why
	// as a selector; we assert the rendered ORDER has no verdict element.
	body := bodyOf(vt.NewClient(t, server, "/?key=noverdictc").HTML())
	require.Contains(t, body, "PKT#1", "the order still renders")
	require.NotContains(t, body, "board-row__send-why",
		"an order with no persisted verdict shows no why element")
}
