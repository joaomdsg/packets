package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
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
	fab, err := resolveFabric(cmd, "list")
	if err != nil {
		return err
	}

	// A malformed or empty fabric.yaml must surface as a clear error, not
	// as an empty listing: the fabric's own config being unreadable is
	// distinct from it simply having no packets yet.
	if _, err := fabric.Load(filepath.Join(filepath.Dir(fab.PacketsDir), "fabric.yaml")); err != nil {
		return fmt.Errorf("list: %s", err)
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
	var broken int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		st, err := state.LoadForSlug(filepath.Join(fab.PacketsDir, e.Name(), "state.json"), e.Name())
		if err != nil {
			// A packet's own error is reported by name so a corrupt
			// state.json doesn't vanish behind an empty-looking listing.
			fmt.Fprintf(cmd.ErrOrStderr(), "list: %s: %s\n", e.Name(), err)
			broken++
			continue
		}
		fmt.Fprintf(out, "%s\t%d\t%s\t%s\n", st.Slug, st.Version, st.State, st.UpdatedAt.Format(time.RFC3339))
	}

	// list has no single packet to log against, so it goes to the
	// fabric-level log instead (fabriclog.go).
	if err := logFabricEvent(fab, journal.EventList, nil); err != nil {
		return fmt.Errorf("list: %s", err)
	}
	if broken > 0 {
		// Non-zero exit so a corrupt packet is never confused with an
		// empty fabric; the healthy packets were still printed above.
		return fmt.Errorf("list: %d packet(s) failed to load", broken)
	}
	return nil
}
