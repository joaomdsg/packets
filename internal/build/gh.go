package build

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joaomdsg/packets/internal/procexec"
)

// GH is the gh external-process boundary the build node needs (§12 step
// 9). Unlike bootstrap.GH, CreatePR reports the PR number directly since
// that's what state.json.pr_number stores.
type GH interface {
	CreatePR(dir, title, body, head, base string) (prNumber int, err error)
}

// ExecGH is the real GH, shelling out to the host's gh binary.
type ExecGH struct{}

func (ExecGH) CreatePR(dir, title, body, head, base string) (int, error) {
	out, err := procexec.Run(dir, "gh", "pr", "create",
		"--title", title, "--body", body, "--head", head, "--base", base)
	if err != nil {
		return 0, err
	}
	return parsePRNumber(out)
}

// parsePRNumber extracts the trailing number from a PR URL, which is what
// `gh pr create` prints on success.
func parsePRNumber(url string) (int, error) {
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	idx := strings.LastIndex(url, "/")
	if idx == -1 {
		return 0, fmt.Errorf("gh: cannot find PR number in output: %q", url)
	}
	n, err := strconv.Atoi(url[idx+1:])
	if err != nil {
		return 0, fmt.Errorf("gh: cannot parse PR number from %q: %s", url, err)
	}
	return n, nil
}
