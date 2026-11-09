package task

import apperrors "todoapp/internal/errors"

// Status of a task. New states can be added here later — all transition
// rules are centralized in validTransitions below.
type Status string

const (
	StatusTodo     Status = "todo"
	StatusProgress Status = "progress"
	StatusDone     Status = "done"
	StatusClose    Status = "close" // close == soft delete
)

// validTransitions defines the task state machine.
// A task can never jump straight from todo to done; close (soft delete)
// is allowed from any active state.
var validTransitions = map[Status][]Status{
	StatusTodo:     {StatusProgress, StatusClose},
	StatusProgress: {StatusDone, StatusClose},
	StatusDone:     {StatusClose},
	StatusClose:    {}, // dead state
}

// CanTransition reports whether moving a task from -> to is allowed.
func CanTransition(from, to Status) bool {
	for _, next := range validTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// ValidateTransition enforces the state machine (same -> same is a no-op).
func ValidateTransition(from, to Status) error {
	if from == to {
		return nil
	}
	if !CanTransition(from, to) {
		return apperrors.InvalidTransition(string(from), string(to))
	}
	return nil
}

// ValidStatus reports whether s is one of the const statuses.
func ValidStatus(s Status) bool {
	switch s {
	case StatusTodo, StatusProgress, StatusDone, StatusClose:
		return true
	}
	return false
}

// IsActive reports whether the task is visible (not soft deleted).
func (s Status) IsActive() bool { return s != StatusClose }
