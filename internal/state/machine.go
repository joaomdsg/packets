package state

import (
	"fmt"

	"github.com/joaomdsg/agntpr/internal/db"
)

type Event string

const (
	EventStartPlanning         Event = "start_planning"
	EventSkipPlanning          Event = "skip_planning"
	EventPlanComplete          Event = "plan_complete"
	EventPlanApproved          Event = "plan_approved"
	EventPlanRejected          Event = "plan_rejected"
	EventImplementationComplete Event = "implementation_complete"
	EventPROpened              Event = "pr_opened"
	EventPRMerged              Event = "pr_merged"
	EventPRClosed              Event = "pr_closed"
	EventReviewComment         Event = "review_comment"
)

type transition struct {
	from  db.IssueState
	event Event
	to    db.IssueState
}

var transitions = []transition{
	{db.StateNew, EventStartPlanning, db.StatePlanning},
	{db.StateNew, EventSkipPlanning, db.StateImplementing},
	{db.StatePlanning, EventPlanComplete, db.StatePlanReview},
	{db.StatePlanReview, EventPlanApproved, db.StateImplementing},
	{db.StatePlanReview, EventPlanRejected, db.StatePlanning},
	{db.StateImplementing, EventImplementationComplete, db.StatePRCreated},
	{db.StatePRCreated, EventPROpened, db.StatePRReview},
	{db.StatePRCreated, EventPRMerged, db.StateDone},
	{db.StatePRCreated, EventPRClosed, db.StateRejected},
	{db.StatePRReview, EventPRMerged, db.StateDone},
	{db.StatePRReview, EventPRClosed, db.StateRejected},
	{db.StatePRReview, EventReviewComment, db.StatePRReview},
}

type Machine struct {
	current db.IssueState
}

func NewMachine(initial db.IssueState) *Machine {
	return &Machine{current: initial}
}

func (m *Machine) Current() db.IssueState {
	return m.current
}

func (m *Machine) Transition(event Event) (db.IssueState, error) {
	for _, t := range transitions {
		if t.from == m.current && t.event == event {
			m.current = t.to
			return t.to, nil
		}
	}
	return m.current, fmt.Errorf(
		"invalid transition: %s + %s", m.current, event)
}

func (m *Machine) CanTransition(event Event) bool {
	for _, t := range transitions {
		if t.from == m.current && t.event == event {
			return true
		}
	}
	return false
}

func (m *Machine) IsTerminal() bool {
	return m.current == db.StateDone ||
		m.current == db.StateRejected ||
		m.current == db.StateErrored
}

func (m *Machine) AvailableEvents() []Event {
	var events []Event
	for _, t := range transitions {
		if t.from == m.current {
			events = append(events, t.event)
		}
	}
	return events
}
