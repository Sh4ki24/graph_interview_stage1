package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "todoapp/internal/errors"
	"todoapp/internal/events"
)

const testUserID int64 = 1

func newTestService(t *testing.T) (*Service, <-chan events.Event, *MockRepository) {
	t.Helper()
	repo := NewMockRepository()
	eventCh := make(chan events.Event, 16)
	return NewService(repo, eventCh, Options{}), eventCh, repo
}

func TestCreate(t *testing.T) {
	svc, eventCh, _ := newTestService(t)
	ctx := context.Background()

	t1, err := svc.Create(ctx, testUserID, "write tests", "unit tests")
	require.NoError(t, err)
	assert.NotZero(t, t1.ID)
	assert.Equal(t, StatusTodo, t1.Status)

	select {
	case e := <-eventCh:
		assert.Equal(t, "created", e.Action)
		assert.Equal(t, t1.ID, e.TaskID)
	case <-time.After(time.Second):
		t.Fatal("expected create event")
	}
}

func TestUpdateStatusHappyPath(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	t1, err := svc.Create(ctx, testUserID, "task", "")
	require.NoError(t, err)

	next := StatusProgress
	t1, err = svc.Update(ctx, t1.ID, nil, nil, &next)
	require.NoError(t, err)
	assert.Equal(t, StatusProgress, t1.Status)

	next = StatusDone
	t1, err = svc.Update(ctx, t1.ID, nil, nil, &next)
	require.NoError(t, err)
	assert.Equal(t, StatusDone, t1.Status)
}

func TestUpdateCannotSkipProgress(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	t1, err := svc.Create(ctx, testUserID, "task", "")
	require.NoError(t, err)

	done := StatusDone
	_, err = svc.Update(ctx, t1.ID, nil, nil, &done)
	assert.ErrorIs(t, err, apperrors.ErrInvalidTransition)
}

func TestUpdateInvalidStatusRejected(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	t1, _ := svc.Create(ctx, testUserID, "task", "")

	bad := Status("archived")
	_, err := svc.Update(ctx, t1.ID, nil, nil, &bad)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatus)
}

func TestUpdateFields(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	t1, _ := svc.Create(ctx, testUserID, "old title", "old desc")

	title := "new title"
	t1, err := svc.Update(ctx, t1.ID, &title, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "new title", t1.Title)
	assert.Equal(t, "old desc", t1.Description) // untouched
}

func TestUpdateNotFound(t *testing.T) {
	svc, _, _ := newTestService(t)
	title := "x"
	_, err := svc.Update(context.Background(), 999, &title, nil, nil)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestCloseIsSoftDelete(t *testing.T) {
	svc, eventCh, _ := newTestService(t)
	ctx := context.Background()
	t1, _ := svc.Create(ctx, testUserID, "task", "")

	closed, err := svc.Close(ctx, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusClose, closed.Status)

	// soft deleted: invisible to get and list
	_, err = svc.Get(ctx, t1.ID)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	tasks, total, err := svc.List(ctx, TaskFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, tasks)

	// drain events: "created" then "closed"
	for i := 0; i < 2; i++ {
		select {
		case e := <-eventCh:
			if i == 1 {
				assert.Equal(t, "closed", e.Action)
			}
		case <-time.After(time.Second):
			t.Fatal("expected close event")
		}
	}
}

func TestListWithStatusFilterAndPagination(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	progress := StatusProgress
	for i := 0; i < 5; i++ {
		t1, err := svc.Create(ctx, testUserID, "task", "")
		require.NoError(t, err)
		if i < 2 {
			_, err = svc.Update(ctx, t1.ID, nil, nil, &progress)
			require.NoError(t, err)
		}
	}

	// filter
	tasks, total, err := svc.List(ctx, TaskFilter{Status: StatusProgress, Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, tasks, 2)

	// pagination
	tasks, total, err = svc.List(ctx, TaskFilter{Page: 2, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, tasks, 2)

	// invalid status rejected
	_, _, err = svc.List(ctx, TaskFilter{Status: "bogus", Page: 1, Limit: 10})
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatus)
}

func TestPublishDoesNotBlockWhenChannelFull(t *testing.T) {
	// nobody drains the events channel; requests must not stall
	repo := NewMockRepository()
	eventCh := make(chan events.Event) // unbuffered, no reader
	svc := NewService(repo, eventCh, Options{})
	_, err := svc.Create(context.Background(), testUserID, "task", "")
	require.NoError(t, err)
}
