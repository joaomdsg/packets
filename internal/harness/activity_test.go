package harness_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/translate"
)

// The Lead must see a live agent's activity AS IT STREAMS, not only in the batch
// of turns returned when the whole run finishes. The WithActivity callback fires
// per stream line, in order, the moment each activity event is read — so the card
// can show "editing token.go" while the agent is still working.
func TestSupervisor_streamsActivityToTheCallbackAsEachLineIsRead(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := strings.NewReader(strings.Join([]string{
		thinking("considering the error path"),
		editing("internal/auth/token.go"),
		turnEnd(),
		thinking("now the test"),
		turnEnd(),
	}, "\n") + "\n")

	var got [][]translate.UIEvent
	turns, err := harness.New(dir, base, harness.WithActivity(func(evs []translate.UIEvent) {
		got = append(got, evs)
	})).Run(context.Background(), stream)
	require.NoError(t, err)

	want := [][]translate.UIEvent{
		{{Type: "activity.agent", Kind: "thinking", Detail: "considering the error path"}},
		{{Type: "activity.agent", Kind: "editing", Detail: "internal/auth/token.go"}},
		{{Type: "activity.agent", Kind: "thinking", Detail: "now the test"}},
	}
	assert.Equal(t, want, got, "each line's activity streams to the callback in order; turn-ends emit nothing")

	// The streamed callback must not disturb the settled turn records.
	require.Len(t, turns, 2, "two turn boundaries still settle two turns")
	assert.Len(t, turns[0].Events, 2, "turn 1 still records its thinking + editing activity")
	assert.Len(t, turns[1].Events, 1, "turn 2 still records its thinking activity")
}

// The turn-ended signal is a boundary marker, not agent activity — it must never
// reach the activity callback (which would surface a phantom beat).
func TestSupervisor_doesNotStreamTheTurnEndSignalAsActivity(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := strings.NewReader(turnEnd() + "\n") // a turn-end with no preceding activity

	calls := 0
	_, err := harness.New(dir, base, harness.WithActivity(func([]translate.UIEvent) {
		calls++
	})).Run(context.Background(), stream)
	require.NoError(t, err)
	assert.Zero(t, calls, "a bare turn-end line is not activity and must not fire the callback")
}

// Activity must stream the moment it is read, independent of the turn settling —
// so a Lead watching an in-progress turn (no turn-end yet) still sees the agent
// working. An incomplete trailing turn settles nothing but its activity streamed.
func TestSupervisor_streamsActivityForAnUnsettledInProgressTurn(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := strings.NewReader(thinking("starting") + "\n" + editing("a.go") + "\n") // no turn-end: still in progress

	var got [][]translate.UIEvent
	turns, err := harness.New(dir, base, harness.WithActivity(func(evs []translate.UIEvent) {
		got = append(got, evs)
	})).Run(context.Background(), stream)
	require.NoError(t, err)

	assert.Empty(t, turns, "an in-progress turn settles nothing")
	want := [][]translate.UIEvent{
		{{Type: "activity.agent", Kind: "thinking", Detail: "starting"}},
		{{Type: "activity.agent", Kind: "editing", Detail: "a.go"}},
	}
	assert.Equal(t, want, got, "the in-progress turn's activity still streamed live")
}

// New must stay usable with no options (the live-activity callback is optional);
// a run with no callback still settles its turns.
func TestSupervisor_runsWithNoActivityCallback(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	stream := strings.NewReader(thinking("ok") + "\n" + turnEnd() + "\n")

	turns, err := harness.New(dir, base).Run(context.Background(), stream)
	require.NoError(t, err)
	require.Len(t, turns, 1, "a run without the activity option still settles its turn")
}
