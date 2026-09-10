package ci

import (
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
)

type pollOutcome int

const (
	pollGreen pollOutcome = iota
	pollRed
	pollTimeout
)

const checkSuccess = "SUCCESS"

var redCheckStates = map[string]bool{"FAILURE": true, "ERROR": true, "CANCELLED": true}

// poll repeats `gh pr checks` until the check named fab.ci.workflow_name
// resolves to SUCCESS or a red state, or the timeout deadline passes. It
// waits between polls through deps.Clock.Sleep so tests can drive a full
// poll sequence with a virtual clock instead of a real wait.
func poll(deps Deps, fab *fabric.Config, prNumber int, log logFunc) (Check, pollOutcome, error) {
	deadline := deps.Clock.Now().Add(time.Duration(fab.CI.TimeoutMinutes) * time.Minute)
	interval := time.Duration(fab.CI.PollSeconds) * time.Second

	for {
		checks, err := deps.GH.PRChecks(fab.RepoPath, prNumber)
		if err != nil {
			return Check{}, 0, err
		}
		found, ok := findCheck(checks, fab.CI.WorkflowName)
		if err := log(journal.EventCIPoll, "", 0, map[string]any{
			"name": found.Name, "state": found.State,
		}); err != nil {
			return Check{}, 0, err
		}
		if ok {
			if found.State == checkSuccess {
				return found, pollGreen, nil
			}
			if redCheckStates[found.State] {
				return found, pollRed, nil
			}
		}
		if !deps.Clock.Now().Before(deadline) {
			return Check{}, pollTimeout, nil
		}
		deps.Clock.Sleep(interval)
	}
}

func findCheck(checks []Check, name string) (Check, bool) {
	for _, c := range checks {
		if c.Name == name {
			return c, true
		}
	}
	return Check{}, false
}
