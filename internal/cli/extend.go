package cli

import (
	"fmt"
	"path/filepath"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/spf13/cobra"
)

func newExtendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extend <slug>",
		Short: "Extend a packet's retry budget and resume it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			retries, err := cmd.Flags().GetInt("retries")
			if err != nil {
				return fmt.Errorf("extend: %s", err)
			}
			return runExtend(cmd, args[0], retries)
		},
	}
	cmd.Flags().Int("retries", 0, "retries to add to budget.retries")
	return cmd
}

func runExtend(cmd *cobra.Command, slug string, retries int) error {
	if retries <= 0 {
		return fmt.Errorf("extend: --retries must be > 0")
	}

	fab, err := resolveFabric(cmd)
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)
	statePath := filepath.Join(packetDir, "state.json")
	packetPath := filepath.Join(packetDir, "packet.yaml")

	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("extend: %s", err)
	}
	if st.State != state.Halted || st.HaltReason == nil || *st.HaltReason != "budget" {
		return fmt.Errorf("extend: %s is not halted with halt_reason=budget", slug)
	}

	pkt, err := packet.Load(packetPath)
	if err != nil {
		return fmt.Errorf("extend: %s", err)
	}
	// No new version: budget is bumped in place, unlike amend.
	pkt.Budget.Retries += retries
	if err := packet.Save(packetPath, pkt); err != nil {
		return fmt.Errorf("extend: %s", err)
	}

	if err := resumeBuilding(statePath, st, journal.EventExtend); err != nil {
		return fmt.Errorf("extend: %s", err)
	}
	return nil
}
