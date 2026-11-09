package routes

import (
	"time"

	"todoapp/internal/task"
)

// CreateTaskInput is the request schema for POST /tasks.
type CreateTaskInput struct {
	Title       string `json:"title" binding:"required,min=1,max=200" example:"Buy milk"`
	Description string `json:"description" example:"2 liters"`
}

// UpdateTaskInput is the request schema for PATCH /tasks/{id}.
// All fields are optional (partial update).
type UpdateTaskInput struct {
	Title       *string `json:"title" binding:"omitempty,min=1,max=200" example:"Buy milk now"`
	Description *string `json:"description" example:"3 liters instead"`
	Status      *string `json:"status" binding:"omitempty,oneof=todo progress done close" example:"progress"`
}

// ListResponse is the response schema for GET /tasks.
type ListResponse struct {
	Data  []task.Task `json:"data"`
	Page  int         `json:"page" example:"1"`
	Limit int         `json:"limit" example:"20"`
	Total int         `json:"total" example:"42"`
}

// TaskResponse mirrors task.Task for swagger docs.
type TaskResponse struct {
	ID          int64     `json:"id" example:"1"`
	Title       string    `json:"title" example:"Buy milk"`
	Description string    `json:"description" example:"2 liters, lactose free"`
	Status      string    `json:"status" example:"todo"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
