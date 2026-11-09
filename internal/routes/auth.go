package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"todoapp/internal/auth"
	"todoapp/internal/user"
)

// AuthInput is the request body for signup and login.
type AuthInput struct {
	Username string `json:"username" binding:"required,min=1,max=50" example:"alice"`
	Password string `json:"password" binding:"required,min=1,max=100" example:"secret123"`
}

// TokenResponse is returned on successful signup or login.
type TokenResponse struct {
	Token    string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	Username string `json:"username" example:"alice"`
	Role     string `json:"role" example:"user"`
	UserID   int64  `json:"user_id" example:"3"`
}

// signup godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body AuthInput true "credentials"
// @Success      201 {object} TokenResponse
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /auth/signup [post]
func (s *Server) signup(c *gin.Context) {
	var in AuthInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := user.HashPassword(in.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	u := &user.User{
		Username:     in.Username,
		PasswordHash: hash,
		Role:         user.RoleUser,
	}
	if err := s.users.Create(c.Request.Context(), u); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
		return
	}

	token, err := auth.GenerateToken(auth.JWTSecret, u.ID, u.Username, string(u.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, TokenResponse{
		Token:    token,
		Username: u.Username,
		Role:     string(u.Role),
		UserID:   u.ID,
	})
}

// login godoc
// @Summary      Log in and receive a JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body AuthInput true "credentials"
// @Success      200 {object} TokenResponse
// @Failure      401 {object} map[string]string
// @Router       /auth/login [post]
func (s *Server) login(c *gin.Context) {
	var in AuthInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := s.users.GetByUsername(c.Request.Context(), in.Username)
	if err != nil || u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !user.CheckPassword(u.PasswordHash, in.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(auth.JWTSecret, u.ID, u.Username, string(u.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		Token:    token,
		Username: u.Username,
		Role:     string(u.Role),
		UserID:   u.ID,
	})
}
