package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type OpenCodeRunner struct {
	timeout time.Duration
	model   string
}

func NewOpenCodeRunner(timeout time.Duration, model string) *OpenCodeRunner {
	return &OpenCodeRunner{timeout: timeout, model: model}
}

func (o *OpenCodeRunner) Model() string {
	return o.model
}

func (o *OpenCodeRunner) Run(
	ctx context.Context, workDir, prompt string,
) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	// OpenCode run subcommand with prompt as positional argument
	args := []string{"run", prompt}
	if o.model != "" {
		args = append(args, "-m", o.model)
	}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &Result{
				Output:  stdout.String(),
				Success: false,
				Error:   "timeout exceeded",
			}, nil
		}
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return &Result{
			Output:  stdout.String(),
			Success: false,
			Error:   fmt.Sprintf("%v: %s", err, errMsg),
		}, nil
	}

	return &Result{
		Output:  stdout.String(),
		Success: true,
	}, nil
}
