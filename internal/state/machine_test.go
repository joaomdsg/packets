package state_test

import (
	"testing"

	"github.com/joaomdsg/agntpr/internal/db"
	"github.com/joaomdsg/agntpr/internal/state"
)

func TestTransitions_ValidTransitions(t *testing.T) {
	validTransitions := []struct {
		from   db.IssueState
		event  state.Event
		to     db.IssueState
	}{
		{db.StateNew, state.EventStartPlanning, db.StatePlanning},
		{db.StatePlanning, state.EventPlanComplete, db.StatePlanReview},
		{db.StatePlanReview, state.EventPlanApproved, db.StateImplementing},
		{db.StatePlanReview, state.EventPlanRejected, db.StatePlanning},
		{db.StateImplementing, state.EventImplementationComplete, db.StatePRCreated},
		{db.StatePRCreated, state.EventPROpened, db.StatePRReview},
		{db.StatePRReview, state.EventPRMerged, db.StateDone},
		{db.StatePRReview, state.EventPRClosed, db.StateRejected},
		{db.StatePRReview, state.EventReviewComment, db.StatePRReview},
	}

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"_"+string(tt.event), func(t *testing.T) {
			machine := state.NewMachine(tt.from)
			newState, err := machine.Transition(tt.event)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if newState != tt.to {
				t.Errorf("expected state %s, got %s", tt.to, newState)
			}
		})
	}
}

func TestTransitions_InvalidTransitions(t *testing.T) {
	invalidTransitions := []struct {
		from  db.IssueState
		event state.Event
	}{
		{db.StateNew, state.EventPlanApproved},
		{db.StatePlanning, state.EventPRMerged},
		{db.StateDone, state.EventStartPlanning},
		{db.StateRejected, state.EventStartPlanning},
	}

	for _, tt := range invalidTransitions {
		t.Run(string(tt.from)+"_"+string(tt.event), func(t *testing.T) {
			machine := state.NewMachine(tt.from)
			_, err := machine.Transition(tt.event)
			if err == nil {
				t.Error("expected error for invalid transition")
			}
		})
	}
}

func TestMachine_CurrentState(t *testing.T) {
	machine := state.NewMachine(db.StateNew)
	if machine.Current() != db.StateNew {
		t.Errorf("expected state new, got %s", machine.Current())
	}

	if _, err := machine.Transition(state.EventStartPlanning); err != nil {
		t.Fatalf("failed to transition: %v", err)
	}
	if machine.Current() != db.StatePlanning {
		t.Errorf("expected state planning, got %s", machine.Current())
	}
}

func TestMachine_CanTransition(t *testing.T) {
	machine := state.NewMachine(db.StateNew)

	if !machine.CanTransition(state.EventStartPlanning) {
		t.Error("expected to be able to transition with EventStartPlanning")
	}

	if machine.CanTransition(state.EventPRMerged) {
		t.Error("should not be able to transition with EventPRMerged from new")
	}
}

func TestMachine_IsTerminal(t *testing.T) {
	t.Run("done is terminal", func(t *testing.T) {
		machine := state.NewMachine(db.StateDone)
		if !machine.IsTerminal() {
			t.Error("done should be terminal")
		}
	})

	t.Run("rejected is terminal", func(t *testing.T) {
		machine := state.NewMachine(db.StateRejected)
		if !machine.IsTerminal() {
			t.Error("rejected should be terminal")
		}
	})

	t.Run("new is not terminal", func(t *testing.T) {
		machine := state.NewMachine(db.StateNew)
		if machine.IsTerminal() {
			t.Error("new should not be terminal")
		}
	})
}

func TestMachine_SkipPlanning(t *testing.T) {
	machine := state.NewMachine(db.StateNew)

	newState, err := machine.Transition(state.EventSkipPlanning)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if newState != db.StateImplementing {
		t.Errorf("expected implementing, got %s", newState)
	}
}
