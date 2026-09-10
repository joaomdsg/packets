package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every packet's slug, version, state, and updated_at",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd)
		},
	}
}

func runList(cmd *cobra.Command) error {
	fab, err := resolveFabric(cmd)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(fab.PacketsDir)
	if err != nil {
		if os.IsNotExist(err) {
			entries = nil
		} else {
			return fmt.Errorf("list: %s", err)
		}
	}

	out := cmd.OutOrStdout()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		st, err := state.Load(filepath.Join(fab.PacketsDir, e.Name(), "state.json"))
		if err != nil {
			continue // not a packet dir (or unreadable); skip rather than fail the whole listing
		}
		fmt.Fprintf(out, "%s\t%d\t%s\t%s\n", st.Slug, st.Version, st.State, st.UpdatedAt.Format(time.RFC3339))
	}

	// list has no single packet to log against, so it goes to the
	// fabric-level log instead (fabriclog.go).
	if err := logFabricEvent(fab, journal.EventList, nil); err != nil {
		return fmt.Errorf("list: %s", err)
	}
	return nil
}
