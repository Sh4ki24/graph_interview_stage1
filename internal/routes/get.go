package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"todoapp/internal/auth"
	apperrors "todoapp/internal/errors"
	"todoapp/internal/task"
)

// listTasks godoc
// @Summary      List tasks with pagination and optional status filter
// @Tags         tasks
// @Produce      json
// @Param        Authorization header string true "Bearer <token>"
// @Security     BearerAuth
// @Param        page query int false "1-based page" default(1)
// @Param        limit query int false "page size (max 100)" default(20)
// @Param        status query string false "todo | progress | done" default("")
// @Success      200 {object} ListResponse
// @Failure      400 {object} map[string]string
// @Router       /tasks [get]
func (s *Server) listTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	// handler-side defaults keep the response payload consistent with what
	// the service will actually apply (both come from the same config)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > s.pagination.MaxLimit {
		limit = s.pagination.DefaultLimit
	}

	tasks, total, err := s.tasks.ListForUser(c.Request.Context(), auth.CurrentUserID(c), task.TaskFilter{
		Status: task.Status(c.Query("status")),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, apperrors.ErrInvalidStatus) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ListResponse{Data: tasks, Page: page, Limit: limit, Total: total})
}

// getTask godoc
// @Summary      Get a task by id
// @Tags         tasks
// @Produce      json
// @Param        Authorization header string true "Bearer <token>"
// @Security     BearerAuth
// @Param        id path int true "task id"
// @Success      200 {object} TaskResponse
// @Failure      404 {object} map[string]string
// @Router       /tasks/{id} [get]
func (s *Server) getTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	t, err := s.tasks.Get(c.Request.Context(), id)
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrNotFound.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	case t.UserID != auth.CurrentUserID(c) && auth.CurrentRole(c) != auth.RoleAdmin:
		c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrNotFound.Error()})
	default:
		c.JSON(http.StatusOK, t)
	}
}
