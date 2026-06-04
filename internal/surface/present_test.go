package surface_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/surface"
)

func TestPresentVerdict_inFlightWinsOverAnyOutcomeWhileRunning(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", surface.PresentVerdict(true, catch.Catch, 1, 0), "a running cycle has no verdict yet")
	assert.Equal(t, "", surface.PresentVerdict(true, catch.NoCatch, 3, 2))
}

func TestPresentVerdict_mapsCatchOutcomesToTheirOwnTokens(t *testing.T) {
	t.Parallel()
	assert.Equal(t, string(catch.Catch), surface.PresentVerdict(false, catch.Catch, 1, 0))
	assert.Equal(t, string(catch.PartialCatch), surface.PresentVerdict(false, catch.PartialCatch, 2, 1))
	assert.Equal(t, string(catch.NoOracleSignal), surface.PresentVerdict(false, catch.NoOracleSignal, 0, 0))
}

func TestPresentVerdict_mapsConstrainedNoCatchToTestedNotBlind(t *testing.T) {
	t.Parallel()
	got := surface.PresentVerdict(false, catch.NoCatch, 3, 0)
	assert.Equal(t, surface.Tested, got, "a fully-constrained line is the affirmative calm-win, not blind silence")
	assert.NotEqual(t, string(catch.NoOracleSignal), got)
	assert.NotEqual(t, string(catch.NoCatch), got)
	assert.NotEqual(t, "", got)
}

func TestPresentVerdict_mapsNoCatchWithSurvivorsToNoCatch(t *testing.T) {
	t.Parallel()
	assert.Equal(t, string(catch.NoCatch), surface.PresentVerdict(false, catch.NoCatch, 3, 2),
		"a fix that left survivors is not tested")
}

func TestPresentVerdict_doesNotCallZeroConsideredNoCatchTested(t *testing.T) {
	t.Parallel()
	got := surface.PresentVerdict(false, catch.NoCatch, 0, 0)
	assert.Equal(t, string(catch.NoCatch), got, "nothing considered must not borrow the verified-strong affirmation")
	assert.NotEqual(t, surface.Tested, got)
}

func TestPresentVerdict_unknownOutcomeFallsToInFlightNotSuccess(t *testing.T) {
	t.Parallel()
	got := surface.PresentVerdict(false, catch.Outcome("wat"), 9, 9)
	assert.Equal(t, "", got, "an unrecognized outcome fails closed to in-flight, never a borrowed success")
	assert.NotEqual(t, string(catch.Catch), got)
	assert.NotEqual(t, surface.Tested, got)
}

func TestPresentVerdict_onlyProducesTokensTheCardRenders(t *testing.T) {
	t.Parallel()
	known := map[string]bool{
		"":                           true, // in-flight default
		surface.Tested:               true,
		string(catch.Catch):          true,
		string(catch.NoCatch):        true,
		string(catch.NoOracleSignal): true,
		string(catch.PartialCatch):   true,
	}
	inputs := []struct {
		running               bool
		outcome               catch.Outcome
		considered, survivors int
	}{
		{true, catch.Catch, 1, 0},
		{false, catch.Catch, 1, 0},
		{false, catch.NoCatch, 3, 0},
		{false, catch.NoCatch, 3, 2},
		{false, catch.NoCatch, 0, 0},
		{false, catch.NoOracleSignal, 0, 0},
		{false, catch.PartialCatch, 2, 1},
		{false, catch.Outcome("wat"), 9, 9},
	}
	for _, in := range inputs {
		got := surface.PresentVerdict(in.running, in.outcome, in.considered, in.survivors)
		assert.Truef(t, known[got], "token %q is not a state the card renders", got)
	}
}
