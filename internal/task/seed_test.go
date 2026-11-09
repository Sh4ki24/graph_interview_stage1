package task

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedSpreadsStatuses(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	require.NoError(t, Seed(ctx, repo, 1, 16))

	tasks, total, err := repo.List(ctx, TaskFilter{Page: 1, Limit: 100})
	require.NoError(t, err)

	// 16 tasks: 2 closed (i=7, 15) are invisible; the rest spread across
	// todo / progress / done following the state machine.
	assert.Equal(t, 14, total)
	assert.Len(t, tasks, 14)

	byStatus := map[Status]int{}
	for _, t1 := range tasks {
		byStatus[t1.Status]++
	}
	assert.Equal(t, 8, byStatus[StatusTodo])    // even i, not i%4==3
	assert.Equal(t, 4, byStatus[StatusProgress]) // odd i, not i%4==3
	assert.Equal(t, 2, byStatus[StatusDone])     // i%4==3, not i%8==7

	// every task got a unique, non-empty title
	titles := map[string]bool{}
	for _, t1 := range tasks {
		assert.NotEmpty(t, t1.Title)
		titles[t1.Title] = true
	}
	assert.Len(t, titles, 14)
}

func TestSeedZeroAndNegative(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	require.NoError(t, Seed(ctx, repo, 1, 0))
	_, total, err := repo.List(ctx, TaskFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, total)

	assert.Error(t, Seed(ctx, repo, 1, -1))
}

func TestSeedRepositoryErrorPropagates(t *testing.T) {
	repo := NewMockRepository()
	repo.Err = assert.AnError
	assert.Error(t, Seed(context.Background(), repo, 1, 5))
}
