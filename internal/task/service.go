// Package task holds the task domain: model, statuses, persistence and
// business rules.
package task

import (
	"context"

	apperrors "todoapp/internal/errors"
	"todoapp/internal/events"
)

// Service holds business rules: status transitions, soft delete via
// close, and async event publishing through a channel.
type Service struct {
	repo        Repository
	events      chan<- events.Event
	defaultLim  int
	maxLim      int
}

// Options tunes service behaviour. Zero values fall back to the defaults
// below; production passes the values from config.
type Options struct {
	DefaultLimit int // page size when the client sends none (default 20)
	MaxLimit     int // hard cap on page size (default 100)
}

func withDefaults(o Options) Options {
	if o.DefaultLimit < 1 {
		o.DefaultLimit = 20
	}
	if o.MaxLimit < 1 {
		o.MaxLimit = 100
	}
	if o.DefaultLimit > o.MaxLimit {
		o.DefaultLimit = o.MaxLimit
	}
	return o
}

func NewService(repo Repository, events chan<- events.Event, opts Options) *Service {
	opts = withDefaults(opts)
	return &Service{repo: repo, events: events, defaultLim: opts.DefaultLimit, maxLim: opts.MaxLimit}
}

func (s *Service) Create(ctx context.Context, userID int64, title, description string) (*Task, error) {
	t := &Task{UserID: userID, Title: title, Description: description, Status: StatusTodo}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	s.publish(events.Event{TaskID: t.ID, Action: "created", Status: string(t.Status)})
	return t, nil
}

// Get returns a task by id. It does not enforce ownership — callers should
// ensure the task belongs to the current user (or is an admin).
func (s *Service) Get(ctx context.Context, id int64) (*Task, error) {
	return s.repo.GetByID(ctx, id)
}

// ListForUser lists tasks owned by userID with the given filter.
func (s *Service) ListForUser(ctx context.Context, userID int64, f TaskFilter) ([]Task, int, error) {
	if userID == 0 {
		return nil, 0, apperrors.ErrNotFound
	}
	f.UserID = userID
	return s.List(ctx, f)
}

func (s *Service) List(ctx context.Context, f TaskFilter) ([]Task, int, error) {
	if f.Status != "" && !ValidStatus(f.Status) {
		return nil, 0, apperrors.InvalidStatus(string(f.Status))
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > s.maxLim {
		f.Limit = s.defaultLim
	}
	return s.repo.List(ctx, f)
}

// Update applies partial updates. A status change must follow the state
// machine: todo -> progress -> done -> close.
func (s *Service) Update(ctx context.Context, id int64, title, description *string, status *Status) (*Task, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if status != nil {
		if !ValidStatus(*status) {
			return nil, apperrors.InvalidStatus(string(*status))
		}
		if err := ValidateTransition(t.Status, *status); err != nil {
			return nil, err
		}
		t.Status = *status
	}
	if title != nil {
		t.Title = *title
	}
	if description != nil {
		t.Description = *description
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	s.publish(events.Event{TaskID: t.ID, Action: "updated", Status: string(t.Status)})
	return t, nil
}

// Close soft-deletes the task. Allowed from any active state.
func (s *Service) Close(ctx context.Context, id int64) (*Task, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Status = StatusClose
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	s.publish(events.Event{TaskID: t.ID, Action: "closed", Status: string(t.Status)})
	return t, nil
}

// publish is non-blocking: if no worker is draining the channel yet we
// drop the event rather than stalling the request.
func (s *Service) publish(e events.Event) {
	select {
	case s.events <- e:
	default:
	}
}
