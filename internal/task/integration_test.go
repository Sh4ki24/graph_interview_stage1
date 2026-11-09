//go:build integration

// Integration tests run against a real postgres. Skipped unless
// TEST_DATABASE_URL is set. Run via: make test-integration
package task_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "todoapp/internal/errors"
	"todoapp/internal/storage"
	"todoapp/internal/task"
)

func newRepo(t *testing.T) task.Repository {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	repo := storage.NewPostgresTaskRepository(pool)
	require.NoError(t, repo.Init(ctx))

	_, err = pool.Exec(ctx, "TRUNCATE tasks RESTART IDENTITY")
	require.NoError(t, err)
	return repo
}

func TestIntegrationTaskCRUD(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	// create
	t1 := &task.Task{UserID: 1, Title: "integration", Description: "created by test"}
	require.NoError(t, repo.Create(ctx, t1))
	require.NotZero(t, t1.ID)

	// get
	got, err := repo.GetByID(ctx, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, "integration", got.Title)
	assert.Equal(t, task.StatusTodo, got.Status)

	// update: todo -> progress
	got.Status = task.StatusProgress
	require.NoError(t, repo.Update(ctx, got))
	got, err = repo.GetByID(ctx, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, task.StatusProgress, got.Status)

	// seed more tasks to exercise list
	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, &task.Task{UserID: 1, Title: "extra"}))
	}

	// list: pagination
	tasks, total, err := repo.List(ctx, task.TaskFilter{Page: 2, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 6, total)
	assert.Len(t, tasks, 2)

	// list: status filter
	tasks, total, err = repo.List(ctx, task.TaskFilter{Status: task.StatusProgress, Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, tasks, 1)
	assert.Equal(t, task.StatusProgress, tasks[0].Status)

	// close (soft delete)
	got.Status = task.StatusClose
	require.NoError(t, repo.Update(ctx, got))

	// soft deleted: get -> not found, excluded from list
	_, err = repo.GetByID(ctx, t1.ID)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	_, total, err = repo.List(ctx, task.TaskFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
}

func TestIntegrationUpdateNotFound(t *testing.T) {
	repo := newRepo(t)
	ghost := &task.Task{ID: 424242, UserID: 1, Title: "ghost", Status: task.StatusTodo}
	err := repo.Update(context.Background(), ghost)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}
