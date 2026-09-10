package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/spf13/cobra"
)

func newProposalsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "proposals <slug>",
		Short: "List a packet's proposals/*.md files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProposals(cmd, args[0])
		},
	}
}

func runProposals(cmd *cobra.Command, slug string) error {
	fab, err := resolveFabric(cmd, "proposals")
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)

	entries, err := os.ReadDir(filepath.Join(packetDir, "proposals"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("proposals: %s", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := cmd.OutOrStdout()
	for _, name := range names {
		fmt.Fprintln(out, name)
	}

	if err := journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Slug:      slug,
		Event:     journal.EventProposalsListed,
		Actor:     journal.ActorHuman,
	}); err != nil {
		return fmt.Errorf("proposals: %s", err)
	}
	return nil
}
