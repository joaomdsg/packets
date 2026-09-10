package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/ledger"
)

// The fleet board ranges liveReg and ranks cards by QUEUED-awaiting-drain — an
// honest, log-derived "where motion is" signal — so a card with work in flight
// sorts above an idle one. (Relative order of THIS test's keys; liveReg is a
// shared global polluted by other tests, so we filter to our own keys.)
func TestBoardRows_PacketsByQueuedActivitySoTheLeadSeesWhereTheWorkIsMoving(t *testing.T) {
	_, _ = bootDefaultServer(t, defaultBootCfg)
	t1, t2 := woTargetN(1), woTargetN(2)
	logB := boardSession(t, "brd-B", 3, []ledger.Target{t1, t2})
	require.NoError(t, logB.AppendSend("d", t1, ownTargetOf(app.LiveConfig{BaseRev: "own-b-brd-B", FixRev: "own-f", Anchor: anchorForCap()})))
	require.NoError(t, logB.AppendSend("d", t2, ownTargetOf(app.LiveConfig{BaseRev: "own-b-brd-B", FixRev: "own-f", Anchor: anchorForCap()})))
	boardSession(t, "brd-A", 1, nil) // a balance, no dispatch → 0 queued

	rows := app.BoardRows()
	requireBefore(t, rows, "brd-B", "brd-A") // brd-B (2 queued) sorts above brd-A (0 queued) — activity, not hoard size

	b := rowFor(t, rows, "brd-B")
	require.Equal(t, 2, b.Queued, "brd-B has two funded, undrained orders")
	require.Equal(t, 1, b.Balance, "brd-B: 3 catches − 2 dispatched debits")
	require.Equal(t, 3, b.Confirmed)
	require.Greater(t, b.BacklogRemaining, 0, "both config targets are funded, but from-catch supply keeps fundable candidate work — the faucet refills, no silent dead-end")

	a := rowFor(t, rows, "brd-A")
	require.Equal(t, 0, a.Queued)
	require.Equal(t, 1, a.Balance)
}

func TestBoardRows_reSortsAsWorkDrainsAndIsFundedElsewhere(t *testing.T) {
	_, _ = bootDefaultServer(t, defaultBootCfg)
	t1, t2 := woTargetN(1), woTargetN(2)
	own := ownTargetOf(app.LiveConfig{BaseRev: "own-b-rsB", FixRev: "own-f", Anchor: anchorForCap()})
	logB := boardSession(t, "rsB", 3, []ledger.Target{t1, t2})
	require.NoError(t, logB.AppendSend("d", t1, own))
	require.NoError(t, logB.AppendSend("d", t2, own))
	logA := boardSession(t, "rsA", 2, []ledger.Target{t1})

	requireBefore(t, app.BoardRows(), "rsB", "rsA") // rsB leads while it holds the queued work

	// rsB's orders run to done (no longer queued); rsA funds one.
	require.NoError(t, logB.AppendStatus(1, "done"))
	require.NoError(t, logB.AppendStatus(2, "done"))
	require.NoError(t, logA.AppendSend("d", t1, ownTargetOf(app.LiveConfig{BaseRev: "own-b-rsA", FixRev: "own-f", Anchor: anchorForCap()})))

	requireBefore(t, app.BoardRows(), "rsA", "rsB") // attention follows the queued work — rsA now leads
}

func TestBoardRows_tieBreaksDeterministicallyByRegistrationPacketNotMapRandomness(t *testing.T) {
	// Equal queued counts must NOT order by sync.Map's nondeterministic Range — the
	// earlier-registered card precedes the later, stably across renders, so the
	// board never flickers (and never fabricates an order from a missing timestamp).
	_, _ = bootDefaultServer(t, defaultBootCfg)
	boardSession(t, "tieEarly", 0, nil)
	boardSession(t, "tieLate", 0, nil)
	for i := 0; i < 5; i++ {
		rows := app.BoardRows()
		requireBefore(t, rows, "tieEarly", "tieLate") // equal-queued cards hold registration order on every render
	}
}

func TestBoardRows_surfacesADonePacketThatMintedNothingAsAVisibleMiss(t *testing.T) {
	// The honest loss must be VISIBLE on the board, never a silent discard: a done
	// order that minted no catch (Done counted it, but no "wo:" catch joined the
	// stock) shows as a MISS — the spend was a bet that did not pay, and the Lead
	// can see it. Misses = Done − Caught (the exact ScoutingReport count).
	_, _ = bootDefaultServer(t, defaultBootCfg)
	log := boardSession(t, "missK", 1, []ledger.Target{woTargetN(1)})
	require.NoError(t, log.AppendSend("d", woTargetN(1), ownTargetOf(app.LiveConfig{BaseRev: "own-b-missK", FixRev: "own-f", Anchor: anchorForCap()})))
	require.NoError(t, log.AppendStatus(1, "done")) // the order ran to done but minted NOTHING

	r := rowFor(t, app.BoardRows(), "missK")
	require.Equal(t, 1, r.Done, "the order reached done")
	require.Equal(t, 0, r.Reinvested, "it minted no catch")
	require.Equal(t, 1, r.Misses, "a done-but-no-mint order is a VISIBLE miss — the honest loss, not a silent discard")
}
