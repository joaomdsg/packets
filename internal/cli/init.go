package cli

import (
	"fmt"
	"os"

	"github.com/joaomdsg/packets/internal/bootstrap"
	"github.com/joaomdsg/packets/internal/xdgpath"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Bootstrap a fabric from the current repo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd)
		},
	}
}

func runInit(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("init: %s", err)
	}
	configDir, err := xdgpath.ConfigDir()
	if err != nil {
		return fmt.Errorf("init: %s", err)
	}
	dataDir, err := xdgpath.DataDir()
	if err != nil {
		return fmt.Errorf("init: %s", err)
	}

	deps := bootstrap.Deps{
		Git:    bootstrap.ExecGit{},
		GH:     bootstrap.ExecGH{},
		Docker: bootstrap.ExecDocker{},
		Getenv: os.Getenv,
		Exists: fileExists,
		Stdin:  cmd.InOrStdin(),
		Stdout: cmd.OutOrStdout(),
	}
	result, err := bootstrap.Init(deps, bootstrap.InitOptions{RepoDir: cwd, ConfigDir: configDir, DataDir: dataDir})
	if err != nil {
		return fmt.Errorf("init: %s", err)
	}
	if result.AlreadyInit {
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), result.Slug)
	fmt.Fprintln(cmd.OutOrStdout(), result.PRURL)
	fmt.Fprintln(cmd.OutOrStdout(), "merge the packets CI workflow PR above to finish enabling packets for this repo")
	return nil
}
