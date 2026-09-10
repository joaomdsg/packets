package bootstrap

// Docker is the docker external-process boundary init needs (§11 steps 5, 6).
type Docker interface {
	// Build runs `docker build -t tag dir`.
	Build(dir, tag string) error
	// Run executes `docker run <args...>` and returns its stdout, used
	// for the smoke test (§11 step 6).
	Run(args []string) (string, error)
}

// ExecDocker is the real Docker, shelling out to the host's docker binary.
type ExecDocker struct{}

func (ExecDocker) Build(dir, tag string) error {
	_, err := runCommand(dir, "docker", "build", "-t", tag, ".")
	return err
}

func (ExecDocker) Run(args []string) (string, error) {
	return runCommand("", "docker", args...)
}
