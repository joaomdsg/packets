package app_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// gitInitNoRemote initializes a bare local repo with no origin remote, so
// packet.ParseAddr on it deterministically falls back to the honest
// "local/<dir>" identity instead of resolving a real remote.
func gitInitNoRemote(t testing.TB, dir string) {
	t.Helper()
	require.NoError(t, exec.Command("git", "init", "-q", dir).Run())
}

// "/" grows a 3-column Console shell around the untouched
// live-card content — a needs-you rail, the preserved center column, and a
// settled+watches rail. Without the grid the Lead has nowhere honest to see
// open questions or settled work at a glance. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_rendersConsoleGridWithThreeRegions(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, `class="console"`, "the card renders the console grid shell")
	require.Contains(t, body, "console__rail--needs-you", "the left rail is the needs-you region")
	require.Contains(t, body, "console__main", "the center column carries the preserved content")
	require.Contains(t, body, "console__rail--settled", "the right rail is the settled+watches region")
}

// The needs-you rail is now driven by the
// packet fold, not the session's raw connect-cycle findings — a HELD packet
// (blocking or advisory) surfaces as a card the Lead can click straight into
// its own /review?wo=<id>. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_needsYouRailShowsAHeldPacketCardWithItsReason(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "needsyou", app.LiveConfig{BaseRev: "own-b-needsyou", FixRev: "own-f", Anchor: anchorForCap()})

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "failed")) // a run failure — held, blocking

	body := bodyOf(vt.NewClient(t, server, "/?key=needsyou").HTML())
	require.Contains(t, body, "needs you · 1", "the panel header counts the held packet")
	require.Contains(t, body, "run failed", "the card names the one-clause hold reason")
	require.Contains(t, body, `href="/review?key=needsyou&amp;wo=1"`, "the card links straight into the packet's own review")
	require.Contains(t, body, "inspect →", "the link carries a trailing arrow naming its destination, per the house voice")
}

// Blocking holds are the most attention-worthy — they must render ahead of
// advisory holds regardless of dispatch recency. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_needsYouRailPacketsBlockingPacketsBeforeAdvisoryOnes(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "needsyouorder", app.LiveConfig{BaseRev: "own-b-needsyouorder", FixRev: "own-f", Anchor: anchorForCap()})
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}

	// Order 1 (oldest — RecentSends is newest-first, so without the
	// blocking-before-advisory sort this would render SECOND): a run failure.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 1}, own))
	require.NoError(t, log.AppendStatus(1, "failed"))
	// Order 2 (newest): done with no catch — an advisory gap, not a failure.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 2}, own))
	require.NoError(t, log.AppendStatus(2, "done"))

	body := bodyOf(vt.NewClient(t, server, "/?key=needsyouorder").HTML())
	blockingIdx := strings.Index(body, "run failed")
	advisoryIdx := strings.Index(body, "gap found")
	require.True(t, blockingIdx >= 0 && advisoryIdx >= 0, "both hold reasons render")
	require.Less(t, blockingIdx, advisoryIdx, "the blocking packet renders before the advisory one — most attention-worthy first")
}

// A session drowning in open questions must stay scannable: the rail shows
// only the first few full cards and collapses the rest into a single "and N
// more" line, rather than growing the rail unboundedly. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_needsYouRailCapsFullCardsAndCollapsesTheRestIntoAMoreLine(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "needsyoucap", app.LiveConfig{BaseRev: "own-b-needsyoucap", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	for i := 1; i <= 5; i++ {
		require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100 + i, ReasonTag: "catch"}))
		require.NoError(t, log.AppendSend("d", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: i}, own))
		require.NoError(t, log.AppendStatus(i, "failed"))
	}

	body := bodyOf(vt.NewClient(t, server, "/?key=needsyoucap").HTML())
	require.Contains(t, body, "needs you · 5", "the header honestly counts every held packet, not just the shown ones")
	require.Equal(t, 4, strings.Count(body, "console__thread-title"),
		"only the capped number of full cards render")
	require.Contains(t, body, "and 1 more", "the rest collapse into a single more-line")
}

// An empty needs-you queue is a VICTORY, not a dead screen — the honest empty
// state must say so calmly, never with an alarm or a fabricated placeholder.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_needsYouRailShowsVictoryEmptyStateWhenNoOpenThreads(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "needs you · 0", "the panel header honestly counts zero open threads")
	require.Contains(t, body, "nothing needs you", "the empty state reads as a victory, not a dead end")
	require.Contains(t, body, "console__card--dashed", "the empty state uses the dashed empty-card treatment")
}

// A session with no Verified (auto-forwarded) packets has nothing honest to
// draw from — the calibration mechanic must show the "no draws
// yet" empty state rather than a fabricated sample. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_calibrationRailShowsHonestEmptyStateBeforeItsMechanicExists(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "calibration", "the calibration region is present")
	require.Contains(t, body, "no calibration draws yet", "no fake sample before the mechanic ships")
}

