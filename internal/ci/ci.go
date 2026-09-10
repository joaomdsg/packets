// Package ci implements the "ci" node of the loop: it polls the fabric's
// CI check to a conclusion and either sends the packet back to build,
// halts it, or dispatches it to approval/merge.
package ci

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
)

// Deps are the ci node's external-process and clock boundaries.
type Deps struct {
	GH    GH
	Git   Git
	Clock Clock
}

// logFunc appends one log.jsonl line for the packet currently in ci.
type logFunc func(event, reason string, tokens int, detail map[string]any) error

func newLogFunc(logPath string, st *state.State, clock Clock, actor string) logFunc {
	return func(event, reason string, tokens int, detail map[string]any) error {
		var reasonPtr *string
		if reason != "" {
			reasonPtr = &reason
		}
		node := state.NodeCI
		return journal.Append(logPath, journal.Entry{
			Timestamp: clock.Now().UTC().Format(time.RFC3339),
			Slug:      st.Slug,
			Version:   st.Version,
			Node:      &node,
			Event:     event,
			Reason:    reasonPtr,
			Tokens:    tokens,
			Actor:     actor,
			Detail:    detail,
		})
	}
}

// Run executes the "ci" case once the build node has handed off: poll the
// fabric's CI check to a conclusion, and either send the packet back to
// build (red, retries left), halt it (red with no retries left, or the
// poll deadline passing), or dispatch it to approval/merge (green).
// packetDir is the packet's directory; fabDir is the fabric's data
// directory, where registry.jsonl lives. st is mutated and persisted as
// the run progresses, so a crash mid-poll resumes correctly: a rerun
// starts a fresh poll loop from node=ci.
func Run(deps Deps, fab *fabric.Config, pkt *packet.Packet, st *state.State, packetDir, fabDir string) (*state.State, error) {
	statePath := filepath.Join(packetDir, "state.json")
	logPath := filepath.Join(packetDir, "log.jsonl")
	log := newLogFunc(logPath, st, deps.Clock, journal.ActorHarness)

	save := func() error {
		st.UpdatedAt = deps.Clock.Now().UTC()
		return state.Save(statePath, st)
	}
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

	check, outcome, err := poll(deps, fab, st.PRNumber, log)
	if err != nil {
		return nil, err
	}

	switch outcome {
	case pollTimeout:
		// §10's event vocabulary has no dedicated "ci timeout" event;
		// budget_halt is the nearest fit, the same reasoning build.go
		// applies to its own halt reasons with no matching event.
		return halt("ci_timeout", journal.EventBudgetHalt)
	case pollRed:
		return runRed(deps, fab, pkt, st, packetDir, log, save, halt, check)
	case pollGreen:
		return runGreen(deps, fab, pkt, st, packetDir, fabDir, log, save)
	default:
		return nil, fmt.Errorf("ci: unknown poll outcome")
	}
}
