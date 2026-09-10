package ci

import (
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
)

const humanApproval = "human_approval"

// runGreen handles a green CI check: if the terminal requires human
// approval and it hasn't been given, the packet waits; otherwise it
// merges immediately (if the fabric auto-merges) or waits for approval
// anyway.
func runGreen(
	deps Deps, fab *fabric.Config, pkt *packet.Packet, st *state.State, packetDir, fabDir string,
	log logFunc, save func() error,
) (*state.State, error) {
	if err := log(journal.EventCIPred, "", 0, map[string]any{
		"cmd": fab.CI.WorkflowName, "exit": 0, "kind": "ci",
	}); err != nil {
		return nil, err
	}

	if containsHumanApproval(pkt.Terminal) && !st.Approved {
		return toAwaitingApproval(st, save)
	}
	if fab.Merge.Auto {
		return Merge(deps, fab, pkt, st, packetDir, fabDir, journal.ActorHarness)
	}
	return toAwaitingApproval(st, save)
}

func toAwaitingApproval(st *state.State, save func() error) (*state.State, error) {
	next, err := st.Transition(state.AwaitingApproval)
	if err != nil {
		return nil, err
	}
	*st = next
	if err := save(); err != nil {
		return nil, err
	}
	return st, nil
}

func containsHumanApproval(terminal []string) bool {
	for _, t := range terminal {
		if t == humanApproval {
			return true
		}
	}
	return false
}
