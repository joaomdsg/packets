package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/gate"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/slug"
	"github.com/spf13/cobra"
)

func newEmitCommand(llm gate.LLM) *cobra.Command {
	return &cobra.Command{
		Use:   "emit <file>",
		Short: "Run the emit gate on a packet.yaml and create the packet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEmit(cmd, args[0], llm)
		},
	}
}

func runEmit(cmd *cobra.Command, path string, llm gate.LLM) error {
	p, err := packet.Load(path)
	if err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), err)
		return fmt.Errorf("emit: %s did not parse", path)
	}

	// Stage A is purely mechanical and must not require a fabric: a user
	// authoring their first packet before `packets init` still needs their
	// rule violations reported. Fabric resolution is deferred until we know
	// whether it's actually needed (caused_by check, then Stage B/create).
	fab, fabErr := resolveFabric(cmd, "emit")
	packetsDir := ""
	if fabErr == nil {
		packetsDir = fab.PacketsDir
	}

	failures := packet.Validate(p, packetsDir)
	if fabErr != nil && p.CausedBy != nil && *p.CausedBy != "" {
		failures = append(failures, fmt.Sprintf(
			"caused_by: cannot verify packet %q exists: %s", *p.CausedBy, fabErr))
	}

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintln(cmd.OutOrStdout(), f)
		}
		// No packet dir exists yet at Stage A, so this rejection has
		// nowhere else to live; see fabriclog.go. Only possible once a
		// fabric resolved, since the fabric log lives under its data dir.
		if fabErr == nil {
			if err := logFabricEvent(fab, journal.EventEmitReject, map[string]any{"failures": failures}); err != nil {
				return fmt.Errorf("emit: %s", err)
			}
		}
		return fmt.Errorf("emit: %s failed validation", path)
	}

	// Stage A passed; everything past here (slug/dir creation, repo
	// listing for Stage B) genuinely needs a fabric.
	if fabErr != nil {
		return fabErr
	}

	// Slug depends only on the goal, so it's resolved before Stage B and
	// its collision (if any) is the first thing logged once the packet is
	// actually created.
	finalSlug, collided := slug.Deconflict(slug.Packet(p.Goal), time.Now(), func(candidate string) bool {
		_, err := os.Stat(filepath.Join(fab.PacketsDir, candidate))
		return err == nil
	})

	rawYAML, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("emit: read %s: %s", path, err)
	}
	repoFiles, err := repoListing(fab.RepoPath)
	if err != nil {
		return fmt.Errorf("emit: %s", err)
	}

	outcome := runGate(llm, string(rawYAML), p.Terminal, repoFiles, cmd.InOrStdin(), cmd.OutOrStdout())
	if !outcome.proceed {
		return fmt.Errorf("emit: not emitted, warnings not overridden")
	}
	p.Terminal = outcome.terminal

	var collisionEntry *journal.Entry
	if collided {
		collisionEntry = &journal.Entry{Event: journal.EventSlugCollision, Actor: journal.ActorHuman}
	}

	if err := createPacket(fab.PacketsDir, finalSlug, p, outcome, collisionEntry); err != nil {
		return fmt.Errorf("emit: %s", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), finalSlug)
	return nil
}
