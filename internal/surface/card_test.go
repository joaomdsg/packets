package surface_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/surface"
)

// newCard mounts a fresh ReviewCard and returns a typed handle (for action
// references) plus a test client driving it over HTTP — the same path a real
// browser takes, so assertions hit rendered HTML / SSE frames, never internal
// state.
func newCard(t *testing.T) (*surface.ReviewCard, *vt.Client) {
	t.Helper()
	var server *httptest.Server
	app := via.New(via.WithTestServer(&server))
	via.Mount[surface.ReviewCard](app, "/")
	return &surface.ReviewCard{}, vt.NewClient(t, server, "/")
}

// post drives a verdict into the card through its server action — the path the
// orchestrator will use to deliver an oracle result.
func post(t *testing.T, c *surface.ReviewCard, tc *vt.Client, verdict string) {
	t.Helper()
	require.Equal(t, 200, tc.Action(c.Post).WithSignal("verdict", verdict).Fire())
}

// resolveTo posts a verdict on a fresh card tab and returns the live SSE
// fragment the card resolves to — the patch a browser would apply. The card's
// verdict is per-tab server state, so it is observed over the tab's SSE stream
// (a fresh Reload would land on a new, pristine tab).
func resolveTo(t *testing.T, verdict, marker string) string {
	t.Helper()
	c, tc := newCard(t)
	frames, cancel := tc.SSEReady()
	defer cancel()
	post(t, c, tc, verdict)
	return vt.AwaitFrame(t, frames, 2*time.Second, marker)
}

// Before any oracle verdict arrives, the card must render a designed in-flight
// state — not a blank/empty card a reviewer would read as "broken".
func TestReviewCard_rendersDesignedInFlightStateBeforeAnyVerdict(t *testing.T) {
	t.Parallel()
	_, tc := newCard(t)
	html := tc.Reload()
	assert.Contains(t, html, `data-state="in-flight"`)
	assert.Contains(t, html, "running")
}

// A Catch is the economy's reward beat: it must resolve to an affirmative
// "caught" state, distinctly marked.
func TestReviewCard_rendersCatchAsAffirmativeCaughtState(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, string(catch.Catch), `data-state="catch"`)
	assert.Contains(t, frame, "Caught")
}

// The most common screen — the oracle ran and the line is fully constrained
// (zero survivors) — must read as an affirmative "tested, ship it" beat, never
// as empty or error chrome (the UX non-negotiable from Round 8).
func TestReviewCard_rendersZeroSurvivorAsTestedNotEmpty(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, surface.Tested, `data-state="tested"`)
	assert.Contains(t, frame, "ship it")
}

// NoOracleSignal ("the oracle is blind here") must be VISUALLY DISTINCT from a
// Catch and must never wear a success claim — conflating "no signal" with
// "verified caught" is the exact lie the oracle exists to prevent. (This also
// defeats a dump-everything View: a card that rendered every state at once
// would carry the catch markers into the blind frame.)
func TestReviewCard_rendersNoOracleSignalDistinctFromCatch(t *testing.T) {
	t.Parallel()
	caught := resolveTo(t, string(catch.Catch), `data-state="catch"`)
	blind := resolveTo(t, string(catch.NoOracleSignal), `data-state="no-oracle-signal"`)

	assert.Contains(t, caught, `data-state="catch"`)
	assert.NotContains(t, blind, `data-state="catch"`)
	assert.NotContains(t, blind, "Caught", "a no-signal line must never claim a catch")
}

// Each of the four catch outcomes maps to its own distinct rendered state, so a
// reviewer can tell them apart at a glance.
func TestReviewCard_rendersEachCatchOutcomeAsADistinctState(t *testing.T) {
	t.Parallel()
	cases := map[catch.Outcome]string{
		catch.Catch:          `data-state="catch"`,
		catch.NoCatch:        `data-state="no-catch"`,
		catch.NoOracleSignal: `data-state="no-oracle-signal"`,
		catch.PartialCatch:   `data-state="partial-catch"`,
	}
	seen := map[string]catch.Outcome{}
	for outcome, marker := range cases {
		frame := resolveTo(t, string(outcome), marker)
		assert.Containsf(t, frame, marker, "outcome %q must render %q", outcome, marker)
		if prev, dup := seen[marker]; dup {
			t.Errorf("outcomes %q and %q collided on %q", outcome, prev, marker)
		}
		seen[marker] = outcome
	}
}

// The first review screen carries NO economy meters (Focus/Trust/Treasury) —
// the surface is built before, and validated independently of, the economy.
func TestReviewCard_carriesNoEconomyMetersOnTheFirstScreen(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, string(catch.Catch), `data-state="catch"`)
	for _, meter := range []string{"Focus", "Trust", "Treasury", "Ledger"} {
		assert.NotContainsf(t, frame, meter, "no %q meter belongs on the first screen", meter)
	}
}

// A verdict arriving after the in-flight state must stream in as a live SSE
// patch — the reviewer sees the card resolve in place, with no full reload —
// and this must hold for every outcome, not just a catch.
func TestReviewCard_streamsEachVerdictAsLivePatch(t *testing.T) {
	t.Parallel()
	c, tc := newCard(t)
	require.Contains(t, tc.Reload(), `data-state="in-flight"`)

	frames, cancel := tc.SSE()
	defer cancel()
	for _, step := range []struct {
		outcome string
		marker  string
	}{
		{string(catch.Catch), `data-state="catch"`},
		{string(catch.NoCatch), `data-state="no-catch"`},
		{string(catch.NoOracleSignal), `data-state="no-oracle-signal"`},
		{string(catch.PartialCatch), `data-state="partial-catch"`},
	} {
		post(t, c, tc, step.outcome)
		vt.AwaitFrame(t, frames, 2*time.Second, step.marker)
	}
}

// An unrecognized verdict (a buggy or hostile orchestrator delivering a string
// outside the known vocabulary) must resolve to the neutral in-flight state —
// never panic, and never borrow a catch/success beat the oracle did not earn.
// "The oracle said something we don't understand" is the same as "the oracle
// has not spoken", not "verified caught".
func TestReviewCard_unknownVerdictResolvesToInFlightNotASuccessClaim(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, "wat-is-this-verdict", `data-state="in-flight"`)
	for _, claim := range []string{
		`data-state="catch"`,
		`data-state="tested"`,
		`data-state="partial-catch"`,
		"Caught",
		"ship it",
	} {
		assert.NotContainsf(t, frame, claim, "an unknown verdict must not claim %q", claim)
	}
}

// The verdict string is client-supplied (it rides in over the action signal),
// so a hostile value must never break out of the data-state attribute or inject
// markup. The rendered state is a fixed server-chosen token, not the raw
// signal: a verdict packed with attribute-breakout chars still lands on a clean
// in-flight token with no surviving raw quote/bracket sequence.
func TestReviewCard_hostileVerdictCannotBreakOutOfTheStateAttribute(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, `"><script>alert(1)</script>`, `data-state="in-flight"`)
	assert.NotContains(t, frame, "<script>", "raw markup must not survive into the frame")
	assert.NotContains(t, frame, `state=""`, "the verdict must not break the attribute open")
	assert.NotContains(t, frame, "alert(1)", "the raw signal must not be reflected as the state token")
}
