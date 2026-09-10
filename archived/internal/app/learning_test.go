package app_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// A fresh repo has produced no real settled judgment yet —
// the console's "learning" card must show the honest running count against
// the real threshold, never a fabricated converged state. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_learningCardShowsHonestProgressBeforeConvergence(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "learningc", app.LiveConfig{BaseRev: "own-b-learningc", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Source: "wo:1"}))

	body := bodyOf(vt.NewClient(t, server, "/?key=learningc").HTML())
	require.Contains(t, body, "learning", "the learning region is present")
	require.Contains(t, body, "1/5 settled", "the card shows the real running count against the real threshold")
	require.Contains(t, body, `data-state="learning"`, "the card's state hook names the honest pre-convergence state")
	require.NotContains(t, body, "converged", "convergence must not render before the real threshold is reached")
}

// Reaching the real threshold's worth of settled packets — verified, held,
// AND delivered all count, mirroring the settled rail exactly — must flip the
// card to "converged", never lingering on a stale progress fraction. NOT
// parallel (shared liveReg/liveFabric).
func TestLiveCard_learningCardShowsConvergedOnceTheThresholdIsReached(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "learningconv", app.LiveConfig{BaseRev: "own-b-learningconv", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	for i := 1; i <= 3; i++ {
		require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100 + i, ReasonTag: "catch"}))
		require.NoError(t, log.AppendSend("d", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: i}, own))
		require.NoError(t, log.AppendStatus(i, "done"))
		require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: i, ReasonTag: "catch", Source: "wo:" + strconv.Itoa(i)}))
	}
	// A held-blocking packet (a run failure) — settled, and counts too.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 200, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d4", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "gamma.go", Line: 20}, own))
	require.NoError(t, log.AppendStatus(4, "failed"))
	// A delivered packet (a real ACK) — settled, and counts too.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 201, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d5", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "delta.go", Line: 21}, own))
	require.NoError(t, log.AppendStatus(5, "deployed"))
	// A 6th, still-queued packet — total dispatched packets (6) now EXCEEDS the
	// settled count (5), so convergence can only be real if it counts settled
	// judgment specifically, not merely how many packets exist.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 202, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d6", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "epsilon.go", Line: 30}, own))

	body := bodyOf(vt.NewClient(t, server, "/?key=learningconv").HTML())
	require.Contains(t, body, "converged", "5 real settled packets (verified, held-blocking, and delivered) reach the threshold")
	require.Contains(t, body, `data-state="converged"`, "the card's state hook flips once real history clears the threshold")
	require.NotContains(t, body, `data-state="learning"`, "the card's state hook no longer shows the pre-convergence state")
	require.NotContains(t, body, "/5 settled", "no stale progress fraction lingers once converged")
}
