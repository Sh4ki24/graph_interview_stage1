// Package routes assembles the HTTP layer: server, per-method route
// files and request/response schemas.
package routes

import (
	"github.com/gin-gonic/gin"

	"todoapp/internal/config"
	"todoapp/internal/task"
	"todoapp/internal/user"
)

// Server wires the task service and user repository into the gin routes.
type Server struct {
	tasks      *task.Service
	users      user.Repository
	pagination config.Pagination
}

func New(tasks *task.Service, users user.Repository, pagination config.Pagination) *Server {
	return &Server{tasks: tasks, users: users, pagination: pagination}
}

// Router builds the gin engine. Exposed for route unit tests.
func (s *Server) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), Metrics())

	s.registerRoutes(r)
	return r
}
