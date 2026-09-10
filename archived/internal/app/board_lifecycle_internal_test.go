package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"
)

// The fleet board surfaces the §29.2 land lifecycle only for TERMINAL outcomes (merged /
// bounced) — the routine "landed, not yet merged" transient stays off the board so it
// stays calm, exactly like boardLand only surfaces a BLOCKED verdict.
func TestBoardLifecycle_surfacesOnlyTerminalOutcomes(t *testing.T) {
	cases := []struct {
		lc        string
		wantShow  bool
		wantState string
		wantLabel string
	}{
		{string(lifecycleMerged), true, "merged", "forwarded"},
		{string(lifecycleBounced), true, "bounced", "closed, not forwarded"},
		{string(lifecycleLanded), false, "", ""}, // routine transient — board stays calm
		{"", false, "", ""},
		{"weird", false, "", ""}, // never surface an unrecognized state
	}
	for _, tc := range cases {
		state, label, show := boardLifecycle(tc.lc)
		assert.Equal(t, tc.wantShow, show, "show for %q", tc.lc)
		if show {
			assert.Equal(t, tc.wantState, state, "state for %q", tc.lc)
			assert.Equal(t, tc.wantLabel, label, "label for %q", tc.lc)
		}
	}
}

// A merged session shows its outcome across the fleet; a routine landed-not-merged
// session shows NO lifecycle badge (the board stays calm). NOT parallel (shared globals).
func TestBoardCard_surfacesMergedButNotLandedLifecycle(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	merged, err := AddSession("mergedS", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = merged.Close() })
	lookupLiveEntry("mergedS").setLandLifecycle(string(lifecycleMerged))

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	assert.Contains(t, body, "board-row__lifecycle", "a merged session surfaces its lifecycle across the fleet")
	assert.Contains(t, body, `data-state="merged"`, "with its honest-color state hook")
}

// A landed-not-merged (routine, non-terminal) session shows NO lifecycle badge — the
// board only flags terminal merge outcomes. NOT parallel (shared globals).
func TestBoardCard_omitsLifecycleForRoutineLanded(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	landed, err := AddSession("landedS", LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = landed.Close() })
	lookupLiveEntry("landedS").setLandLifecycle(string(lifecycleLanded))

	body := bodyOf(vt.NewClient(t, server, "/board").HTML())
	assert.NotContains(t, body, "board-row__lifecycle",
		"a routine landed-not-merged session flags nothing — the board stays calm")
}
