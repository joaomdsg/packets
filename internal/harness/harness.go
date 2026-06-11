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
	repoDir    string
	baseRev    string
	onActivity func([]translate.UIEvent)
}

// Option configures a Supervisor at construction.
type Option func(*Supervisor)

// WithActivity registers a callback invoked with each stream line's activity
// events the moment they are read — BEFORE the turn settles — so a live agent's
// thinking/editing/tool beats can be surfaced as they stream, not only in the
// batch of turns Run returns at completion.
func WithActivity(fn func([]translate.UIEvent)) Option {
	return func(s *Supervisor) { s.onActivity = fn }
}

// New constructs a Supervisor that settles turns in repoDir, diffing the first
// turn against baseRev.
func New(repoDir, baseRev string, opts ...Option) *Supervisor {
	s := &Supervisor{repoDir: repoDir, baseRev: baseRev}
	for _, o := range opts {
		o(s)
	}
	return s
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
		var activity []translate.UIEvent
		for _, e := range events {
			if e.Type != "turn.ended" {
				pending = append(pending, e)
				activity = append(activity, e)
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
		// Stream this line's activity live, the moment it is read — before the turn
		// settles — so the surface can show the agent working in real time.
		if s.onActivity != nil && len(activity) > 0 {
			s.onActivity(activity)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return turns, nil
}
