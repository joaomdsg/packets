package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
)

// A fresh session has interrupted the Lead zero times this week — the KPI
// must show the honest zero against the LOCKED weekly cap of 10
// (the console's "N/10 interrupts" header),
// never omit the stat or invent a different cap. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_interruptKPIShowsHonestZeroOnAFreshSession(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "0/10 interrupts", "a session that has raised no interrupts shows the honest zero against the fixed cap")
}

// The KPI must reflect a REAL count of raised interrupts (ledger blocks), not
// a fabricated number — it changes as real blocks are logged. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_interruptKPIReflectsRealLoggedInterrupts(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "interruptkpi", app.LiveConfig{BaseRev: "own-b-interruptkpi", FixRev: "own-f", Anchor: anchorForCap()})
	require.NoError(t, log.AppendBlock("q:1", time.Now()))
	require.NoError(t, log.AppendBlock("q:2", time.Now()))

	body := bodyOf(vt.NewClient(t, server, "/?key=interruptkpi").HTML())
	require.Contains(t, body, "2/10 interrupts", "the KPI counts the two real interrupts this session logged, against the fixed cap")
}

// The KPI must show the REAL count even past the fixed cap, never clamp it —
// clamping would hide from the Lead exactly how far over budget they ran.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_interruptKPIShowsTheRealCountUncappedWhenOverBudget(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "interruptover", app.LiveConfig{BaseRev: "own-b-interruptover", FixRev: "own-f", Anchor: anchorForCap()})
	for i := 0; i < 11; i++ {
		require.NoError(t, log.AppendBlock("q:"+string(rune('a'+i)), time.Now()))
	}

	body := bodyOf(vt.NewClient(t, server, "/?key=interruptover").HTML())
	require.Contains(t, body, "11/10 interrupts", "the real over-budget count is shown honestly, never clamped at the cap")
}

// A new interrupt raised while a tab is connected must update the KPI over
// the LIVE SSE stream, not just on the next full reload — the same live-refresh
// pattern already proven for the hero stat's dispatch-tally signature. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_interruptKPIRefreshesLiveWhenANewInterruptIsRaised(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "interruptlive", app.LiveConfig{BaseRev: "own-b-interruptlive", FixRev: "own-f", Anchor: anchorForCap()})

	tc := vt.NewClient(t, server, "/?key=interruptlive")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, "0/10 interrupts")

	require.NoError(t, log.AppendBlock("q:1", time.Now()))

	vt.AwaitFrame(t, frames, 10*time.Second, "1/10 interrupts")
}
