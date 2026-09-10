package surface_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/surface"
)

func TestPresentVerdict_inFlightWinsOverAnyOutcomeWhileRunning(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", surface.PresentVerdict(true, catch.Catch, pipe.ReasonNone, 1, 0), "a running cycle has no verdict yet")
	assert.Equal(t, "", surface.PresentVerdict(true, catch.NoCatch, pipe.ReasonNone, 3, 2))
}

func TestPresentVerdict_mapsCatchOutcomesToTheirOwnTokens(t *testing.T) {
	t.Parallel()
	assert.Equal(t, string(catch.Catch), surface.PresentVerdict(false, catch.Catch, pipe.ReasonNone, 1, 0))
	assert.Equal(t, string(catch.PartialCatch), surface.PresentVerdict(false, catch.PartialCatch, pipe.ReasonNone, 2, 1))
	assert.Equal(t, string(catch.NoOracleSignal), surface.PresentVerdict(false, catch.NoOracleSignal, pipe.ReasonNoMutableOperator, 0, 0))
}

func TestPresentVerdict_mapsConstrainedNoCatchToTestedNotBlind(t *testing.T) {
	t.Parallel()
	got := surface.PresentVerdict(false, catch.NoCatch, pipe.ReasonNone, 3, 0)
	assert.Equal(t, surface.Tested, got, "a fully-constrained line is the affirmative calm-win, not blind silence")
	assert.NotEqual(t, string(catch.NoOracleSignal), got)
	assert.NotEqual(t, string(catch.NoCatch), got)
	assert.NotEqual(t, "", got)
}

func TestPresentVerdict_mapsNoCatchWithSurvivorsToNoCatch(t *testing.T) {
	t.Parallel()
	assert.Equal(t, string(catch.NoCatch), surface.PresentVerdict(false, catch.NoCatch, pipe.ReasonNone, 3, 2),
		"a fix that left survivors is not tested")
}

func TestPresentVerdict_doesNotCallZeroConsideredNoCatchTested(t *testing.T) {
	t.Parallel()
	got := surface.PresentVerdict(false, catch.NoCatch, pipe.ReasonNone, 0, 0)
	assert.Equal(t, string(catch.NoCatch), got, "nothing considered must not borrow the verified-strong affirmation")
	assert.NotEqual(t, surface.Tested, got)
}

func TestPresentVerdict_unknownOutcomeFallsToInFlightNotSuccess(t *testing.T) {
	t.Parallel()
	got := surface.PresentVerdict(false, catch.Outcome("wat"), pipe.ReasonNone, 9, 9)
	assert.Equal(t, "", got, "an unrecognized outcome fails closed to in-flight, never a borrowed success")
	assert.NotEqual(t, string(catch.Catch), got)
	assert.NotEqual(t, surface.Tested, got)
}

func TestPresentVerdict_splitsQuietReasonsIntoDistinctHonestTokens(t *testing.T) {
	t.Parallel()
	renamed := surface.PresentVerdict(false, catch.NoOracleSignal, pipe.ReasonFileRenamed, 0, 0)
	edited := surface.PresentVerdict(false, catch.NoOracleSignal, pipe.ReasonAnchorEdited, 0, 0)
	deleted := surface.PresentVerdict(false, catch.NoOracleSignal, pipe.ReasonAnchorDeleted, 0, 0)
	operatorFree := surface.PresentVerdict(false, catch.NoOracleSignal, pipe.ReasonNoMutableOperator, 0, 0)
	incomplete := surface.PresentVerdict(false, catch.NoOracleSignal, pipe.ReasonOracleIncomplete, 1, 0)

	assert.Equal(t, surface.LostViaRename, renamed, "a renamed anchor must carry its own token, not the operator-free one")
	assert.Equal(t, surface.AnchorEdited, edited, "an edited anchor must carry its own token, not the operator-free one")
	assert.Equal(t, surface.AnchorDeleted, deleted, "a deleted/vanished file must carry its own token, not the edited-in-place one")
	assert.Equal(t, string(catch.NoOracleSignal), operatorFree, "only a genuinely operator-free line keeps the no-oracle-signal token")
	assert.Equal(t, surface.OracleIncomplete, incomplete, "an incomplete oracle run carries its own token — the line HAS operators, the run just didn't finish")
	assert.NotEqual(t, string(catch.NoOracleSignal), incomplete, "an incomplete run must NOT read as 'no mutable operator' — that is a false reason")

	tokens := []string{renamed, edited, deleted, operatorFree, incomplete}
	assert.Len(t, dedupTokens(tokens), 5, "the five quiet causes must map to five distinct tokens")
}

