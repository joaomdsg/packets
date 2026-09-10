package ci

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/registry"
	"github.com/joaomdsg/packets/internal/state"
)

// Merge performs the merge block: gh pr merge, accretion registry
// writes, and the terminal state transition. It is shared by the ci
// node's auto-merge path and `packets approve`, which differ only in
// actor ("harness" vs "human").
func Merge(deps Deps, fab *fabric.Config, pkt *packet.Packet, st *state.State, packetDir, fabDir, actor string) (*state.State, error) {
	statePath := filepath.Join(packetDir, "state.json")
	logPath := filepath.Join(packetDir, "log.jsonl")
	log := newLogFunc(logPath, st, deps.Clock, actor)
	save := func() error {
		st.UpdatedAt = deps.Clock.Now().UTC()
		return state.Save(statePath, st)
	}

	if err := deps.GH.MergePR(fab.RepoPath, st.PRNumber, fab.Merge.Strategy); err != nil {
		return nil, fmt.Errorf("ci: gh pr merge: %s", err)
	}
	if err := log(journal.EventMerge, "", 0, nil); err != nil {
		return nil, err
	}

	writeRegistryEntries(deps, fab, st, fabDir, log)

	next, err := st.Transition(state.Terminated)
	if err != nil {
		return nil, err
	}
	*st = next
	st.Node = nil
	if err := save(); err != nil {
		return nil, err
	}
	if err := log(journal.EventTerminate, "", 0, nil); err != nil {
		return nil, err
	}
	return st, nil
}

// writeRegistryEntries records this merge's accretions. Detection failure
// must never block the merge: any error is logged as
// registry_detect_failed and the merge proceeds.
func writeRegistryEntries(deps Deps, fab *fabric.Config, st *state.State, fabDir string, log logFunc) {
	registryPath := filepath.Join(fabDir, "registry.jsonl")
	commit, err := deps.Git.CurrentCommit(fab.RepoPath)
	if err != nil {
		logRegistryDetectFailed(log, err)
		return
	}
	ts := deps.Clock.Now().UTC().Format(time.RFC3339)

	gateSuggested := registry.DetectGateSuggested(fab.RepoPath, st.GateSuggestedTerminals)
	for _, path := range gateSuggested {
		appendAccretion(log, registryPath, ts, fab.Slug, st, path, registry.CauseGateSuggested, commit)
	}

	ciFailurePaths, err := registry.DetectCIFailure(deps.Git, fab.RepoPath, "origin/"+fab.DefaultBranch, st.Branch, gateSuggested)
	if err != nil {
		logRegistryDetectFailed(log, err)
		return
	}
	for _, path := range ciFailurePaths {
		appendAccretion(log, registryPath, ts, fab.Slug, st, path, registry.CauseCIFailure, commit)
	}
}

func appendAccretion(log logFunc, registryPath, ts, fabricSlug string, st *state.State, path, cause, commit string) {
	entry := registry.Entry{
		Timestamp: ts, Fabric: fabricSlug, Packet: st.Slug, Version: st.Version,
		Path: path, Cause: cause, Commit: commit,
	}
	if err := registry.Append(registryPath, entry); err != nil {
		logRegistryDetectFailed(log, err)
		return
	}
	_ = log(journal.EventAccrete, "", 0, map[string]any{"path": path, "cause": cause})
}

func logRegistryDetectFailed(log logFunc, err error) {
	reason := err.Error()
	_ = log(journal.EventRegistryDetectFailed, reason, 0, nil)
}
