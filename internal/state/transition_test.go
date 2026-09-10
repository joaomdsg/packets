package state_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestTransition_allowsEveryLegalMove(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"emit", state.Draft, state.Emitted},
		{"run", state.Emitted, state.Building},
		{"local green", state.Building, state.CI},
		{"building halt", state.Building, state.Halted},
		{"ci red retry", state.CI, state.Building},
		{"ci red halt", state.CI, state.Halted},
		{"ci green needs approval", state.CI, state.AwaitingApproval},
		{"ci green terminates", state.CI, state.Terminated},
		{"approve", state.AwaitingApproval, state.Terminated},
		{"resume or extend", state.Halted, state.Building},
		{"amend from halted", state.Halted, state.Emitted},
		{"amend from awaiting approval", state.AwaitingApproval, state.Emitted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := state.State{State: tt.from}

			got, err := s.Transition(tt.to)

			assert.NoError(t, err)
			assert.Equal(t, tt.to, got.State)
		})
	}
}

func TestTransition_allowsKillFromAnyNonFinalState(t *testing.T) {
	nonFinal := []string{
		state.Draft, state.Emitted, state.Building, state.CI,
		state.Halted, state.AwaitingApproval,
	}
	for _, from := range nonFinal {
		t.Run(from, func(t *testing.T) {
			t.Parallel()
			s := state.State{State: from}

			got, err := s.Transition(state.Killed)

			assert.NoError(t, err)
			assert.Equal(t, state.Killed, got.State)
		})
	}
}

func TestTransition_rejectsIllegalMoves(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"cannot skip straight to building", state.Draft, state.Building},
		{"cannot go backwards from ci to emitted", state.CI, state.Emitted},
		{"cannot leave terminated", state.Terminated, state.Building},
		{"cannot leave killed", state.Killed, state.Building},
		{"terminated cannot even be killed again", state.Terminated, state.Killed},
		{"halted cannot jump straight to ci", state.Halted, state.CI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := state.State{State: tt.from}

			_, err := s.Transition(tt.to)

			assert.Error(t, err)
		})
	}
}
