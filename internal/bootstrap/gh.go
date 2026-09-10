package bootstrap

// GH is the gh external-process boundary init needs (§11 steps 3, 7).
type GH interface {
	DefaultBranch(dir string) (string, error)
	CreatePR(dir, title, body, head, base string) (url string, err error)
}

// ExecGH is the real GH, shelling out to the host's gh binary.
type ExecGH struct{}

func (ExecGH) DefaultBranch(dir string) (string, error) {
	return runCommand(dir, "gh", "repo", "view", "--json", "defaultBranchRef", "-q", ".defaultBranchRef.name")
}

func (ExecGH) CreatePR(dir, title, body, head, base string) (string, error) {
	return runCommand(dir, "gh", "pr", "create", "--title", title, "--body", body, "--head", head, "--base", base)
}
