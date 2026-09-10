package ci_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joaomdsg/packets/internal/ci"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFabric() *fabric.Config {
	return &fabric.Config{
		Slug:          "acme",
		RepoPath:      "/repo",
		DefaultBranch: "main",
		CI:            fabric.CI{PollSeconds: 30, TimeoutMinutes: 30, WorkflowName: "packets"},
		Merge:         fabric.Merge{Auto: true, Strategy: "squash"},
	}
}

func testPacket() *packet.Packet {
	return &packet.Packet{
		Goal:     "fix the login timeout",
		Terminal: []string{"true"},
		Budget:   packet.Budget{Retries: 3, Minutes: 30},
	}
}

func writeFixture(t *testing.T, dir string, pkt *packet.Packet, st *state.State) {
	t.Helper()
	require.NoError(t, packet.Save(filepath.Join(dir, "packet.yaml"), pkt))
	require.NoError(t, state.Save(filepath.Join(dir, "state.json"), st))
}

func readLogEvents(t *testing.T, dir string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "log.jsonl"))
	require.NoError(t, err)
	var events []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var e struct {
			Event string `json:"event"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &e))
		events = append(events, e.Event)
	}
	return events
}

func readRegistryLines(t *testing.T, fabDir string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fabDir, "registry.jsonl"))
	require.NoError(t, err)
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m))
		lines = append(lines, m)
	}
	return lines
}

func ciNode() *string {
	n := state.NodeCI
	return &n
}

func TestRun_redCIWritesFailureMDAndReturnsToBuild(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fabDir := t.TempDir()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.CI, Node: ciNode(), PRNumber: 42, Branch: "packets/s-v1"}
	writeFixture(t, dir, pkt, st)

	gh := &stubGH{checksSeq: [][]ci.Check{{{Name: "packets", State: "FAILURE", Link: "https://github.com/o/r/actions/runs/555/job/1"}}},
		logFailedOutput: "panic: boom\n"}
	deps := ci.Deps{GH: gh, Git: &stubGit{diffStat: " file.go | 2 +-\n"}, Clock: &stubClock{}}

	got, err := ci.Run(deps, testFabric(), pkt, st, dir, fabDir)

	require.NoError(t, err)
	assert.Equal(t, state.Building, got.State)
	require.NotNil(t, got.Node)
	assert.Equal(t, state.NodeBuild, *got.Node)
	assert.Equal(t, 1, got.RetriesUsed)

	failureMD, err := os.ReadFile(filepath.Join(dir, "failure.md"))
	require.NoError(t, err)
	assert.Contains(t, string(failureMD), "attempt: 0\n")
	assert.Contains(t, string(failureMD), "exit: 1\n")
	assert.Contains(t, string(failureMD), "--- last 200 lines ---\npanic: boom")
	assert.Contains(t, string(failureMD), "--- diff since previous attempt ---\n file.go | 2 +-")

	assert.Contains(t, readLogEvents(t, dir), "ci_pred")
	assert.Contains(t, readLogEvents(t, dir), "enter_node")
}

func TestRun_redCIExhaustsRetriesAndHaltsBudget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fabDir := t.TempDir()
	pkt := testPacket()
	pkt.Budget.Retries = 1
	st := &state.State{Slug: "s", Version: 1, State: state.CI, Node: ciNode(), PRNumber: 42, RetriesUsed: 0}
	writeFixture(t, dir, pkt, st)

	gh := &stubGH{checksSeq: [][]ci.Check{{{Name: "packets", State: "FAILURE", Link: "https://x/actions/runs/1/job/1"}}}}
	deps := ci.Deps{GH: gh, Git: &stubGit{}, Clock: &stubClock{}}

	got, err := ci.Run(deps, testFabric(), pkt, st, dir, fabDir)

	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	require.NotNil(t, got.HaltReason)
	assert.Equal(t, "budget", *got.HaltReason)
	assert.Contains(t, readLogEvents(t, dir), "budget_halt")
}

func TestRun_greenCIAutoMergesAndTerminates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fabDir := t.TempDir()
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "spec_helper.go"), []byte("x"), 0o600))

	pkt := testPacket()
	st := &state.State{
		Slug: "s", Version: 2, State: state.CI, Node: ciNode(), PRNumber: 42,
		Branch: "packets/s-v2", GateSuggestedTerminals: []string{"go test spec_helper.go"},
	}
	writeFixture(t, dir, pkt, st)

	fab := testFabric()
	fab.RepoPath = repoDir
	gh := &stubGH{checksSeq: [][]ci.Check{{{Name: "packets", State: "SUCCESS"}}}}
	git := &stubGit{currentCommit: "abc123", addedFiles: []string{"test/new_case_test.go", "spec_helper.go"}}
	deps := ci.Deps{GH: gh, Git: git, Clock: &stubClock{}}

	got, err := ci.Run(deps, fab, pkt, st, dir, fabDir)

	require.NoError(t, err)
	assert.Equal(t, state.Terminated, got.State)
	assert.Nil(t, got.Node)
	assert.Equal(t, []int{42}, gh.merged)

	lines := readRegistryLines(t, fabDir)
	var causes []string
	for _, l := range lines {
		causes = append(causes, l["cause"].(string))
		assert.Equal(t, "abc123", l["commit"])
		assert.Equal(t, "acme", l["fabric"])
		assert.Equal(t, "s", l["packet"])
	}
	assert.Contains(t, causes, "gate_suggested")
	assert.Contains(t, causes, "ci_failure")

	events := readLogEvents(t, dir)
	assert.Contains(t, events, "merge")
	assert.Contains(t, events, "terminate")
	assert.Contains(t, events, "accrete")
}

func TestRun_greenCIWithHumanApprovalWaitsForApproval(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fabDir := t.TempDir()
	pkt := testPacket()
	pkt.Terminal = []string{"human_approval"}
	st := &state.State{Slug: "s", Version: 1, State: state.CI, Node: ciNode(), PRNumber: 42, Approved: false}
	writeFixture(t, dir, pkt, st)

	gh := &stubGH{checksSeq: [][]ci.Check{{{Name: "packets", State: "SUCCESS"}}}}
	deps := ci.Deps{GH: gh, Git: &stubGit{}, Clock: &stubClock{}}

	got, err := ci.Run(deps, testFabric(), pkt, st, dir, fabDir)

	require.NoError(t, err)
	assert.Equal(t, state.AwaitingApproval, got.State)
	assert.Empty(t, gh.merged, "must not merge before approval")
}

func TestRun_pollDeadlineHaltsWithCITimeoutReason(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fabDir := t.TempDir()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.CI, Node: ciNode(), PRNumber: 42}
	writeFixture(t, dir, pkt, st)

	fab := testFabric()
	fab.CI.TimeoutMinutes = 0
	gh := &stubGH{checksSeq: [][]ci.Check{{}}}
	deps := ci.Deps{GH: gh, Git: &stubGit{}, Clock: &stubClock{now: time.Unix(0, 0)}}

	got, err := ci.Run(deps, fab, pkt, st, dir, fabDir)

	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	require.NotNil(t, got.HaltReason)
	assert.Equal(t, "ci_timeout", *got.HaltReason)
	assert.Equal(t, 1, gh.pollCalls, "a zero-minute timeout must not sleep/poll again")
}

func TestRun_pollAdvancesThroughPendingWithoutRealSleep(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fabDir := t.TempDir()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.CI, Node: ciNode(), PRNumber: 42}
	writeFixture(t, dir, pkt, st)

	gh := &stubGH{checksSeq: [][]ci.Check{
		{{Name: "packets", State: "IN_PROGRESS"}},
		{{Name: "packets", State: "IN_PROGRESS"}},
		{{Name: "packets", State: "SUCCESS"}},
	}}
	deps := ci.Deps{GH: gh, Git: &stubGit{}, Clock: &stubClock{}}

	start := time.Now()
	got, err := ci.Run(deps, testFabric(), pkt, st, dir, fabDir)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, state.Terminated, got.State)
	assert.Equal(t, 3, gh.pollCalls)
	assert.Less(t, elapsed, time.Second, "polling must use the injected clock, not a real sleep")
}
