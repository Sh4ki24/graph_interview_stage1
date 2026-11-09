package task

import (
	"context"
	"sync"

	apperrors "todoapp/internal/errors"
)

// MockRepository is an in-memory Repository used by unit tests.
type MockRepository struct {
	mu     sync.Mutex
	items  map[int64]*Task
	nextID int64
	Err    error // when set, every call fails with this error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{items: make(map[int64]*Task), nextID: 1}
}

func (m *MockRepository) Init(_ context.Context) error { return m.Err }

func (m *MockRepository) Create(_ context.Context, t *Task) error {
	if m.Err != nil {
		return m.Err
	}
	if t.Status == "" {
		t.Status = StatusTodo
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t.ID = m.nextID
	m.nextID++
	cp := *t
	m.items[t.ID] = &cp
	return nil
}

func (m *MockRepository) GetByID(_ context.Context, id int64) (*Task, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.items[id]
	if !ok || !t.Status.IsActive() {
		return nil, apperrors.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (m *MockRepository) Update(_ context.Context, t *Task) error {
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.items[t.ID]
	if !ok || !cur.Status.IsActive() {
		return apperrors.ErrNotFound
	}
	cp := *t
	m.items[t.ID] = &cp
	return nil
}

func (m *MockRepository) List(_ context.Context, f TaskFilter) ([]Task, int, error) {
	if m.Err != nil {
		return nil, 0, m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var visible []Task
	for _, t := range m.items {
		if !t.Status.IsActive() {
			continue
		}
		if f.Status != "" && t.Status != f.Status {
			continue
		}
		if f.UserID != 0 && t.UserID != f.UserID {
			continue
		}
		visible = append(visible, *t)
	}

	total := len(visible)
	start := (f.Page - 1) * f.Limit
	if start > total {
		start = total
	}
	end := start + f.Limit
	if end > total {
		end = total
	}
	return visible[start:end], total, nil
}
