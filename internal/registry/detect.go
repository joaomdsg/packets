package registry

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var testDirSegment = regexp.MustCompile(`(^|/)(test|tests|spec|__tests__|e2e)(/|$)`)

// Git is the git external-process boundary DetectCIFailure needs.
type Git interface {
	DiffAddedFiles(dir, base, head string) ([]string, error)
}

// DetectGateSuggested finds, best effort, the repo-relative file each
// gate-suggested terminal command references: any whitespace-separated
// token in the command that exists as a path under repoPath. Never
// errors: a failed-to-detect terminal is simply skipped.
func DetectGateSuggested(repoPath string, terminals []string) []string {
	var found []string
	seen := map[string]bool{}
	for _, cmd := range terminals {
		for _, tok := range strings.Fields(cmd) {
			tok = strings.Trim(tok, `"'`)
			if tok == "" || seen[tok] {
				continue
			}
			if _, err := os.Stat(filepath.Join(repoPath, tok)); err == nil {
				found = append(found, tok)
				seen[tok] = true
			}
		}
	}
	return found
}

// DetectCIFailure finds new files under a test directory that this
// branch added relative to base and that were not already recorded as
// gate-suggested. Test directories are any path segment matching
// test|tests|spec|__tests__|e2e.
func DetectCIFailure(g Git, repoPath, base, head string, exclude []string) ([]string, error) {
	added, err := g.DiffAddedFiles(repoPath, base, head)
	if err != nil {
		return nil, err
	}
	excluded := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		excluded[e] = true
	}
	var out []string
	for _, path := range added {
		if testDirSegment.MatchString(path) && !excluded[path] {
			out = append(out, path)
		}
	}
	return out, nil
}
