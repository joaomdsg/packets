package cli

import (
	"fmt"
	"path/filepath"

	"github.com/joaomdsg/packets/internal/report"
	"github.com/spf13/cobra"
)

func newReportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Print gate, termination, accretion, amend, and halt metrics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fab, err := resolveFabric(cmd)
			if err != nil {
				return err
			}
			r, err := report.Compute(filepath.Dir(fab.PacketsDir))
			if err != nil {
				return fmt.Errorf("report: %s", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), report.FormatText(r))
			return nil
		},
	}
}
