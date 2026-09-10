package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/spf13/cobra"
)

func newResumeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <slug>",
		Short: "Resume a halted packet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResume(cmd, args[0])
		},
	}
}

func runResume(cmd *cobra.Command, slug string) error {
	fab, err := resolveFabric(cmd)
	if err != nil {
		return err
	}
	packetDir := filepath.Join(fab.PacketsDir, slug)
	statePath := filepath.Join(packetDir, "state.json")

	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("resume: %s", err)
	}
	if st.State != state.Halted {
		return fmt.Errorf("resume: %s is %s, not halted", slug, st.State)
	}

	if err := resumeBuilding(statePath, st, journal.EventResume); err != nil {
		return fmt.Errorf("resume: %s", err)
	}
	return nil
}

// resumeBuilding applies the "resume" half of §5's halted --resume/extend-->
// building transition shared by `packets resume` and `packets extend`: set
// state=building, clear halt_reason, persist, and log event.
func resumeBuilding(statePath string, st *state.State, event string) error {
	next, err := st.Transition(state.Building)
	if err != nil {
		return err
	}
	*st = next
	st.HaltReason = nil
	st.UpdatedAt = time.Now().UTC()
	if err := state.Save(statePath, st); err != nil {
		return err
	}
	return journal.Append(filepath.Join(filepath.Dir(statePath), "log.jsonl"), journal.Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Slug:      st.Slug,
		Version:   st.Version,
		Event:     event,
		Actor:     journal.ActorHuman,
	})
}
