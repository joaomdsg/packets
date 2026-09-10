package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/gate"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/joaomdsg/packets/internal/xdgpath"
	"github.com/spf13/cobra"
)

func newAmendCommand(llm gate.LLM, editor Editor) *cobra.Command {
	return &cobra.Command{
		Use:   "amend <slug>",
		Short: "Edit a packet's packet.yaml and re-emit as a new version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAmend(cmd, args[0], llm, editor)
		},
	}
}

// runAmend implements §7's amend seam: open packet.yaml in $EDITOR, run
// the same gate flow emit uses, and on pass bump the version, snapshot
// it, and restart the packet at build. §5 allows amend from halted or
// awaiting_approval only.
func runAmend(cmd *cobra.Command, slug string, llm gate.LLM, editor Editor) error {
	fab, err := resolveFabric(cmd)
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)
	statePath := filepath.Join(packetDir, "state.json")
	packetPath := filepath.Join(packetDir, "packet.yaml")
	logPath := filepath.Join(packetDir, "log.jsonl")

	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("amend: %s", err)
	}
	if st.State != state.Halted && st.State != state.AwaitingApproval {
		return fmt.Errorf("amend: %s is %s, not halted or awaiting_approval", slug, st.State)
	}

	before, err := os.ReadFile(packetPath)
	if err != nil {
		return fmt.Errorf("amend: %s", err)
	}
	if err := editor.Open(packetPath); err != nil {
		return err
	}
	after, err := os.ReadFile(packetPath)
	if err != nil {
		return fmt.Errorf("amend: %s", err)
	}

	p, err := packet.Load(packetPath)
	if err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), err)
		return fmt.Errorf("amend: %s did not parse", packetPath)
	}
	failures := packet.Validate(p, fab.PacketsDir)
	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintln(cmd.OutOrStdout(), f)
		}
		if err := journal.Append(logPath, journal.Entry{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Slug:      slug,
			Version:   st.Version,
			Event:     journal.EventEmitReject,
			Actor:     journal.ActorHuman,
			Detail:    map[string]any{"failures": failures},
		}); err != nil {
			return fmt.Errorf("amend: %s", err)
		}
		return fmt.Errorf("amend: %s failed validation", packetPath)
	}

	repoFiles, err := repoListing(fab.RepoPath)
	if err != nil {
		return fmt.Errorf("amend: %s", err)
	}
	outcome := runGate(llm, string(after), p.Terminal, repoFiles, cmd.InOrStdin(), cmd.OutOrStdout())
	if !outcome.proceed {
		return fmt.Errorf("amend: not amended, warnings not overridden")
	}
	p.Terminal = outcome.terminal

	nextVersion := st.Version + 1
	versionsDir := filepath.Join(packetDir, "versions")
	if err := xdgpath.EnsureDir(versionsDir); err != nil {
		return fmt.Errorf("amend: %s", err)
	}
	versionPath := filepath.Join(versionsDir, fmt.Sprintf("v%d.yaml", nextVersion))
	if err := packet.Save(versionPath, p); err != nil {
		return fmt.Errorf("amend: %s", err)
	}
	if err := packet.Save(packetPath, p); err != nil {
		return fmt.Errorf("amend: %s", err)
	}

	next, err := st.Transition(state.Emitted)
	if err != nil {
		return fmt.Errorf("amend: %s", err)
	}
	*st = next
	st.Version = nextVersion
	st.RetriesUsed = 0
	st.HaltReason = nil
	st.Node = nil
	st.GateSuggestedTerminals = append(st.GateSuggestedTerminals, outcome.gateSuggestedTerminals...)
	st.UpdatedAt = time.Now().UTC()
	// Branch and pr_number are left as-is: §15 requires amend to restart
	// at build with the previous branch checked out, which build.Run does
	// automatically whenever state.branch is already set.
	if err := state.Save(statePath, st); err != nil {
		return fmt.Errorf("amend: %s", err)
	}

	for _, e := range outcome.entries {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339)
		e.Slug = slug
		e.Version = nextVersion
		e.Actor = journal.ActorHuman
		if err := journal.Append(logPath, e); err != nil {
			return fmt.Errorf("amend: %s", err)
		}
	}

	summary := firstDiffLine(string(before), string(after))
	if err := journal.Append(logPath, journal.Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Slug:      slug,
		Version:   nextVersion,
		Event:     journal.EventAmend,
		Actor:     journal.ActorHuman,
		Detail:    map[string]any{"summary": summary},
	}); err != nil {
		return fmt.Errorf("amend: %s", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), slug)
	return nil
}
