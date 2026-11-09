package events

import (
	"context"
	"log/slog"
)

// StartWorker drains the events channel in a goroutine until ctx is
// cancelled or the channel is closed.
func StartWorker(ctx context.Context, events <-chan Event, logger *slog.Logger) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-events:
				logger.Info("task event",
					"task_id", e.TaskID,
					"action", e.Action,
					"status", e.Status,
				)
			}
		}
	}()
}
