package cli

import (
	"fmt"
	"path/filepath"

	"github.com/joaomdsg/packets/internal/state"
	"github.com/spf13/cobra"
)

func newCheckApprovedCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check-approved <slug>",
		Short: "Exit 0 iff the packet's state.json has approved == true",
		Args:  cobra.ExactArgs(1),
		// Runs from inside a shell predicate, possibly many times per
		// attempt, so it stays quiet: no stdout noise and no log line
		// (§7's "every seam logs" rule is for human-driven seams; this one
		// is exercised by generated predicate scripts, not a human typing
		// a command).
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckApproved(cmd, args[0])
		},
	}
}

func runCheckApproved(cmd *cobra.Command, slug string) error {
	fab, err := resolveFabric(cmd)
	if err != nil {
		return err
	}
	st, err := state.Load(filepath.Join(fab.PacketsDir, slug, "state.json"))
	if err != nil {
		return err
	}
	if !st.Approved {
		return fmt.Errorf("check-approved: %s not approved", slug)
	}
	return nil
}
