package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status <slug>",
		Short: "Print a packet's state.json fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, args[0])
		},
	}
}

func runStatus(cmd *cobra.Command, slug string) error {
	fab, err := resolveFabric(cmd)
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)
	st, err := state.Load(filepath.Join(packetDir, "state.json"))
	if err != nil {
		return fmt.Errorf("status: %s", err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "slug: %s\n", st.Slug)
	fmt.Fprintf(out, "version: %d\n", st.Version)
	fmt.Fprintf(out, "state: %s\n", st.State)
	fmt.Fprintf(out, "node: %s\n", derefStr(st.Node))
	fmt.Fprintf(out, "attempt: %d\n", st.Attempt)
	fmt.Fprintf(out, "retries_used: %d\n", st.RetriesUsed)
	fmt.Fprintf(out, "minutes_used: %g\n", st.MinutesUsed)
	fmt.Fprintf(out, "tokens_used: %d\n", st.TokensUsed)
	fmt.Fprintf(out, "branch: %s\n", st.Branch)
	fmt.Fprintf(out, "pr_number: %d\n", st.PRNumber)
	fmt.Fprintf(out, "approved: %t\n", st.Approved)
	fmt.Fprintf(out, "halt_reason: %s\n", derefStr(st.HaltReason))
	fmt.Fprintf(out, "created_at: %s\n", st.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(out, "updated_at: %s\n", st.UpdatedAt.Format(time.RFC3339))

	if err := journal.Append(filepath.Join(packetDir, "log.jsonl"), journal.Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Slug:      st.Slug,
		Version:   st.Version,
		Event:     journal.EventStatus,
		Actor:     journal.ActorHuman,
	}); err != nil {
		return fmt.Errorf("status: %s", err)
	}
	return nil
}

func derefStr(s *string) string {
	if s == nil {
		return "(none)"
	}
	return *s
}
