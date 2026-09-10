package ci

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joaomdsg/packets/internal/build"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
)

// runRed handles a red CI check: it writes failure.md from the failed
// run's logs, counts the retry, and either halts on budget or sends the
// packet back to build.
func runRed(
	deps Deps, fab *fabric.Config, pkt *packet.Packet, st *state.State, packetDir string,
	log logFunc, save func() error, halt func(reason, event string) (*state.State, error),
	check Check,
) (*state.State, error) {
	runID, _ := parseRunID(check.Link)
	logs, err := deps.GH.RunViewLogFailed(fab.RepoPath, runID)
	if err != nil {
		return nil, fmt.Errorf("ci: gh run view --log-failed: %s", err)
	}
	diffStat, err := deps.Git.DiffStat(fab.RepoPath, "origin/"+fab.DefaultBranch)
	if err != nil {
		return nil, fmt.Errorf("ci: git diff --stat: %s", err)
	}

	failureMD := build.FormatFailure(st.Attempt, "ci", fab.CI.WorkflowName, 1, tail(logs, failureTailLines), diffStat)
	if err := os.WriteFile(filepath.Join(packetDir, "failure.md"), []byte(failureMD), 0o600); err != nil {
		return nil, fmt.Errorf("ci: write failure.md: %s", err)
	}
	// The workflow check is a single aggregate job (§13.9), not one
	// predicate at a time, so "kind" here is neither "constraint" nor
	// "terminal" as local_pred/ci_pred elsewhere in §10 assume.
	if err := log(journal.EventCIPred, "", 0, map[string]any{
		"cmd": fab.CI.WorkflowName, "exit": 1, "kind": "ci",
	}); err != nil {
		return nil, err
	}

	st.RetriesUsed++
	if err := save(); err != nil {
		return nil, err
	}
	if st.RetriesUsed >= pkt.Budget.Retries {
		return halt("budget", journal.EventBudgetHalt)
	}

	next, err := st.Transition(state.Building)
	if err != nil {
		return nil, err
	}
	*st = next
	node := state.NodeBuild
	st.Node = &node
	if err := save(); err != nil {
		return nil, err
	}
	if err := log(journal.EventEnterNode, "", 0, nil); err != nil {
		return nil, err
	}
	return st, nil
}
