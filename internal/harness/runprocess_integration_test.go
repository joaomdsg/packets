package harness_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/translate"
)

// fakeClaude writes an executable named "claude" to a fresh dir and prepends that
// dir to PATH, so RunProcess (which spawns the bare "claude") resolves it. The
// script stands in for the real harness: it edits a file in its working dir (the
// repo, since RunProcess sets cmd.Dir) and emits the stream-json a real run would —
// proving the host-subprocess path end-to-end without an API key.
func fakeClaude(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "claude")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// RunProcess must drive a REAL Claude Code subprocess end-to-end: spawn it in the
// repo, stream its activity live, and settle the file it actually wrote into a
// reviewable revision — the whole live pipe, proven against a real process (not a
// fixture reader) with no API key.
func TestRunProcess_settlesARealSubprocessEditIntoARevisionAndStreamsItsActivity(t *testing.T) {
	// NOT parallel: t.Setenv mutates process-wide PATH.
	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	// The fake agent narrates, writes a real file in its cwd (the repo), then ends
	// the turn — exactly the shape RunProcess + the Supervisor reduce.
	fakeClaude(t, `
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"adding a feature"}]}}'
printf 'package main\n' > feature.go
echo '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"feature.go"}}]}}'
echo '{"type":"result","subtype":"success"}'
`)

	var activity []translate.UIEvent
	turns, err := harness.RunProcess(context.Background(), dir, "add feature.go", func(evs []translate.UIEvent) {
		activity = append(activity, evs...)
	})
	require.NoError(t, err)
	require.Len(t, turns, 1, "the one turn-end settles one revision")

	out := turns[0].Outcome
	assert.True(t, out.Minted, "the subprocess's real file edit must settle into a revision")
	assert.Equal(t, runGit(t, dir, "rev-parse", "HEAD"), out.SHA, "the minted SHA is the new HEAD")
	assert.NotEqual(t, base, out.SHA, "HEAD moved — a real revision was produced")
	var sawFeature bool
	for _, f := range out.Diff.Files {
		if f.Path == "feature.go" {
			sawFeature = true
		}
	}
	assert.True(t, sawFeature, "the revision's diff includes the file the agent wrote, got %+v", out.Diff.Files)

	assert.Contains(t, activity, translate.UIEvent{Type: "activity.agent", Kind: "editing", Detail: "feature.go"},
		"the agent's edit activity streamed live during the real run")
}

// A subprocess that exits non-zero must surface as an error, never a silent
// success — a crashed harness is not a completed run.
func TestRunProcess_surfacesANonZeroSubprocessExit(t *testing.T) {
	dir := initRepo(t)
	fakeClaude(t, `
echo '{"type":"result","subtype":"success"}'
exit 3
`)
	_, err := harness.RunProcess(context.Background(), dir, "do nothing", nil)
	assert.Error(t, err, "a non-zero claude exit must surface")
}
