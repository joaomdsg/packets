package bootstrap

import "github.com/joaomdsg/packets/internal/procexec"

// runCommand executes name with args in dir (empty dir means the caller's
// cwd) and returns trimmed stdout; stderr is folded into the error since
// every caller here just needs to report the failure to the human.
func runCommand(dir, name string, args ...string) (string, error) {
	return procexec.Run(dir, name, args...)
}
