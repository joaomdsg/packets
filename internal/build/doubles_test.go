package build_test

import (
	"strings"
	"time"

	"github.com/joaomdsg/packets/internal/build"
)

// stubGit is a hand-rolled Git double: fields configure what each method
// returns, and calls* fields record what was passed to the mutating ones.
type stubGit struct {
	fetchErr             error
	checkoutNewBranchErr error
	checkoutErr          error
	workingDiff          string
	workingDiffErr       error
	committed            bool
	commitErr            error
	rebaseConflict       bool
	rebaseErr            error
	rebaseAbortErr       error
	pushErr              error

	fetched        bool
	newBranches    []string
	checkedOut     []string
	commitMessages []string
	rebasedOnto    []string
	rebaseAborted  bool
	pushedBranches []string
}

func (s *stubGit) Fetch(dir string) error { s.fetched = true; return s.fetchErr }

func (s *stubGit) CheckoutNewBranch(dir, branch, from string) error {
	s.newBranches = append(s.newBranches, branch+" from "+from)
	return s.checkoutNewBranchErr
}

func (s *stubGit) Checkout(dir, branch string) error {
	s.checkedOut = append(s.checkedOut, branch)
	return s.checkoutErr
}

func (s *stubGit) WorkingDiffNameStatus(dir, from string) (string, error) {
	return s.workingDiff, s.workingDiffErr
}

func (s *stubGit) CommitAll(dir, message string) (bool, error) {
	s.commitMessages = append(s.commitMessages, message)
	return s.committed, s.commitErr
}

func (s *stubGit) RebaseOnto(dir, base string) (bool, error) {
	s.rebasedOnto = append(s.rebasedOnto, base)
	return s.rebaseConflict, s.rebaseErr
}

func (s *stubGit) RebaseAbort(dir string) error {
	s.rebaseAborted = true
	return s.rebaseAbortErr
}

func (s *stubGit) PushForceWithLease(dir, branch string) error {
	s.pushedBranches = append(s.pushedBranches, branch)
	return s.pushErr
}

// stubGH is a hand-rolled GH double.
type stubGH struct {
	prNumber int
	err      error
	titles   []string
	bodies   []string
}

func (s *stubGH) CreatePR(dir, title, body, head, base string) (int, error) {
	s.titles = append(s.titles, title)
	s.bodies = append(s.bodies, body)
	return s.prNumber, s.err
}

// stubDocker is a hand-rolled Docker double. runFunc, when set, lets a
// test inspect the argv (e.g. to find the signals dir bind mount and
// write files into it, standing in for what hooks would have written
// inside the container) before returning canned output.
type stubDocker struct {
	calls   [][]string
	runFunc func(args []string) (string, int, error)
}

func (s *stubDocker) Run(args []string) (string, int, error) {
	s.calls = append(s.calls, args)
	if s.runFunc != nil {
		return s.runFunc(args)
	}
	return `{"usage":{"input_tokens":10,"output_tokens":5},"total_cost_usd":0.02}`, 0, nil
}

// signalsDirFromArgs extracts the host path bind mounted at /signals from
// a docker run argv built by containerArgs.
func signalsDirFromArgs(args []string) string {
	for i, a := range args {
		if a == "-v" && i+1 < len(args) {
			if v, ok := strings.CutSuffix(args[i+1], ":/signals:rw"); ok {
				return v
			}
		}
	}
	return ""
}

// stubClock advances by step every time Now is called, giving
// deterministic elapsed durations without sleeping.
type stubClock struct {
	now  time.Time
	step time.Duration
}

func (c *stubClock) Now() time.Time {
	t := c.now
	c.now = c.now.Add(c.step)
	return t
}

func newStubDeps() (build.Deps, *stubGit, *stubGH, *stubDocker) {
	git := &stubGit{committed: true}
	gh := &stubGH{prNumber: 42}
	docker := &stubDocker{}
	deps := build.Deps{
		Git:    git,
		GH:     gh,
		Docker: docker,
		Clock:  &stubClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), step: time.Minute},
		Getenv: func(key string) string {
			if key == "ANTHROPIC_API_KEY" {
				return "fake-key"
			}
			return ""
		},
		Exists: func(string) bool { return false },
	}
	return deps, git, gh, docker
}
