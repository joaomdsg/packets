package build_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/build"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderClaudeMD_substitutesHumanApprovalAndFlagsNewTerminal(t *testing.T) {
	t.Parallel()
	pkt := &packet.Packet{
		Goal:        "fix the login timeout",
		Context:     "",
		Constraints: []string{"go test ./...", "human_approval"},
		Terminal:    []string{"go test ./login/...", "human_approval"},
	}
	st := &state.State{
		Slug:                   "fix-the-login-timeout",
		Version:                2,
		Attempt:                3,
		GateSuggestedTerminals: []string{"go test ./login/..."},
	}

	out, err := build.RenderClaudeMD(pkt, st, "", "")

	require.NoError(t, err)
	assert.Contains(t, out, "# Packet: fix-the-login-timeout v2 (attempt 3)")
	assert.Contains(t, out, "fix the login timeout")
	assert.Contains(t, out, "(none)")
	assert.Contains(t, out, "`go test ./login/...`  ← NEW: this check does not exist yet. You must create it.")
	assert.Contains(t, out, "`packets check-approved fix-the-login-timeout`")
	assert.NotContains(t, out, "human_approval")
	// Not gate-suggested, so no NEW marker on this line.
	assert.Contains(t, out, "- `go test ./...`\n")
}

func TestRenderClaudeMD_splicesInPreviousFailureAndHalt(t *testing.T) {
	t.Parallel()
	pkt := &packet.Packet{Goal: "g", Terminal: []string{"true"}}
	st := &state.State{Slug: "s", Version: 1, Attempt: 1}

	out, err := build.RenderClaudeMD(pkt, st, "attempt: 1\nfailed: terminal `x`", "smell:max_files\n\ndetails")

	require.NoError(t, err)
	assert.Contains(t, out, "## Previous CI failure")
	assert.Contains(t, out, "attempt: 1\nfailed: terminal `x`")
	assert.Contains(t, out, "## Previous halt (human has resumed you)")
	assert.Contains(t, out, "smell:max_files")
}

func TestRenderClaudeMD_omitsFailureAndHaltSectionsWhenAbsent(t *testing.T) {
	t.Parallel()
	pkt := &packet.Packet{Goal: "g", Terminal: []string{"true"}}
	st := &state.State{Slug: "s", Version: 1, Attempt: 1}

	out, err := build.RenderClaudeMD(pkt, st, "", "")

	require.NoError(t, err)
	assert.NotContains(t, out, "Previous CI failure")
	assert.NotContains(t, out, "Previous halt")
}

func TestFormatFailure_matchesTheSpecFormat(t *testing.T) {
	t.Parallel()

	out := build.FormatFailure(4, "terminal", "go test ./...", 1, "panic: boom", " file.go | 2 +-")

	assert.Contains(t, out, "attempt: 4\n")
	assert.Contains(t, out, "failed: terminal `go test ./...`\n")
	assert.Contains(t, out, "exit: 1\n")
	assert.Contains(t, out, "--- last 200 lines ---\npanic: boom\n")
	assert.Contains(t, out, "--- diff since previous attempt ---\n file.go | 2 +-\n")
}
