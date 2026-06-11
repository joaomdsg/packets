package surface_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/surface"
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

func TestReviewCard_rendersDesignedInFlightStateBeforeAnyVerdict(t *testing.T) {
	t.Parallel()
	_, tc := newCard(t)
	html := tc.Reload()
	assert.Contains(t, html, `data-state="in-flight"`)
	assert.Contains(t, html, "running")
}

func TestReviewCard_rendersCatchAsAffirmativeCaughtState(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, string(catch.Catch), `data-state="catch"`)
	assert.Contains(t, frame, "Caught")
}

func TestReviewCard_rendersZeroSurvivorAsTestedNotEmpty(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, surface.Tested, `data-state="tested"`)
	assert.Contains(t, frame, "ship it")
}

func TestReviewCard_rendersNoOracleSignalDistinctFromCatch(t *testing.T) {
	t.Parallel()
	caught := resolveTo(t, string(catch.Catch), `data-state="catch"`)
	blind := resolveTo(t, string(catch.NoOracleSignal), `data-state="no-oracle-signal"`)

	assert.Contains(t, caught, `data-state="catch"`)
	assert.NotContains(t, blind, `data-state="catch"`)
	assert.NotContains(t, blind, "Caught", "a no-signal line must never claim a catch")
}

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

func TestReviewCard_rendersLostViaRenameWithoutClaimingNoOperator(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, surface.LostViaRename, `data-state="lost-via-rename"`)
	assert.Contains(t, strings.ToLower(frame), "rename", "a lost-via-rename card must name the rename as the reason it is quiet")
	assert.NotContains(t, frame, "no mutable operator", "a renamed anchor must never claim the line had no operators — the confidently-wrong terminal this build exists to kill")
	assert.NotContains(t, frame, `data-state="no-oracle-signal"`, "a renamed anchor is a distinct quiet state from operator-free")
	assert.NotContains(t, frame, "Caught", "a lost anchor is not a catch")
}

func TestReviewCard_rendersAnchorEditedDistinctFromNoOracleSignal(t *testing.T) {
	t.Parallel()
	edited := resolveTo(t, surface.AnchorEdited, `data-state="anchor-edited"`)
	blind := resolveTo(t, string(catch.NoOracleSignal), `data-state="no-oracle-signal"`)

	assert.NotContains(t, edited, `data-state="no-oracle-signal"`, "an edited anchor is a distinct quiet state from operator-free")
	assert.NotContains(t, edited, "no mutable operator", "an edited anchor must not claim the line had no operators")
	assert.Contains(t, strings.ToLower(edited), "edit", "the edited-anchor card must say the line was edited")
	assert.Contains(t, blind, "no mutable operator", "a genuinely operator-free line keeps the true 'no mutable operator' copy")
}

// A vanished file must render as its own honest state: it must NOT claim the
// line was "edited in place" (false for a gone file), must NOT assert a rename
// it could not detect, and must admit the deletion-or-rename uncertainty.
func TestReviewCard_rendersAnchorDeletedWithoutClaimingEditedOrRenamed(t *testing.T) {
	t.Parallel()
	deleted := resolveTo(t, surface.AnchorDeleted, `data-state="anchor-deleted"`)

	assert.NotContains(t, deleted, `data-state="anchor-edited"`, "a deleted file is distinct from an in-place edit")
	assert.NotContains(t, deleted, `data-state="lost-via-rename"`, "a deleted file must not assert a rename it could not detect")
	assert.NotContains(t, deleted, "no mutable operator", "a deleted file must not claim the line had no operators")
	low := strings.ToLower(deleted)
	assert.Contains(t, low, "delet", "the card must say the file was deleted")
	assert.Contains(t, low, "renamed", "the card must admit a sub-threshold rename is also possible")
}

func TestReviewCard_carriesNoEconomyMetersOnTheFirstScreen(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, string(catch.Catch), `data-state="catch"`)
	for _, meter := range []string{"Focus", "Trust", "Treasury", "Ledger"} {
		assert.NotContainsf(t, frame, meter, "no %q meter belongs on the first screen", meter)
	}
}

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

func TestReviewCard_hostileVerdictCannotBreakOutOfTheStateAttribute(t *testing.T) {
	t.Parallel()
	frame := resolveTo(t, `"><script>alert(1)</script>`, `data-state="in-flight"`)
	assert.NotContains(t, frame, "<script>", "raw markup must not survive into the frame")
	assert.NotContains(t, frame, `state=""`, "the verdict must not break the attribute open")
	assert.NotContains(t, frame, "alert(1)", "the raw signal must not be reflected as the state token")
}
