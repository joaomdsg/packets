package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The authoring assist runs as the Lead types, so the draft read must be FAST: it
// uses Haiku (the low-latency model), not the default, and plain-text one-shot
// output. A slow default model would make the live read lag the typing.
func TestAnalysisArgs_useHaikuForAFastDraftRead(t *testing.T) {
	t.Parallel()
	args := analysisArgs("Add retry logic to the uploader.", "")
	assert.Equal(t, []string{
		"-p", "Add retry logic to the uploader.",
		"--output-format", "text",
		"--model", "haiku",
		"--effort", "low",
	}, args, "the assist runs Haiku at LOW effort in one-shot text mode for a fast read")
}

// When the session has a warm harness, the assist RESUMES it (forking a branch) so
// the read carries the explored repo context — the same remembered session the live
// orders use. --fork-session keeps each read an isolated branch, so concurrent reads
// and an order fill never collide on the one base id.
func TestAnalysisArgs_resumesTheWarmSessionWhenGiven(t *testing.T) {
	t.Parallel()
	args := analysisArgs("Sharpen this order.", "sess-abc")
	assert.Equal(t, []string{
		"-p", "Sharpen this order.",
		"--output-format", "text",
		"--model", "haiku",
		"--effort", "low",
		"--resume", "sess-abc",
		"--fork-session",
	}, args, "a warm session is resumed + forked so the read reuses the explored context")
}
