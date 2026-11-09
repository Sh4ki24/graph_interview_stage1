package task

import (
	"context"
)

// Repository is the persistence contract. Unit tests use the in-package
// Mock; production uses Postgres.
type Repository interface {
	Init(ctx context.Context) error
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	Update(ctx context.Context, t *Task) error
	// List returns visible tasks matching the filter plus the total count
	// (for pagination). Closed tasks are excluded. When f.UserID is set,
	// only that user's tasks are returned.
	List(ctx context.Context, f TaskFilter) ([]Task, int, error)
}
