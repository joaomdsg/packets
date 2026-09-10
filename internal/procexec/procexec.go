// Package procexec is the shared shelling-out boundary for git/gh/docker
// process wrappers across bootstrap and build.
package procexec

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes name with args in dir (empty dir means the caller's cwd)
// and returns trimmed stdout; stderr is folded into the error since most
// callers just need to report the failure to the human.
func Run(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %s: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunCode executes name with args in dir, returning combined stdout+stderr
// and the process's exit code. Unlike Run, a non-zero exit is not an
// error: err is non-nil only when the process could not be started or run
// at all, so callers that care about specific exit codes (e.g. a `timeout`
// wrapper's 124) can inspect them directly.
func RunCode(dir, name string, args ...string) (output string, exitCode int, err error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	runErr := cmd.Run()
	output = buf.String()
	if runErr == nil {
		return output, 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		return output, exitErr.ExitCode(), nil
	}
	return output, -1, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), runErr)
}
