// Package harness is the live-harness supervisor: the stateful turn-reducer
// that drives a Claude Code harness's stream-json output into reviewable
// revisions. It reads the harness event stream, surfaces each turn's live
// activity (via internal/translate), and at every turn boundary settles the
// working tree into a revision (via internal/orchestrator), threading the
// minted SHA forward as the next turn's base.
//
// The harness mints nothing itself — only the host's settle step produces a
// revision (the economy firewall). Spawning the real `claude` subprocess is a
// separate slice; this reducer reads any io.Reader (a subprocess stdout, a
// fixture stream), which is the harness's true I/O boundary.
package harness

import (
	"bufio"
	"context"
	"io"

	"github.com/joaomdsg/packets/internal/orchestrator"
	"github.com/joaomdsg/packets/internal/translate"
)

// Turn is one completed harness turn: the live activity the user watched
// (Events) and the settled outcome (a minted revision, a no-edit turn, or a
// secret-blocked turn — see orchestrator.TurnOutcome).
type Turn struct {
	Events  []translate.UIEvent
	Outcome orchestrator.TurnOutcome
}

// Supervisor reduces one harness stream into settled turns against a repo.
type Supervisor struct {
	repoDir string
	baseRev string
}

// New constructs a Supervisor that settles turns in repoDir, diffing the first
// turn against baseRev.
func New(repoDir, baseRev string) *Supervisor {
	return &Supervisor{repoDir: repoDir, baseRev: baseRev}
}

// Run reads the harness stream from r to completion, surfacing each turn's
// activity and settling a revision at every turn boundary. The minted SHA of a
// turn becomes the base for the next, so each turn's diff shows only what that
// turn changed. An incomplete trailing turn (no turn-end before EOF) settles
// nothing. A malformed stream line is an error.
func (s *Supervisor) Run(ctx context.Context, r io.Reader) ([]Turn, error) {
	var turns []Turn
	var pending []translate.UIEvent

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		events, err := translate.Translate(line)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if e.Type != "turn.ended" {
				pending = append(pending, e)
				continue
			}
			out, err := orchestrator.SettleTurn(ctx, s.repoDir, s.baseRev, "harness turn")
			if err != nil {
				return nil, err
			}
			if out.Minted {
				s.baseRev = out.SHA
			}
			turns = append(turns, Turn{Events: pending, Outcome: out})
			pending = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return turns, nil
}
