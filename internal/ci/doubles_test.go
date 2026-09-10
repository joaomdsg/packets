package ci_test

import (
	"time"

	"github.com/joaomdsg/packets/internal/ci"
)

// stubGH is a hand-rolled GH double. checksSeq holds one []Check per poll
// call so a test can script a pending->success (or ->red) sequence; once
// exhausted, the last entry repeats.
type stubGH struct {
	checksSeq [][]ci.Check
	checksErr error

	logFailedOutput string
	logFailedErr    error

	mergeErr error
	merged   []int

	prURL    string
	prURLErr error

	closed   []int
	closeErr error

	pollCalls int
}

func (g *stubGH) PRChecks(dir string, pr int) ([]ci.Check, error) {
	if g.checksErr != nil {
		return nil, g.checksErr
	}
	if len(g.checksSeq) == 0 {
		return nil, nil
	}
	idx := g.pollCalls
	if idx >= len(g.checksSeq) {
		idx = len(g.checksSeq) - 1
	}
	g.pollCalls++
	return g.checksSeq[idx], nil
}

func (g *stubGH) RunViewLogFailed(dir, runID string) (string, error) {
	return g.logFailedOutput, g.logFailedErr
}

func (g *stubGH) MergePR(dir string, pr int, strategy string) error {
	g.merged = append(g.merged, pr)
	return g.mergeErr
}

func (g *stubGH) PRURL(dir string, pr int) (string, error) {
	return g.prURL, g.prURLErr
}

func (g *stubGH) ClosePR(dir string, pr int) error {
	g.closed = append(g.closed, pr)
	return g.closeErr
}

// stubGit is a hand-rolled Git double.
type stubGit struct {
	diffStat string

	currentCommit    string
	currentCommitErr error

	addedFiles []string
	addedErr   error
}

func (g *stubGit) DiffStat(dir, from string) (string, error) { return g.diffStat, nil }

func (g *stubGit) CurrentCommit(dir string) (string, error) {
	return g.currentCommit, g.currentCommitErr
}

func (g *stubGit) DiffAddedFiles(dir, base, head string) ([]string, error) {
	return g.addedFiles, g.addedErr
}

// stubClock advances only through Sleep, so a test can assert a poll loop
// crossed its deadline without ever waiting in real time.
type stubClock struct{ now time.Time }

func (c *stubClock) Now() time.Time        { return c.now }
func (c *stubClock) Sleep(d time.Duration) { c.now = c.now.Add(d) }
