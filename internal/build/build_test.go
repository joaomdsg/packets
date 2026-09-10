package build_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/build"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFabric() *fabric.Config {
	return &fabric.Config{
		RepoPath:      "/repo",
		DefaultBranch: "main",
		Image:         "packets/x:v0",
		SmellTriggers: fabric.SmellTriggers{
			MaxFilesTouched:      10,
			DenyTestModification: true,
			DenyDependencyAdd:    true,
		},
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

func TestRun_happyPathPushesOpensPRAndEntersCI(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "fix-the-login-timeout", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)
	deps, git, gh, docker := newStubDeps()

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.CI, got.State)
	require.NotNil(t, got.Node)
	assert.Equal(t, state.NodeCI, *got.Node)
	assert.Equal(t, 1, got.Attempt)
	assert.Equal(t, 42, got.PRNumber)
	assert.Equal(t, 15, got.TokensUsed)
	assert.Positive(t, got.MinutesUsed)

	assert.True(t, git.fetched)
	require.Len(t, git.newBranches, 1)
	assert.Contains(t, git.newBranches[0], "packets/fix-the-login-timeout-v1 from origin/main")
	assert.Equal(t, []string{"packets/fix-the-login-timeout-v1"}, git.pushedBranches)

	require.Len(t, gh.titles, 1)
	assert.Equal(t, "fix the login timeout", gh.titles[0])
	assert.Contains(t, gh.bodies[0], "```yaml")

	require.Len(t, docker.calls, 1)
	assert.Contains(t, docker.calls[0], "packets/x:v0")

	reloaded, err := state.Load(filepath.Join(dir, "state.json"))
	require.NoError(t, err)
	assert.Equal(t, state.CI, reloaded.State)

	claudeMD, err := os.ReadFile(filepath.Join(dir, "runs", "attempt-1", "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeMD), "fix the login timeout")

	assert.Contains(t, readLogEvents(t, dir), "enter_node")
}

func TestRun_agentHaltCopiesHaltMDAndLogsAgentHalt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)
	deps, _, _, docker := newStubDeps()
	docker.runFunc = func(args []string) (string, int, error) {
		signals := signalsDirFromArgs(args)
		require.NoError(t, os.WriteFile(filepath.Join(signals, "halt.md"),
			[]byte("this packet seems wrong\nthe goal contradicts the terminal\n"), 0o600))
		return "", 0, nil
	}

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	require.NotNil(t, got.HaltReason)
	assert.Equal(t, "this packet seems wrong", *got.HaltReason)

	copied, err := os.ReadFile(filepath.Join(dir, "halt.md"))
	require.NoError(t, err)
	assert.Contains(t, string(copied), "this packet seems wrong")
	assert.Contains(t, readLogEvents(t, dir), "agent_halt")
}

func TestRun_apiErrorHaltsBeforeCommitAndPreservesRetryBudget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)
	deps, git, _, docker := newStubDeps()
	// Real observed response: a credit-exhaustion failure still reports
	// subtype "success" and exit code 0, so is_error is the only signal.
	docker.runFunc = func(args []string) (string, int, error) {
		return `{"is_error":true,"subtype":"success","api_error_status":400,` +
			`"terminal_reason":"api_error","result":"Credit balance is too low",` +
			`"num_turns":1,"total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0}}`, 0, nil
	}

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	require.NotNil(t, got.HaltReason)
	assert.Equal(t, "agent:api_error", *got.HaltReason)
	assert.Equal(t, 0, got.RetriesUsed)
	assert.Empty(t, git.commitMessages)

	haltMD, err := os.ReadFile(filepath.Join(dir, "halt.md"))
	require.NoError(t, err)
	assert.Contains(t, string(haltMD), "Credit balance is too low")
	assert.Contains(t, string(haltMD), "api_error_status: 400")

	assert.Contains(t, readLogEvents(t, dir), "agent_halt")
}

func TestRun_isNewPredicatePassedBeforeImplementationHaltsAsSmell(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)
	deps, _, _, docker := newStubDeps()
	docker.runFunc = func(args []string) (string, int, error) {
		signals := signalsDirFromArgs(args)
		require.NoError(t, os.WriteFile(filepath.Join(signals, "halt.md"),
			[]byte("smell:new_predicate_passed_before_implementation\n\ndetails\n"), 0o600))
		return "", 0, nil
	}

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	assert.Equal(t, "smell:new_predicate_passed_before_implementation", *got.HaltReason)
	assert.Contains(t, readLogEvents(t, dir), "smell_halt")
}

func TestRun_containerTimeoutHaltsWithReasonBudget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)
	deps, _, _, docker := newStubDeps()
	docker.runFunc = func(args []string) (string, int, error) {
		return "", 124, nil
	}

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	require.NotNil(t, got.HaltReason)
	assert.Equal(t, "budget", *got.HaltReason)
	assert.Contains(t, readLogEvents(t, dir), "budget_halt")
}

