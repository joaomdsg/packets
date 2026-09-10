package ci

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/joaomdsg/packets/internal/procexec"
)

// Check is one entry of `gh pr checks --json name,state,link`.
type Check struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Link  string `json:"link"`
}

// GH is the gh external-process boundary the ci node needs.
type GH interface {
	PRChecks(dir string, pr int) ([]Check, error)
	RunViewLogFailed(dir, runID string) (string, error)
	MergePR(dir string, pr int, strategy string) error
	PRURL(dir string, pr int) (string, error)
	ClosePR(dir string, pr int) error
}

// ExecGH is the real GH, shelling out to the host's gh binary.
type ExecGH struct{}

func (ExecGH) PRChecks(dir string, pr int) ([]Check, error) {
	out, err := procexec.Run(dir, "gh", "pr", "checks", fmt.Sprint(pr), "--json", "name,state,link")
	if err != nil {
		return nil, err
	}
	var checks []Check
	if err := json.Unmarshal([]byte(out), &checks); err != nil {
		return nil, fmt.Errorf("ci: parse gh pr checks output: %s", err)
	}
	return checks, nil
}

func (ExecGH) RunViewLogFailed(dir, runID string) (string, error) {
	return procexec.Run(dir, "gh", "run", "view", runID, "--log-failed")
}

func (ExecGH) MergePR(dir string, pr int, strategy string) error {
	_, err := procexec.Run(dir, "gh", "pr", "merge", fmt.Sprint(pr), "--"+strategy, "--delete-branch=false")
	return err
}

func (ExecGH) PRURL(dir string, pr int) (string, error) {
	return procexec.Run(dir, "gh", "pr", "view", fmt.Sprint(pr), "--json", "url", "-q", ".url")
}

func (ExecGH) ClosePR(dir string, pr int) error {
	_, err := procexec.Run(dir, "gh", "pr", "close", fmt.Sprint(pr))
	return err
}

var runIDPattern = regexp.MustCompile(`/actions/runs/(\d+)`)

// parseRunID extracts the numeric run id from a check's details link, as
// printed by `gh pr checks --json link`.
func parseRunID(link string) (string, bool) {
	m := runIDPattern.FindStringSubmatch(link)
	if m == nil {
		return "", false
	}
	return m[1], true
}
