package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/registry"
	"github.com/spf13/cobra"
)

func newApproveProposalCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "approve-proposal <slug> <n>",
		Short: "Draft a new packet.yaml from an existing proposal",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApproveProposal(cmd, args[0], args[1])
		},
	}
}

func runApproveProposal(cmd *cobra.Command, slug, n string) error {
	fab, err := resolveFabric(cmd)
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)
	proposalPath := filepath.Join(packetDir, "proposals", n+".md")

	data, err := os.ReadFile(proposalPath)
	if err != nil {
		return fmt.Errorf("approve-proposal: %s", err)
	}
	title, body := splitProposal(string(data))

	draft := &packet.Packet{
		Goal:     title,
		Context:  body,
		CausedBy: &slug,
	}
	const draftPath = "packet.yaml"
	if err := packet.Save(draftPath, draft); err != nil {
		return fmt.Errorf("approve-proposal: %s", err)
	}

	fabDir := filepath.Dir(fab.PacketsDir)
	if err := registry.Append(filepath.Join(fabDir, "registry.jsonl"), registry.Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fabric:    fab.Slug,
		Packet:    slug,
		Path:      draftPath,
		Cause:     registry.CauseProposal,
	}); err != nil {
		return fmt.Errorf("approve-proposal: %s", err)
	}

	if err := journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Slug:      slug,
		Event:     journal.EventProposal,
		Actor:     journal.ActorHuman,
		Detail:    map[string]any{"proposal": n, "draft": draftPath},
	}); err != nil {
		return fmt.Errorf("approve-proposal: %s", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), draftPath)
	return nil
}

// splitProposal separates a proposal file's title (first line, per §13.7's
// "write /signals/proposal-<n>.md with a title on the first line and
// rationale below") from its body.
func splitProposal(contents string) (title, body string) {
	lines := strings.SplitN(contents, "\n", 2)
	title = strings.TrimSpace(lines[0])
	if len(lines) > 1 {
		body = strings.TrimSpace(lines[1])
	}
	return title, body
}
