package build

import (
	"path/filepath"
	"regexp"

	"github.com/joaomdsg/packets/internal/fabric"
)

var testDirSegment = regexp.MustCompile(`(^|/)(test|tests|spec|__tests__|e2e)(/|$)`)

var dependencyFiles = map[string]bool{
	"go.mod":            true,
	"package.json":      true,
	"package-lock.json": true,
	"requirements.txt":  true,
	"pyproject.toml":    true,
}

// hostSmellCheck is §12 step 6: the authoritative, host-side re-run of the
// in-container smell hooks. It runs before the attempt's changes are
// committed, so it diffs the fabric's default branch against the working
// tree (not HEAD, which wouldn't yet include this attempt's edits) and
// returns the halt_reason to use and whether any trigger fired.
func hostSmellCheck(g Git, repoPath, defaultBranch string, triggers fabric.SmellTriggers) (string, bool, error) {
	out, err := g.WorkingDiffNameStatus(repoPath, "origin/"+defaultBranch)
	if err != nil {
		return "", false, err
	}
	lines := parseDiffNameStatus(out)

	if triggers.MaxFilesTouched > 0 && len(lines) > triggers.MaxFilesTouched {
		return "smell:max_files", true, nil
	}

	for _, l := range lines {
		if triggers.DenyTestModification && l.Status == "M" && testDirSegment.MatchString(l.Path) {
			return "smell:test_modified", true, nil
		}
		if triggers.DenyDependencyAdd && dependencyFiles[filepath.Base(l.Path)] {
			return "smell:dependency", true, nil
		}
	}
	return "", false, nil
}
