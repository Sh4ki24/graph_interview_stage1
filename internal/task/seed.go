package task

import (
	"context"
	"fmt"
)

// seederTitles / seederDescriptions are combined to generate varied fake
// tasks without pulling in an external faker dependency.
var (
	seederTitles = []string{
		"Write the design doc", "Review the open PRs", "Fix the flaky test",
		"Update dependencies", "Refactor the handler", "Add integration tests",
		"Clean up the backlog", "Pair on the bug", "Ship the release", "Retrospective",
	}
	seederDescriptions = []string{
		"", "small, focused change", "needs a second pair of eyes",
		"blocked on review", "picked up from the backlog", "timeboxed to one hour",
	}
)

// Seed populates the database with n tasks belonging to userID, spread
// across the status state machine (roughly half todo, a quarter progress,
// the rest done, plus a few soft-deleted). It is used by cmd/seed
// (`make seed`) to give the API realistic data to list, filter and paginate.
func Seed(ctx context.Context, repo Repository, userID int64, n int) error {
	if n < 0 {
		return fmt.Errorf("seed: negative count %d", n)
	}
	for i := 0; i < n; i++ {
		t := &Task{
			UserID:      userID,
			Title:       fmt.Sprintf("%s #%d", seederTitles[i%len(seederTitles)], i+1),
			Description: seederDescriptions[i%len(seederDescriptions)],
			Status:      StatusTodo,
		}
		if err := repo.Create(ctx, t); err != nil {
			return fmt.Errorf("seed: create task %d: %w", i+1, err)
		}
		// walk the state machine: every 4th task ends up done, every 8th
		// is additionally soft-deleted, the rest stay todo/progress
		if err := seedAdvance(ctx, repo, t, i); err != nil {
			return err
		}
	}
	return nil
}

func seedAdvance(ctx context.Context, repo Repository, t *Task, i int) error {
	if i%4 != 3 {
		if i%2 == 1 {
			t.Status = StatusProgress
			if err := repo.Update(ctx, t); err != nil {
				return fmt.Errorf("seed: advance task %d to progress: %w", t.ID, err)
			}
		}
		return nil
	}
	t.Status = StatusProgress
	if err := repo.Update(ctx, t); err != nil {
		return fmt.Errorf("seed: advance task %d to progress: %w", t.ID, err)
	}
	t.Status = StatusDone
	if err := repo.Update(ctx, t); err != nil {
		return fmt.Errorf("seed: advance task %d to done: %w", t.ID, err)
	}
	if i%8 == 7 {
		t.Status = StatusClose
		if err := repo.Update(ctx, t); err != nil {
			return fmt.Errorf("seed: close task %d: %w", t.ID, err)
		}
	}
	return nil
}
