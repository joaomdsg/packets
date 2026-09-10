package app_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/packet"
)

// failedDispatch mints a catch to cover the dispatch's cost (AppendSend
// refuses a zero balance — mirrors the calibration fixture),
// funds one work order, and fails it — a "failed" status folds to a
// HoldBlocking packet, the cheapest real trigger for
// WatchBlockingHold without needing a real mutation/build cycle. Returns the
// new order's id.
func failedDispatch(t *testing.T, log *ledger.Log, name string) int {
	t.Helper()
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "cover-" + name + ".go", Line: 1, ReasonTag: "catch"}))
	target := ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Prompt: name, Path: "alpha.go", Line: 7}
	require.NoError(t, log.AppendSend(name, target, own))
	rows, err := log.RecentSends(0)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	id := rows[0].ID
	require.NoError(t, log.AppendStatus(id, "failed"))
	return id
}

// A watch with nothing recorded yet must say so literally — never a
// fabricated score. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_watchesRailReportsNoHistoryYetForEveryKindBeforeAnythingFires(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{RepoDir: "."})

	body := bodyOf(vt.NewClient(t, server, "/").HTML())

	assert.Contains(t, body, "strict-lane")
	assert.Contains(t, body, "gate-failure")
	assert.Contains(t, body, "blocking-hold")
	assert.Equal(t, 3, strings.Count(body, "no history yet"),
		"all three canonical watches report the honest empty state before any fire is recorded")
}

// A packet that trips a watch's predicate must surface a mark prompt — the
// human judgment Precision is computed from. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_watchesRailOffersAMarkPromptForAnUnmarkedFire(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{RepoDir: "."})
	log := addFundedSession(t, "watchfireprompt", app.LiveConfig{BaseRev: "own-b-watchfireprompt", FixRev: "own-f", Anchor: anchorForCap()})
	failedDispatch(t, log, "d1")

	body := bodyOf(vt.NewClient(t, server, "/?key=watchfireprompt").HTML())

	assert.Contains(t, body, "/_action/MarkWatchFire", "an unmarked fire offers the mark affordance")
	assert.Contains(t, body, "d1", "the prompt names the packet that tripped the watch")
}

// Marking a fire useful must move it out of "no history yet" and into a real
// precision score, and the mark prompt for that fire must disappear (it's
// resolved — asking again would be noise on the human, not the trigger). NOT
// parallel (shared liveReg/liveFabric).
func TestLiveCard_markingAFireUsefulUpdatesPrecisionAndClearsThePrompt(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{RepoDir: "."})
	log := addFundedSession(t, "watchfiremark", app.LiveConfig{BaseRev: "own-b-watchfiremark", FixRev: "own-f", Anchor: anchorForCap()})
	id := failedDispatch(t, log, "d1")

	// A render records the fire before it can be marked.
	_ = bodyOf(vt.NewClient(t, server, "/?key=watchfiremark").HTML())

	tc := vt.NewClient(t, server, "/?key=watchfiremark")
	require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "watchfiremark"}).MarkWatchFire).
		WithSignal("markwatchkind", strconv.Itoa(int(packet.WatchBlockingHold))).
		WithSignal("markwatchwo", strconv.Itoa(id)).
		WithSignal("markuseful", "true").
		Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=watchfiremark").HTML())
	assert.Contains(t, body, "1/1 useful", "one marked-useful fire out of one sampled")
	assert.NotContains(t, body, "/_action/MarkWatchFire", "a resolved fire no longer offers a mark prompt")
}

