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
			fab, err := resolveFabric(cmd, "report")
			if err != nil {
				return err
			}
			r, err := report.Compute(filepath.Dir(fab.PacketsDir))
			if err != nil {
				return fmt.Errorf("report: %s", err)
			}
			// A skipped line is a torn write, not a defect in a specific
			// packet the way a corrupt state.json is for `list` — the
			// only actionable unit here is the aggregate metrics, which
			// still computed from every line that did parse. Warn and
			// keep exit 0 rather than making one bad byte anywhere in the
			// fabric's history block every future report.
			for _, s := range r.Skipped {
				fmt.Fprintf(cmd.ErrOrStderr(), "report: skipped %d unparseable line(s) in %s\n", s.Skipped, s.Path)
			}
			fmt.Fprint(cmd.OutOrStdout(), report.FormatText(r))
			return nil
		},
	}
}
