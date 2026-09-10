package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/ci"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/spf13/cobra"
)

func newKillCommand(ciDeps ci.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "kill <slug>",
		Short: "Kill a packet in any non-final state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKill(cmd, args[0], ciDeps)
		},
	}
}

func runKill(cmd *cobra.Command, slug string, ciDeps ci.Deps) error {
	fab, err := resolveFabric(cmd, "kill")
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)
	fabDir := filepath.Dir(fab.PacketsDir)
	statePath := filepath.Join(packetDir, "state.json")

	st, err := state.LoadForSlug(statePath, slug)
	if err != nil {
		return fmt.Errorf("kill: %s", err)
	}

	next, err := st.Transition(state.Killed)
	if err != nil {
		return fmt.Errorf("kill: %s", err)
	}
	*st = next

	// Closing the PR does not delete the branch (§7): the harness never
	// removes work, only the intent to keep pursuing it.
	if st.PRNumber != 0 {
		fabCfg, err := fabric.Load(filepath.Join(fabDir, "fabric.yaml"))
		if err != nil {
			return fmt.Errorf("kill: %s", err)
		}
		if err := ciDeps.GH.ClosePR(fabCfg.RepoPath, st.PRNumber); err != nil {
			return fmt.Errorf("kill: gh pr close: %s", err)
		}
	}

	st.UpdatedAt = time.Now().UTC()
	if err := state.Save(statePath, st); err != nil {
		return fmt.Errorf("kill: %s", err)
	}
	if err := journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Slug:      st.Slug,
		Version:   st.Version,
		Event:     journal.EventKill,
		Actor:     journal.ActorHuman,
	}); err != nil {
		return fmt.Errorf("kill: %s", err)
	}
	return nil
}