// Rendering the same held packet across many polls must record ONE fire, not
// one per render — otherwise Precision's sample size would inflate with
// nothing but repeated renders, corrupting the score. NOT parallel (shared
// liveReg/liveFabric).
func TestLiveCard_repeatedRendersRecordExactlyOneFirePerPacket(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{RepoDir: "."})
	log := addFundedSession(t, "watchfiredupe", app.LiveConfig{BaseRev: "own-b-watchfiredupe", FixRev: "own-f", Anchor: anchorForCap()})
	id := failedDispatch(t, log, "d1")

	for i := 0; i < 5; i++ {
		_ = bodyOf(vt.NewClient(t, server, "/?key=watchfiredupe").HTML())
	}

	tc := vt.NewClient(t, server, "/?key=watchfiredupe")
	require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "watchfiredupe"}).MarkWatchFire).
		WithSignal("markwatchkind", strconv.Itoa(int(packet.WatchBlockingHold))).
		WithSignal("markwatchwo", strconv.Itoa(id)).
		WithSignal("markuseful", "false").
		Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=watchfiredupe").HTML())
	assert.Contains(t, body, "0/1 useful",
		"exactly one fire was ever recorded for this packet across 5 renders — a dedup bug would inflate the sample")

	// A SECOND mark attempt for the same (kind, packet) must find nothing left
	// to mark — proving there was only ever ONE fire, not "0/1 useful" by
	// coincidence while 4 leftover unmarked duplicates still sat unexcluded.
	// If dedup were broken, this second mark would find one of those leftovers
	// and flip the score to "1/2 useful".
	tc2 := vt.NewClient(t, server, "/?key=watchfiredupe")
	require.Equal(t, 200, tc2.Action((&app.LiveCard{Key: "watchfiredupe"}).MarkWatchFire).
		WithSignal("markwatchkind", strconv.Itoa(int(packet.WatchBlockingHold))).
		WithSignal("markwatchwo", strconv.Itoa(id)).
		WithSignal("markuseful", "true").
		Fire())

	body = bodyOf(vt.NewClient(t, server, "/?key=watchfiredupe").HTML())
	assert.Contains(t, body, "0/1 useful",
		"a second mark on an already-resolved (kind,packet) pair must be a no-op — proves dedup, not just a lucky score")
}

// Marking one packet's fire must not affect a DIFFERENT packet's own,
// independent mark prompt — mirrors gauntlet's "confirming a second order
// doesn't clobber the first". NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_markingOneFireDoesNotClobberAnotherPacketsFire(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{RepoDir: "."})
	log := addFundedSession(t, "watchfiretwo", app.LiveConfig{BaseRev: "own-b-watchfiretwo", FixRev: "own-f", Anchor: anchorForCap()})
	id1 := failedDispatch(t, log, "d1")
	id2 := failedDispatch(t, log, "d2")

	_ = bodyOf(vt.NewClient(t, server, "/?key=watchfiretwo").HTML())

	tc := vt.NewClient(t, server, "/?key=watchfiretwo")
	require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "watchfiretwo"}).MarkWatchFire).
		WithSignal("markwatchkind", strconv.Itoa(int(packet.WatchBlockingHold))).
		WithSignal("markwatchwo", strconv.Itoa(id1)).
		WithSignal("markuseful", "true").
		Fire())

	body := bodyOf(vt.NewClient(t, server, "/?key=watchfiretwo").HTML())
	assert.Contains(t, body, "1/1 useful", "only d1's fire is marked so far")
	assert.Contains(t, body, "d2", "d2's own fire still needs a mark — untouched by d1's action")
	assert.Contains(t, body, "/_action/MarkWatchFire", "d2's mark prompt is still offered")

	tc2 := vt.NewClient(t, server, "/?key=watchfiretwo")
	require.Equal(t, 200, tc2.Action((&app.LiveCard{Key: "watchfiretwo"}).MarkWatchFire).
		WithSignal("markwatchkind", strconv.Itoa(int(packet.WatchBlockingHold))).
		WithSignal("markwatchwo", strconv.Itoa(id2)).
		WithSignal("markuseful", "false").
		Fire())

	body = bodyOf(vt.NewClient(t, server, "/?key=watchfiretwo").HTML())
	assert.Contains(t, body, "1/2 useful", "d1's earlier mark and d2's new mark BOTH count — neither clobbered the other")
}