// The settled rail now shows lifecycle-colored rows straight
// from the packet fold — verified, held-advisory, and held-blocking (a run
// failure IS settled-red) all count; only composing/in-flight are excluded.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_settledRailShowsLifecycleColoredRows(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "settledc", app.LiveConfig{BaseRev: "own-b-settledc", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Source: "wo:1"}))

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9}, own))
	require.NoError(t, log.AppendStatus(2, "done")) // left with no matching catch — a settled advisory gap

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 102, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d3", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "gamma.go", Line: 11}, own))
	require.NoError(t, log.AppendStatus(3, "failed")) // a run failure — settled, blocking

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 103, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d4", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "delta.go", Line: 13}, own))
	require.NoError(t, log.AppendStatus(4, "deployed")) // a real ACK — settled, delivered

	body := bodyOf(vt.NewClient(t, server, "/?key=settledc").HTML())
	require.Contains(t, body, "settled · 4", "verified, held-advisory, held-blocking, AND delivered orders all count as settled")
	require.Contains(t, body, `data-state="verified"`, "the caught order's state square is verified")
	require.Contains(t, body, `data-state="held-blocking"`, "the failed order's state square is the blocking-red hue")
	require.Contains(t, body, `data-state="held"`, "the missed order's state square is the advisory-amber hue")
	require.Contains(t, body, `data-state="delivered"`, "the deployed order's state square fills only on a real ACK")
	require.Contains(t, body, "delivered", "the row names its lifecycle state, matching the vocabulary elsewhere")
}

// A session with nothing settled yet must show the honest dashed empty state,
// not an empty region masquerading as content. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_settledRailShowsDashedEmptyWhenNothingSettled(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "settled · 0", "the settled rail honestly counts zero")
	require.Contains(t, body, "nothing settled yet", "the dashed empty state names the honest absence")
}

// The watches region is always present, even before any watch has fired —
// its per-kind "no history yet" content is covered in depth elsewhere.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_watchesRailIsPresentBeforeAnyWatchHasFired(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "your watches", "the watches region is present")
}

// The hero stat counts ONLY packets whose State is Verified
// (done AND the order's own catch minted AND no open questions) — a
// done-but-missed order no longer counts, unlike the old raw Done tally.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_heroStatRendersTheVerifiedDoneCount(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "heroc", app.LiveConfig{BaseRev: "own-b-heroc", FixRev: "own-f", Anchor: anchorForCap()})

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendSend("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.AppendStatus(2, "done")) // done but left with no matching catch — NOT verified
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Source: "wo:1"}))

	body := bodyOf(vt.NewClient(t, server, "/?key=heroc").HTML())
	require.Contains(t, body, `console__hero-stat">1<`, "only the genuinely verified order counts — the done-but-missed one does not")
	require.Contains(t, body, "packets verified", "the hero stat is labelled with the honest mechanized state")
}

// A fresh session with zero done orders must show an honest zero, never omit
// the hero stat — a missing stat would read as a bug, not calm silence.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_heroStatShowsHonestZeroOnAFreshSession(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, `console__hero-stat">0<`, "a fresh session honestly shows zero verified, not a hidden stat")
	require.Contains(t, body, "packets verified", "the label is present even at zero")
}

// The center column gains an "in flight · N" strip under the
// hero — one pulsing signal cell per running order, one ghost-outline cell
// per queued (composing) order. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_inFlightStripShowsPulsingRunningAndGhostQueuedRows(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "inflight", app.LiveConfig{BaseRev: "own-b-inflight", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "running"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9}, own))
	// order 2 gets no status appended — it defaults to "queued" (composing).

	body := bodyOf(vt.NewClient(t, server, "/?key=inflight").HTML())
	require.Contains(t, body, "in flight · 2", "the strip counts both the running and the queued order")
	require.Contains(t, body, `data-state="in-flight"`, "the running order renders the pulsing signal cell")
	require.Contains(t, body, `data-state="composing"`, "the queued order renders the ghost-outline cell")
}

// The strip is a live-activity affordance only — it must be entirely absent
// when nothing is composing or in flight, not an empty header. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_inFlightStripAbsentWhenNothingInFlightOrComposing(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.NotContains(t, body, "console__inflight", "the strip is omitted entirely, not shown empty")
}

// The console hero region carries a mono addr line, using
// the honest repo identity (packet.ParseAddr) rather than a raw folder name —
// a remote-less repo falls back to "local/<dir>", never a fabricated owner.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_addrLineRendersTheLocalFallbackForARemotelessRepo(t *testing.T) {
	dir := t.TempDir()
	gitInitNoRemote(t, dir)

	server, _ := bootDefaultServer(t, defaultBootCfg)
	_ = addFundedSession(t, "addrline", app.LiveConfig{RepoDir: dir})

	body := bodyOf(vt.NewClient(t, server, "/?key=addrline").HTML())
	require.Contains(t, body, "addr local/"+filepath.Base(dir), "a remote-less repo falls back to the honest local/<dir> identity")
}

// The four retired economy meter rows (stock/balance/
// bandwidth/dispatch) must never render on the console — these are retired
// from the UI outright. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_economyMeterRowsAreGoneFromTheConsole(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	for _, hook := range []string{
		`data-state="stock"`, `data-state="balance"`, `data-state="bandwidth"`, `data-state="dispatch"`,
	} {
		require.NotContainsf(t, body, hook, "the retired economy meter %q must not render on the console", hook)
	}
}
