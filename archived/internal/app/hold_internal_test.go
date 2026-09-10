package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/packet"
)

// The needs-you rail must escalate a packet whose measured
// Lane outgrew its authored handshake — the lanes / measured-blast-radius rule's "radius buys a
// STRONGER REQUIRED HANDSHAKE" has teeth only if a lane-floor breach actually
// forces a hold, not just a data fact nobody looks at. This proves BOTH
// halves of the app wiring: (1) BEFORE the lane is ever measured (no
// Inspector visit yet), an otherwise-Verified order stays honestly unheld —
// LaneUnmeasured's floor is StrengthNone, so ReconcileHold must not
// false-positive on data nobody has computed; (2) once the SAME order's lane
// is cached (a real Inspector visit, the only place allowed to exec
// packet.Measure), the console's needs-you rail picks up the cached value on
// its very next poll tick and escalates the packet to a blocking hold that
// sorts AHEAD of a plain lifecycle-advisory one — proving a lane-floor-forced
// blocking hold participates in heldPackets' blocking-before-advisory sort
// exactly like a lifecycle-native one. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_needsYouRailEscalatesOnceALaneFloorBreachIsMeasuredAndSortsAheadOfAdvisory(t *testing.T) {
	resetConsumersForTest()
	repo, base, fix := initMeasurableRepo(t)

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "holdlanefloor", "i")
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}

	// Order 1: measurable against the fixture repo's single package (a
	// 100%-of-graph ripple, LaneStrict once measured), done+caught+no open
	// questions — Fold's own baseline is Verified/HoldNone. No
	// HandshakeStrength is ever authored (stays the zero value, StrengthNone),
	// so once LaneStrict is measured its floor (StrengthProperties) is
	// breached.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "main.go", Line: 1}, own))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "main.go", Line: 1, ReasonTag: "catch", Source: "wo:1"}))
	require.NoError(t, log.AppendStatus(1, "done"))

	// Order 2: a plain lifecycle advisory hold (done, not caught) — its lane
	// is never measured (LaneUnmeasured, floor StrengthNone) so it can never
	// be forced blocking; it stays exactly what Fold set.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 2}, own))
	require.NoError(t, log.AppendStatus(2, "done"))

	registerSession("holdlanefloor", LiveConfig{RepoDir: repo, BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	tc := vt.NewClient(t, server, "/?key=holdlanefloor")
	frames, cancel := tc.SSE()
	defer cancel()

	// Before any Inspector visit, order 1's lane has never been measured —
	// only order 2's advisory hold shows.
	vt.AwaitFrame(t, frames, 10*time.Second, "needs you · 1")

	e := lookupLiveEntry("holdlanefloor")
	require.NotNil(t, e)
	e.laneMu.Lock()
	_, laneCached := e.laneCache[1]
	e.laneMu.Unlock()
	assert.False(t, laneCached, "order 1's lane is unmeasured before any Inspector visit")

	// Visiting order 1's Inspector is the ONLY place allowed to exec
	// packet.Measure — this caches LaneStrict for order 1.
	_ = bodyOf(vt.NewClient(t, server, "/review?key=holdlanefloor&wo=1").HTML())
	e.laneMu.Lock()
	lane, laneCached := e.laneCache[1]
	e.laneMu.Unlock()
	require.True(t, laneCached, "the Inspector visit measured and cached order 1's lane")
	require.Equal(t, packet.LaneStrict, lane, "the single-package fixture's own change ripples through 100% of its 1-package graph")

	// Caching a lane touches no dispatch status/caught/thread count, so the
	// poll's own re-render signature (live.go's OnConnect) would otherwise
	// stay unchanged and never push a fresh frame — a third order's status
	// change is the observable trigger the sibling poll-safety tests
	// use to force the poll to notice and re-render over this connection.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 102, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d3", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "gamma.go", Line: 3}, own))
	require.NoError(t, log.AppendStatus(3, "running"))

	// The console's NEXT poll tick reads the now-cached lane (a pure map
	// read) and reconciles order 1 into a forced blocking hold — sorted
	// ahead of order 2's plain advisory hold. All three needles are awaited
	// in ONE call so the index comparison below reads off the SAME frame,
	// never a later one that might have moved on.
	frame := vt.AwaitFrame(t, frames, 10*time.Second, "needs you · 2", "handshake below lane floor", "gap found")
	blockingIdx := strings.Index(frame, "handshake below lane floor")
	advisoryIdx := strings.Index(frame, "gap found")
	require.True(t, blockingIdx >= 0 && advisoryIdx >= 0, "both hold reasons render")
	assert.Less(t, blockingIdx, advisoryIdx, "the lane-floor-forced blocking hold renders ahead of the plain advisory one")
}
