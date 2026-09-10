// Package cli wires the packets command-line interface.
package cli

import (
	"os"
	"os/exec"

	"github.com/joaomdsg/packets/internal/build"
	"github.com/joaomdsg/packets/internal/ci"
	"github.com/joaomdsg/packets/internal/deps"
	"github.com/joaomdsg/packets/internal/gate"
	"github.com/spf13/cobra"
)

// fileExists is the production Exists boundary for claudeauth.Resolve,
// shared by init and build's Deps wiring.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// rootConfig holds NewRoot's optional dependencies.
type rootConfig struct {
	llm       gate.LLM
	buildDeps build.Deps
	ciDeps    ci.Deps
	editor    Editor
}

// Option configures NewRoot. The zero-value config uses the real,
// production dependencies; tests override individual pieces.
type Option func(*rootConfig)

// WithGateLLM overrides the emit gate's LLM boundary, used by tests to
// inject a stub in place of the real "claude -p" process.
func WithGateLLM(llm gate.LLM) Option {
	return func(c *rootConfig) {
		c.llm = llm
	}
}

// WithBuildDeps overrides the build node's git/gh/docker/clock boundaries,
// used by tests to drive `packets run` without a real container, git
// remote, or gh.
func WithBuildDeps(d build.Deps) Option {
	return func(c *rootConfig) {
		c.buildDeps = d
	}
}

// WithCIDeps overrides the ci node's gh/git/clock boundaries, used by
// tests to drive `packets run` and `packets approve` without a real gh or
// network.
func WithCIDeps(d ci.Deps) Option {
	return func(c *rootConfig) {
		c.ciDeps = d
	}
}

// WithEditor overrides the amend seam's $EDITOR boundary, used by tests to
// inject a stub in place of launching a real editor.
func WithEditor(e Editor) Option {
	return func(c *rootConfig) {
		c.editor = e
	}
}

// NewRoot builds the packets root command with every command from the CLI
// seam table wired in, stubbed commands included.
func NewRoot(opts ...Option) *cobra.Command {
	cfg := rootConfig{
		llm: gate.ClaudeCLI{},
		buildDeps: build.Deps{
			Git:    build.ExecGit{},
			GH:     build.ExecGH{},
			Docker: build.ExecDocker{},
			Clock:  build.RealClock{},
			Getenv: os.Getenv,
			Exists: fileExists,
		},
		ciDeps: ci.Deps{
			GH:    ci.ExecGH{},
			Git:   ci.ExecGit{},
			Clock: ci.RealClock{},
		},
		editor: ExecEditor{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	root := &cobra.Command{
		Use:           "packets",
		Short:         "Run autonomous coding packets against a fabric",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return deps.Check(exec.LookPath)
		},
	}
	root.PersistentFlags().String("fabric", "", "fabric slug (default: fabric whose repo_path contains cwd)")

	root.AddCommand(
		newNewCommand(),
		newEmitCommand(cfg.llm),
		newInitCommand(),
		newRunCommand(cfg.buildDeps, cfg.ciDeps),
		newStatusCommand(),
		newListCommand(),
		newResumeCommand(),
		newExtendCommand(),
		newAmendCommand(cfg.llm, cfg.editor),
		newKillCommand(cfg.ciDeps),
		newApproveCommand(cfg.ciDeps),
		newCheckApprovedCommand(),
		newProposalsCommand(),
		newApproveProposalCommand(),
		newReportCommand(),
	)
	return root
}
