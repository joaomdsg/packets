package cli

import (
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/xdgpath"
)

// fabricLogPath is the fabric-level log.jsonl, alongside registry.jsonl.
// It exists for events that have no packet dir to live in yet: an
// emit_reject happens during Stage A, before a packet dir (and therefore
// any per-packet log.jsonl) is created. See docs/claude-code-facts.md's
// "Additions to the §2 layout".
func fabricLogPath(fab resolvedFabric) string {
	return filepath.Join(filepath.Dir(fab.PacketsDir), "log.jsonl")
}

// logFabricEvent appends one line to the fabric-level log. Every seam
// command's log line uses actor "human"; nothing at this level runs as
// the harness or an agent.
func logFabricEvent(fab resolvedFabric, event string, detail map[string]any) error {
	if err := xdgpath.EnsureDir(filepath.Dir(fab.PacketsDir)); err != nil {
		return err
	}
	return journal.Append(fabricLogPath(fab), journal.Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Event:     event,
		Actor:     journal.ActorHuman,
		Detail:    detail,
	})
}
