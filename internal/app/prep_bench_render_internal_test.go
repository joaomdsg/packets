package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/ledger"
)

// The bench chip GREW INTO A CARD: each fundable target now carries both the fund
// affordance (the curation decision, flow d) AND a sharpen body so the Lead can
// refine the work during dead-air without leaving the card. NOT parallel (globals).
func TestLiveCard_benchItemRendersAsACardWithFundAndSharpenAffordances(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	// The fund affordance survives the grow-into-a-card (flow d is unchanged).
	require.Contains(t, body, "bench__fund", "the card still funds the chosen target")
	require.Contains(t, body, `data-target="pay.go:88"`, "the fund affordance carries the target key")
	// The new sharpen affordances: a collapsible body with an input and a submit.
	require.Contains(t, body, "bench__sharpen", "a sharpen affordance opens the card's body")
	require.Contains(t, body, "bench__criteria", "the sharpen body has an acceptance-criteria input")
	require.Contains(t, body, "bench__refine", "the sharpen body has a refine submit")
	// Scoped human-readable copy: the criteria input is labelled (a11y + stripped-CSS
	// legible), not merely a class that happens to contain the word.
	require.Contains(t, body, `aria-label="acceptance criteria`, "the criteria input carries a plain-words accessible label")
}

// A target the Lead already sharpened shows its attached criteria on the card, so
// the refinement is visible (and survives a reopen — it is a logged fact). NOT
// parallel (shared globals).
func TestLiveCard_benchCardShowsAnAttachedCriterion(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{
		Target: ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88},
		Refine: "criteria", Criteria: []string{"rejects a negative amount"},
	}))

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "bench__anno", "an attached refinement renders as an annotation on the card")
	require.Contains(t, body, "rejects a negative amount", "the criterion the Lead attached is shown on the bench card")
}

// A convention the Lead noted on a target shows on its card too — the symmetric
// annotation to criteria, so both sharpening kinds are visible. NOT parallel.
func TestLiveCard_benchCardShowsAnAttachedConvention(t *testing.T) {
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.AppendRefine(ledger.RefinedOrderRecord{
		Target: ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "pay.go", Line: 88},
		Refine: "convention", Note: "wrap errors with an origin prefix",
	}))

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	require.Contains(t, body, "bench__anno", "the convention renders as an annotation on the card")
	require.Contains(t, body, "wrap errors with an origin prefix", "the convention the Lead noted is shown on the bench card")
}
