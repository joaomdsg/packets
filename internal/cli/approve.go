package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/ci"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/spf13/cobra"
)

func newApproveCommand(ciDeps ci.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "approve <slug>",
		Short: "Approve a packet awaiting human approval, then merge it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApprove(cmd, args[0], ciDeps)
		},
	}
}

func runApprove(cmd *cobra.Command, slug string, ciDeps ci.Deps) error {
	fab, err := resolveFabric(cmd)
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)
	fabDir := filepath.Dir(fab.PacketsDir)
	statePath := filepath.Join(packetDir, "state.json")
	logPath := filepath.Join(packetDir, "log.jsonl")

	fabCfg, err := fabric.Load(filepath.Join(fabDir, "fabric.yaml"))
	if err != nil {
		return fmt.Errorf("approve: %s", err)
	}
	pkt, err := packet.Load(filepath.Join(packetDir, "packet.yaml"))
	if err != nil {
		return fmt.Errorf("approve: %s", err)
	}
	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("approve: %s", err)
	}
	if st.State != state.AwaitingApproval {
		return fmt.Errorf("approve: %s is %s, not awaiting_approval", slug, st.State)
	}

	// Write-before-act: persist approved=true before the merge it gates.
	st.Approved = true
	st.UpdatedAt = ciDeps.Clock.Now().UTC()
	if err := state.Save(statePath, st); err != nil {
		return fmt.Errorf("approve: %s", err)
	}
	if err := journal.Append(logPath, journal.Entry{
		Timestamp: ciDeps.Clock.Now().UTC().Format(time.RFC3339),
		Slug:      st.Slug,
		Version:   st.Version,
		Event:     journal.EventApprove,
		Actor:     journal.ActorHuman,
	}); err != nil {
		return fmt.Errorf("approve: %s", err)
	}

	if _, err := ci.Merge(ciDeps, fabCfg, pkt, st, packetDir, fabDir, journal.ActorHuman); err != nil {
		return fmt.Errorf("approve: %s", err)
	}
	return nil
}
