package ci

import (
	"strings"

	"github.com/joaomdsg/packets/internal/procexec"
)

// Git is the git external-process boundary the ci node needs: enough to
// describe what an attempt changed for failure.md, to stamp the accretion
// registry with a commit hash, and to detect files added on the branch for
// registry.DetectCIFailure.
type Git interface {
	DiffStat(dir, from string) (string, error)
	CurrentCommit(dir string) (string, error)
	DiffAddedFiles(dir, base, head string) ([]string, error)
}

// ExecGit is the real Git, shelling out to the host's git binary.
type ExecGit struct{}

func (ExecGit) DiffStat(dir, from string) (string, error) {
	return procexec.Run(dir, "git", "diff", "--stat", from)
}

func (ExecGit) CurrentCommit(dir string) (string, error) {
	return procexec.Run(dir, "git", "rev-parse", "HEAD")
}

func (ExecGit) DiffAddedFiles(dir, base, head string) ([]string, error) {
	out, err := procexec.Run(dir, "git", "diff", "--name-status", "--diff-filter=A", base, head)
	if err != nil {
		return nil, err
	}
	return parseAddedPaths(out), nil
}

// parseAddedPaths extracts the path column from `git diff --name-status`
// output already filtered to added ("A") entries.
func parseAddedPaths(output string) []string {
	var paths []string
	for _, raw := range strings.Split(output, "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		fields := strings.Fields(raw)
		if len(fields) < 2 {
			continue
		}
		paths = append(paths, fields[len(fields)-1])
	}
	return paths
}
