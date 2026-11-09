// Package errors holds every sentinel error of the application in one place.
package errors

import (
	stderrors "errors"
	"fmt"
)

// Re-exported std helpers so callers can use this package alone.
var (
	Is  = stderrors.Is
	As  = stderrors.As
	New = stderrors.New
)

// Task errors.
var (
	ErrNotFound         = New("task not found")
	ErrInvalidStatus    = New("invalid status")
	ErrInvalidTransition = New("invalid status transition")
)

// Auth errors.
var (
	ErrUnauthorized = New("missing or invalid role")
	ErrForbidden    = New("role not permitted")
)

// InvalidTransition wraps ErrInvalidTransition with from/to context.
func InvalidTransition(from, to string) error {
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
}

// InvalidStatus wraps ErrInvalidStatus with the bad value.
func InvalidStatus(s string) error {
	return fmt.Errorf("%w: %s", ErrInvalidStatus, s)
}
