package cli

import (
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/joaomdsg/packets/internal/xdgpath"
)

// createPacket writes the packet's on-disk artifacts: versions/v1.yaml,
// packet.yaml, state.json, and log.jsonl seeded with whatever the gate
// buffered plus a trailing "emit" event. Nothing is written before this
// point — an aborted emit (override declined) leaves no trace.
func createPacket(packetsDir, slug string, p *packet.Packet, outcome gateOutcome, collisionEntry *journal.Entry) error {
	dir := filepath.Join(packetsDir, slug)
	versionsDir := filepath.Join(dir, "versions")
	if err := xdgpath.EnsureDir(versionsDir); err != nil {
		return err
	}

	if err := packet.Save(filepath.Join(versionsDir, "v1.yaml"), p); err != nil {
		return err
	}
	if err := packet.Save(filepath.Join(dir, "packet.yaml"), p); err != nil {
		return err
	}

	now := time.Now().UTC()
	s := &state.State{
		Slug:                   slug,
		Version:                1,
		State:                  state.Emitted,
		GateSuggestedTerminals: outcome.gateSuggestedTerminals,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := state.Save(filepath.Join(dir, "state.json"), s); err != nil {
		return err
	}

	entries := outcome.entries
	if collisionEntry != nil {
		entries = append([]journal.Entry{*collisionEntry}, entries...)
	}
	entries = append(entries, journal.Entry{Event: journal.EventEmit, Actor: journal.ActorHuman})

	logPath := filepath.Join(dir, "log.jsonl")
	for _, e := range entries {
		e.Timestamp = now.Format(time.RFC3339)
		e.Slug = slug
		e.Version = 1
		e.Actor = journal.ActorHuman
		if err := journal.Append(logPath, e); err != nil {
			return err
		}
	}
	return nil
}
