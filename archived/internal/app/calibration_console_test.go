package app_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

const dryAsideLine = "✱ an empty queue is success, not idleness."

// The needs-you rail's victory empty state must carry the house voice's one
// permitted dry aside (design/guidelines/voice.md rule 10), verbatim from
// concepts.md's attention-economics framing — empty is a success, never a
// dead end. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_needsYouRailShowsTheDryAsideOnlyWhenTrulyEmpty(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Equal(t, 1, strings.Count(body, dryAsideLine),
		"the dry aside appears exactly once on the truly-empty needs-you rail")
}

// A held packet means the queue is NOT empty — the dry aside is a victory
// line for the empty case only, never appended alongside real cards. NOT
// parallel (shared liveReg/liveFabric).
func TestLiveCard_needsYouRailOmitsTheDryAsideWhenPacketsAreHeld(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "dryasideheld", app.LiveConfig{BaseRev: "own-b-dryasideheld", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "failed"))

	body := bodyOf(vt.NewClient(t, server, "/?key=dryasideheld").HTML())
	require.NotContains(t, body, dryAsideLine, "the dry aside never renders alongside real held cards")
}

// A real Verified (auto-forwarded) packet must replace the dashed
// calibration placeholder with a real drawn card: mono kicker, the packet's
// Name, and a trailing "skim →" link into its own review. NOT parallel
// (shared liveReg/liveFabric).
func TestLiveCard_calibrationCardShowsADrawnVerifiedPacketWithASkimLink(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "calibdrawn", app.LiveConfig{BaseRev: "own-b-calibdrawn", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Prompt: "skim me packet", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Source: "wo:1"}))

	body := bodyOf(vt.NewClient(t, server, "/?key=calibdrawn").HTML())
	require.Contains(t, body, "calibration", "the calibration kicker still names the region")
	require.Contains(t, body, "skim-me-packet", "the drawn card names the real verified packet")
	require.Contains(t, body, `href="/review?key=calibdrawn&amp;wo=1"`, "the card links straight into the drawn packet's own review")
	require.Contains(t, body, "skim →", "the link carries the house-voice trailing arrow naming its destination")
	require.NotContains(t, body, "no calibration draws yet", "a real draw replaces the dashed placeholder")
}

// The draw must stay STABLE across repeated renders (the 100ms poll re-runs
// renderConsole constantly) — with more than one qualifying candidate, a
// non-cached implementation would flicker between them across requests. NOT
// parallel (shared liveReg/liveFabric).
func TestLiveCard_calibrationCardStaysStableAcrossRepeatedRenders(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "calibstable", app.LiveConfig{BaseRev: "own-b-calibstable", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Source: "wo:1"}))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9}, own))
	require.NoError(t, log.AppendStatus(2, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "beta.go", Line: 9, ReasonTag: "catch", Source: "wo:2"}))

	first := bodyOf(vt.NewClient(t, server, "/?key=calibstable").HTML())
	firstHref := extractCalibrationHref(t, first)
	for i := 0; i < 10; i++ {
		body := bodyOf(vt.NewClient(t, server, "/?key=calibstable").HTML())
		require.Equal(t, firstHref, extractCalibrationHref(t, body), "the drawn candidate must not flicker across repeated renders")
	}
}

// A held packet and a Verified (auto-forwarded) packet must coexist
// correctly on the same render: the held card keeps naming its own hold
// reason and review link, and the calibration card independently draws from
// the Verified packet — one never masks or gets confused with the other. NOT
// parallel (shared liveReg/liveFabric).
func TestLiveCard_calibrationCardCoexistsWithAHeldNeedsYouCard(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "calibcoexist", app.LiveConfig{BaseRev: "own-b-calibcoexist", FixRev: "own-f", Anchor: anchorForCap()})

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "failed")) // held, blocking

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9}, own))
	require.NoError(t, log.AppendStatus(2, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "beta.go", Line: 9, ReasonTag: "catch", Source: "wo:2"})) // verified

	body := bodyOf(vt.NewClient(t, server, "/?key=calibcoexist").HTML())
	require.Contains(t, body, "run failed", "the held card's own hold reason still renders")
	require.Contains(t, body, `href="/review?key=calibcoexist&amp;wo=1"`, "the held card still links to its own order")
	require.Contains(t, body, "skim →", "the calibration card renders independently alongside the held card")
	require.Equal(t, "/review?key=calibcoexist&amp;wo=2", extractCalibrationHref(t, body),
		"the calibration draw links to the VERIFIED order, never the held one")
}

// extractCalibrationHref pulls the href of the calibration card SPECIFICALLY
// — not the unrelated dispatch "inspect diffs" link the state-history section
// renders with the SAME /review?...&wo=N shape, and not any nav-header href
// preceding the whole console grid. It first isolates the needs-you rail
// (where the calibration slot lives), then within THAT isolated snippet walks
// backward from the kicker text to the nearest enclosing <a href=...>, since
// h.A renders the href attribute before its children — the dashed empty
// placeholder wraps the same kicker text in a plain <div>, which carries no
// href at all, so a still-dashed rail fails this lookup loudly rather than
// silently matching something unrelated.
func extractCalibrationHref(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "console__rail--needs-you")
	require.GreaterOrEqual(t, start, 0, "the needs-you rail must be present")
	// Bounded at the closing </aside>, NOT the next rail's opening marker — the
	// needs-you aside is followed by the WHOLE center column (console__main,
	// which also carries a same-shaped /review?...&wo=N "inspect diffs" link)
	// before the settled aside even starts. Stopping at </aside> keeps that
	// center-column content out of `rail` entirely, rather than relying on
	// kIdx happening to sit before it.
	railEnd := strings.Index(body[start:], "</aside>")
	require.Greater(t, railEnd, 0, "the needs-you aside must close")
	rail := body[start : start+railEnd]

	const kicker = `<div class="console__empty-kicker">calibration</div>`
	kIdx := strings.Index(rail, kicker)
	require.GreaterOrEqual(t, kIdx, 0, "the calibration kicker must be present in the needs-you rail")
	prefix := rail[:kIdx]
	hrefIdx := strings.LastIndex(prefix, `href="`)
	require.GreaterOrEqual(t, hrefIdx, 0, "the calibration card must be a real link (<a href=...>), not the dashed placeholder")
	rest := prefix[hrefIdx+len(`href="`):]
	end := strings.Index(rest, `"`)
	require.Greater(t, end, 0)
	return rest[:end]
}
