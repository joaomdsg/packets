package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// A brand-new session lands the Lead on a card showing nothing but bare zeros —
// 0 confirmed, balance 0, 0 dispatched — with no signal for WHAT to do or WHY
// nothing is happening. That dead first screen strands a first-run user at the
// entry to the core loop. A fresh session must carry a calm onboarding affordance
// that names the next action in the real flow (catch → mint → spend → reinvest).
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_freshSessionShowsOnboardingAffordance(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, `data-state="empty"`,
		"a fresh session card must carry the empty-state onboarding affordance, not a dead screen of zeros")
	require.Contains(t, body, "onboarding",
		"the affordance uses the onboarding class hook so the stylesheet can style it")
	// The affordance must name each honest step of the real loop so the Lead
	// understands WHY the screen is blank and WHAT moves it — never a fabricated
	// metric. (1) the honest current state, (2) how a catch is minted, (3) the
	// next action the Lead takes.
	require.Contains(t, body, "No confirmed catches yet",
		"the affordance names the honest current state of a fresh session")
	require.Contains(t, body, "mints to your balance",
		"the affordance explains how the oracle's catch becomes spendable balance")
	require.Contains(t, body, "Spend",
		"the affordance names the real next action (spend a catch to fund work)")
}

// The onboarding affordance is for the EMPTY state only — once a session has
// activity it is noise that competes with the real economy rows. A session with a
// confirmed catch (stock + balance > 0) must NOT render it.
func TestLiveCard_activeSessionHidesOnboarding(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// Mint one real confirmed catch into the served session's ledger — now the card
	// has stock and a spendable balance, so it is no longer a fresh session.
	require.NoError(t, log.Append(*ledger.NewCatchRecord(
		catch.Catch, "pkg/file.go", 1, "b", "f", nil, nil, false, false)))

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.NotContains(t, body, `data-state="empty"`,
		"an active session must not render the empty-state onboarding affordance")
}
