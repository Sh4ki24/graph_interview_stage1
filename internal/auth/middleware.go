package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apperrors "todoapp/internal/errors"
)

// JWTSecret is set at startup and used by the middleware to validate tokens.
var JWTSecret string

// RequireAuth is a gin middleware that validates the JWT from the
// Authorization header. The claims are stored in the gin context for
// downstream handlers.
func RequireAuth(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := ParseToken(JWTSecret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		if len(allowed) > 0 {
			if _, ok := allowed[claims.Role]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": apperrors.ErrForbidden.Error()})
				return
			}
		}
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// CurrentUserID returns the authenticated user ID from context.
func CurrentUserID(c *gin.Context) int64 {
	id, _ := c.Get("userID")
	if v, ok := id.(int64); ok {
		return v
	}
	return 0
}

// CurrentRole returns the authenticated role.
func CurrentRole(c *gin.Context) string {
	r, _ := c.Get("role")
	if v, ok := r.(string); ok {
		return v
	}
	return ""
}

// CurrentUsername returns the authenticated username.
func CurrentUsername(c *gin.Context) string {
	u, _ := c.Get("username")
	if v, ok := u.(string); ok {
		return v
	}
	return ""
}

// User returns the authenticated username. Kept for backwards compat.
func User(c *gin.Context) string {
	return CurrentUsername(c)
}
