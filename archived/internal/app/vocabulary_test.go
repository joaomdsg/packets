package app_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// bannedWordPattern is the source of truth for the retired vocabulary: it matches
// any of those banned words as a WHOLE word
// (optionally plural/possessive), case-insensitively — never a substring match,
// so "coordinator" doesn't trip on "order" and "island" doesn't trip on "land".
// "review" is banned only as a NOUN (the verb "inspect"/"review a
// packet" is fine) — this pattern can't distinguish the two, so it bans the
// bare word outright: nothing in this app currently needs the verb form, and
// a future false positive here is a cheap prompt to double-check it isn't the
// retired noun creeping back in.
var bannedWordPattern = regexp.MustCompile(
	`(?i)\b(PRs?|merges?|merged|merging|approves?|approved|approving|orders?|sessions?|boards?|oracles?|verdicts?|` +
		`reviews?|lands?|landed|landing|bounced|drafts?|benche?s?|spends?|spent|spending|stocks?|balances?|bets?|LGTM)\b`,
)

// stripNonVisible removes <script>/<style> blocks (JS/CSS/JSON payloads, never
// prose a human reads) and then every remaining tag, leaving only visible text
// nodes — so an href="/board" or a data-state="land-clean" attribute can never
// be mistaken for rendered copy.
func stripNonVisible(html string) string {
	html = regexp.MustCompile(`(?is)<script.*?</script>`).ReplaceAllString(html, " ")
	html = regexp.MustCompile(`(?is)<style.*?</style>`).ReplaceAllString(html, " ")
	html = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(html, " ")
	return html
}

// assertNoBannedVocabulary fails with every offending word if surface (the
// visible text of one page) still carries any of the retired vocabulary.
func assertNoBannedVocabulary(t *testing.T, surface, body string) {
	t.Helper()
	text := stripNonVisible(bodyOf(body))
	hits := bannedWordPattern.FindAllString(text, -1)
	assert.Empty(t, hits, "%s renders retired vocabulary: %v", surface, hits)
}

// vocabularySweepFixture funds an already-registered session's log spanning
// every packet lifecycle state (composing/queued, in-flight/running, verified,
// held-advisory, held-blocking, delivered) plus an open review thread — the
// widest real surface a single fixture can drive, so the banned-word sweep
// exercises the needs-you rail, the settled rail, and the in-flight strip
// all in one pass. Populates log in place (does not create its own fabric),
// since app.AddSession's log is already bound to the server's shared fabric.
func vocabularySweepFixture(t *testing.T, log *ledger.Log) {
	t.Helper()
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	fund := func(name, path string, line int) int {
		require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "cover-" + name + ".go", Line: 1, ReasonTag: "catch"}))
		require.NoError(t, log.AppendSend(name, ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Prompt: name, Path: path, Line: line}, own))
		rows, err := log.RecentSends(0)
		require.NoError(t, err)
		require.NotEmpty(t, rows)
		return rows[0].ID
	}

	fund("queued", "queued.go", 1) // stays queued — no status appended

	running := fund("running", "running.go", 2)
	require.NoError(t, log.AppendStatus(running, "running"))

	verified := fund("verified", "verified.go", 3)
	require.NoError(t, log.AppendStatus(verified, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "verified.go", Line: 3, ReasonTag: "catch", Source: "wo:" + strconv.Itoa(verified)}))

	missed := fund("missed", "missed.go", 4)
	require.NoError(t, log.AppendStatus(missed, "done")) // done, no matching catch → held advisory

	failedOrder := fund("failed", "failed.go", 5)
	require.NoError(t, log.AppendStatus(failedOrder, "failed")) // held blocking

	delivered := fund("delivered", "delivered.go", 6)
	require.NoError(t, log.AppendStatus(delivered, "done"))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "delivered.go", Line: 6, ReasonTag: "catch", Source: "wo:" + strconv.Itoa(delivered)}))
	require.NoError(t, log.AppendStatus(delivered, "deployed"))
}

// The Console, the fleet board, and the utility settings page must never
// render the retired vocabulary (PR/merge/approve/order/session/board/
// oracle/verdict/land/bounced/draft/bench/spend/stock/balance/bet/LGTM) —
// every packet lifecycle state is present in the fixture so the sweep
// exercises the needs-you rail, the settled rail, and the in-flight strip
// together. NOT parallel (shared liveReg/liveFabric).
func TestSurfaces_neverRenderRetiredVocabulary(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "vocabsweep", app.LiveConfig{BaseRev: "own-b-vocabsweep", FixRev: "own-f", Anchor: anchorForCap()})
	vocabularySweepFixture(t, log)

	assertNoBannedVocabulary(t, "/ (Console)", bodyOf(vt.NewClient(t, server, "/?key=vocabsweep").HTML()))
	assertNoBannedVocabulary(t, "/board (fleet)", bodyOf(vt.NewClient(t, server, "/board").HTML()))
	assertNoBannedVocabulary(t, "/review (Inspector, session-scoped)", bodyOf(vt.NewClient(t, server, "/review?key=vocabsweep").HTML()))
	assertNoBannedVocabulary(t, "/settings", bodyOf(vt.NewClient(t, server, "/settings").HTML()))
}

// A per-packet Inspector view (?wo=<id>) is a DISTINCT render path from the
// session-scoped one (a different branch in ReviewCard.View) — it must be
// swept separately. NOT parallel (shared liveReg/liveFabric).
func TestSurfaces_perPacketInspectorNeverRendersRetiredVocabulary(t *testing.T) {
	server, _ := bootDefaultServer(t, defaultBootCfg)
	log := addFundedSession(t, "vocabsweeporder", app.LiveConfig{BaseRev: "own-b-vocabsweeporder", FixRev: "own-f", Anchor: anchorForCap()})
	vocabularySweepFixture(t, log)

	rows, err := log.RecentSends(0)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	for _, r := range rows {
		body := bodyOf(vt.NewClient(t, server, "/review?key=vocabsweeporder&wo="+strconv.Itoa(r.ID)).HTML())
		assertNoBannedVocabulary(t, "/review?wo="+strconv.Itoa(r.ID), body)
	}
}
