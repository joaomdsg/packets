package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// actNowCardBody stands up a session with BOTH a spendable balance (one confirmed
// catch) and earned attention bandwidth (one fast-cleared block), so every act-now
// affordance — spend, bench, authoring, place-order — renders. It returns the live
// card body for that session. NOT parallel (shared liveReg/liveFabric).
func actNowCardBody(t *testing.T) string {
	t.Helper()
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "actnow", "i")

	// A confirmed catch → balance 1 (spend + bench render).
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "seed.go", Line: 1, ReasonTag: "catch"}))
	// A fast-cleared block → bandwidth (authoring + place-order render).
	base := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("wo:1", base))
	require.NoError(t, log.AppendUnblock("wo:1", base.Add(30*time.Second)))
	registerSession("actnow", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"},
		DispatchBacklog: []ledger.Target{
			{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "deck.go", Line: 9},
		},
	}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	return bodyOf(vt.NewClient(t, server, "/?key=actnow").HTML())
}

// FLOW A: the live card mixes act-now controls with retrospective state in a flat
// scroll. Without sub-landmarks an assistive-tech user can't tell "what I act on"
// from "what already happened". Two <section aria-labelledby> regions, each headed
// by a stable-id .pk-section-label, make that split navigable.
func TestLiveCard_splitsActNowFromStateWithLabelledSections(t *testing.T) {
	body := actNowCardBody(t)

	require.Contains(t, body, `id="act-now-label"`, "the act-now heading carries a stable id")
	require.Contains(t, body, `aria-labelledby="act-now-label"`, "the act-now section is labelled by its heading")
	require.Contains(t, body, `id="state-history-label"`, "the state/history heading carries a stable id")
	require.Contains(t, body, `aria-labelledby="state-history-label"`, "the state/history section is labelled by its heading")
	require.Contains(t, strings.ToLower(body), "<section", "the regions are real <section> landmarks")
	require.Contains(t, body, "pk-section-label", "each region heading reuses the PR1 section-label component")
}

// FLOW A: the act-now controls must live INSIDE the act-now section, not scattered
// — otherwise the landmark is a hollow label that doesn't actually group the moves
// the Lead acts on.
func TestLiveCard_actNowControlsRenderInsideTheActNowSection(t *testing.T) {
	body := actNowCardBody(t)

	start := strings.Index(body, `aria-labelledby="act-now-label"`)
	require.GreaterOrEqual(t, start, 0, "the act-now section is present")
	end := strings.Index(body, `aria-labelledby="state-history-label"`)
	require.Greater(t, end, start, "the state/history section follows act-now")
	actNow := body[start:end]

	require.Contains(t, actNow, "spend-action", "the spend control is inside act-now")
	require.Contains(t, actNow, "bench", "the prep bench is inside act-now")
	require.Contains(t, actNow, "compose", "the authoring composer is inside act-now")
	require.Contains(t, actNow, "compose__place", "place-order is inside act-now")
}

// FLOW A (NotRegress): sectioning must NEST inside the live card's main landmark,
// never move or duplicate it — the SSE live region the accessibility contract
// depends on (R61) must survive untouched.
func TestLiveCard_sectioningKeepsTheMainLiveRegionIntact(t *testing.T) {
	body := actNowCardBody(t)

	require.Contains(t, body, `role="main"`, "the live card is still the page main landmark")
	require.Contains(t, body, `aria-live="polite"`, "the SSE live region survives the sectioning")
	require.Equal(t, 1, strings.Count(body, `role="main"`), "main is not duplicated by the sectioning")
	require.Equal(t, 1, strings.Count(body, `aria-live="polite"`), "the live region is not duplicated")
}

// FLOW A (NotRegress): a brand-new session has nothing to act on, so the act-now
// section is OMITTED (not an empty-shouting heading) — but the onboarding guide
// still renders, so a first-run Lead is never stranded.
func TestLiveCard_freshSessionOmitsActNowButKeepsOnboarding(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.NotContains(t, body, `aria-labelledby="act-now-label"`,
		"a fresh session has nothing to act on — the act-now section is omitted, not empty-shouting")
	require.Contains(t, body, `data-state="empty"`, "a fresh session still renders the onboarding guide")
}
