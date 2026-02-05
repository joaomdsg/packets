package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const DefaultModel = "sonnet"

type ClaudeRunner struct {
	timeout time.Duration
	model   string
}

func NewClaudeRunner(timeout time.Duration, model string) *ClaudeRunner {
	if model == "" {
		model = DefaultModel
	}
	return &ClaudeRunner{timeout: timeout, model: model}
}

func (c *ClaudeRunner) Model() string {
	return c.model
}

func (c *ClaudeRunner) Run(
	ctx context.Context, workDir, prompt string,
) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude",
		"--print",
		"--dangerously-skip-permissions",
		"--no-session-persistence",
		"--model", c.model,
		"-p", prompt)
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
		// Include both stdout and stderr - Claude CLI may output errors to either
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
