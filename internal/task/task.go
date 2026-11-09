package task

import "time"

// Task is the todo list item. Each task belongs to a single user.
type Task struct {
	ID          int64     `json:"id" example:"1"`
	UserID      int64     `json:"user_id" example:"7"`
	Title       string    `json:"title" example:"Buy milk"`
	Description string    `json:"description" example:"2 liters, lactose free"`
	Status      Status    `json:"status" example:"todo"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskFilter carries listing options: status filter + pagination. When
// UserID is set, only that user's tasks are returned.
type TaskFilter struct {
	UserID int64
	Status Status
	Page   int
	Limit  int
}