// A malformed signal (blank/non-numeric packet id, or a kind that doesn't
// parse) must be a calm no-op — mirroring ConfirmIntentFidelity's handling of
// bad input. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_markWatchFireWithMalformedSignalsIsASilentNoOp(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{RepoDir: "."})
	log := addFundedSession(t, "watchfirebad", app.LiveConfig{BaseRev: "own-b-watchfirebad", FixRev: "own-f", Anchor: anchorForCap()})
	id := failedDispatch(t, log, "d1")

	before := bodyOf(vt.NewClient(t, server, "/?key=watchfirebad").HTML())
	require.Contains(t, before, "/_action/MarkWatchFire")

	tc := vt.NewClient(t, server, "/?key=watchfirebad")
	require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "watchfirebad"}).MarkWatchFire).
		WithSignal("markwatchkind", "not-a-number").
		WithSignal("markwatchwo", strconv.Itoa(id)).
		WithSignal("markuseful", "true").
		Fire(), "a malformed kind must not error the action itself")

	tc2 := vt.NewClient(t, server, "/?key=watchfirebad")
	require.Equal(t, 200, tc2.Action((&app.LiveCard{Key: "watchfirebad"}).MarkWatchFire).
		WithSignal("markwatchkind", strconv.Itoa(int(packet.WatchBlockingHold))).
		WithSignal("markwatchwo", "").
		WithSignal("markuseful", "true").
		Fire(), "a blank packet id must not error the action itself")

	after := bodyOf(vt.NewClient(t, server, "/?key=watchfirebad").HTML())
	assert.Contains(t, after, "/_action/MarkWatchFire", "malformed input left the fire exactly as unmarked as before")
	assert.Contains(t, after, "no history yet", "no fabricated mark was ever recorded")
}

// A watch that racks up mostly-noise marks across a real sample must lose the
// right to interrupt: its precision line still shows the real score, but new
// fires no longer prompt for a mark. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_aWatchWithMostlyNoiseMarksAcrossARealSampleStopsPromptingForNewFires(t *testing.T) {
	server, _ := bootDefaultServer(t, app.LiveConfig{RepoDir: "."})
	log := addFundedSession(t, "watchnoisy", app.LiveConfig{BaseRev: "own-b-watchnoisy", FixRev: "own-f", Anchor: anchorForCap()})

	ids := make([]int, 5)
	for i := range ids {
		ids[i] = failedDispatch(t, log, "d"+strconv.Itoa(i+1))
	}
	// Record all 5 fires, then mark every one of them noise (a real sample of 5).
	_ = bodyOf(vt.NewClient(t, server, "/?key=watchnoisy").HTML())
	for _, id := range ids {
		tc := vt.NewClient(t, server, "/?key=watchnoisy")
		require.Equal(t, 200, tc.Action((&app.LiveCard{Key: "watchnoisy"}).MarkWatchFire).
			WithSignal("markwatchkind", strconv.Itoa(int(packet.WatchBlockingHold))).
			WithSignal("markwatchwo", strconv.Itoa(id)).
			WithSignal("markuseful", "false").
			Fire())
	}

	resolvedBody := bodyOf(vt.NewClient(t, server, "/?key=watchnoisy").HTML())
	assert.Contains(t, resolvedBody, "0/5 useful", "the real precision score renders")
	assert.Contains(t, resolvedBody, "noisy — lost interrupt rights")
	assert.Equal(t, 0, strings.Count(resolvedBody, "/_action/MarkWatchFire"),
		"all 5 fires are resolved — no mark prompt remains")

	// A sixth failed order trips the SAME watch kind again — it must be
	// recorded (precision tracking continues) but, being noisy, must NOT
	// surface a new mark prompt. Asserting the exact affordance count (not a
	// name-based needle like the dispatch's own id, which could coincidentally
	// appear elsewhere in the page) is the robust check here.
	failedDispatch(t, log, "d6")
	body := bodyOf(vt.NewClient(t, server, "/?key=watchnoisy").HTML())

	assert.Contains(t, body, "0/5 useful", "precision is unaffected by an unmarked new fire")
	assert.Equal(t, 0, strings.Count(body, "/_action/MarkWatchFire"),
		"a noisy watch's new fire must not add a mark prompt")
}
