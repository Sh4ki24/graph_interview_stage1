package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"todoapp/internal/auth"
	_ "todoapp/docs" // swagger docs
)

// registerRoutes attaches every endpoint to the engine. The per-method
// handlers live in get.go / post.go / patch.go / delete.go, auth in auth.go.
func (s *Server) registerRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/signup", s.signup)
		authGroup.POST("/login", s.login)
	}

	tasks := r.Group("/tasks")
	{
		tasks.Use(auth.RequireAuth())
		tasks.POST("", s.createTask)
		tasks.GET("", s.listTasks)
		tasks.GET("/:id", s.getTask)
		tasks.PATCH("/:id", s.updateTask)
		tasks.DELETE("/:id", s.closeTask)
	}
}