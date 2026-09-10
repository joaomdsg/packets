// Package build implements the "build" node of §12's loop: it spawns the
// agent container for one attempt, inspects what it produced, and either
// halts or hands the packet off to the (Phase 5) ci node.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"gopkg.in/yaml.v3"
)

// Deps are the build node's external-process and clock boundaries.
type Deps struct {
	Git    Git
	GH     GH
	Docker Docker
	Clock  Clock
	// Getenv and Exists back claudeauth.Resolve for the container run
	// (docs/claude-code-facts.md); production wires os.Getenv and an
	// os.Stat-backed Exists.
	Getenv func(string) string
	Exists func(string) bool
}

// logFunc appends one log.jsonl line for the packet currently being built.
type logFunc func(event, reason string, tokens int, detail map[string]any) error

// Run executes §12's "build" case once: prepare the branch, spawn the
// container, inspect what it produced, and either halt or push/open a PR
// and hand off to the ci node. packetDir is the packet's directory
// (.../packets/<slug>); st is mutated and persisted as the run progresses
// (§0.5 write-before-act), so a crash mid-run leaves state.json at the
// last completed step and a rerun picks up from there.
func Run(deps Deps, fab *fabric.Config, pkt *packet.Packet, st *state.State, packetDir string) (*state.State, error) {
	statePath := filepath.Join(packetDir, "state.json")
	logPath := filepath.Join(packetDir, "log.jsonl")

	save := func() error {
		st.UpdatedAt = deps.Clock.Now().UTC()
		return state.Save(statePath, st)
	}
	log := logFunc(func(event, reason string, tokens int, detail map[string]any) error {
		var reasonPtr *string
		if reason != "" {
			reasonPtr = &reason
		}
		node := state.NodeBuild
		return journal.Append(logPath, journal.Entry{
			Timestamp: deps.Clock.Now().UTC().Format(time.RFC3339),
			Slug:      st.Slug,
			Version:   st.Version,
			Node:      &node,
			Event:     event,
			Reason:    reasonPtr,
			Tokens:    tokens,
			Actor:     journal.ActorHarness,
			Detail:    detail,
		})
	})
	halt := func(reason, event string) (*state.State, error) {
		next, err := st.Transition(state.Halted)
		if err != nil {
			return nil, err
		}
		*st = next
		st.HaltReason = &reason
		if err := save(); err != nil {
			return nil, err
		}
		if err := log(event, reason, 0, nil); err != nil {
			return nil, err
		}
		return st, nil
	}

	// Step 1: prepare the branch.
	if st.Branch == "" {
		st.Branch = fmt.Sprintf("packets/%s-v%d", st.Slug, st.Version)
		if err := save(); err != nil {
			return nil, err
		}
		if err := deps.Git.Fetch(fab.RepoPath); err != nil {
			return nil, fmt.Errorf("build: fetch: %s", err)
		}
		if err := deps.Git.CheckoutNewBranch(fab.RepoPath, st.Branch, "origin/"+fab.DefaultBranch); err != nil {
			return nil, fmt.Errorf("build: checkout -B %s: %s", st.Branch, err)
		}
	} else if err := deps.Git.Checkout(fab.RepoPath, st.Branch); err != nil {
		return nil, fmt.Errorf("build: checkout %s: %s", st.Branch, err)
	}

	// The attempt counter is bumped before rendering CLAUDE.md, since
	// runs/<attempt>/ is named after the attempt about to run, not the one
	// that just finished.
	st.Attempt++
	if err := save(); err != nil {
		return nil, err
	}
	if err := log(journal.EventContainerStart, "", 0, nil); err != nil {
		return nil, err
	}

	runDir := filepath.Join(packetDir, "runs", fmt.Sprintf("attempt-%d", st.Attempt))
	signalsDir := filepath.Join(runDir, "signals")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, fmt.Errorf("build: create %s: %s", runDir, err)
	}

	// Step 2: render CLAUDE.md, splicing in the previous failure/halt.
	failureMD, _ := os.ReadFile(filepath.Join(packetDir, "failure.md"))
	haltMD, _ := os.ReadFile(filepath.Join(packetDir, "halt.md"))
	claudeMD, err := RenderClaudeMD(pkt, st, string(failureMD), string(haltMD))
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(runDir, "CLAUDE.md"), []byte(claudeMD), 0o600); err != nil {
		return nil, fmt.Errorf("build: write CLAUDE.md: %s", err)
	}
	if err := seedSignals(signalsDir, pkt, st); err != nil {
		return nil, err
	}

	// Step 3, 4, 4b: spawn the container, then check for an agent halt or
	// a budget timeout.
	if st2, err := runContainer(deps, fab, pkt, st, packetDir, runDir, signalsDir, log, save, halt); err != nil || st2 != nil {
		return st2, err
	}

	// Step 5: log local predicate results (whatever hooks/stop.sh last
	// wrote; a run that halted or timed out before it ran leaves none).
	results, err := readPredicateResults(signalsDir)
	if err != nil {
		return nil, err
	}
	for _, r := range results {
		if err := log(journal.EventLocalPred, "", 0, map[string]any{
			"cmd": r.Cmd, "exit": r.Exit, "kind": r.Kind,
		}); err != nil {
			return nil, err
		}
	}

	// Step 6: authoritative host-side smell re-check.
	reason, triggered, err := hostSmellCheck(deps.Git, fab.RepoPath, fab.DefaultBranch, fab.SmellTriggers)
	if err != nil {
		return nil, err
	}
	if triggered {
		return halt(reason, journal.EventSmellHalt)
	}

	// Step 7: commit.
	commitMsg := fmt.Sprintf("packets: %s v%d attempt %d", st.Slug, st.Version, st.Attempt)
	committed, err := deps.Git.CommitAll(fab.RepoPath, commitMsg)
	if err != nil {
		return nil, fmt.Errorf("build: commit: %s", err)
	}
	if !committed {
		// No dedicated log event covers this halt reason (§10's vocabulary
		// enumerates agent/smell/budget/conflict but not no_changes);
		// smell_halt is the nearest fit — a no-op attempt is exactly the
		// kind of signal the smell checks exist to catch.
		return halt("no_changes", journal.EventSmellHalt)
	}

	// Step 8: rebase onto the default branch.
	if err := deps.Git.Fetch(fab.RepoPath); err != nil {
		return nil, fmt.Errorf("build: fetch: %s", err)
	}
	conflict, err := deps.Git.RebaseOnto(fab.RepoPath, "origin/"+fab.DefaultBranch)
	if err != nil {
		return nil, fmt.Errorf("build: rebase: %s", err)
	}
	if conflict {
		if err := deps.Git.RebaseAbort(fab.RepoPath); err != nil {
			return nil, fmt.Errorf("build: rebase --abort: %s", err)
		}
		return halt("conflict", journal.EventConflictHalt)
	}
	if err := log(journal.EventRebase, "", 0, nil); err != nil {
		return nil, err
	}

	// Step 9: push, open a PR if none exists yet, hand off to ci.
	if err := deps.Git.PushForceWithLease(fab.RepoPath, st.Branch); err != nil {
		return nil, fmt.Errorf("build: push: %s", err)
	}
	if err := log(journal.EventPush, "", 0, nil); err != nil {
		return nil, err
	}
	if st.PRNumber == 0 {
		body, err := prBody(pkt)
		if err != nil {
			return nil, err
		}
		prNumber, err := deps.GH.CreatePR(fab.RepoPath, pkt.Goal, body, st.Branch, fab.DefaultBranch)
		if err != nil {
			return nil, fmt.Errorf("build: gh pr create: %s", err)
		}
		st.PRNumber = prNumber
		if err := save(); err != nil {
			return nil, err
		}
		if err := log(journal.EventPROpen, "", 0, nil); err != nil {
			return nil, err
		}
	}

	next, err := st.Transition(state.CI)
	if err != nil {
		return nil, err
	}
	*st = next
	node := state.NodeCI
	st.Node = &node
	if err := save(); err != nil {
		return nil, err
	}
	if err := log(journal.EventEnterNode, "", 0, nil); err != nil {
		return nil, err
	}
	return st, nil
}

// prBody renders the PR body: the packet.yaml contents in a YAML fence,
// per §12 step 9.
func prBody(pkt *packet.Packet) (string, error) {
	data, err := yaml.Marshal(pkt)
	if err != nil {
		return "", fmt.Errorf("build: marshal packet.yaml for PR body: %s", err)
	}
	return fmt.Sprintf("```yaml\n%s```\n", data), nil
}
