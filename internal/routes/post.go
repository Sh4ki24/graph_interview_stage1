package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"todoapp/internal/auth"
)

// createTask godoc
// @Summary      Create a task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer <token>"
// @Security     BearerAuth
// @Param        input body CreateTaskInput true "task payload"
// @Success      201 {object} TaskResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /tasks [post]
func (s *Server) createTask(c *gin.Context) {
	var in CreateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := s.tasks.Create(c.Request.Context(), auth.CurrentUserID(c), in.Title, in.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}
