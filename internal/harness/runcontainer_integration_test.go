package harness_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/harness"
	"github.com/joaomdsg/packets/internal/translate"
)

// fakeAgentImage builds a fake-claude agent image FROM the CI-built cage image
// (so no extra base pull — the flake source). Its `claude` edits the bind-mounted
// /work repo and emits stream-json, standing in for the real harness so the whole
// container path runs in CI with NO API key. Skips when the cage base is absent
// (local without `docker build -f internal/cage/Dockerfile -t packets-cage:dev .`),
// unless PACKETS_REQUIRE_CAGE forces a hard run (CI builds the base).
func fakeAgentImage(t *testing.T) string {
	t.Helper()
	if exec.Command("docker", "image", "inspect", "packets-cage:dev").Run() != nil {
		if os.Getenv("PACKETS_REQUIRE_CAGE") != "" {
			t.Fatal("PACKETS_REQUIRE_CAGE set but packets-cage:dev (the fake-agent base) is absent")
		}
		t.Skip("packets-cage:dev not present (build: docker build -f internal/cage/Dockerfile -t packets-cage:dev .); skipping the container integration test")
	}
	dir := t.TempDir()
	// The fake agent edits /work (the bind-mounted host repo) and narrates in
	// stream-json. It does NOT commit — the HOST settles the working-tree change
	// into a revision (the economy firewall: only the host mints).
	script := `#!/bin/sh
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"adding a feature"}]}}'
printf 'package main\n' > /work/feature.go
echo '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"feature.go"}}]}}'
echo '{"type":"result","subtype":"success"}'
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"),
		[]byte("FROM packets-cage:dev\nCOPY claude /usr/local/bin/claude\nRUN chmod +x /usr/local/bin/claude\n"), 0o644))

	tag := "packets-fake-agent:test"
	out, err := exec.Command("docker", "build", "-t", tag, dir).CombinedOutput()
	require.NoError(t, err, "docker build fake agent image:\n%s", out)
	t.Cleanup(func() { _ = exec.Command("docker", "rmi", "-f", tag).Run() })
	return tag
}

// RunContainer must drive a REAL agent CONTAINER end-to-end: spawn it on the repo,
// stream its activity live, and settle the file the agent actually wrote (inside
// the container, on the bind-mounted repo) into a reviewable revision on the host —
// the whole containerized live pipe, proven against a real `docker run` with no API
// key. NOT parallel (swaps the package agentImage var + runs a container).
func TestRunContainer_settlesARealContainerEditIntoARevision(t *testing.T) {
	tag := fakeAgentImage(t)
	restore := harness.SetAgentImage(tag)
	t.Cleanup(restore)

	dir := initRepo(t)
	base := runGit(t, dir, "rev-parse", "HEAD")

	var activity []translate.UIEvent
	turns, err := harness.RunContainer(context.Background(), dir, "add feature.go", func(evs []translate.UIEvent) {
		activity = append(activity, evs...)
	})
	require.NoError(t, err)
	require.Len(t, turns, 1, "the one turn-end settles one revision")

	out := turns[0].Outcome
	assert.True(t, out.Minted, "the container's real file edit settles into a host revision")
	assert.Equal(t, runGit(t, dir, "rev-parse", "HEAD"), out.SHA, "the minted SHA is the new HEAD")
	assert.NotEqual(t, base, out.SHA, "HEAD moved — a real revision was produced by the containerized agent")
	var sawFeature bool
	for _, f := range out.Diff.Files {
		if f.Path == "feature.go" {
			sawFeature = true
		}
	}
	assert.True(t, sawFeature, "the revision includes the file the containerized agent wrote, got %+v", out.Diff.Files)

	assert.Contains(t, activity, translate.UIEvent{Type: "activity.agent", Kind: "editing", Detail: "feature.go"},
		"the agent's edit activity streamed live from inside the container")
}
