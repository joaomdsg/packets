package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// After a Spend funds a work-order, the Lead on the session card needs to watch
// THAT order resolve caught-or-missed — the payoff of the spend. Today the card
// shows only aggregate dispatch counts (queued/running/done); the per-order
// round-trip lives only on the fleet board, forcing a context-switch off the card
// the Lead is acting on. The live card must surface this session's recent
// work-orders with their caught/missed outcome, closing spend → dispatch → watch
// it resolve on one surface. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_showsThisSessionsSendRoundTripOutcomes(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "cardc", app.LiveConfig{BaseRev: "own-b-cardc", FixRev: "own-f", Anchor: anchorForCap()})

	// Fund two work-orders (two catches → balance 2), run both: PKT#1 mints (caught),
	// PKT#2 does not (missed) — the same round-trip the board already surfaces.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendSend("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.AppendStatus(2, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Source: "wo:1"}))

	body := vt.NewClient(t, server, "/?key=cardc").HTML()
	require.Contains(t, body, "PKT#1", "the card shows the caught order by id")
	require.Contains(t, body, "alpha.go:7", "with its target line")
	require.Contains(t, body, "caught", "PKT#1 minted → caught, shown on the card")
	require.Contains(t, body, "PKT#2", "the card shows the missed order too")
	require.Contains(t, body, "missed", "PKT#2 ran but minted nothing → missed, shown on the card")
	// Each resolved order carries a per-outcome hook so the calm palette can color
	// caught vs missed — the round-trip outcome legible at a glance, not
	// undifferentiated dim text (extends the same per-state honesty to dispatches).
	require.Contains(t, body, `data-outcome="caught"`, "the caught order carries a per-outcome hook")
	require.Contains(t, body, `data-outcome="missed"`, "the missed order carries a per-outcome hook")
	// The stylesheet colors both outcomes in the honest palette (selectors live in
	// the head), never an alarm red/green.
	require.Contains(t, body, `[data-outcome="caught"]`, "the stylesheet colors a caught order")
	require.Contains(t, body, `[data-outcome="missed"]`, "the stylesheet colors a missed order")
}

// A session that has funded no work-orders must NOT render an empty dispatch
// cluster — an empty round-trip block is visual noise that implies activity where
// there is none. The cluster is omitted entirely until there is an order to show.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_omitsTheSendClusterWhenNoPacketsFunded(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)

	// bodyOf scopes past the <head>: the stylesheet contains "board-row__sends"
	// as a CSS selector, so a whole-page check would always match.
	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.NotContains(t, body, "board-row__sends",
		"a session with no funded orders renders no dispatch cluster, not an empty block")
}

// The whole point of a filled order is to inspect the changes the peer made —
// but the only drill-link into a settled order's diff was gated on it leaving open
// questions (surviving mutants). An order that fills cleanly, OR a miss that left
// zero questions (no mutable operator, lost-via-rename, oracle-incomplete), then
// surfaced NO clickable path to its own base→fix diff — the changes were
// unreachable by click. Every settled order must link into its /review diff.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_settledPacketAlwaysLinksToItsDiffEvenWithNoOpenQuestions(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "inspectc", app.LiveConfig{BaseRev: "own-b-inspectc", FixRev: "own-f", Anchor: anchorForCap()})

	// One catch funds one order; it runs to done but mints nothing (missed) and
	// leaves zero findings → zero open questions: exactly the no-link blind spot.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))

	body := bodyOf(vt.NewClient(t, server, "/?key=inspectc").HTML())
	require.Contains(t, body, "missed", "the order ran to done and minted nothing")
	require.Contains(t, body, `href="/review?key=inspectc&amp;wo=1"`,
		"a settled order with zero open questions still links into its base→fix diff")
	require.Contains(t, body, "inspect diffs",
		"the drill link reads as a diff inspection, not an open-questions count")
}

// The inspect link is about reaching the peer's edits, NOT about the catch
// outcome — so a CAUGHT order that left zero open questions must get it just the
// same as a missed one. Without this the link would read as "only failures are
// inspectable", which is wrong: a clean catch's diff is exactly what a Lead wants
// to review before landing. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_caughtPacketAlsoLinksToItsDiffWithZeroQuestions(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "caughtc", app.LiveConfig{BaseRev: "own-b-caughtc", FixRev: "own-f", Anchor: anchorForCap()})

	// One catch funds the order; it runs to done AND mints its own catch (caught),
	// leaving zero findings → zero open questions.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Source: "wo:1"}))

	body := bodyOf(vt.NewClient(t, server, "/?key=caughtc").HTML())
	require.Contains(t, body, "caught", "the order minted its own catch")
	require.Contains(t, body, `href="/review?key=caughtc&amp;wo=1"`,
		"a caught order links into its diff just like a missed one — inspection is outcome-independent")
	require.Contains(t, body, "inspect diffs")
}

// A queued/running order has not produced any edits yet, so it carries no diff to
// inspect — the inspect link must be gated on a SETTLED (done) order, never shown
// while work is still queued/running (which would 404-equivalent into an empty
// diff). NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_unsettledPacketShowsNoInspectLink(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "pendingc", app.LiveConfig{BaseRev: "own-b-pendingc", FixRev: "own-f", Anchor: anchorForCap()})

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))

	body := bodyOf(vt.NewClient(t, server, "/?key=pendingc").HTML())
	require.Contains(t, body, "queued", "the order has not resolved")
	require.NotContains(t, body, "inspect diffs",
		"a queued order has produced no diff yet — no inspect link before it settles")
}

// A queued/running order has NOT resolved, so it must carry NO outcome hook — it
// stays neutral, never colored caught or missed before it has an outcome. Without
// this guard a bug could paint unresolved work as a confirmed catch or a loss.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_aQueuedPacketCarriesNoOutcomeHook(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "queuedc", app.LiveConfig{BaseRev: "own-b-queuedc", FixRev: "own-f", Anchor: anchorForCap()})

	// One catch funds one order, left QUEUED (no done status, no outcome yet).
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))

	// bodyOf scopes past the head — the stylesheet's [data-outcome=...] selectors
	// live there; we assert the rendered queued ORDER carries no outcome attribute.
	body := bodyOf(vt.NewClient(t, server, "/?key=queuedc").HTML())
	require.Contains(t, body, "PKT#1", "the queued order is shown")
	require.Contains(t, body, "queued", "with its unresolved status")
	require.NotContains(t, body, "data-outcome=",
		"a queued order carries no outcome hook — it is never colored before it resolves")
}
