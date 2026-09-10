package state

import "fmt"

var finalStates = map[string]bool{
	Terminated: true,
	Killed:     true,
}

// allowedTransitions lists every (from, to) pair in the state machine table,
// excluding the "any non-terminal --kill--> killed" rule, which Transition
// handles separately since it applies uniformly to every non-final state.
var allowedTransitions = map[string]map[string]bool{
	Draft:            {Emitted: true},
	Emitted:          {Building: true},
	Building:         {CI: true, Halted: true},
	CI:               {Building: true, Halted: true, AwaitingApproval: true, Terminated: true},
	Halted:           {Building: true, Emitted: true},
	AwaitingApproval: {Terminated: true, Emitted: true},
}

// Transition returns a copy of s moved to next, or an error if the move is
// not a legal state machine transition. terminated and killed are final:
// no transition out of them is legal.
func (s State) Transition(next string) (State, error) {
	if finalStates[s.State] {
		return s, fmt.Errorf("state: %s is final, cannot transition to %s", s.State, next)
	}
	if next == Killed || allowedTransitions[s.State][next] {
		s.State = next
		return s, nil
	}
	return s, fmt.Errorf("state: illegal transition from %s to %s", s.State, next)
}
