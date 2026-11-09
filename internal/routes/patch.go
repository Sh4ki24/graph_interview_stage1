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

// updateTask godoc
// @Summary      Update a task (partial). Status must follow todo -> progress -> done -> close
// @Description  A task cannot jump directly from todo to done. Close is the soft delete.
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer <token>"
// @Security     BearerAuth
// @Param        id path int true "task id"
// @Param        input body UpdateTaskInput true "fields to update"
// @Success      200 {object} TaskResponse
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /tasks/{id} [patch]
func (s *Server) updateTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var in UpdateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var status *task.Status
	if in.Status != nil && *in.Status != "" {
		st := task.Status(*in.Status)
		status = &st
	}

	// ownership check: users may only update their own tasks; admins may
	// update any task.
	existing, err := s.tasks.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrNotFound.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	if existing.UserID != auth.CurrentUserID(c) && auth.CurrentRole(c) != auth.RoleAdmin {
		c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrNotFound.Error()})
		return
	}

	t, err := s.tasks.Update(c.Request.Context(), id, in.Title, in.Description, status)
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrNotFound.Error()})
	case errors.Is(err, apperrors.ErrInvalidTransition), errors.Is(err, apperrors.ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusOK, t)
	}
}