func dedupTokens(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func TestPresentVerdict_onlyProducesTokensTheCardRenders(t *testing.T) {
	t.Parallel()
	known := map[string]bool{
		"":                           true, // in-flight default
		surface.Tested:               true,
		surface.LostViaRename:        true,
		surface.AnchorEdited:         true,
		surface.AnchorDeleted:        true,
		surface.OracleIncomplete:     true,
		string(catch.Catch):          true,
		string(catch.NoCatch):        true,
		string(catch.NoOracleSignal): true,
		string(catch.PartialCatch):   true,
	}
	inputs := []struct {
		running               bool
		outcome               catch.Outcome
		reason                pipe.Reason
		considered, survivors int
	}{
		{true, catch.Catch, pipe.ReasonNone, 1, 0},
		{false, catch.Catch, pipe.ReasonNone, 1, 0},
		{false, catch.NoCatch, pipe.ReasonNone, 3, 0},
		{false, catch.NoCatch, pipe.ReasonNone, 3, 2},
		{false, catch.NoCatch, pipe.ReasonNone, 0, 0},
		{false, catch.NoOracleSignal, pipe.ReasonNoMutableOperator, 0, 0},
		{false, catch.NoOracleSignal, pipe.ReasonFileRenamed, 0, 0},
		{false, catch.NoOracleSignal, pipe.ReasonAnchorEdited, 0, 0},
		{false, catch.NoOracleSignal, pipe.ReasonAnchorDeleted, 0, 0},
		{false, catch.NoOracleSignal, pipe.ReasonOracleIncomplete, 1, 0},
		{false, catch.PartialCatch, pipe.ReasonNone, 2, 1},
		{false, catch.Outcome("wat"), pipe.ReasonNone, 9, 9},
	}
	for _, in := range inputs {
		got := surface.PresentVerdict(in.running, in.outcome, in.reason, in.considered, in.survivors)
		assert.Truef(t, known[got], "token %q is not a state the card renders", got)
	}
}

// Other surfaces (the fleet board's "why" tag) must render the SAME human verdict label
// the review card shows — not the raw snake_case token. VerdictLabel maps a persisted
// verdict token to its headline; an unknown/empty token returns the token unchanged
// (forward-compatible — never the misleading "Oracle running…" the in-flight default
// would give).
func TestVerdictLabel_humanizesKnownTokensAndPassesUnknownThrough(t *testing.T) {
	cases := []struct{ token, want string }{
		{"catch", "Caught"},
		{"no_catch", "No catch"},
		{"no_oracle_signal", "No signal"},
		{"partial_catch", "Partially caught"},
		{surface.Tested, "Tested — ship it"},
		{surface.LostViaRename, "Anchor lost: file renamed"},
		{surface.AnchorEdited, "Anchor edited"},
		{surface.AnchorDeleted, "Anchor lost: file gone"},
		{surface.OracleIncomplete, "Gate incomplete"},
		{"future_kind", "future_kind"}, // unknown → raw, never "Gate running…"
		{"", ""},                       // empty → raw
	}
	for _, tc := range cases {
		if got := surface.VerdictLabel(tc.token); got != tc.want {
			t.Errorf("VerdictLabel(%q) = %q, want %q", tc.token, got, tc.want)
		}
	}
}
