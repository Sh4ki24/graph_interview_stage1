package user

import (
	"context"
	"errors"
	"sync"
)

// ErrDuplicate is returned when creating a user that already exists.
var ErrDuplicate = errors.New("username already taken")

// MockRepository is an in-memory user.Repository for unit tests.
type MockRepository struct {
	mu     sync.Mutex
	items  map[int64]*User
	byName map[string]*User
	nextID int64
	Err    error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		items:  make(map[int64]*User),
		byName: make(map[string]*User),
		nextID: 1,
	}
}

func (m *MockRepository) Init(_ context.Context) error { return m.Err }

func (m *MockRepository) Create(_ context.Context, u *User) error {
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byName[u.Username]; ok {
		return ErrDuplicate
	}
	u.ID = m.nextID
	m.nextID++
	cp := *u
	m.items[u.ID] = &cp
	m.byName[u.Username] = &cp
	return nil
}

func (m *MockRepository) GetByUsername(_ context.Context, username string) (*User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byName[username]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (m *MockRepository) GetByID(_ context.Context, id int64) (*User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}
