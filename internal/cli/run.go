package cli

import (
	"fmt"
	"path/filepath"

	"github.com/joaomdsg/packets/internal/build"
	"github.com/joaomdsg/packets/internal/ci"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/spf13/cobra"
)

func newRunCommand(buildDeps build.Deps, ciDeps ci.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "run <slug>",
		Short: "Run the packet loop until it halts, awaits approval, or ends",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRun(cmd, args[0], buildDeps, ciDeps)
		},
	}
}

// alreadyDone are the states §12 exits immediately on: `packets run` is a
// no-op on a packet that isn't waiting to build.
var alreadyDone = map[string]bool{
	state.Terminated:       true,
	state.Killed:           true,
	state.Halted:           true,
	state.AwaitingApproval: true,
}

func runRun(cmd *cobra.Command, slug string, buildDeps build.Deps, ciDeps ci.Deps) error {
	fab, err := resolveFabric(cmd, "run")
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)
	fabDir := filepath.Dir(fab.PacketsDir)
	statePath := filepath.Join(packetDir, "state.json")

	fabCfg, err := fabric.Load(filepath.Join(fabDir, "fabric.yaml"))
	if err != nil {
		return fmt.Errorf("run: %s", err)
	}
	pkt, err := packet.Load(filepath.Join(packetDir, "packet.yaml"))
	if err != nil {
		return fmt.Errorf("run: %s", err)
	}
	st, err := state.LoadForSlug(statePath, slug)
	if err != nil {
		return fmt.Errorf("run: %s", err)
	}

	if alreadyDone[st.State] {
		fmt.Fprintln(cmd.OutOrStdout(), st.State)
		return nil
	}

	// "emitted --run--> building" (§5): the loop always starts building on
	// its first run, entering the build node.
	if st.State == state.Emitted {
		next, err := st.Transition(state.Building)
		if err != nil {
			return fmt.Errorf("run: %s", err)
		}
		*st = next
		node := state.NodeBuild
		st.Node = &node
		if err := state.Save(statePath, st); err != nil {
			return fmt.Errorf("run: %s", err)
		}
	}

	// Drive build->ci->build cycles to completion: each node mutates st in
	// place and hands off by updating st.Node/st.State, so the loop just
	// re-reads them until a terminal-for-this-command state is reached.
	for !alreadyDone[st.State] {
		node := state.NodeBuild
		if st.Node != nil {
			node = *st.Node
		}
		switch node {
		case state.NodeBuild:
			if _, err := build.Run(buildDeps, fabCfg, pkt, st, packetDir); err != nil {
				return fmt.Errorf("run: %s", err)
			}
		case state.NodeCI:
			if _, err := ci.Run(ciDeps, fabCfg, pkt, st, packetDir, fabDir); err != nil {
				return fmt.Errorf("run: %s", err)
			}
		default:
			return fmt.Errorf("run: unknown node %q", node)
		}
	}

	// The only path into awaiting_approval in this phase is green CI with
	// human_approval still outstanding (§5); print the PR so the human has
	// it at hand to review.
	if st.State == state.AwaitingApproval && st.PRNumber != 0 {
		if url, err := ciDeps.GH.PRURL(fabCfg.RepoPath, st.PRNumber); err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), url)
		}
	}
	return nil
}
