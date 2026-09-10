package harness_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/harness"
)

// The harness is useless unless launched in headless streaming mode: -p carries
// the task, and --output-format stream-json + --verbose are BOTH required for
// the CLI to emit the per-event stream the reducer consumes. A regression that
// drops any of them would silently starve the supervisor of events, so the argv
// must pin them.
func TestClaudeArgs_launchesHeadlessStreamingWithTheGivenPrompt(t *testing.T) {
	t.Parallel()
	args := harness.ClaudeArgs("fix the auth bug", "")

	require.Contains(t, args, "-p", "print mode is required for a headless run")
	assert.Contains(t, args, "fix the auth bug", "the prompt must be passed to the harness")

	requireFlagValue(t, args, "--output-format", "stream-json")
	assert.Contains(t, args, "--verbose", "stream-json requires --verbose to emit all events")
	requireFlagValue(t, args, "--permission-mode", "bypassPermissions")
}

// When the session has a warm explored harness, the order fill RESUMES it (forking
// a branch) so the agent works with the repo context the warm-up built, instead of
// cold-starting. --fork-session keeps the fill an isolated branch off the warm base.
func TestClaudeArgs_resumesTheWarmSessionWhenGiven(t *testing.T) {
	t.Parallel()
	args := harness.ClaudeArgs("fix it", "sess-7")
	requireFlagValue(t, args, "--resume", "sess-7")
	assert.Contains(t, args, "--fork-session", "the order fill forks the warm explored session")
	assert.Contains(t, args, "fix it", "the prompt is still carried")
}

// The prompt must be a distinct argv element from -p (not concatenated), or the
// CLI would treat the whole thing as an unknown flag.
func TestClaudeArgs_passesThePromptAsItsOwnArgument(t *testing.T) {
	t.Parallel()
	args := harness.ClaudeArgs("do the thing", "")
	i := indexOf(args, "-p")
	require.GreaterOrEqual(t, i, 0, "-p must be present")
	require.Less(t, i+1, len(args), "-p must be followed by an argument")
	assert.Equal(t, "do the thing", args[i+1], "the prompt directly follows -p")
}

// A prompt that happens to start with a dash must still be positioned as -p's
// value, never emitted loose where a flag parser would read it as an option.
func TestClaudeArgs_keepsADashLeadingPromptAttachedToTheFlag(t *testing.T) {
	t.Parallel()
	args := harness.ClaudeArgs("--rewrite everything", "")
	i := indexOf(args, "-p")
	require.GreaterOrEqual(t, i, 0)
	require.Less(t, i+1, len(args))
	assert.Equal(t, "--rewrite everything", args[i+1], "the prompt stays -p's value")
}

func indexOf(s []string, want string) int {
	for i, v := range s {
		if v == want {
			return i
		}
	}
	return -1
}

func requireFlagValue(t *testing.T, args []string, flag, value string) {
	t.Helper()
	i := indexOf(args, flag)
	require.GreaterOrEqual(t, i, 0, "flag %s must be present", flag)
	require.Less(t, i+1, len(args), "flag %s must have a value", flag)
	assert.Equal(t, value, args[i+1], "flag %s value", flag)
}