func TestRun_proposalFileIsCopiedAndLogged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)
	deps, _, _, docker := newStubDeps()
	docker.runFunc = func(args []string) (string, int, error) {
		signals := signalsDirFromArgs(args)
		require.NoError(t, os.WriteFile(filepath.Join(signals, "proposal-1.md"),
			[]byte("Add a lint check\nrationale here\n"), 0o600))
		return `{"usage":{"input_tokens":1,"output_tokens":1}}`, 0, nil
	}

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.CI, got.State)
	copied, err := os.ReadFile(filepath.Join(dir, "proposals", "proposal-1.md"))
	require.NoError(t, err)
	assert.Contains(t, string(copied), "Add a lint check")
	assert.Contains(t, readLogEvents(t, dir), "proposal")
}

func TestRun_hostSmellRecheckHaltsBeforeCommitting(t *testing.T) {
	tests := []struct {
		name        string
		workingDiff string
		triggers    fabric.SmellTriggers
		wantReason  string
	}{
		{
			name:        "max files touched",
			workingDiff: "M\ta.go\nM\tb.go\nM\tc.go\n",
			triggers:    fabric.SmellTriggers{MaxFilesTouched: 2, DenyTestModification: true, DenyDependencyAdd: true},
			wantReason:  "smell:max_files",
		},
		{
			name:        "existing test file modified",
			workingDiff: "M\ttests/example_test.go\n",
			triggers:    fabric.SmellTriggers{MaxFilesTouched: 10, DenyTestModification: true, DenyDependencyAdd: true},
			wantReason:  "smell:test_modified",
		},
		{
			name:        "dependency file changed",
			workingDiff: "M\tgo.mod\n",
			triggers:    fabric.SmellTriggers{MaxFilesTouched: 10, DenyTestModification: true, DenyDependencyAdd: true},
			wantReason:  "smell:dependency",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			fab := testFabric()
			fab.SmellTriggers = tt.triggers
			pkt := testPacket()
			st := &state.State{Slug: "s", Version: 1, State: state.Building}
			writeFixture(t, dir, pkt, st)
			deps, git, _, _ := newStubDeps()
			git.workingDiff = tt.workingDiff

			got, err := build.Run(deps, fab, pkt, st, dir)

			require.NoError(t, err)
			assert.Equal(t, state.Halted, got.State)
			assert.Equal(t, tt.wantReason, *got.HaltReason)
			assert.Empty(t, git.commitMessages, "must not commit after a smell halt")
		})
	}
}

func TestRun_noChangesHaltsInsteadOfCommittingEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)
	deps, git, _, _ := newStubDeps()
	git.committed = false

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	assert.Equal(t, "no_changes", *got.HaltReason)
	assert.Empty(t, git.pushedBranches)
}

func TestRun_rebaseConflictAbortsAndLeavesBranchIntact(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)
	deps, git, _, _ := newStubDeps()
	git.rebaseConflict = true

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	assert.Equal(t, "conflict", *got.HaltReason)
	assert.True(t, git.rebaseAborted)
	assert.Empty(t, git.pushedBranches, "must not push after an aborted rebase")
	assert.Contains(t, readLogEvents(t, dir), "conflict_halt")
}

func TestRun_reusesExistingBranchInsteadOfRecreatingIt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building, Branch: "packets/s-v1"}
	writeFixture(t, dir, pkt, st)
	deps, git, _, _ := newStubDeps()

	got, err := build.Run(deps, fab, pkt, st, dir)

	require.NoError(t, err)
	assert.Equal(t, state.CI, got.State)
	assert.Equal(t, []string{"packets/s-v1"}, git.checkedOut)
	assert.Empty(t, git.newBranches)
}

func TestRun_killedMidRunThenRerunResumesWithoutCorruptingState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fab := testFabric()
	pkt := testPacket()
	st := &state.State{Slug: "s", Version: 1, State: state.Building}
	writeFixture(t, dir, pkt, st)

	deps, _, _, docker := newStubDeps()
	docker.runFunc = func(args []string) (string, int, error) {
		return "", 0, assert.AnError // stands in for the process dying mid-run
	}
	_, err := build.Run(deps, fab, pkt, st, dir)
	require.Error(t, err)

	// The "kill" happened after step 3's attempt++ + save; state.json must
	// still be valid and reflect that checkpoint, not be torn or stale.
	afterKill, err := state.Load(filepath.Join(dir, "state.json"))
	require.NoError(t, err)
	assert.Equal(t, 1, afterKill.Attempt)
	assert.Equal(t, state.Building, afterKill.State)

	deps2, _, _, _ := newStubDeps()
	pkt2, err := packet.Load(filepath.Join(dir, "packet.yaml"))
	require.NoError(t, err)
	got, err := build.Run(deps2, fab, pkt2, afterKill, dir)

	require.NoError(t, err)
	assert.Equal(t, state.CI, got.State)
	assert.Equal(t, 2, got.Attempt)
}
