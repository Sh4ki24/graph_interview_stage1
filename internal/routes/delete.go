package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"todoapp/internal/auth"
	apperrors "todoapp/internal/errors"
)

// closeTask godoc
// @Summary      Close a task (soft delete). Admin only.
// @Tags         tasks
// @Produce      json
// @Param        Authorization header string true "Bearer <token> (admin)"
// @Security     BearerAuth
// @Param        id path int true "task id"
// @Success      200 {object} TaskResponse
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /tasks/{id} [delete]
func (s *Server) closeTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if auth.CurrentRole(c) != auth.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}
	t, err := s.tasks.Close(c.Request.Context(), id)
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrNotFound.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusOK, t)
	}
}
