package bootstrap

// Git is the git external-process boundary init needs (§11 steps 1, 3, 7).
type Git interface {
	RemoteURL(dir string) (string, error)
	Toplevel(dir string) (string, error)
	CreateBranch(dir, branch string) error
	CommitAll(dir, message string) error
	Push(dir, branch string) error
}

// ExecGit is the real Git, shelling out to the host's git binary.
type ExecGit struct{}

func (ExecGit) RemoteURL(dir string) (string, error) {
	return runCommand(dir, "git", "remote", "get-url", "origin")
}

func (ExecGit) Toplevel(dir string) (string, error) {
	return runCommand(dir, "git", "rev-parse", "--show-toplevel")
}

func (ExecGit) CreateBranch(dir, branch string) error {
	_, err := runCommand(dir, "git", "checkout", "-B", branch)
	return err
}

func (ExecGit) CommitAll(dir, message string) error {
	if _, err := runCommand(dir, "git", "add", "-A"); err != nil {
		return err
	}
	_, err := runCommand(dir, "git", "commit", "-m", message)
	return err
}

func (ExecGit) Push(dir, branch string) error {
	_, err := runCommand(dir, "git", "push", "-u", "origin", branch)
	return err
}
