package build

import (
	"strings"

	"github.com/joaomdsg/packets/internal/procexec"
)

// Git is the git external-process boundary the build node needs (§12
// steps 1, 6, 7, 8, 9). It is a wider interface than bootstrap.Git, whose
// methods serve packets init instead.
type Git interface {
	Fetch(dir string) error
	CheckoutNewBranch(dir, branch, from string) error
	Checkout(dir, branch string) error
	// WorkingDiffNameStatus runs `git diff --name-status from`, one
	// "STATUS\tpath" line per touched file. It compares from against the
	// working tree (not a second ref) since §12 step 6 runs before the
	// attempt's edits are committed.
	WorkingDiffNameStatus(dir, from string) (string, error)
	// CommitAll stages everything and commits; committed is false when
	// there was nothing staged, letting the caller halt reason=no_changes
	// instead of treating an empty commit as an error.
	CommitAll(dir, message string) (committed bool, err error)
	// RebaseOnto rebases the current branch onto base; conflict is true
	// when the rebase stopped on a conflict, in which case the caller must
	// still call RebaseAbort.
	RebaseOnto(dir, base string) (conflict bool, err error)
	RebaseAbort(dir string) error
	PushForceWithLease(dir, branch string) error
}

// ExecGit is the real Git, shelling out to the host's git binary.
type ExecGit struct{}

func (ExecGit) Fetch(dir string) error {
	_, err := procexec.Run(dir, "git", "fetch", "origin")
	return err
}

func (ExecGit) CheckoutNewBranch(dir, branch, from string) error {
	_, err := procexec.Run(dir, "git", "checkout", "-B", branch, from)
	return err
}

func (ExecGit) Checkout(dir, branch string) error {
	_, err := procexec.Run(dir, "git", "checkout", branch)
	return err
}

func (ExecGit) WorkingDiffNameStatus(dir, from string) (string, error) {
	return procexec.Run(dir, "git", "diff", "--name-status", from)
}

func (ExecGit) CommitAll(dir, message string) (bool, error) {
	if _, err := procexec.Run(dir, "git", "add", "-A"); err != nil {
		return false, err
	}
	// A quiet, zero-exit `diff --cached` means nothing is staged; distinct
	// from a real error so the caller can tell "nothing to commit" apart
	// from a broken git invocation.
	_, exitCode, err := procexec.RunCode(dir, "git", "diff", "--cached", "--quiet")
	if err != nil {
		return false, err
	}
	if exitCode == 0 {
		return false, nil
	}
	if _, err := procexec.Run(dir, "git", "commit", "-m", message); err != nil {
		return false, err
	}
	return true, nil
}

func (ExecGit) RebaseOnto(dir, base string) (bool, error) {
	_, exitCode, err := procexec.RunCode(dir, "git", "rebase", base)
	if err != nil {
		return false, err
	}
	return exitCode != 0, nil
}

func (ExecGit) RebaseAbort(dir string) error {
	_, err := procexec.Run(dir, "git", "rebase", "--abort")
	return err
}

func (ExecGit) PushForceWithLease(dir, branch string) error {
	_, err := procexec.Run(dir, "git", "push", "--force-with-lease", "origin", branch)
	return err
}

// diffNameStatusLine is one parsed line of `git diff --name-status`
// output.
type diffNameStatusLine struct {
	Status string
	Path   string
}

func parseDiffNameStatus(output string) []diffNameStatusLine {
	var lines []diffNameStatusLine
	for _, raw := range strings.Split(output, "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		fields := strings.Fields(raw)
		if len(fields) < 2 {
			continue
		}
		lines = append(lines, diffNameStatusLine{Status: fields[0], Path: fields[len(fields)-1]})
	}
	return lines
}
